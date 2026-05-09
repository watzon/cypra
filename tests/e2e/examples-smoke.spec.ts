import { test, expect } from "@playwright/test";

import {
  configureEmailProvider,
  configureUpstream,
  createProject,
  createTenant,
  enrollAndVerifyWebAuthn2FA,
  enrollTenantPasskey,
  redeemBootstrapToken,
  redeemTenantInviteForSession,
  reachOIDCConsentViaGoogle,
  reachOIDCConsentViaMagicLink,
  reachOIDCConsentViaPasskey,
  reachOIDCConsentViaPassword,
  signInViaGoogleUpstream,
  signInViaPasskey,
  startHarness,
  startNextjsExample,
} from "./harness";

test("examples smoke through Cypra", async () => {
  test.setTimeout(8 * 60 * 1000);
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
    await createProject(harness, "console");
    await configureEmailProvider(harness);
    await configureUpstream(harness);
    const user = await redeemTenantInviteForSession(harness);
    await enrollTenantPasskey(harness, user.userID);
    await enrollAndVerifyWebAuthn2FA(harness, user.userID);
    const nextjs = await startNextjsExample(harness);
    try {
      await reachOIDCConsentViaPassword(harness, nextjs.client, user);
      await reachOIDCConsentViaMagicLink(harness, nextjs.client, user);
      await reachOIDCConsentViaPasskey(harness, nextjs.client);
      await reachOIDCConsentViaGoogle(harness, nextjs.client);
      await harness.context.clearCookies();
      await harness.page.goto(nextjs.url);
      await harness.page.getByRole("link", { name: "Sign in with Cypra" }).click();
      await harness.page.getByRole("button", { name: "Cypra" }).click();
      if (await harness.page.getByRole("heading", { name: "Sign in" }).isVisible()) {
        await harness.page.getByRole("button", { name: "Use passkey" }).click();
      }
      try {
        await expect(
          harness.page.getByRole("heading", { name: "Sign in to this application" }),
        ).toBeVisible({ timeout: 15_000 });
      } catch (error) {
        throw new Error(
          `Next.js sign-in did not reach Cypra consent at ${harness.page.url()}\n${await harness.page.locator("body").innerText()}\n${nextjs.logs()}\n${String(error)}`,
        );
      }
      await harness.page.getByRole("button", { name: "Allow" }).click();
      await expect(harness.page).toHaveURL(new RegExp(`^${nextjs.url.replaceAll(".", "\\.")}`));
      const session = await harness.page.evaluate(async () => {
        const response = await fetch("/api/auth/session");
        return response.json();
      });
      if (!session?.cypra?.sub) {
        throw new Error(
          `Auth.js session missing Cypra claims at ${harness.page.url()}: ${JSON.stringify(session)}\n${await harness.page.locator("body").innerText()}\n${nextjs.logs()}`,
        );
      }
      expect(session?.cypra?.sub).toBeTruthy();
      expect(session?.cypra?.email).toBe(user.email);
    } finally {
      await nextjs.close();
    }
    await signInViaPasskey(harness);
    await signInViaGoogleUpstream(harness);
    if (process.env.CYPRA_E2E_MANAGED !== "1") {
      expect(harness.page.url()).toMatch(/accounts\.google\.com|google/);
    }
  } finally {
    await harness.close();
  }
});
