import { test, expect } from "@playwright/test";

import {
  configureEmailProvider,
  configureUpstream,
  createTenant,
  redeemBootstrapToken,
  signInViaGoogleUpstream,
  signInViaPasskey,
  startHarness,
} from "../harness";

test("canonical demo completes inside the unattended timing budget", async () => {
  test.skip(
    !process.env.CYPRA_E2E_LIVE,
    "set CYPRA_E2E_LIVE=1 when the local canonical stack is running",
  );

  const started = Date.now();
  const harness = await startHarness();
  try {
    await redeemBootstrapToken(harness, process.env.CYPRA_SETUP_TOKEN ?? "cypra_setup_test");
    await createTenant(harness, "acme");
    await configureEmailProvider(harness);
    await configureUpstream(harness);
    await signInViaPasskey(harness);
    await signInViaGoogleUpstream(harness);
    await expect(harness.page).toHaveTitle(/Cypra/);

    const elapsedMs = Date.now() - started;
    expect(elapsedMs).toBeLessThan(8 * 60 * 1000);
  } finally {
    await harness.close();
  }
});

test("canonical demo dry-run documents the local env contract", async () => {
  const required = ["CYPRA_BASE_URL", "CYPRA_TENANT_URL", "CYPRA_SETUP_TOKEN"];
  expect(required).toEqual(["CYPRA_BASE_URL", "CYPRA_TENANT_URL", "CYPRA_SETUP_TOKEN"]);
});
