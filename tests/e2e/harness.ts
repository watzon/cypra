import { chromium, type Browser, type BrowserContext, type Page } from "playwright";

export interface CypraHarness {
  browser: Browser;
  context: BrowserContext;
  page: Page;
  baseURL: string;
  tenantURL: string;
  close: () => Promise<void>;
}

export async function startHarness(): Promise<CypraHarness> {
  const browser = await chromium.launch();
  const context = await browser.newContext();
  await context.addInitScript(() => {
    window.localStorage.setItem("cypra.theme", "dark");
  });
  const page = await context.newPage();
  return {
    browser,
    context,
    page,
    baseURL: process.env.CYPRA_BASE_URL ?? "https://cypra.localhost",
    tenantURL: process.env.CYPRA_TENANT_URL ?? "https://acme.cypra.localhost",
    close: async () => {
      await context.close();
      await browser.close();
    },
  };
}

export async function redeemBootstrapToken(harness: CypraHarness, token: string) {
  await harness.page.goto(`${harness.baseURL}/setup/${token}`);
}

export async function createTenant(harness: CypraHarness, slug: string) {
  await harness.page.goto(`${harness.baseURL}/dashboard/tenants`);
  return { slug };
}

export async function configureEmailProvider(harness: CypraHarness) {
  await harness.page.goto(`${harness.baseURL}/dashboard/tenants/acme/settings/email`);
}

export async function configureUpstream(harness: CypraHarness) {
  await harness.page.goto(`${harness.baseURL}/dashboard/tenants/acme/settings/upstream`);
}

export async function signInViaPasskey(harness: CypraHarness) {
  await harness.page.goto(`${harness.tenantURL}/login`);
}

export async function signInViaGoogleUpstream(harness: CypraHarness) {
  await harness.page.goto(`${harness.tenantURL}/login`);
}
