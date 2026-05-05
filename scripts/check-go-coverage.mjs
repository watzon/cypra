import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";

const floor = 85;
const packages = [
  "./internal/crypto",
  "./internal/oidc",
  "./internal/auth",
  "./internal/db",
  "./internal/sessions",
  "./internal/bootstrap",
];

const tempDir = mkdtempSync(join(tmpdir(), "cypra-coverage-"));
const failures = [];

try {
  for (const packageName of packages) {
    const profile = join(tempDir, `${packageName.replaceAll(/[/.]/g, "_")}.cover`);
    run("go", ["test", `-coverprofile=${profile}`, packageName]);
    const output = run("go", ["tool", "cover", `-func=${profile}`]);
    const total = output
      .trim()
      .split("\n")
      .find((line) => line.startsWith("total:"));
    if (!total) {
      failures.push(`${packageName}: missing total coverage line`);
      continue;
    }
    const match = total.match(/([0-9]+(?:\.[0-9]+)?)%$/);
    if (!match) {
      failures.push(`${packageName}: could not parse coverage from ${total}`);
      continue;
    }
    const coverage = Number(match[1]);
    console.log(`${packageName}: ${coverage.toFixed(1)}%`);
    if (coverage < floor) {
      failures.push(`${packageName}: ${coverage.toFixed(1)}% < ${floor}%`);
    }
  }
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}

if (failures.length > 0) {
  console.error(`Coverage floor failed:\n${failures.join("\n")}`);
  process.exit(1);
}

function run(command, args) {
  const result = spawnSync(command, args, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
  if (result.status !== 0) {
    process.stdout.write(result.stdout);
    process.stderr.write(result.stderr);
    throw new Error(`${command} ${args.join(" ")} failed`);
  }
  return result.stdout;
}
