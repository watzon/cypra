import { expect } from "@playwright/test";
import { spawn, spawnSync, type ChildProcess } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { createServer, type Server as HTTPServer } from "node:http";
import { createServer as createNetServer, type Server as NetServer, type Socket } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { chromium, type Browser, type BrowserContext, type Page } from "playwright";

export interface CypraHarness {
  browser: Browser;
  context: BrowserContext;
  page: Page;
  baseURL: string;
  tenantURL: string;
  smtpURL?: string;
  setupToken?: string;
  waitForEmail?: (predicate: (message: string) => boolean, timeoutMs?: number) => Promise<string>;
  managedStack?: ManagedStack;
  close: () => Promise<void>;
}

export interface ExampleClientConfig {
  issuer: string;
  clientID: string;
  clientSecret: string;
  redirectURI: string;
}

export interface NextjsExample {
  url: string;
  client: ExampleClientConfig;
  logs: () => string;
  close: () => Promise<void>;
}

export interface RedeemedTenantUser {
  email: string;
  password: string;
  sessionID: string;
  userID: string;
}

export interface ManagedStack {
  baseURL: string;
  tenantURL: string;
  smtpURL: string;
  setupToken?: string;
  env: NodeJS.ProcessEnv;
  waitForEmail: (predicate: (message: string) => boolean, timeoutMs?: number) => Promise<string>;
  close: () => Promise<void>;
}

export interface ManagedBackup {
  path: string;
  passphrasePath: string;
  close: () => Promise<void>;
}

export async function startHarness(): Promise<CypraHarness> {
  const managedStack =
    process.env.CYPRA_E2E_MANAGED === "1" ? await startManagedStack() : undefined;
  const browser = await chromium.launch();
  const context = await browser.newContext();
  await context.addInitScript(() => {
    window.localStorage.setItem("cypra.theme", "dark");
  });
  const page = await context.newPage();
  const cdp = await context.newCDPSession(page);
  await cdp.send("WebAuthn.enable");
  await cdp.send("WebAuthn.addVirtualAuthenticator", {
    options: {
      protocol: "ctap2",
      transport: "internal",
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
  return {
    browser,
    context,
    page,
    baseURL: managedStack?.baseURL ?? process.env.CYPRA_BASE_URL ?? "https://cypra.localhost",
    tenantURL:
      managedStack?.tenantURL ?? process.env.CYPRA_TENANT_URL ?? "https://acme.cypra.localhost",
    smtpURL: managedStack?.smtpURL,
    setupToken: managedStack?.setupToken,
    waitForEmail: managedStack?.waitForEmail,
    managedStack,
    close: async () => {
      await context.close();
      await browser.close();
      await managedStack?.close();
    },
  };
}

async function startManagedStack(backup?: ManagedBackup): Promise<ManagedStack> {
  const postgresPort = await freePort();
  const cypraPort = await freePort();
  const googleStub = await startGoogleStub();
  const smtpStub = await startSMTPStub();
  const projectName = `cypra-e2e-${process.pid}-${Date.now()}`;
  const storageDir = await mkdtemp(join(tmpdir(), "cypra-e2e-storage-"));
  const databaseURL = `postgres://cypra:cypra@127.0.0.1:${postgresPort}/cypra?sslmode=disable`;
  const baseURL = `http://localhost:${cypraPort}`;
  const env = {
    ...process.env,
    POSTGRES_HOST_PORT: String(postgresPort),
    POSTGRES_PASSWORD: "cypra",
    DATABASE_URL: databaseURL,
    MIGRATE_DATABASE_URL: databaseURL,
    MASTER_KEY: "dev-only-change-me-dev-only-change-me-32b",
    LISTEN_ADDR: `:${cypraPort}`,
    PUBLIC_BASE_URL: baseURL,
    CYPRA_DEV_INSECURE_HTTP: "true",
    TRUSTED_PROXY_HEADERS: "none",
    STORAGE_BACKEND: "local-disk",
    STORAGE_LOCAL_PATH: storageDir,
    LOG_LEVEL: "debug",
    CYPRA_GOOGLE_AUTH_URL: googleStub.authURL,
    VITE_DEV_SERVER: process.env.VITE_DEV_SERVER ?? "http://localhost:4173",
  };
  const composeFiles = ["-f", "deploy/docker-compose.yml", "-f", "deploy/docker-compose.dev.yml"];

  runChecked(
    "docker",
    ["compose", "-p", projectName, ...composeFiles, "up", "-d", "--wait", "postgres"],
    env,
  );
  try {
    runChecked("go", ["run", "./cmd/cypra", "migrate"], env);
    let setupToken: string | undefined;
    if (backup) {
      runChecked(
        "go",
        ["run", "./cmd/cypra", "import", "--passphrase-file", backup.passphrasePath, backup.path],
        env,
      );
    } else {
      const tokenOutput = runChecked("go", ["run", "./cmd/cypra", "admin", "reset-bootstrap"], env);
      setupToken = tokenOutput.match(/setup_token=(\S+)/)?.[1];
      if (!setupToken) {
        throw new Error(`admin reset-bootstrap did not print setup token: ${tokenOutput}`);
      }
    }
    const server = spawn("go", ["run", "./cmd/cypra", "serve", "--skip-migrate"], {
      env,
      stdio: ["ignore", "pipe", "pipe"],
    });
    const logs: string[] = [];
    server.stdout?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));
    server.stderr?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));
    await waitForURL(`${baseURL}/readyz`, 30_000, true, server, logs);
    return {
      baseURL,
      tenantURL: `http://acme.localhost:${cypraPort}`,
      smtpURL: smtpStub.url,
      setupToken,
      env,
      waitForEmail: smtpStub.waitForMessage,
      close: async () => {
        await stopProcess(server);
        runChecked("docker", ["compose", "-p", projectName, ...composeFiles, "down", "-v"], env);
        await rm(storageDir, { recursive: true, force: true });
        await googleStub.close();
        await smtpStub.close();
      },
    };
  } catch (error) {
    runChecked("docker", ["compose", "-p", projectName, ...composeFiles, "down", "-v"], env);
    await rm(storageDir, { recursive: true, force: true });
    await googleStub.close();
    await smtpStub.close();
    throw error;
  }
}

