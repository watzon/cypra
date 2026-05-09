import { createServer } from "node:net";
import { spawn, spawnSync } from "node:child_process";

const image = process.env.CYPRA_COLD_START_IMAGE ?? "cypra:image-size";
const maxMs = Number(process.env.CYPRA_COLD_START_MAX_MS ?? 3000);
const postgresPort = await freePort();
const cypraPort = await freePort();
const project = `cypra-cold-start-${process.pid}-${Date.now()}`;
const databaseURL = `postgres://cypra:cypra@host.docker.internal:${postgresPort}/cypra?sslmode=disable`;
const composeEnv = { ...process.env, POSTGRES_HOST_PORT: String(postgresPort) };
const dockerEnvArgs = [
  "--add-host=host.docker.internal:host-gateway",
  "-e",
  `DATABASE_URL=${databaseURL}`,
  "-e",
  `MIGRATE_DATABASE_URL=${databaseURL}`,
  "-e",
  "MASTER_KEY=dev-only-change-me-dev-only-change-me-32b",
  "-e",
  "PUBLIC_BASE_URL=http://localhost:8080",
  "-e",
  "CYPRA_DEV_INSECURE_HTTP=true",
  "-e",
  "LISTEN_ADDR=:8080",
  "-e",
  "TRUSTED_PROXY_HEADERS=none",
  "-e",
  "STORAGE_BACKEND=local-disk",
  "-e",
  "STORAGE_LOCAL_PATH=/tmp/cypra-storage",
];

runChecked(
  "docker",
  ["compose", "-p", project, "-f", "deploy/docker-compose.yml", "up", "-d", "--wait", "postgres"],
  composeEnv,
);

let container;
try {
  runChecked("docker", ["run", "--rm", ...dockerEnvArgs, image, "migrate"], process.env);
  const started = performance.now();
  container = spawn(
    "docker",
    [
      "run",
      "--rm",
      "-p",
      `127.0.0.1:${cypraPort}:8080`,
      ...dockerEnvArgs,
      image,
      "serve",
      "--skip-migrate",
    ],
    { stdio: ["ignore", "pipe", "pipe"] },
  );
  const logs = [];
  container.stdout.on("data", (chunk) => logs.push(chunk.toString()));
  container.stderr.on("data", (chunk) => logs.push(chunk.toString()));
  await waitForReady(`http://127.0.0.1:${cypraPort}/readyz`, container, logs);
  const elapsedMs = Math.round(performance.now() - started);
  console.log(`cold-start readyz: ${elapsedMs}ms`);
  if (elapsedMs > maxMs) {
    throw new Error(`cold-start ${elapsedMs}ms exceeds ${maxMs}ms`);
  }
} finally {
  if (container?.exitCode === null) container.kill("SIGTERM");
  runChecked(
    "docker",
    ["compose", "-p", project, "-f", "deploy/docker-compose.yml", "down", "-v"],
    composeEnv,
  );
}

function runChecked(command, args, env) {
  const result = spawnSync(command, args, { encoding: "utf8", env });
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed\n${result.stdout}\n${result.stderr}`);
  }
  return result.stdout;
}

async function waitForReady(url, processToWatch, logs) {
  const timeoutAt = Date.now() + 30_000;
  while (Date.now() < timeoutAt) {
    if (processToWatch.exitCode !== null) {
      throw new Error(`container exited before readyz\n${logs.join("")}`);
    }
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // Retry until the container is listening.
    }
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error(`timed out waiting for readyz\n${logs.join("")}`);
}

async function freePort() {
  return await new Promise((resolve, reject) => {
    const server = createServer();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close(() => reject(new Error("could not allocate port")));
        return;
      }
      server.close(() => resolve(address.port));
    });
  });
}
