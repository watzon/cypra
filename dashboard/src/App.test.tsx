import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { App } from "./App";

describe("App", () => {
  it("renders the dashboard foundation", () => {
    render(<App />);

    expect(screen.getByText("Cypra dashboard foundation")).toBeTruthy();
  });
});