async function startGoogleStub(): Promise<{ authURL: string; close: () => Promise<void> }> {
  const port = await freePort();
  const server = createServer((request, response) => {
    const url = new URL(request.url ?? "/", `http://${request.headers.host ?? "localhost"}`);
    if (url.pathname !== "/oauth2/v2/auth") {
      response.writeHead(404).end("not found");
      return;
    }
    response.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
    response.end(
      `<!doctype html><title>Google OAuth Stub</title><h1>Google OAuth Stub</h1><p>state=${url.searchParams.get("state") ?? ""}</p>`,
    );
  });
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, "localhost", () => resolve());
  });
  return {
    authURL: `http://127.0.0.1:${port}/oauth2/v2/auth`,
    close: () => closeHTTPServer(server),
  };
}

async function closeHTTPServer(server: HTTPServer) {
  await new Promise<void>((resolve, reject) => {
    server.close((error) => {
      if (error) reject(error);
      else resolve();
    });
  });
}

async function startSMTPStub(): Promise<{
  url: string;
  waitForMessage: (predicate: (message: string) => boolean, timeoutMs?: number) => Promise<string>;
  close: () => Promise<void>;
}> {
  const port = await freePort();
  const messages: string[] = [];
  const waiters: Array<{
    predicate: (message: string) => boolean;
    resolve: (message: string) => void;
    reject: (error: Error) => void;
    timer: ReturnType<typeof setTimeout>;
  }> = [];
  const server = createNetServer((socket) =>
    handleSMTPConnection(socket, (message) => {
      messages.push(message);
      for (const waiter of [...waiters]) {
        if (!waiter.predicate(message)) continue;
        clearTimeout(waiter.timer);
        waiters.splice(waiters.indexOf(waiter), 1);
        waiter.resolve(message);
      }
    }),
  );
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, "127.0.0.1", () => resolve());
  });
  return {
    url: `smtp://cypra:cypra@localhost:${port}`,
    waitForMessage: (predicate, timeoutMs = 15_000) => {
      const existing = messages.find(predicate);
      if (existing) return Promise.resolve(existing);
      return new Promise<string>((resolve, reject) => {
        const waiter = {
          predicate,
          resolve,
          reject,
          timer: setTimeout(() => {
            waiters.splice(waiters.indexOf(waiter), 1);
            reject(new Error(`timed out waiting for SMTP message; saw ${messages.length}`));
          }, timeoutMs),
        };
        waiters.push(waiter);
      });
    },
    close: () => closeNetServer(server),
  };
}

