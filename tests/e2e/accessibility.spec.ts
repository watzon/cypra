import { expect, test, type Page } from "@playwright/test";
import { createRequire } from "node:module";

import { createTenant, redeemBootstrapToken, startHarness } from "./harness";

const require = createRequire(import.meta.url);
const axePath = require.resolve("axe-core/axe.min.js");

test("authenticated dashboard and hosted-login routes have no axe violations", async () => {
  test.setTimeout(4 * 60 * 1000);
  test.skip(
    !process.env.CYPRA_E2E_LIVE && process.env.CYPRA_E2E_MANAGED !== "1",
    "set CYPRA_E2E_LIVE=1 for an existing stack or CYPRA_E2E_MANAGED=1 for a managed local stack",
  );

  const harness = await startHarness();
  try {
    await redeemBootstrapToken(
      harness,
      harness.setupToken ?? process.env.CYPRA_SETUP_TOKEN ?? "cypra_setup_test",
    );
    await createTenant(harness, "acme");

    const dashboardRoutes = [
      "/dashboard",
      "/dashboard/account",
      "/dashboard/tenants",
      "/dashboard/tenants/acme",
      "/dashboard/tenants/acme/projects",
      "/dashboard/tenants/acme/users",
      "/dashboard/tenants/acme/auth-providers",
      "/dashboard/tenants/acme/signing-keys",
      "/dashboard/tenants/acme/audit",
      "/dashboard/tenants/acme/settings/branding",
      "/dashboard/tenants/acme/settings/api-tokens",
      "/dashboard/tenants/acme/settings/members",
      "/dashboard/tenants/acme/settings/email",
      "/dashboard/tenants/acme/settings/danger",
      "/dashboard/instance/admins",
      "/dashboard/instance/diagnostics",
      "/dashboard/instance/audit",
    ];

    for (const path of dashboardRoutes) {
      const url = new URL(path, harness.baseURL);
      url.searchParams.set("state", "demo");
      await harness.page.goto(url.toString());
      await expect(harness.page.locator("main")).toBeVisible();
      await expectNoAxeViolations(harness.page, path);
    }

    const hostedRoutes = [
      "/login",
      "/signup",
      "/reset",
      "/2fa",
      "/oidc/consent?scope=openid%20email&continue=demo",
      "/error?message=Demo%20error",
      "/invite",
    ];

    for (const path of hostedRoutes) {
      await harness.page.goto(new URL(path, harness.tenantURL).toString());
      await expect(harness.page.locator("main")).toBeVisible();
      await expectNoAxeViolations(harness.page, path);
    }
  } finally {
    await harness.close();
  }
});

async function expectNoAxeViolations(page: Page, routeName: string) {
  await page.addScriptTag({ path: axePath });
  const results = await page.evaluate(async () => {
    const axe = (window as unknown as { axe: { run: (target: Element) => Promise<unknown> } }).axe;
    return axe.run(document.body);
  });
  expect((results as { violations: unknown[] }).violations, routeName).toEqual([]);
}
