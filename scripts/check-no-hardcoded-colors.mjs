import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const root = process.cwd();
const allowed = new Set(["dashboard/src/tokens.css"]);
const ignoredDirs = new Set([".git", "node_modules", "dist", "bin"]);
const checkedExtensions = new Set([".css", ".ts", ".tsx", ".js", ".jsx"]);
const hexColor = /#[0-9a-fA-F]{3,8}\b/g;
const failures = [];

function extension(path) {
  const dot = path.lastIndexOf(".");
  return dot === -1 ? "" : path.slice(dot);
}

function walk(dir) {
  for (const entry of readdirSync(dir)) {
    if (ignoredDirs.has(entry)) continue;
    const path = join(dir, entry);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      walk(path);
      continue;
    }
    const rel = relative(root, path);
    if (allowed.has(rel) || !checkedExtensions.has(extension(path))) continue;
    const content = readFileSync(path, "utf8");
    const matches = content.match(hexColor);
    if (matches) failures.push(`${rel}: ${matches.join(", ")}`);
  }
}

walk(join(root, "dashboard", "src"));

if (failures.length > 0) {
  console.error("Hardcoded hex colors are only allowed in dashboard/src/tokens.css");
  for (const failure of failures) console.error(failure);
  process.exit(1);
}