function handleSMTPConnection(socket: Socket, onMessage: (message: string) => void) {
  socket.setEncoding("utf8");
  let buffer = "";
  let dataMode = false;
  const dataLines: string[] = [];
  const write = (line: string) => socket.write(`${line}\r\n`);
  write("220 cypra-e2e-smtp ESMTP");
  socket.on("data", (chunk) => {
    buffer += chunk;
    for (;;) {
      const newline = buffer.indexOf("\n");
      if (newline < 0) break;
      const raw = buffer.slice(0, newline).replace(/\r$/, "");
      buffer = buffer.slice(newline + 1);
      if (dataMode) {
        if (raw === ".") {
          dataMode = false;
          onMessage(dataLines.join("\n"));
          dataLines.length = 0;
          write("250 queued");
        } else {
          dataLines.push(raw.startsWith("..") ? raw.slice(1) : raw);
        }
        continue;
      }
      const command = raw.split(" ")[0]?.toUpperCase();
      switch (command) {
        case "EHLO":
        case "HELO":
          socket.write("250-cypra-e2e-smtp\r\n250 AUTH PLAIN\r\n");
          break;
        case "AUTH":
          write("235 authenticated");
          break;
        case "MAIL":
        case "RCPT":
        case "RSET":
        case "NOOP":
          write("250 ok");
          break;
        case "DATA":
          dataMode = true;
          write("354 end with <CR><LF>.<CR><LF>");
          break;
        case "QUIT":
          write("221 bye");
          socket.end();
          break;
        default:
          write("250 ok");
      }
    }
  });
}

async function closeNetServer(server: NetServer) {
  await new Promise<void>((resolve, reject) => {
    server.close((error) => {
      if (error) reject(error);
      else resolve();
    });
  });
}

