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
  return vi.fn(() =>
    Promise.resolve(
      new Response(JSON.stringify({ version: "dev", commit: "test" }), { status: 200 }),
    ),
  );
}

function renderApp(path = "/dashboard", fetchMock = defaultFetchMock()) {
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
