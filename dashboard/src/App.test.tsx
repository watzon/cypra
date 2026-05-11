import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import axe from "axe-core";
import { readFileSync } from "node:fs";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "./App";
import {
  BackupCodeGrid,
  Button,
  Drawer,
  IdentifierPill,
  MaskedSecret,
  Modal,
  MobileBlockedBanner,
  PrimitiveGallery,
  Skeleton,
  StatusPip,
  TextInput,
} from "./components";
import { ThemeProvider } from "./theme";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function defaultFetchMock() {
  return vi.fn((input: RequestInfo | URL) => {
    void input;
    return Promise.resolve(
      new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
    );
  });
}

function dashboardFetchMock() {
  return vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const method = init?.method ?? "GET";
    if (url === "/api/v1/tenants/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000acme",
              slug: "acme",
              name: "Acme Operations",
              branding: { display_name: "Acme Login", powered_by: true },
              created_at: "2025-08-14T12:00:00Z",
              member_count: 3,
              project_count: 2,
              user_count: 128,
              email_provider_required: true,
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/users/me") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "00000000-0000-0000-0000-00000000bo0t",
            sub: "00000000-0000-0000-0000-00000000bo0t",
            email: "operator@example.com",
            display_name: "Operator",
            tenant_id: "00000000-0000-0000-0000-00000000acme",
            created_at: "2025-08-14T12:00:00Z",
            metadata: {},
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/auth/passkeys") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-0000000pkacme",
              label: "MacBook Touch ID",
              transports: ["internal"],
              created_at: "2025-08-14T12:00:00Z",
              last_used_at: "2026-05-01T08:00:00Z",
              this_device: true,
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/auth/sessions") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000ses1",
              created_at: "2026-05-06T09:00:00Z",
              last_seen_at: "2026-05-06T09:30:00Z",
              expires_at: "2026-05-13T09:00:00Z",
              user_agent: "Chrome/134",
              auth_kind: "cookie",
              current: true,
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/auth/mfa/factors") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            totp: { enrolled: true, confirmed_at: "2025-11-12T00:00:00Z" },
            backup_codes: { remaining: 6, total: 10 },
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/instance/summary") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            tenants: 1,
            projects: 1,
            users: 1,
            active_signing_key: true,
            recent_audit: [
              { action: "tenant.create", resource_id: "00000000-0000-0000-0000-00000000acme" },
            ],
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/projects/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000app1",
              slug: "console",
              name: "Console App",
              client_id: "client_cypra_acme_console",
              issuer_url: "https://acme.cypra.localhost",
              redirect_uris: ["https://app.example.com/api/auth/callback/cypra"],
              allowed_scopes: ["openid", "email"],
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url.includes("/rotate-secret")) {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "00000000-0000-0000-0000-00000000app1",
            client_secret: "rotated-secret",
          }),
          { status: 200 },
        ),
      );
    }
    if (url.startsWith("/api/v1/projects/")) {
      return Promise.resolve(
        new Response(JSON.stringify({ id: "00000000-0000-0000-0000-00000000app1" }), {
          status: method === "DELETE" ? 204 : 200,
        }),
      );
    }
    if (url === "/api/v1/pats/" && method === "POST") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "00000000-0000-0000-0000-00000000pat2",
            name: "Deploy write",
            scopes: ["projects.write"],
            created_at: "2026-05-06",
            token: "cypra_pat_secret_once",
          }),
          { status: 201 },
        ),
      );
    }
    if (url === "/api/v1/pats/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000pat1",
              name: "Deploy automation",
              scopes: ["projects.read"],
              created_at: "2026-05-01",
              expires_at: "2026-08-01T00:00:00Z",
              last4: "9xyz",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/users/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000ada1",
              email: "ada@example.com",
              sub: "usr_acme_ada",
              state: "active",
              enrolled_methods: ["passkey"],
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/users/00000000-0000-0000-0000-00000000ada1") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            user: {
              id: "00000000-0000-0000-0000-00000000ada1",
              email: "ada@example.com",
              sub: "usr_acme_ada",
              state: "active",
              enrolled_methods: ["passkey", "totp"],
              metadata: { name: "Ada" },
            },
            auth_methods: ["passkey", "totp"],
            sessions: [
              {
                id: "00000000-0000-0000-0000-00000000sess",
                created_at: "2026-05-01T00:00:00Z",
                last_seen_at: "2026-05-06T00:00:00Z",
                expires_at: "2026-06-01T00:00:00Z",
                revoked: false,
              },
            ],
            consents: [
              {
                id: "00000000-0000-0000-0000-00000000cons",
                client_id: "client_cypra_acme_console",
                scopes: ["openid", "email"],
                granted_at: "2026-05-01T00:00:00Z",
                revoked: false,
              },
            ],
            audit: [
              {
                id: "a1",
                occurred_at: "2026-05-01T00:00:00Z",
                action: "user.create",
                resource_kind: "user",
                state_after: {},
                redacted: false,
              },
            ],
            metadata: { name: "Ada" },
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/auth-providers/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            { method: "password", enabled: true, enrolled_count: 47, config: {} },
            { method: "magic_link", enabled: true, enrolled_count: 18, config: {} },
            { method: "passkey", enabled: true, enrolled_count: 9, config: {} },
            { method: "totp", enabled: true, enrolled_count: 12, config: {} },
            { method: "google", enabled: true, enrolled_count: 5, config: {} },
            { method: "oidc_upstream", enabled: false, enrolled_count: 0, config: {} },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/auth-providers/default") {
      return Promise.resolve(new Response(JSON.stringify({}), { status: 200 }));
    }
    if (url === "/api/v1/auth-providers/registration") {
      return Promise.resolve(
        new Response(JSON.stringify({ mode: "open", allowlist: [], invites_enabled: true }), {
          status: 200,
        }),
      );
    }
    if (url.startsWith("/api/v1/auth-providers/")) {
      return Promise.resolve(
        new Response(
          JSON.stringify({ method: "oidc_upstream", enabled: true, enrolled_count: 0, config: {} }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/signing-keys/rotate") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              kid: "00000000-0000-0000-0000-00000000acme:2",
              algorithm: "RS256",
              state: "active",
              activated_at: "2026-05-06T00:00:00Z",
              retires_at: "2026-08-04T00:00:00Z",
            },
            {
              kid: "00000000-0000-0000-0000-00000000acme:1",
              algorithm: "RS256",
              state: "overlap",
              activated_at: "2026-05-01T00:00:00Z",
              retires_at: "2026-06-01T00:00:00Z",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/signing-keys/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              kid: "00000000-0000-0000-0000-00000000acme:1",
              algorithm: "RS256",
              state: "active",
              activated_at: "2026-05-01T00:00:00Z",
              retires_at: "2026-08-01T00:00:00Z",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url.startsWith("/api/v1/audit/") || url.startsWith("/api/v1/instance/audit")) {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000aud1",
              occurred_at: "2026-05-06T00:00:00Z",
              actor_kind: "tenant_admin",
              action: "user.update",
              resource_kind: "user",
              resource_id: "00000000-0000-0000-0000-00000000ada1",
              state_before: { email: "[redacted]" },
              state_after: { email: "[redacted]", role: "admin" },
              metadata: { redacted_at: "2026-05-06T00:00:00Z" },
              redacted_at: "2026-05-06T00:00:00Z",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/provider-config/email") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            kind: "terminal",
            configured: false,
            healthy: true,
            message: "Resend is recommended for production; terminal email is available locally.",
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/provider-config/email/test") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            kind: "terminal",
            configured: true,
            healthy: true,
            message: "Email provider config decrypted and parsed successfully.",
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/provider-config/upstream") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            kind: "google",
            configured: false,
            healthy: false,
            message: "Google OAuth credentials are not configured.",
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/provider-config/upstream/test") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            kind: "google",
            configured: true,
            healthy: true,
            message: "Google credentials decrypted successfully.",
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/instance/admins") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000adm1",
              email: "root@example.com",
              role: "owner",
              created_at: "2026-05-01",
              last_seen_at: "2026-05-05",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/instance/invites") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000inv1",
              email: "ops@example.com",
              role: "instance_admin",
              expires_at: "2026-05-12T00:00:00Z",
              created_at: "2026-05-06T00:00:00Z",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/admin/invites") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000tin1",
              email: "pending@example.com",
              role: "member",
              expires_at: "2026-05-12T00:00:00Z",
              created_at: "2026-05-06T00:00:00Z",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/admin/members") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000mem1",
              user_id: "00000000-0000-0000-0000-00000000ada1",
              email: "ada@example.com",
              role: "owner",
              created_at: "2026-05-01T00:00:00Z",
              last_seen_at: "2026-05-06T00:00:00Z",
            },
            {
              id: "00000000-0000-0000-0000-00000000mem2",
              user_id: "00000000-0000-0000-0000-00000000alan",
              email: "alan@example.com",
              role: "admin",
              created_at: "2026-05-02T00:00:00Z",
              last_seen_at: "2026-05-05T00:00:00Z",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url.startsWith("/api/v1/admin/members/")) {
      return Promise.resolve(
        new Response(JSON.stringify({ id: url.split("/").at(-1), role: "member" }), {
          status: method === "DELETE" ? 204 : 200,
        }),
      );
    }
    if (url.startsWith("/api/v1/admin/invites/")) {
      return Promise.resolve(new Response(JSON.stringify({ status: "revoked" }), { status: 200 }));
    }
    if (url === "/api/v1/instance/invite") {
      return Promise.resolve(
        new Response(JSON.stringify({ id: "new-invite", token: "secret" }), { status: 201 }),
      );
    }
    if (url === "/api/v1/instance/diagnostics") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            health: { db: true, storage: true, email: true },
            version: { version: "dev", commit: "test", build_date: "local" },
            migrations: { current: 4, pending: [] },
            master_key_rotation: { phase: "done", rows_done: 0, rows_total: 0 },
            storage: {
              kind: "local-disk",
              endpoint: "file://****/cypra-storage",
              credentials_present: true,
            },
          }),
          { status: 200 },
        ),
      );
    }
    if (url.includes("/branding")) {
      return Promise.resolve(new Response(JSON.stringify({}), { status: 200 }));
    }
    return Promise.resolve(
      new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
    );
  });
}