function runChecked(command: string, args: string[], env: NodeJS.ProcessEnv, cwd?: string) {
  const result = spawnSync(command, args, { cwd, encoding: "utf8", env });
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed\n${result.stdout}\n${result.stderr}`);
  }
  return result.stdout;
}

export async function exportManagedBackup(harness: CypraHarness): Promise<ManagedBackup> {
  if (!harness.managedStack) throw new Error("managed backup export requires CYPRA_E2E_MANAGED=1");
  const dir = await mkdtemp(join(tmpdir(), "cypra-e2e-backup-"));
  const passphrasePath = join(dir, "passphrase.txt");
  const path = join(dir, "backup.json");
  await writeFile(passphrasePath, "secret", { mode: 0o600 });
  runChecked(
    "go",
    ["run", "./cmd/cypra", "export", "--out", path, "--passphrase-file", passphrasePath],
    harness.managedStack.env,
  );
  return {
    path,
    passphrasePath,
    close: () => rm(dir, { recursive: true, force: true }),
  };
}

export function startRestoredManagedStack(backup: ManagedBackup): Promise<ManagedStack> {
  return startManagedStack(backup);
}

async function waitForURL(
  url: string,
  timeoutMs: number,
  requireOK: boolean,
  processToWatch?: ChildProcess,
  logs: string[] = [],
) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (processToWatch?.exitCode !== null) {
      throw new Error(`process exited while waiting for ${url}\n${logs.join("")}`);
    }
    try {
      const response = await fetch(url);
      if (!requireOK || response.ok) return;
    } catch {
      // Retry until the deadline.
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`timed out waiting for ${url}\n${logs.join("")}`);
}

async function freePort() {
  const { createServer } = await import("node:net");
  return await new Promise<number>((resolve, reject) => {
    const server = createServer();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close(() => reject(new Error("could not allocate port")));
        return;
      }
      const port = address.port;
      server.close(() => resolve(port));
    });
  });
}

async function stopProcess(child: ChildProcess) {
  if (child.exitCode !== null) return;
  child.kill("SIGTERM");
  await new Promise<void>((resolve) => {
    const timeout = setTimeout(() => {
      child.kill("SIGKILL");
      resolve();
    }, 5_000);
    child.once("exit", () => {
      clearTimeout(timeout);
      resolve();
    });
  });
}

export async function redeemBootstrapToken(harness: CypraHarness, token: string) {
  await harness.page.goto(`${harness.baseURL}/setup/${token}`);
  await expect(harness.page.getByRole("heading", { name: "Set up Cypra" })).toBeVisible();
  await harness.page
    .getByLabel("Admin email")
    .fill(process.env.CYPRA_ADMIN_EMAIL ?? "root@example.com");
  await harness.page.getByLabel("Display name").fill(process.env.CYPRA_ADMIN_NAME ?? "Root Admin");
  await harness.page.getByRole("button", { name: "Continue" }).click();
  await harness.page.getByRole("button", { name: "Enroll passkey" }).click();
  const savedButton = harness.page.getByRole("button", { name: "I have saved these" });
  try {
    await expect(savedButton).toBeVisible();
  } catch (error) {
    throw new Error(
      `setup passkey enrollment did not reach backup codes\n${await harness.page.locator("body").innerText()}\n${String(error)}`,
    );
  }
  await savedButton.click();
  await harness.page.getByRole("button", { name: "Go to dashboard" }).click();
  await expect(harness.page.getByText("Next step: create your first tenant")).toBeVisible();
  await harness.page.getByRole("button", { name: "Open dashboard" }).click();
  await expect(harness.page).toHaveURL(/\/dashboard$/);
  await mirrorSessionCookieToTenantHost(harness);
}

async function mirrorSessionCookieToTenantHost(harness: CypraHarness) {
  const tenantURL = new URL(harness.tenantURL);
  const cookies = await harness.context.cookies(harness.baseURL);
  const session = cookies.find((cookie) => cookie.name === "cypra_session");
  if (!session || tenantURL.hostname === new URL(harness.baseURL).hostname) return;
  await harness.context.addCookies([
    {
      name: session.name,
      value: session.value,
      domain: tenantURL.hostname,
      path: "/",
      httpOnly: true,
      sameSite: "Lax",
      expires: Math.floor(Date.now() / 1000) + 24 * 60 * 60,
      secure: tenantURL.protocol === "https:",
    },
  ]);
}

export async function createTenant(harness: CypraHarness, slug: string) {
  await harness.page.goto(`${harness.baseURL}/dashboard/tenants`);
  await expect(harness.page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();
  await harness.page.getByRole("button", { name: "Create tenant" }).first().click();
  const dialog = harness.page.getByRole("dialog", { name: "Create tenant" });
  await dialog.getByLabel("Display name").fill(process.env.CYPRA_TENANT_NAME ?? "Acme Operations");
  await dialog.getByLabel("Slug").fill(slug);
  await dialog.getByRole("button", { name: "Create tenant" }).click();
  await expect(harness.page).toHaveURL(new RegExp(`/dashboard/tenants/${slug}$`));
  return { slug };
}

export async function createProject(harness: CypraHarness, slug: string) {
  await harness.page.goto(`${harness.tenantURL}/dashboard/tenants/acme/projects`);
  try {
    await expect(
      harness.page.getByRole("heading", { name: "Projects", exact: true }),
    ).toBeVisible();
  } catch (error) {
    throw new Error(
      `project list did not render\n${await harness.page.locator("body").innerText()}\n${String(error)}`,
    );
  }
  await harness.page.getByRole("button", { name: "Create project" }).first().click();
  const dialog = harness.page.getByRole("dialog", { name: "Create project" });
  await dialog.getByLabel("Project name").fill(process.env.CYPRA_PROJECT_NAME ?? "Console App");
  await dialog.getByLabel("Slug").fill(slug);
  await dialog.getByRole("button", { name: "Create project" }).click();
  await expect(harness.page).toHaveURL(new RegExp(`/dashboard/tenants/acme/projects/${slug}$`));
  return { slug };
}

export async function startNextjsExample(harness: CypraHarness): Promise<NextjsExample> {
  const port = await freePort();
  const url = `http://localhost:${port}`;
  const redirectURI = `${url}/api/auth/callback/cypra`;
  const client = await configureDownstreamProject(harness, redirectURI);
  const distDir = `.next-${port}`;
  const env = {
    ...process.env,
    AUTH_SECRET: process.env.AUTH_SECRET ?? "dev-secret-change-me-dev-secret-change-me",
    AUTH_URL: url,
    AUTH_TRUST_HOST: "true",
    CYPRA_ISSUER: client.issuer,
    CYPRA_CLIENT_ID: client.clientID,
    CYPRA_CLIENT_SECRET: client.clientSecret,
    NEXT_DIST_DIR: distDir,
  };
  const cwd = join(process.cwd(), "examples", "nextjs");
  const tsconfigPath = join(cwd, "tsconfig.json");
  const originalTSConfig = await readFile(tsconfigPath, "utf8");
  const server = spawn("bun", ["run", "dev", "--", "-H", "127.0.0.1", "-p", String(port)], {
    cwd,
    env,
    stdio: ["ignore", "pipe", "pipe"],
  });
  const logs: string[] = [];
  server.stdout?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));
  server.stderr?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));
  await waitForURL(url, 60_000, false, server, logs);
  return {
    url,
    client,
    logs: () => logs.join(""),
    close: async () => {
      await stopProcess(server);
      await writeFile(tsconfigPath, originalTSConfig);
      await rm(join(cwd, distDir), { recursive: true, force: true });
    },
  };
}

