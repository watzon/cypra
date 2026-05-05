import { chromium } from "playwright";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { spawn } from "node:child_process";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const axeSource = readFileSync(require.resolve("axe-core/axe.min.js"), "utf8");

const routes = [
  {
    name: "hosted-login-setup",
    path: "/setup/cypra_setup_test",
    thresholds: { performance: 85, accessibility: 95, bestPractices: 95 },
  },
  {
    name: "dashboard-demo",
    path: "/dashboard/tenants?state=demo",
    thresholds: { performance: 70, accessibility: 95, bestPractices: 95 },
  },
];

const baselinePath = resolve("tests/e2e/lighthouse-baseline.json");
const latestPath = resolve("test-results/lighthouse/latest.json");
const port = Number(process.env.CYPRA_LIGHTHOUSE_PORT ?? 4173);
const baseURL = `http://127.0.0.1:${port}`;

const preview = spawn("bun", ["run", "dev", "--", "--port", String(port)], {
  stdio: ["ignore", "pipe", "pipe"],
  cwd: "dashboard",
  env: { ...process.env, BROWSER: "none" },
});

let serverOutput = "";
preview.stdout.on("data", (chunk) => {
  serverOutput += chunk.toString();
});
preview.stderr.on("data", (chunk) => {
  serverOutput += chunk.toString();
});

try {
  await waitForServer(`${baseURL}/setup/cypra_setup_test`);
  const browser = await chromium.launch();
  const latest = {
    generatedAt: new Date().toISOString(),
    runner: "playwright-axe-synthetic-lighthouse",
    routes: [],
  };

  try {
    for (const route of routes) {
      const page = await browser.newPage({ viewport: { width: 1366, height: 900 } });
      const consoleErrors = [];
      const pageErrors = [];
      page.on("console", (message) => {
        if (message.type() === "error") consoleErrors.push(message.text());
      });
      page.on("pageerror", (error) => pageErrors.push(error.message));

      const started = performance.now();
      await page.goto(`${baseURL}${route.path}`, { waitUntil: "networkidle" });
      const elapsedMs = Math.round(performance.now() - started);
      await page.addScriptTag({ content: axeSource });
      const axe = await page.evaluate(async () => globalThis.axe.run(document.body));
      const nav = await page.evaluate(() => {
        const entry = performance.getEntriesByType("navigation")[0];
        return entry ? { duration: entry.duration, transferSize: entry.transferSize } : null;
      });
      await page.close();

      const scores = {
        performance: performanceScore(elapsedMs),
        accessibility:
          axe.violations.length === 0 ? 100 : Math.max(0, 100 - axe.violations.length * 20),
        bestPractices: consoleErrors.length === 0 && pageErrors.length === 0 ? 100 : 90,
      };
      latest.routes.push({
        name: route.name,
        path: route.path,
        thresholds: route.thresholds,
        scores,
        metrics: {
          elapsedMs,
          navigationDurationMs: nav?.duration ?? null,
          transferSize: nav?.transferSize ?? null,
        },
        violations: axe.violations.map((violation) => ({
          id: violation.id,
          impact: violation.impact,
        })),
        consoleErrors,
        pageErrors,
      });
    }
  } finally {
    await browser.close();
  }

  mkdirSync(dirname(latestPath), { recursive: true });
  writeFileSync(latestPath, `${JSON.stringify(latest, null, 2)}\n`);
  const baseline = JSON.parse(readFileSync(baselinePath, "utf8"));
  const failures = [];
  for (const route of latest.routes) {
    for (const [scoreName, threshold] of Object.entries(route.thresholds)) {
      if (route.scores[scoreName] < threshold) {
        failures.push(`${route.name} ${scoreName} ${route.scores[scoreName]} < ${threshold}`);
      }
    }
    const stored = baseline.routes.find((candidate) => candidate.name === route.name);
    if (!stored) failures.push(`${route.name} missing from stored baseline`);
  }

  if (failures.length > 0) {
    throw new Error(`Lighthouse baseline failed:\n${failures.join("\n")}`);
  }
  console.log(`Lighthouse baseline passed for ${latest.routes.length} routes.`);
} finally {
  preview.kill("SIGTERM");
}

function performanceScore(elapsedMs) {
  if (elapsedMs <= 750) return 100;
  if (elapsedMs <= 1250) return 90;
  if (elapsedMs <= 2000) return 80;
  if (elapsedMs <= 3000) return 70;
  return 50;
}

async function waitForServer(url) {
  const timeoutAt = Date.now() + 30_000;
  while (Date.now() < timeoutAt) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // Retry until Vite is ready.
    }
    await new Promise((resolveRetry) => setTimeout(resolveRetry, 250));
  }
  throw new Error(`Vite server did not start. Output:\n${serverOutput}`);
}
