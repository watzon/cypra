# ADR-0010: Bun Workspaces with pnpm Fallback

Status: accepted

Cypra will use pinned Bun workspaces for frontend reproducibility while documenting a pnpm fallback path if Bun creates CI risk.

## Context

The dashboard is a Vite React SPA embedded into the Go binary. The repository needs one install command that works for Go and frontend contributors, while keeping CI deterministic.

## Decision

Use Bun workspaces as the primary package manager. The root `package.json` owns the workspace list and delegates dashboard build, lint, typecheck, and test scripts through `bun run --filter dashboard ...`. The root lockfile is committed and `make install` uses `bun install --frozen-lockfile`.

If Bun workspace resolution blocks CI, the fallback is pnpm with the same workspace shape and script names. The fallback must preserve the root script contract before it is merged.

## Consequences

Frontend dependencies install from the repository root, matching CI and local development. The dashboard package remains independently runnable for Vite dev server work, but the root scripts are the supported automation surface.
