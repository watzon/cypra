import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import axe from "axe-core";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "./App";
import {
  BackupCodeGrid,
  Button,
  IdentifierPill,
  MaskedSecret,
  PrimitiveGallery,
  StatusPip,
  TextInput,
} from "./components";
import { ThemeProvider } from "./theme";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function defaultFetchMock() {
  return vi.fn((input: RequestInfo | URL) => {
    void input;
    return Promise.resolve(
      new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
    );
  });
}

function dashboardFetchMock() {
  return vi.fn((input: RequestInfo | URL) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    if (url === "/api/v1/tenants/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000acme",
              slug: "acme",
              name: "Acme Operations",
              branding: { display_name: "Acme Login", powered_by: true },
              member_count: 3,
              project_count: 2,
              user_count: 128,
              email_provider_required: true,
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/projects/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000app1",
              slug: "console",
              name: "Console App",
              client_id: "client_cypra_acme_console",
              issuer_url: "https://acme.cypra.localhost",
              redirect_uris: ["https://app.example.com/api/auth/callback/cypra"],
              allowed_scopes: ["openid", "email"],
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/pats/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000pat1",
              name: "Deploy automation",
              scopes: ["tenants:read"],
              created_at: "2026-05-01",
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url === "/api/v1/users/") {
      return Promise.resolve(
        new Response(
          JSON.stringify([
            {
              id: "00000000-0000-0000-0000-00000000ada1",
              email: "ada@example.com",
              sub: "usr_acme_ada",
              state: "active",
              enrolled_methods: ["passkey"],
            },
          ]),
          { status: 200 },
        ),
      );
    }
    if (url.includes("/branding")) {
      return Promise.resolve(new Response(JSON.stringify({}), { status: 200 }));
    }
    return Promise.resolve(
      new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
    );
  });
}

function renderApp(
  path = "/dashboard",
  fetchMock: (input: RequestInfo | URL) => Promise<Response> = defaultFetchMock(),
) {
  history.replaceState(null, "", path);
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <ThemeProvider>
      <QueryClientProvider client={client}>
        <App />
      </QueryClientProvider>
    </ThemeProvider>,
  );
}

