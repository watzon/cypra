import { test } from "@playwright/test";

import {
  configureEmailProvider,
  createProject,
  createTenant,
  enrollTenantPasskey,
  exportManagedBackup,
  redeemBootstrapToken,
  redeemTenantInviteForSession,
  signInViaPasskey,
  startHarness,
  startRestoredManagedStack,
} from "./harness";

test("backup import restores browser-generated tenant passkeys", async () => {
  test.setTimeout(8 * 60 * 1000);
  test.skip(process.env.CYPRA_E2E_MANAGED !== "1", "set CYPRA_E2E_MANAGED=1");

  const harness = await startHarness();
  let backup: Awaited<ReturnType<typeof exportManagedBackup>> | undefined;
  let restored: Awaited<ReturnType<typeof startRestoredManagedStack>> | undefined;
  try {
    await redeemBootstrapToken(harness, harness.setupToken ?? "cypra_setup_test");
    await createTenant(harness, "acme");
    await createProject(harness, "console");
    await configureEmailProvider(harness);
    const user = await redeemTenantInviteForSession(harness);
    await enrollTenantPasskey(harness, user.userID);

    backup = await exportManagedBackup(harness);
    restored = await startRestoredManagedStack(backup);
    harness.baseURL = restored.baseURL;
    harness.tenantURL = restored.tenantURL;

    await signInViaPasskey(harness);
  } finally {
    await restored?.close();
    await backup?.close();
    await harness.close();
  }
});
