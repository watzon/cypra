import { test, expect } from "@playwright/test";

import {
  configureEmailProvider,
  configureUpstream,
  createTenant,
  redeemBootstrapToken,
  signInViaGoogleUpstream,
  signInViaPasskey,
  startHarness,
} from "./harness";

test("examples smoke through Cypra", async () => {
  test.skip(
    !process.env.CYPRA_E2E_LIVE,
    "set CYPRA_E2E_LIVE=1 when the Cypra example stack is running",
  );

  const harness = await startHarness();
  try {
    await redeemBootstrapToken(harness, process.env.CYPRA_SETUP_TOKEN ?? "cypra_setup_test");
    await createTenant(harness, "acme");
    await configureEmailProvider(harness);
    await configureUpstream(harness);
    await signInViaPasskey(harness);
    await signInViaGoogleUpstream(harness);
    await expect(harness.page).toHaveTitle(/Cypra/);
  } finally {
    await harness.close();
  }
});
