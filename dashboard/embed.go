// Package dashboard embeds the built dashboard SPA into the Cypra binary.
package dashboard

import "embed"

// Files is the embedded Vite build output. The explicit index pattern makes a
// raw Go build fail loudly until `make build-frontend` has populated dist.
//
//go:embed dist/index.html dist/assets/*
var Files embed.FS