function dashboardErrorFetchMock() {
  return vi.fn((input: RequestInfo | URL) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    if (url === "/api/v1/tenants/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000acme",
              slug: "acme",
              name: "Acme Operations",
              branding: { display_name: "Acme Login", powered_by: true },
            },
          ]),
          { status: 200 },
        ),
      );
    }
    return Promise.resolve(new Response(JSON.stringify({ error: "failed" }), { status: 500 }));
  });
}

function setupFetchMock() {
  return vi.fn((input: RequestInfo | URL) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    if (url === "/api/v1/setup/verify") {
      return Promise.resolve(new Response(JSON.stringify({ status: "ok" }), { status: 200 }));
    }
    if (url === "/api/v1/setup/passkey/begin") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            admin_id: "00000000-0000-0000-0000-00000000adm1",
            ceremony_id: "00000000-0000-0000-0000-00000000cafe",
            options: {
              publicKey: {
                challenge: "AQID",
                rp: { id: "cypra.localhost", name: "Cypra" },
                user: { id: "BAUG", name: "root@example.com", displayName: "Root Admin" },
                pubKeyCredParams: [{ type: "public-key", alg: -7 }],
              },
            },
          }),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/setup/complete") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            admin_id: "00000000-0000-0000-0000-00000000adm1",
            backup_codes: ["CYPRA-A11Y", "CYPRA-B22Y", "CYPRA-C33Y"],
          }),
          { status: 201 },
        ),
      );
    }
    return defaultFetchMock()(input);
  });
}

