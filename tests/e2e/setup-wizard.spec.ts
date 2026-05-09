import { expect, test } from "@playwright/test";

interface SetupCompletePayload {
  token: string;
  email: string;
  display_name: string;
  ceremony_id: string;
  response: {
    id: string;
    rawId: string;
    type: string;
    response: {
      clientDataJSON: string;
      attestationObject: string;
      transports: string[];
    };
  };
}

test("setup wizard completes with a browser-generated WebAuthn attestation", async ({ page }) => {
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("WebAuthn.enable");
  await cdp.send("WebAuthn.addVirtualAuthenticator", {
    options: {
      protocol: "ctap2",
      transport: "internal",
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });

  let completePayload: SetupCompletePayload | undefined;

  await page.route("**/api/v1/version", async (route) => {
    await route.fulfill({ json: { version: "e2e", commit: "setup-wizard" } });
  });
  await page.route("**/api/v1/setup/verify", async (route) => {
    await route.fulfill({ json: { status: "ok" } });
  });
  await page.route("**/api/v1/setup/passkey/begin", async (route) => {
    await route.fulfill({
      json: {
        admin_id: "00000000-0000-0000-0000-00000000adm1",
        ceremony_id: "00000000-0000-0000-0000-00000000cafe",
        options: {
          publicKey: {
            challenge: "AQIDBAUGBwgJCgsMDQ4PEA",
            rp: { id: "localhost", name: "Cypra" },
            user: { id: "BAUGBwgJ", name: "root@example.com", displayName: "Root Admin" },
            pubKeyCredParams: [
              { type: "public-key", alg: -7 },
              { type: "public-key", alg: -257 },
            ],
            authenticatorSelection: {
              residentKey: "preferred",
              userVerification: "preferred",
            },
            timeout: 60_000,
            attestation: "none",
          },
        },
      },
    });
  });
  await page.route("**/api/v1/setup/complete", async (route) => {
    completePayload = route.request().postDataJSON() as SetupCompletePayload;
    await route.fulfill({
      status: 201,
      json: {
        admin_id: "00000000-0000-0000-0000-00000000adm1",
        backup_codes: ["CYPRA-A11Y", "CYPRA-B22Y", "CYPRA-C33Y"],
      },
    });
  });

  await page.goto("/setup/cypra_setup_browser");
  await page.getByLabel("Admin email").fill("root@example.com");
  await page.getByLabel("Display name").fill("Root Admin");
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByRole("button", { name: "Enroll passkey" }).click();

  await expect(page.getByText("CYPRA-A11Y")).toBeVisible();
  expect(completePayload).toBeDefined();
  expect(completePayload).toMatchObject({
    token: "cypra_setup_browser",
    email: "root@example.com",
    display_name: "Root Admin",
    ceremony_id: "00000000-0000-0000-0000-00000000cafe",
  });
  expect(completePayload?.response.type).toBe("public-key");
  expect(completePayload?.response.id).toBeTruthy();
  expect(completePayload?.response.rawId).toBeTruthy();
  expect(completePayload?.response.response.clientDataJSON).toBeTruthy();
  expect(completePayload?.response.response.attestationObject).toBeTruthy();
  expect(completePayload?.response.response.transports).toContain("internal");

  await page.getByRole("button", { name: "I have saved these" }).click();
  await page.getByRole("button", { name: "Go to dashboard" }).click();
  await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Create tenant" })).toBeVisible();
});
