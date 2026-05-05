import { createRoot } from "react-dom/client";

export function App() {
  return <main>Cypra dashboard foundation</main>;
}

const rootElement = document.getElementById("root");

if (rootElement) {
  createRoot(rootElement).render(<App />);
}