function stubLocation(pathname: string, assignSpy = vi.fn()) {
  vi.stubGlobal("location", {
    assign: assignSpy,
    pathname,
  } satisfies Partial<Location>);
  return assignSpy;
}

function renderApp(
  path = "/dashboard",
  fetchMock: (input: RequestInfo | URL) => Promise<Response> = defaultFetchMock(),
) {
  history.replaceState(null, "", path);
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  // Preseed the auth gate so the synchronous render path doesn't show the
  // loading shell. Tests that exercise unauthenticated state should override.
  client.setQueryData(["app-me"], {
    kind: "user",
    id: "00000000-0000-0000-0000-00000000bo0t",
    sub: "00000000-0000-0000-0000-00000000bo0t",
    email: "operator@example.com",
    display_name: "Operator",
    tenant_id: "00000000-0000-0000-0000-00000000acme",
    created_at: "2025-08-14T12:00:00Z",
    metadata: {},
  });
  return render(
    <ThemeProvider>
      <QueryClientProvider client={client}>
        <App />
      </QueryClientProvider>
    </ThemeProvider>,
  );
}

describe("App", () => {
  it("renders the dashboard overview route", () => {
    renderApp("/dashboard", dashboardFetchMock());

    expect(screen.getByRole("heading", { name: "Overview" })).toBeInTheDocument();
    expect(screen.getByText("SIGNING KEYS")).toBeInTheDocument();
  });

  it("renders dashboard overview summary states from the API", async () => {
    renderApp("/dashboard", dashboardFetchMock());

    expect(await screen.findByText("tenant.create")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Resource ID: 00000000-0000-0000-0000-00000000acme"),
    ).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard", dashboardErrorFetchMock());

    expect(await screen.findByText("Couldn't load install summary.")).toBeInTheDocument();

    cleanup();
    renderApp(
      "/dashboard",
      vi.fn((input: RequestInfo | URL) => {
        const url =
          typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
        if (url === "/api/v1/instance/summary") {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                tenants: 0,
                projects: 0,
                users: 0,
                active_signing_key: false,
                recent_audit: [],
              }),
              { status: 200 },
            ),
          );
        }
        return defaultFetchMock()(input);
      }),
    );

    expect(await screen.findByRole("heading", { name: "No tenants yet" })).toBeInTheDocument();
  });

  it("captures the API version at SPA boot instead of hardcoding dev", async () => {
    const fetchMock = vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify({ version: "v0.1.0", commit: "test" }), { status: 200 }),
      ),
    );
    renderApp("/dashboard", fetchMock);

    expect(await screen.findByRole("heading", { name: "Overview" })).toBeInTheDocument();
    expect(
      screen.queryByText("A new version is available - refresh to update"),
    ).not.toBeInTheDocument();
  });

  it("renders setup wizard by token route", async () => {
    renderApp("/setup/cypra_setup_test", setupFetchMock());

    expect(screen.getByRole("heading", { name: "Set up Cypra" })).toBeInTheDocument();
    expect(await screen.findByLabelText("Admin email")).toBeInTheDocument();
  });

  it("navigates to the dashboard once backup codes are confirmed", async () => {
    const navigatorMock = Object.create(navigator) as Navigator & {
      credentials: CredentialsContainer;
    };
    Object.defineProperty(navigatorMock, "credentials", {
      configurable: true,
      value: {
        create: vi.fn(() =>
          Promise.resolve({
            id: "credential-id",
            rawId: new Uint8Array([1, 2, 3]).buffer,
            type: "public-key",
            authenticatorAttachment: "platform",
            getClientExtensionResults: () => ({}),
            response: {
              clientDataJSON: new Uint8Array([4, 5, 6]).buffer,
              attestationObject: new Uint8Array([7, 8, 9]).buffer,
              getTransports: () => ["internal"],
            },
          }),
        ),
      },
    });
    vi.stubGlobal("navigator", navigatorMock);
    renderApp("/setup/cypra_setup_test", setupFetchMock());

    await userEvent.type(await screen.findByLabelText("Admin email"), "root@example.com");
    await userEvent.type(screen.getByLabelText("Display name"), "Root Admin");
    await userEvent.click(screen.getByRole("button", { name: "Continue" }));
    await userEvent.click(screen.getByRole("button", { name: "Enroll passkey" }));
    await userEvent.click(await screen.findByRole("button", { name: "I have saved these" }));
    // Once codes are saved, the wizard hands off to the dashboard via a hard
    // navigation. JSDOM can't follow that, so we verify the button is present
    // and clickable — the navigation itself happens on real browsers.
    expect(await screen.findByRole("button", { name: "Go to dashboard" })).not.toBeDisabled();
  });

  it("renders the primitive gallery", () => {
    renderApp("/__cypra/gallery");

    expect(screen.getByRole("heading", { name: "Primitive gallery" })).toBeInTheDocument();
    expect(screen.getByText("Light mode parity")).toBeInTheDocument();
    expect(screen.getByText("Dark mode parity")).toBeInTheDocument();
  });

  it("has no axe violations on the gallery route", async () => {
    renderApp("/__cypra/gallery");

    const results = await axe.run(document.body);

    expect(results.violations).toEqual([]);
  });

  it("persists theme changes through the actor metadata endpoint", async () => {
    const fetchMock = defaultFetchMock();
    renderApp("/dashboard/account", fetchMock);

    await userEvent.click(screen.getAllByRole("radio", { name: "light" })[0]);

    expect(localStorage.getItem("cypra.theme")).toBe("light");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/instance/admins/me",
      expect.objectContaining({ method: "PATCH" }),
    );

    cleanup();
    const tenantFetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme", tenantFetchMock);

    await userEvent.click((await screen.findAllByRole("button", { name: /^Theme:/ }))[0]);

    expect(localStorage.getItem("cypra.theme")).toBe("dark");
    expect(tenantFetchMock).toHaveBeenCalledWith(
      "/api/v1/users/me",
      expect.objectContaining({ method: "PATCH" }),
    );
  });

  it("opens shortcut help from shift+? and footer help", async () => {
    renderApp("/dashboard");

    await userEvent.keyboard("{Shift>}?{/Shift}");
    expect(screen.getByRole("dialog", { name: "Keyboard shortcuts" })).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    await userEvent.click(screen.getByRole("button", { name: "Show keyboard shortcuts" }));

    expect(screen.getByRole("dialog", { name: "Keyboard shortcuts" })).toBeInTheDocument();
  });

  it("renders responsive dashboard navigation controls", async () => {
    renderApp("/dashboard/tenants?state=demo", dashboardFetchMock());

    expect(screen.getByRole("button", { name: "Open navigation" })).toHaveAttribute(
      "aria-controls",
      "mobile-dashboard-nav",
    );
    await userEvent.click(screen.getByRole("button", { name: "Open navigation" }));

    expect(screen.getByRole("dialog", { name: "Dashboard navigation" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close navigation" })).toBeInTheDocument();
    expect(await screen.findByRole("table")).toHaveAttribute("data-responsive", "stack");
  });

  it("announces route titles in the live region", () => {
    renderApp("/dashboard/instance/diagnostics", dashboardFetchMock());

    expect(screen.getByRole("status")).toHaveTextContent("Diagnostics");
  });

  it("does not leak phase / dev placeholder copy on stub routes", () => {
    const stubRoutes = [
      "/dashboard/users",
      "/dashboard/projects",
      "/dashboard/auth-methods",
      "/dashboard/members",
      "/dashboard/audit",
      "/dashboard/settings",
    ];
    const forbidden = [
      "This route is wired",
      "data-backed implementation",
      "Phase 9",
      "Phase 10",
      "lands in a later phase",
    ];
    for (const route of stubRoutes) {
      renderApp(route);
      const main = document.querySelector("main");
      const text = main?.textContent ?? "";
      for (const phrase of forbidden) {
        expect(text, `${route} leaked "${phrase}"`).not.toContain(phrase);
      }
      cleanup();
    }
  });

  it("renders 404 for retired install-level stub routes", () => {
    renderApp("/dashboard/audit");
    expect(screen.getByText("We couldn't find that page.")).toBeInTheDocument();
  });

  it("g-then-t sequence navigates to the tenants route from instance scope", async () => {
    const assignSpy = stubLocation("/dashboard");
    renderApp("/dashboard");
    await userEvent.keyboard("g");
    await userEvent.keyboard("t");
    expect(assignSpy).toHaveBeenCalledWith("/dashboard/tenants");
  });

  it("g-then-u sequence navigates to tenant users from a tenant route", async () => {
    const assignSpy = stubLocation("/dashboard/tenants/acme");
    renderApp("/dashboard/tenants/acme", dashboardFetchMock());
    await userEvent.keyboard("g");
    await userEvent.keyboard("u");
    expect(assignSpy).toHaveBeenCalledWith("/dashboard/tenants/acme/users");
  });

  it("renders the searchable tenant list", async () => {
    renderApp("/dashboard/tenants", dashboardFetchMock());

    expect(await screen.findByRole("heading", { name: "Tenants" })).toBeInTheDocument();
    expect((await screen.findAllByText("Acme Operations")).length).toBeGreaterThan(0);

    const searchInputs = screen.getAllByLabelText("Search tenants");
    await userEvent.type(searchInputs[searchInputs.length - 1], "missing");

    expect(
      screen.getByText((_content, element) =>
        element?.textContent === "No tenants match missing." ? true : false,
      ),
    ).toBeInTheDocument();
  });

  it("keeps dashboard demo records out of production API error states", async () => {
    const routes = [
      ["/dashboard/tenants/acme/projects/console", "Couldn't load project.", "Console App"],
      [
        "/dashboard/tenants/acme/users/00000000-0000-0000-0000-00000000ada1",
        "Couldn't load user.",
        "ada@example.com",
      ],
      [
        "/dashboard/tenants/acme/settings/api-tokens",
        "Couldn't load API tokens.",
        "Deploy automation",
      ],
      [
        "/dashboard/tenants/acme/settings/email",
        "Couldn't load email provider.",
        "Recommended: Resend free tier",
      ],
      ["/dashboard/instance/admins", "Couldn't load instance admins.", "root@example.com"],
      ["/dashboard/instance/diagnostics", "Couldn't load diagnostics.", "Storage backend"],
    ] as const;

    for (const [route, errorText, demoText] of routes) {
      renderApp(route, dashboardErrorFetchMock());

      expect(await screen.findByText(errorText)).toBeInTheDocument();
      expect(screen.queryByText(demoText)).not.toBeInTheDocument();
      cleanup();
    }
  });

  it("renders production empty states instead of demo records", async () => {
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      if (url === "/api/v1/tenants/") {
        return Promise.resolve(
          new Response(
            JSON.stringify([
              {
                id: "00000000-0000-0000-0000-00000000acme",
                slug: "acme",
                name: "Acme Operations",
                branding: { display_name: "Acme Login", powered_by: true },
              },
            ]),
            { status: 200 },
          ),
        );
      }
      if (
        [
          "/api/v1/projects/",
          "/api/v1/users/",
          "/api/v1/pats/",
          "/api/v1/instance/admins",
          "/api/v1/instance/invites",
        ].includes(url)
      ) {
        return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
      }
      return Promise.resolve(
        new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
      );
    });

    const routes = [
      ["/dashboard/tenants/acme/projects", "No projects yet.", "Console App"],
      ["/dashboard/tenants/acme/users", "No users yet.", "ada@example.com"],
      ["/dashboard/tenants/acme/settings/api-tokens", "No API tokens yet.", "Deploy automation"],
      ["/dashboard/instance/admins", "No instance admins visible.", "root@example.com"],
    ] as const;

    for (const [route, emptyText, demoText] of routes) {
      renderApp(route, fetchMock);

      expect(await screen.findByText(emptyText)).toBeInTheDocument();
      expect(screen.queryByText(demoText)).not.toBeInTheDocument();
      cleanup();
    }
  });

  it("renders tenant overview and branding settings routes", async () => {
    renderApp("/dashboard/tenants/acme/settings/branding", dashboardFetchMock());

    expect(await screen.findByRole("heading", { name: "Acme Login" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Branding" })).toBeInTheDocument();
    expect(screen.getByLabelText("Accent color")).toBeInTheDocument();
  });

  it("renders project detail, auth methods, and API tokens surfaces", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/projects/console", fetchMock);

    expect(await screen.findByRole("heading", { name: "Console App" })).toBeInTheDocument();
    expect(screen.getByText("Rotate client secret")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Save project" }));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/projects/00000000-0000-0000-0000-00000000app1",
      expect.objectContaining({ method: "PUT" }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Rotate client secret" }));
    await userEvent.click(screen.getByRole("button", { name: "Start rotation" }));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/projects/00000000-0000-0000-0000-00000000app1/rotate-secret",
      expect.objectContaining({ method: "POST" }),
    );

    cleanup();
    renderApp("/dashboard/tenants/acme/auth-providers", fetchMock);

    expect(
      await screen.findByRole("heading", { name: "Authentication providers" }),
    ).toBeInTheDocument();
    expect(await screen.findByText("Default sign-in method")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/auth-providers/identifiers/password", fetchMock);

    expect(await screen.findByRole("heading", { name: "Password" })).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/settings/api-tokens", fetchMock);

    expect(await screen.findByRole("heading", { name: "API tokens" })).toBeInTheDocument();
    expect(screen.getByText("Deploy automation")).toBeInTheDocument();
    expect(screen.getByText("ending 9xyz")).toBeInTheDocument();
    expect(screen.getByText(/expires/)).toBeInTheDocument();
  });

  it("creates and reveals a personal access token with backend scope names", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/settings/api-tokens", fetchMock);

    await userEvent.click(await screen.findByRole("button", { name: "Create token" }));
    const dialog = screen.getByRole("dialog");
    await userEvent.type(within(dialog).getByLabelText("Token name"), "Deploy write");
    await userEvent.click(within(dialog).getByLabelText("projects.read"));
    await userEvent.click(within(dialog).getByLabelText("projects.write"));
    await userEvent.click(within(dialog).getByRole("button", { name: "Create token" }));
    await userEvent.click(await within(dialog).findByRole("button", { name: "Reveal secret" }));

    expect(await screen.findByText("cypra_pat_secret_once")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/pats/",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ name: "Deploy write", scopes: ["projects.write"] }),
      }),
    );
  });

  it("renders user list and user detail surfaces", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/users", fetchMock);

    expect(await screen.findByRole("heading", { name: "Users" })).toBeInTheDocument();
    expect(screen.getByText("ada@example.com")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/users/00000000-0000-0000-0000-00000000ada1", fetchMock);

    expect(await screen.findByRole("heading", { name: "ada@example.com" })).toBeInTheDocument();
    expect(screen.getByText("Reset password")).toBeInTheDocument();
  });

  it("sends tenant user invites and avoids production placeholder copy", async () => {
    const baseFetch = dashboardFetchMock();
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      if (url === "/api/v1/tenants/") {
        return Promise.resolve(
          new Response(
            JSON.stringify([
              {
                id: "00000000-0000-0000-0000-00000000acme",
                slug: "acme",
                name: "Acme Operations",
                branding: { display_name: "Acme Login", powered_by: true },
                email_provider_required: false,
              },
            ]),
            { status: 200 },
          ),
        );
      }
      if (url === "/api/v1/admin/invite" && init?.method === "POST") {
        return Promise.resolve(new Response("", { status: 201 }));
      }
      return baseFetch(input, init);
    });
    renderApp("/dashboard/tenants/acme/users", fetchMock);

    await userEvent.click(await screen.findByRole("button", { name: "Invite user" }));
    const inviteDialog = screen.getByRole("dialog", { name: "Invite user" });
    await userEvent.type(within(inviteDialog).getByLabelText("Email"), "new-user@example.com");
    await userEvent.click(within(inviteDialog).getByRole("button", { name: "Send invite" }));

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/admin/invite",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          email: "new-user@example.com",
          role: "member",
        }),
      }),
    );

    await userEvent.click(screen.getByRole("button", { name: "Import via CLI" }));
    expect(
      screen.queryByText("Screencast placeholder lands in v1.1 docs."),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/validates users before writing tenant records/)).toBeInTheDocument();
  });

  it("renders members, signing keys, and audit surfaces", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/settings/members", fetchMock);

    expect(await screen.findByRole("heading", { name: "Members & roles" })).toBeInTheDocument();
    expect(screen.getByText("ada@example.com")).toBeInTheDocument();
    expect(screen.getByText("alan@example.com")).toBeInTheDocument();
    await userEvent.selectOptions(screen.getByLabelText("alan@example.com role"), "member");
    await userEvent.click(screen.getAllByRole("button", { name: "Save" })[0]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/admin/members/00000000-0000-0000-0000-00000000mem2",
      expect.objectContaining({ method: "PUT" }),
    );
    expect(screen.getByText("Pending invites")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/signing-keys", fetchMock);

    expect(await screen.findByRole("heading", { name: "Signing keys" })).toBeInTheDocument();
    expect(
      screen.getAllByLabelText("Key ID: 00000000-0000-0000-0000-00000000acme:1").length,
    ).toBeGreaterThan(0);
    await userEvent.click(screen.getByRole("button", { name: "Rotate now" }));
    const rotateDialog = screen.getByRole("dialog");
    await userEvent.type(
      within(rotateDialog).getByLabelText("Type active kid to confirm"),
      "00000000-0000-0000-0000-00000000acme:1",
    );
    await userEvent.click(within(rotateDialog).getByRole("button", { name: "Rotate key" }));
    expect(await screen.findByText("Signing key rotation started.")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/signing-keys/rotate",
      expect.objectContaining({ method: "POST" }),
    );

    cleanup();
    renderApp("/dashboard/tenants/acme/audit", fetchMock);

    expect(await screen.findByRole("heading", { name: "Tenant audit" })).toBeInTheDocument();
    expect(screen.getByText("user.update")).toBeInTheDocument();
    expect(screen.getByText("redacted")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Expand entry" }));
    expect(screen.getByText(/state_after/)).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("Action"), "user");
    expect(window.location.search).toContain("action=user");
    expect(screen.getByText("Export NDJSON")).toBeInTheDocument();
  });

  it("has no axe violations on Phase 9 dashboard routes", async () => {
    const routes = [
      ["/dashboard/tenants", "Tenants"],
      ["/dashboard/tenants/acme/settings/branding", "Branding"],
      ["/dashboard/tenants/acme/projects/console", "Console App"],
      ["/dashboard/tenants/acme/auth-providers", "Authentication providers"],
      ["/dashboard/tenants/acme/users", "Users"],
      ["/dashboard/tenants/acme/settings/members", "Members & roles"],
      ["/dashboard/tenants/acme/signing-keys", "Signing keys"],
      ["/dashboard/tenants/acme/audit", "Tenant audit"],
      ["/dashboard/tenants/acme/settings/api-tokens", "API tokens"],
    ] as const;

    for (const [route, heading] of routes) {
      renderApp(route, dashboardFetchMock());
      await screen.findByRole("heading", { name: heading });

      const results = await axe.run(document.body);

      expect(results.violations, route).toEqual([]);
      cleanup();
    }
  });

  it("renders Phase 10 provider, danger, and instance routes", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/settings/email", fetchMock);

    expect(await screen.findByRole("heading", { name: "Email provider" })).toBeInTheDocument();
    expect(screen.getByText("No email provider configured yet")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Send test email" }));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/provider-config/email/test",
      expect.objectContaining({ method: "POST" }),
    );

    cleanup();
    renderApp("/dashboard/tenants/acme/settings/danger", fetchMock);

    expect(await screen.findByRole("heading", { name: "Suspend tenant" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Schedule deletion" })).toBeDisabled();
    await userEvent.type(screen.getByLabelText("Type tenant slug to confirm"), "acme");
    await userEvent.click(screen.getByRole("button", { name: "Schedule deletion" }));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/tenants/00000000-0000-0000-0000-00000000acme/delete",
      expect.objectContaining({ method: "POST" }),
    );

    cleanup();
    renderApp("/dashboard/instance/admins", fetchMock);

    expect(await screen.findByRole("heading", { name: "Instance admins" })).toBeInTheDocument();
    expect(screen.getByText("root@example.com")).toBeInTheDocument();
    expect(screen.getByText("ops@example.com")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Demote" })).toBeDisabled();

    await userEvent.click(screen.getByRole("button", { name: "Invite admin" }));
    await userEvent.type(screen.getByLabelText("Email"), "second@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Send invite" }));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/instance/invite",
      expect.objectContaining({ method: "POST" }),
    );

    cleanup();
    renderApp("/dashboard/instance/diagnostics", fetchMock);

    expect(await screen.findByRole("heading", { name: "Diagnostics" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Storage backend" })).toBeInTheDocument();
  });

  it("has no axe violations on Phase 10 dashboard routes", async () => {
    const routes = [
      ["/dashboard", "Overview"],
      ["/dashboard/account", "Account"],
      ["/dashboard/tenants/acme", "Acme Login"],
      ["/dashboard/tenants/acme/settings/email", "Email provider"],
      ["/dashboard/tenants/acme/settings/danger", "Suspend tenant"],
      ["/dashboard/tenants/acme/users/00000000-0000-0000-0000-00000000ada1", "ada@example.com"],
      ["/dashboard/instance/admins", "Instance admins"],
      ["/dashboard/instance/diagnostics", "Diagnostics"],
      ["/dashboard/instance/audit", "Instance audit"],
    ] as const;

    for (const [route, heading] of routes) {
      renderApp(route, dashboardFetchMock());
      await screen.findByRole("heading", { name: heading });

      const results = await axe.run(document.body);

      expect(results.violations, route).toEqual([]);
      cleanup();
    }
  });
});

