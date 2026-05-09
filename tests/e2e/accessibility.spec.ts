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
      ["/dashboard", "Dashboard"],
      ["/dashboard/account", "Account"],
      ["/dashboard/tenants", "Tenants"],
      ["/dashboard/tenants/acme", "Acme Login"],
      ["/dashboard/tenants/acme/projects", "Projects"],
      ["/dashboard/tenants/acme/projects/console", "Console App"],
      ["/dashboard/tenants/acme/users", "Users"],
      ["/dashboard/tenants/acme/users/00000000-0000-0000-0000-00000000ada1", "ada@example.com"],
      ["/dashboard/tenants/acme/auth-methods", "Email + Password"],
      ["/dashboard/tenants/acme/signing-keys", "Signing keys"],
      ["/dashboard/tenants/acme/audit", "Tenant audit"],
      ["/dashboard/tenants/acme/settings/branding", "Branding"],
      ["/dashboard/tenants/acme/settings/api-tokens", "API tokens"],
      ["/dashboard/tenants/acme/settings/members", "Members & roles"],
      ["/dashboard/tenants/acme/settings/email", "Email provider"],
      ["/dashboard/tenants/acme/settings/upstream", "Google upstream"],
      ["/dashboard/tenants/acme/settings/danger", "Suspend tenant"],
      ["/dashboard/instance/admins", "Instance admins"],
      ["/dashboard/instance/diagnostics", "Diagnostics"],
      ["/dashboard/instance/audit", "Instance audit"],
    ] as const;

    for (const [path, heading] of dashboardRoutes) {
      const url = new URL(path, harness.baseURL);
      url.searchParams.set("state", "demo");
      await harness.page.goto(url.toString());
      await expect(harness.page.getByRole("heading", { name: heading })).toBeVisible();
      await expectNoAxeViolations(harness.page, path);
    }

    const hostedRoutes = [
      ["/login", "Sign in"],
      ["/signup", "Create account"],
      ["/reset", "Reset password"],
      ["/2fa", "Two-factor authentication"],
      ["/oidc/consent?scope=openid%20email&continue=demo", "Sign in to this application"],
      ["/error?message=Demo%20error", "Sign in could not continue."],
      ["/invite", "Accept invite"],
    ] as const;

    for (const [path, heading] of hostedRoutes) {
      await harness.page.goto(new URL(path, harness.tenantURL).toString());
      await expect(harness.page.getByRole("heading", { name: heading })).toBeVisible();
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
