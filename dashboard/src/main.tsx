import { createRoot } from "react-dom/client";

import { App } from "@/App";
import "@/index.css";
import { QueryProvider } from "@/query";
import { ThemeProvider } from "@/theme";

if (import.meta.env.DEV) {
  void import("axe-core").then(({ default: axe }) => {
    window.setTimeout(() => {
      void axe
        .run(document)
        .then((results) => {
          if (results.violations.length > 0) {
            console.warn("axe violations", results.violations);
          }
        })
        .catch((error: unknown) => console.error(error));
    }, 500);
  });
}

const rootElement = document.getElementById("root");

if (rootElement) {
  createRoot(rootElement).render(
    <ThemeProvider>
      <QueryProvider>
        <App />
      </QueryProvider>
    </ThemeProvider>,
  );
}