async function configureDownstreamProject(
  harness: CypraHarness,
  redirectURI: string,
): Promise<ExampleClientConfig> {
  const projects = await apiJSON<
    Array<{
      id: string;
      slug: string;
      name: string;
      client_id?: string;
      issuer_url?: string;
      allowed_scopes?: string[];
      token_endpoint_auth_method?: "client_secret_basic" | "client_secret_post" | "none";
    }>
  >(harness, `${harness.tenantURL}/api/v1/projects/`, {
    headers: { "X-Cypra-Tenant-Role": "admin" },
  });
  const project = projects.find((item) => item.slug === "console") ?? projects[0];
  if (!project) throw new Error("no project exists for downstream example configuration");
  await apiJSON(harness, `${harness.tenantURL}/api/v1/projects/${project.id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", "X-Cypra-Tenant-Role": "admin" },
    body: JSON.stringify({
      name: project.name,
      redirect_uris: [redirectURI],
      allowed_scopes: project.allowed_scopes ?? ["openid", "email", "profile"],
      token_endpoint_auth_method: project.token_endpoint_auth_method ?? "client_secret_basic",
    }),
  });
  const rotated = await apiJSON<{ client_secret: string }>(
    harness,
    `${harness.tenantURL}/api/v1/projects/${project.id}/rotate-secret`,
    { method: "POST", headers: { "X-Cypra-Tenant-Role": "admin" } },
  );
  const discovery = await apiJSON<{ issuer: string }>(
    harness,
    `${harness.tenantURL}/.well-known/openid-configuration`,
  );
  return {
    issuer: discovery.issuer,
    clientID: project.client_id ?? "client_acme_console",
    clientSecret: rotated.client_secret,
    redirectURI,
  };
}

export async function redeemTenantInviteForSession(
  harness: CypraHarness,
): Promise<RedeemedTenantUser> {
  const email = process.env.CYPRA_E2E_USER_EMAIL ?? "demo-user@example.com";
  const password = "correct horse battery staple";
  const invite = await apiJSON<{ token: string }>(
    harness,
    `${harness.tenantURL}/api/v1/admin/invite`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Cypra-Tenant-Role": "admin" },
      body: JSON.stringify({ email, role: "member" }),
    },
  );
  if (harness.waitForEmail) {
    await harness.waitForEmail(
      (message) => message.includes(email) && message.toLowerCase().includes("subject:"),
    );
  }
  await harness.page.goto(`${harness.tenantURL}/login`);
  const redeemed = await harness.page.evaluate(
    async ({ token, email, password }) => {
      const response = await fetch("/api/v1/auth/invite/redeem", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          token,
          display_name: "Demo User",
          password,
        }),
      });
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as {
        invite?: { UserID?: string; user_id?: string };
        session_id?: string;
      };
      return {
        email,
        password,
        sessionID: body.session_id,
        userID: body.invite?.UserID ?? body.invite?.user_id,
      };
    },
    { token: invite.token, email, password },
  );
  if (!redeemed.sessionID) throw new Error("invite redemption did not create a user session");
  if (!redeemed.userID) throw new Error("invite redemption did not return a user id");
  return redeemed;
}

export async function startOIDCAuthorize(harness: CypraHarness, client: ExampleClientConfig) {
  const params = new URLSearchParams({
    response_type: "code",
    client_id: client.clientID,
    redirect_uri: client.redirectURI,
    scope: "openid email profile",
    state: `state-${Date.now()}`,
    code_challenge: "phase16-e2e-code-challenge",
    code_challenge_method: "S256",
  });
  await harness.page.goto(`${harness.tenantURL}/oidc/authorize?${params.toString()}`);
  await expect(harness.page.getByRole("heading", { name: "Sign in" })).toBeVisible();
}

export async function expectOIDCConsent(harness: CypraHarness) {
  await expect(
    harness.page.getByRole("heading", { name: "Sign in to this application" }),
  ).toBeVisible({ timeout: 15_000 });
}

export async function reachOIDCConsentViaPassword(
  harness: CypraHarness,
  client: ExampleClientConfig,
  user: RedeemedTenantUser,
) {
  await harness.context.clearCookies();
  await startOIDCAuthorize(harness, client);
  await harness.page.getByLabel("Email").fill(user.email);
  await harness.page.getByLabel("Password").fill(user.password);
  await harness.page.getByRole("button", { name: "Continue" }).click();
  await expectOIDCConsent(harness);
}

export async function reachOIDCConsentViaMagicLink(
  harness: CypraHarness,
  client: ExampleClientConfig,
  user: RedeemedTenantUser,
) {
  await harness.context.clearCookies();
  await startOIDCAuthorize(harness, client);
  const continuation = new URL(harness.page.url()).searchParams.get("continue");
  if (!continuation)
    throw new Error(`OIDC login did not include continuation: ${harness.page.url()}`);
  await harness.page.getByPlaceholder("you@example.com").fill(user.email);
  await harness.page.getByRole("button", { name: "Send magic link" }).click();
  const message = await harness.waitForEmail?.(
    (candidate) =>
      candidate.includes(user.email) &&
      candidate.includes("Your Cypra sign-in link") &&
      candidate.includes("Token:"),
  );
  if (!message) throw new Error("managed SMTP stub did not capture magic link email");
  const token = message.match(/Token:\s*([A-Za-z0-9_-]+)/)?.[1];
  if (!token) throw new Error(`magic link email did not include token: ${message}`);
  const result = await harness.page.evaluate(async (token) => {
    const response = await fetch("/api/v1/auth/magic-link/verify", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    });
    const body = await response.text();
    if (!response.ok) throw new Error(`${response.status} ${body}`);
    return JSON.parse(body) as { session_id?: string };
  }, token);
  if (!result.session_id) throw new Error("magic link verification did not establish session");
  await harness.page.goto(
    `${harness.tenantURL}/oidc/authorize?continue=${encodeURIComponent(continuation)}`,
  );
  await expectOIDCConsent(harness);
}

export async function reachOIDCConsentViaPasskey(
  harness: CypraHarness,
  client: ExampleClientConfig,
) {
  await harness.context.clearCookies();
  await startOIDCAuthorize(harness, client);
  await harness.page.getByRole("button", { name: "Use passkey" }).click();
  await expectOIDCConsent(harness);
}

export async function reachOIDCConsentViaGoogle(
  harness: CypraHarness,
  client: ExampleClientConfig,
) {
  await harness.context.clearCookies();
  await startOIDCAuthorize(harness, client);
  const continuation = new URL(harness.page.url()).searchParams.get("continue");
  if (!continuation)
    throw new Error(`OIDC login did not include continuation: ${harness.page.url()}`);
  await signInViaGoogleUpstream(harness);
  await harness.page.goto(
    `${harness.tenantURL}/oidc/authorize?continue=${encodeURIComponent(continuation)}`,
  );
  await expectOIDCConsent(harness);
}

export async function enrollTenantPasskey(harness: CypraHarness, userID: string) {
  await harness.page.goto(`${harness.tenantURL}/login`);
  const message = await harness.page.evaluate(async (userID) => {
    const target = document.createElement("button");
    target.dataset.userId = userID;
    const passkey = (
      window as unknown as {
        cypraPasskey?: (action: string, target: HTMLElement) => Promise<string>;
      }
    ).cypraPasskey;
    if (!passkey) throw new Error("passkey helper was not loaded");
    return passkey("create", target);
  }, userID);
  expect(message).toBe("Passkey added.");
}

export async function enrollAndVerifyWebAuthn2FA(harness: CypraHarness, userID: string) {
  await harness.page.goto(`${harness.tenantURL}/2fa`);
  const message = await harness.page.evaluate(async (userID) => {
    const tools = window as unknown as {
      publicKeyFromEnvelope?: (envelope: unknown) => PublicKeyCredentialCreationOptions;
      credentialToJSON?: (credential: PublicKeyCredential) => unknown;
      cypraPasskey?: (action: string, target: HTMLElement) => Promise<string>;
    };
    if (!tools.publicKeyFromEnvelope || !tools.credentialToJSON || !tools.cypraPasskey) {
      throw new Error("passkey helper was not loaded");
    }
    const beginResponse = await fetch("/api/v1/auth/webauthn2fa/enroll", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ user_id: userID }),
    });
    const begin = await beginResponse.json();
    if (!beginResponse.ok)
      throw new Error(begin.detail || begin.error || "2fa enroll begin failed");
    const credential = (await navigator.credentials.create({
      publicKey: tools.publicKeyFromEnvelope(begin),
    })) as PublicKeyCredential | null;
    if (!credential) throw new Error("2fa credential creation failed");
    const finishResponse = await fetch("/api/v1/auth/webauthn2fa/enroll", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        user_id: userID,
        ceremony_id: begin.ceremony_id,
        response: tools.credentialToJSON(credential),
      }),
    });
    const finish = await finishResponse.json().catch(() => ({}));
    if (!finishResponse.ok) throw new Error(finish.detail || finish.error || "2fa enroll failed");
    const target = document.createElement("button");
    target.dataset.userId = userID;
    return tools.cypraPasskey("2fa", target);
  }, userID);
  expect(message).toBe("Factor accepted.");
}

async function apiJSON<T = unknown>(harness: CypraHarness, url: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers);
  const cookies = await harness.context.cookies(url);
  if (cookies.length > 0) {
    headers.set("Cookie", cookies.map((cookie) => `${cookie.name}=${cookie.value}`).join("; "));
  }
  const response = await fetch(url, { ...init, headers });
  if (!response.ok) {
    throw new Error(
      `${init.method ?? "GET"} ${url} failed: ${response.status} ${await response.text()}`,
    );
  }
  return (await response.json()) as T;
}

export async function configureEmailProvider(harness: CypraHarness) {
  await harness.page.goto(`${harness.tenantURL}/dashboard/tenants/acme/settings/email`);
  try {
    await expect(harness.page.getByRole("heading", { name: "Email provider" })).toBeVisible();
  } catch (error) {
    throw new Error(
      `email provider screen did not render\n${await harness.page.locator("body").innerText()}\n${String(error)}`,
    );
  }
  await harness.page
    .getByLabel("Email provider kind")
    .selectOption(harness.smtpURL ? "smtp" : "terminal");
  await harness.page
    .getByLabel("From address")
    .fill(process.env.CYPRA_FROM_ADDRESS ?? "auth@example.com");
  await harness.page.getByLabel("From name").fill(process.env.CYPRA_FROM_NAME ?? "Cypra Auth");
  if (harness.smtpURL) {
    await harness.page.getByLabel("SMTP URL").fill(harness.smtpURL);
  }
  await harness.page.getByRole("button", { name: "Save" }).click();
  await expect(harness.page.getByText("1 unsaved changes")).toBeHidden();
}

export async function configureUpstream(harness: CypraHarness) {
  await harness.page.goto(`${harness.tenantURL}/dashboard/tenants/acme/settings/upstream`);
  await expect(harness.page.getByRole("heading", { name: "Google upstream" })).toBeVisible();
  await harness.page
    .getByLabel("Google client ID")
    .fill(process.env.CYPRA_GOOGLE_CLIENT_ID ?? "stub.apps.googleusercontent.com");
  await harness.page
    .getByLabel("Google client secret")
    .fill(process.env.CYPRA_GOOGLE_CLIENT_SECRET ?? "stub-secret");
  const enabled = harness.page.getByRole("switch", { name: "Google enabled" });
  if (!(await enabled.isChecked())) {
    await enabled.click();
  }
  await harness.page.getByRole("button", { name: "Save" }).click();
  await expect(harness.page.getByText("1 unsaved changes")).toBeHidden();
}

export async function signInViaPasskey(harness: CypraHarness) {
  await harness.context.clearCookies();
  await harness.page.goto(`${harness.tenantURL}/login`);
  await expect(harness.page.getByRole("heading", { name: "Sign in" })).toBeVisible();
  await harness.page.getByRole("button", { name: "Use passkey" }).click();
  await expect(harness.page.locator("#login-result")).toContainText(
    "Signed in. Continue to your application.",
  );
  const cookies = await harness.context.cookies(harness.tenantURL);
  expect(cookies.some((cookie) => cookie.name === "cypra_session" && cookie.value !== "")).toBe(
    true,
  );
}

export async function signInViaGoogleUpstream(harness: CypraHarness) {
  await harness.page.goto(`${harness.tenantURL}/login`);
  await expect(harness.page.getByRole("heading", { name: "Sign in" })).toBeVisible();
  const googleLink = harness.page.getByRole("link", { name: "Sign in with Google" });
  const href = await googleLink.getAttribute("href");
  if (!href) throw new Error("Google upstream link did not include an href");
  if (process.env.CYPRA_E2E_MANAGED !== "1") {
    await googleLink.click();
    return;
  }

  const authURL = new URL(href);
  const state = authURL.searchParams.get("state");
  const nonce = authURL.searchParams.get("nonce");
  if (!state || !nonce)
    throw new Error(`stub Google URL missing state or nonce: ${harness.page.url()}`);

  await harness.page.goto(`${harness.tenantURL}/login`);
  const result = await harness.page.evaluate(
    async ({ state, nonce }) => {
      const payload = {
        sub: "stub-google-user",
        email: "google-user@example.com",
        name: "Google User",
        nonce,
      };
      const encodedPayload = btoa(JSON.stringify(payload))
        .replaceAll("+", "-")
        .replaceAll("/", "_")
        .replaceAll("=", "");
      const response = await fetch("/api/v1/auth/google/callback", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ state, id_token: `header.${encodedPayload}.signature` }),
      });
      const body = await response.text();
      if (!response.ok) throw new Error(`${response.status} ${body}`);
      return JSON.parse(body) as { session_id?: string; user_id?: string };
    },
    { state, nonce },
  );
  if (!result.session_id || !result.user_id) {
    throw new Error(`Google callback did not establish a session: ${JSON.stringify(result)}`);
  }
}
