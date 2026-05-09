import lighthouse from "lighthouse";
import * as chromeLauncher from "chrome-launcher";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { spawn } from "node:child_process";

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

let chrome;
try {
  await waitForServer(`${baseURL}/setup/cypra_setup_test`);
  chrome = await chromeLauncher.launch({ chromeFlags: ["--headless=new", "--no-sandbox"] });
  const latest = {
    generatedAt: new Date().toISOString(),
    runner: "lighthouse",
    routes: [],
  };

  for (const route of routes) {
    const result = await lighthouse(`${baseURL}${route.path}`, {
      port: chrome.port,
      output: "json",
      logLevel: "error",
      onlyCategories: ["performance", "accessibility", "best-practices"],
      throttlingMethod: "provided",
    });
    if (!result?.lhr) throw new Error(`Lighthouse did not return an LHR for ${route.name}`);
    const categories = result.lhr.categories;
    latest.routes.push({
      name: route.name,
      path: route.path,
      thresholds: route.thresholds,
      scores: {
        performance: score(categories.performance?.score),
        accessibility: score(categories.accessibility?.score),
        bestPractices: score(categories["best-practices"]?.score),
      },
      metrics: {
        firstContentfulPaintMs: auditNumeric(result.lhr, "first-contentful-paint"),
        largestContentfulPaintMs: auditNumeric(result.lhr, "largest-contentful-paint"),
        totalBlockingTimeMs: auditNumeric(result.lhr, "total-blocking-time"),
        cumulativeLayoutShift: auditNumeric(result.lhr, "cumulative-layout-shift"),
      },
    });
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
  await chrome?.kill();
  preview.kill("SIGTERM");
}

function score(value) {
  return Math.round((value ?? 0) * 100);
}

function auditNumeric(lhr, id) {
  const value = lhr.audits[id]?.numericValue;
  return typeof value === "number" ? Math.round(value * 100) / 100 : null;
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
