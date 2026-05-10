import { test, expect } from "@playwright/test";

import {
  configureEmailProvider,
  configurePasskeyProvider,
  configureUpstream,
  createProject,
  createTenant,
  enrollAndVerifyWebAuthn2FA,
  enrollTenantPasskey,
  redeemBootstrapToken,
  redeemTenantInviteForSession,
  reachOIDCConsentViaPassword,
  signInViaGoogleUpstream,
  signInViaPasskey,
  startHarness,
  startNextjsExample,
} from "../harness";

test("canonical demo completes inside the unattended timing budget", async () => {
  test.setTimeout(8 * 60 * 1000);
  test.skip(
    !process.env.CYPRA_E2E_LIVE && process.env.CYPRA_E2E_MANAGED !== "1",
    "set CYPRA_E2E_LIVE=1 for an existing stack or CYPRA_E2E_MANAGED=1 for a managed local stack",
  );

  const started = Date.now();
  const timings: { step: string; elapsedMs: number }[] = [];
  const timedStep = async (step: string, action: () => Promise<void>) => {
    const stepStarted = Date.now();
    await action();
    timings.push({ step, elapsedMs: Date.now() - stepStarted });
  };
  const harness = await startHarness();
  let nextjs: Awaited<ReturnType<typeof startNextjsExample>> | undefined;
  try {
    await timedStep("redeem bootstrap + enroll passkey", () =>
      redeemBootstrapToken(
        harness,
        harness.setupToken ?? process.env.CYPRA_SETUP_TOKEN ?? "cypra_setup_test",
      ),
    );
    await timedStep("create tenant", () => createTenant(harness, "acme"));
    await timedStep("create project", () => createProject(harness, "console"));
    await timedStep("configure email provider", () => configureEmailProvider(harness));
    await timedStep("configure passkey provider", () => configurePasskeyProvider(harness));
    await timedStep("configure Google upstream", () => configureUpstream(harness));
    await timedStep("Next.js downstream sign-in", async () => {
      const user = await redeemTenantInviteForSession(harness);
      await enrollTenantPasskey(harness, user.userID);
      await enrollAndVerifyWebAuthn2FA(harness, user.userID);
      nextjs = await startNextjsExample(harness);
      await reachOIDCConsentViaPassword(harness, nextjs.client, user);
      await harness.page.goto(nextjs.url);
      await harness.page.getByRole("link", { name: "Sign in with Cypra" }).click();
      await harness.page.getByRole("button", { name: "Cypra" }).click();
      await expect(harness.page.getByText("Review requested access")).toBeVisible({
        timeout: 15_000,
      });
      await harness.page.getByRole("button", { name: "Allow" }).click();
      await expect(harness.page).toHaveURL(new RegExp(`^${nextjs.url.replaceAll(".", "\\.")}`));
      const session = await harness.page.evaluate(async () => {
        const response = await fetch("/api/auth/session");
        return response.json();
      });
      expect(session?.cypra?.sub).toBeTruthy();
      expect(session?.cypra?.email).toBe(user.email);
    });
    await timedStep("hosted passkey entry", () => signInViaPasskey(harness));
    await timedStep("Google upstream sign-in", () => signInViaGoogleUpstream(harness));
    if (process.env.CYPRA_E2E_MANAGED !== "1") {
      expect(harness.page.url()).toMatch(/accounts\.google\.com|google/);
    }

    const elapsedMs = Date.now() - started;
    console.info(JSON.stringify({ event: "canonical-demo-timings", elapsedMs, timings }, null, 2));
    expect(elapsedMs).toBeLessThan(8 * 60 * 1000);
  } finally {
    // `nextjs` is closed here instead of inside the timed step so its process
    // remains available if an assertion captures page/server logs.
    await nextjs?.close();
    await harness.close();
  }
});

test("canonical demo dry-run documents the local env contract", async () => {
  const existingStack = ["CYPRA_BASE_URL", "CYPRA_TENANT_URL", "CYPRA_SETUP_TOKEN"];
  const managedStack = ["CYPRA_E2E_MANAGED"];
  expect({ existingStack, managedStack }).toEqual({
    existingStack: ["CYPRA_BASE_URL", "CYPRA_TENANT_URL", "CYPRA_SETUP_TOKEN"],
    managedStack: ["CYPRA_E2E_MANAGED"],
  });
});