describe("App", () => {
  it("renders the dashboard overview route", () => {
    renderApp("/dashboard");

    expect(screen.getByRole("heading", { name: "Dashboard" })).toBeInTheDocument();
    expect(screen.getByText("Signing key health")).toBeInTheDocument();
  });

  it("renders setup wizard by token route", () => {
    renderApp("/setup/cypra_setup_test");

    expect(screen.getByRole("heading", { name: "Set up Cypra" })).toBeInTheDocument();
    expect(screen.getByLabelText("Setup token: cypra_setup_test")).toBeInTheDocument();
  });

  it("renders the primitive gallery", () => {
    renderApp("/__cypra/gallery");

    expect(screen.getByRole("heading", { name: "Primitive gallery" })).toBeInTheDocument();
    expect(screen.getByText("Light mode parity")).toBeInTheDocument();
    expect(screen.getByText("Dark mode parity")).toBeInTheDocument();
  });

  it("has no axe violations on the gallery route", async () => {
    renderApp("/__cypra/gallery");

    const results = await axe.run(document.body);

    expect(results.violations).toEqual([]);
  });

  it("persists theme changes through the actor metadata endpoint", async () => {
    const fetchMock = defaultFetchMock();
    renderApp("/dashboard/account", fetchMock);

    await userEvent.click(screen.getAllByRole("button", { name: "light" })[0]);

    expect(localStorage.getItem("cypra.theme")).toBe("light");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/instance/admins/me",
      expect.objectContaining({ method: "PATCH" }),
    );
  });

  it("renders the searchable tenant list", async () => {
    renderApp("/dashboard/tenants", dashboardFetchMock());

    expect(await screen.findByRole("heading", { name: "Tenants" })).toBeInTheDocument();
    expect((await screen.findAllByText("Acme Operations")).length).toBeGreaterThan(0);

    const searchInputs = screen.getAllByLabelText("Search tenants");
    await userEvent.type(searchInputs[searchInputs.length - 1], "missing");

    expect(screen.getByText("No tenants match missing.")).toBeInTheDocument();
  });

  it("renders tenant overview and branding settings routes", async () => {
    renderApp("/dashboard/tenants/acme/settings/branding", dashboardFetchMock());

    expect(await screen.findByRole("heading", { name: "Acme Login" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Branding" })).toBeInTheDocument();
    expect(screen.getByLabelText("Accent color")).toBeInTheDocument();
  });

  it("renders project detail, auth methods, and API tokens surfaces", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/projects/console", fetchMock);

    expect(await screen.findByRole("heading", { name: "Console App" })).toBeInTheDocument();
    expect(screen.getByText("Rotate client secret")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/auth-methods", fetchMock);

    expect(await screen.findByText("Email + Password")).toBeInTheDocument();
    expect(screen.getByText("OIDC upstreams")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/settings/api-tokens", fetchMock);

    expect(await screen.findByRole("heading", { name: "API tokens" })).toBeInTheDocument();
    expect(screen.getByText("Deploy automation")).toBeInTheDocument();
  });

  it("renders user list and user detail surfaces", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/users", fetchMock);

    expect(await screen.findByRole("heading", { name: "Users" })).toBeInTheDocument();
    expect(screen.getByText("ada@example.com")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/users/00000000-0000-0000-0000-00000000ada1", fetchMock);

    expect(await screen.findByRole("heading", { name: "ada@example.com" })).toBeInTheDocument();
    expect(screen.getByText("Reset password")).toBeInTheDocument();
  });

  it("renders members, signing keys, and audit surfaces", async () => {
    const fetchMock = dashboardFetchMock();
    renderApp("/dashboard/tenants/acme/settings/members", fetchMock);

    expect(await screen.findByRole("heading", { name: "Members & roles" })).toBeInTheDocument();
    expect(screen.getByText("Pending invites")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/signing-keys", fetchMock);

    expect(await screen.findByRole("heading", { name: "Signing keys" })).toBeInTheDocument();
    expect(screen.getByText("Rotate now")).toBeInTheDocument();

    cleanup();
    renderApp("/dashboard/tenants/acme/audit", fetchMock);

    expect(await screen.findByRole("heading", { name: "Tenant audit" })).toBeInTheDocument();
    expect(screen.getByText("Export NDJSON")).toBeInTheDocument();
  });

  it("has no axe violations on Phase 9 dashboard routes", async () => {
    const routes = [
      ["/dashboard/tenants", "Tenants"],
      ["/dashboard/tenants/acme/settings/branding", "Branding"],
      ["/dashboard/tenants/acme/projects/console", "Console App"],
      ["/dashboard/tenants/acme/auth-methods", "Email + Password"],
      ["/dashboard/tenants/acme/users", "Users"],
      ["/dashboard/tenants/acme/settings/members", "Members & roles"],
      ["/dashboard/tenants/acme/signing-keys", "Signing keys"],
      ["/dashboard/tenants/acme/audit", "Tenant audit"],
      ["/dashboard/tenants/acme/settings/api-tokens", "API tokens"],
    ] as const;

    for (const [route, heading] of routes) {
      renderApp(route, dashboardFetchMock());
      await screen.findByRole("heading", { name: heading });

      const results = await axe.run(document.body);

      expect(results.violations, route).toEqual([]);
      cleanup();
    }
  });
});

describe("primitives", () => {
  it("renders button variants and loading state", () => {
    render(<Button loading>Save</Button>);

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("renders text input error state", () => {
    render(<TextInput label="Email" error="Email is required." />);

    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByText("Email is required.")).toBeInTheDocument();
  });

  it("refuses masked secret copy until revealed", async () => {
    render(<MaskedSecret name="client_secret" value="real-secret" />);

    await userEvent.click(screen.getByRole("button", { name: "Copy secret" }));

    expect(screen.getByText("Reveal first to copy")).toBeInTheDocument();
  });

  it("copies identifier pills", async () => {
    render(<IdentifierPill value="client_0123456789abcdef" label="Client ID" />);

    const copyButtons = screen.getAllByRole("button", { name: "Copy Client ID" });
    const copyButton = copyButtons[0];
    await userEvent.click(copyButton);

    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it("renders status variants with labels", () => {
    render(<StatusPip variant="pending" label="Pending" />);

    expect(screen.getByText("Pending")).toBeInTheDocument();
  });

  it("renders backup codes with confirmation gate", async () => {
    render(<BackupCodeGrid codes={["CYPRA-1111", "CYPRA-2222"]} />);

    const savedButtons = screen.getAllByRole("button", { name: "I've saved these" });
    const savedButton = savedButtons[0];
    await userEvent.click(savedButton);

    expect(screen.getByRole("button", { name: "Saved" })).toBeInTheDocument();
  });

  it("renders all gallery composites", () => {
    render(
      <ThemeProvider>
        <PrimitiveGallery />
      </ThemeProvider>,
    );

    expect(screen.getAllByText("Permission matrix").length).toBeGreaterThan(0);
  });
});