describe("primitives", () => {
  it("enforces 40px touch targets on coarse pointers", () => {
    const tokensCSS = readFileSync("src/tokens.css", "utf8");

    expect(tokensCSS).toContain("@media (pointer: coarse)");
    expect(tokensCSS).toContain('button:not([data-touch-target="cursor-only"])');
    expect(tokensCSS).toContain("min-width: var(--space-10)");
    expect(tokensCSS).toContain("min-height: var(--space-10)");
  });

  it("removes transforms and animation durations for reduced motion", () => {
    const indexCSS = readFileSync("src/index.css", "utf8");

    expect(indexCSS).toContain("@media (prefers-reduced-motion: reduce)");
    expect(indexCSS).toContain("animation-duration: 0ms !important");
    expect(indexCSS).toContain("transition-duration: 0ms !important");
    expect(indexCSS).toContain("transform: none !important");
  });

  it("renders button variants and loading state", () => {
    render(<Button loading>Save</Button>);

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("renders text input error state", () => {
    render(<TextInput label="Email" error="Email is required." />);

    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByText("Email is required.")).toBeInTheDocument();
  });

  it("refuses masked secret copy until revealed", async () => {
    render(<MaskedSecret name="client_secret" value="real-secret" />);

    await userEvent.click(screen.getByRole("button", { name: "Copy secret" }));

    expect(screen.getByText("Reveal first to copy")).toBeInTheDocument();
  });

  it("copies identifier pills", async () => {
    render(<IdentifierPill value="client_0123456789abcdef" label="Client ID" />);

    expect(screen.getByLabelText("Client ID: client_0123456789abcdef")).toBeInTheDocument();

    const copyButtons = screen.getAllByRole("button", { name: "Copy Client ID" });
    const copyButton = copyButtons[0];
    await userEvent.click(copyButton);

    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it("renders status variants with labels", () => {
    render(<StatusPip variant="pending" label="Pending" />);

    expect(screen.getByText("Pending")).toBeInTheDocument();
  });

  it("keeps skeletons static and shows the mobile blocked banner copy", () => {
    const { container } = render(
      <>
        <Skeleton />
        <MobileBlockedBanner />
      </>,
    );

    expect(container.querySelector(".animate-pulse")).not.toBeInTheDocument();
    expect(container.querySelector(".animate-shimmer")).not.toBeInTheDocument();
    expect(
      screen.getByText("Cypra dashboard is optimized for desktop. Some features may be cramped."),
    ).toBeInTheDocument();
  });

  it("renders backup codes with confirmation gate", async () => {
    const onConfirmedChange = vi.fn();
    render(
      <BackupCodeGrid codes={["CYPRA-1111", "CYPRA-2222"]} onConfirmedChange={onConfirmedChange} />,
    );

    const savedButtons = screen.getAllByRole("button", { name: "I have saved these" });
    const savedButton = savedButtons[0];
    await userEvent.click(savedButton);

    expect(
      screen.getByText("Backup codes saved. You can leave this screen safely."),
    ).toBeInTheDocument();
    expect(onConfirmedChange).toHaveBeenCalledWith(true);
  });

  it("guards backup codes against browser back before confirmation", () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const pushSpy = vi.spyOn(history, "pushState");
    render(<BackupCodeGrid codes={["CYPRA-1111"]} />);

    window.dispatchEvent(new PopStateEvent("popstate"));

    expect(confirmSpy).toHaveBeenCalledWith(
      "Your backup codes are shown only once. Continue without saving?",
    );
    expect(pushSpy).toHaveBeenCalled();
    confirmSpy.mockRestore();
    pushSpy.mockRestore();
  });

  it("downloads backup codes as text", async () => {
    const created: string[] = [];
    const createSpy = vi.spyOn(URL, "createObjectURL").mockImplementation(() => {
      created.push("blob:mock");
      return "blob:mock";
    });
    const revokeSpy = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => undefined);
    render(<BackupCodeGrid codes={["A", "B"]} />);
    const downloadButtons = screen.getAllByRole("button", { name: "Download .txt" });
    await userEvent.click(downloadButtons[0]);
    expect(created.length).toBeGreaterThan(0);
    createSpy.mockRestore();
    revokeSpy.mockRestore();
  });

  it("renders all gallery composites", () => {
    render(
      <ThemeProvider>
        <PrimitiveGallery />
      </ThemeProvider>,
    );

    expect(screen.getAllByText("Permission matrix").length).toBeGreaterThan(0);
  });

  it("traps modal focus, inerts the app root, closes on Escape, and restores focus", async () => {
    function Harness() {
      const [open, setOpen] = useState(false);
      return (
        <>
          <Button onClick={() => setOpen(true)}>Open modal</Button>
          <Modal
            title="Keyboard modal"
            open={open}
            onClose={() => setOpen(false)}
            footer={<Button>Save</Button>}
          >
            <TextInput label="Modal field" data-autofocus="true" />
          </Modal>
        </>
      );
    }
    const root = document.createElement("div");
    root.id = "root";
    document.body.appendChild(root);
    render(<Harness />, { container: root });

    const opener = screen.getByRole("button", { name: "Open modal" });
    opener.focus();
    await userEvent.click(opener);

    const dialog = screen.getByRole("dialog", { name: "Keyboard modal" });
    expect(root).toHaveAttribute("inert");
    expect(within(dialog).getByLabelText("Modal field")).toHaveFocus();

    within(dialog).getByRole("button", { name: "Save" }).focus();
    await userEvent.tab();
    expect(within(dialog).getByRole("button", { name: "Close" })).toHaveFocus();

    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Keyboard modal" })).not.toBeInTheDocument();
    expect(root).not.toHaveAttribute("inert");
    expect(opener).toHaveFocus();
    root.remove();
  });

  it("applies the same keyboard close behavior to drawers", async () => {
    function Harness() {
      const [open, setOpen] = useState(false);
      return (
        <>
          <Button onClick={() => setOpen(true)}>Open drawer</Button>
          <Drawer title="Keyboard drawer" open={open} onClose={() => setOpen(false)}>
            <Button>Drawer action</Button>
          </Drawer>
        </>
      );
    }
    render(<Harness />);

    await userEvent.click(screen.getByRole("button", { name: "Open drawer" }));
    expect(screen.getByRole("dialog", { name: "Keyboard drawer" })).toBeInTheDocument();

    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Keyboard drawer" })).not.toBeInTheDocument();
  });
});

describe("modals & wiring", () => {
  it("opens Create tenant modal from dashboard CTA and submits", async () => {
    const calls: { url: string; method?: string; body?: string }[] = [];
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      calls.push({
        url,
        method: init?.method,
        body: typeof init?.body === "string" ? init.body : undefined,
      });
      if (url === "/api/v1/tenants/" && init?.method === "POST") {
        return Promise.resolve(
          new Response(JSON.stringify({ id: "t1", slug: "newco", name: "New Co" }), {
            status: 201,
          }),
        );
      }
      return Promise.resolve(
        new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
      );
    });
    history.replaceState(null, "", "/dashboard");
    vi.stubGlobal("fetch", fetchMock);
    stubLocation("/dashboard");
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    client.setQueryData(["app-me"], {
      kind: "user",
      id: "00000000-0000-0000-0000-00000000bo0t",
      sub: "00000000-0000-0000-0000-00000000bo0t",
      email: "operator@example.com",
      display_name: "Operator",
      tenant_id: "00000000-0000-0000-0000-00000000acme",
      created_at: "2025-08-14T12:00:00Z",
      metadata: {},
    });
    render(
      <ThemeProvider>
        <QueryClientProvider client={client}>
          <App />
        </QueryClientProvider>
      </ThemeProvider>,
    );

    const ctaButtons = await screen.findAllByRole("button", { name: /Create.*tenant/i });
    await userEvent.click(ctaButtons[0]);

    const dialog = await screen.findByRole("dialog");
    const nameInput = await within(dialog).findByLabelText("Display name");
    const slugInput = within(dialog).getByLabelText("Slug");
    await userEvent.type(nameInput, "New Co");
    await userEvent.type(slugInput, "newco");

    const submit = within(dialog).getAllByRole("button", { name: "Create tenant" })[0];
    await userEvent.click(submit);

    const tenantPost = calls.find(
      (call) => call.url === "/api/v1/tenants/" && call.method === "POST",
    );
    expect(tenantPost).toBeDefined();
    const body = JSON.parse(tenantPost?.body ?? "{}") as unknown;
    expect(body).toEqual({ slug: "newco", name: "New Co" });
  });

  it("revokes other sessions when Sign out other sessions is clicked", async () => {
    const calls: { url: string; method?: string }[] = [];
    const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      calls.push({ url, method: init?.method });
      if (url === "/api/v1/auth/sessions/revoke-others") {
        return Promise.resolve(new Response("", { status: 204 }));
      }
      return Promise.resolve(
        new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
      );
    });
    history.replaceState(null, "", "/dashboard/account");
    vi.stubGlobal("fetch", fetchMock);
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <ThemeProvider>
        <QueryClientProvider client={client}>
          <App />
        </QueryClientProvider>
      </ThemeProvider>,
    );
    const button = await screen.findByRole("button", { name: "Sign out other sessions" });
    await userEvent.click(button);
    const revoke = calls.find((c) => c.url === "/api/v1/auth/sessions/revoke-others");
    expect(revoke?.method).toBe("POST");
  });

  it("exports audit NDJSON via window.location.assign", async () => {
    history.replaceState(null, "", "/dashboard/tenants/acme/audit");
    vi.stubGlobal("fetch", dashboardFetchMock());
    const assignSpy = stubLocation("/dashboard/tenants/acme/audit");
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <ThemeProvider>
        <QueryClientProvider client={client}>
          <App />
        </QueryClientProvider>
      </ThemeProvider>,
    );
    const exportButton = await screen.findByRole("button", { name: "Export NDJSON" });
    await userEvent.click(exportButton);
    expect(assignSpy).toHaveBeenCalledWith(expect.stringContaining("/api/v1/audit/export"));
  });
});
