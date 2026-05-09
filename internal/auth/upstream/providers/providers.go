// Package providers imports every social-SSO provider package so that their
// init() functions register them in the upstream registry. Import for side
// effects from the binary entrypoint (and from tests that need the registry
// populated).
package providers

import (
	_ "github.com/watzon/cypra/internal/auth/upstream/apple"     // register Apple provider
	_ "github.com/watzon/cypra/internal/auth/upstream/discord"   // register Discord provider
	_ "github.com/watzon/cypra/internal/auth/upstream/github"    // register GitHub provider
	_ "github.com/watzon/cypra/internal/auth/upstream/google"    // register Google provider
	_ "github.com/watzon/cypra/internal/auth/upstream/microsoft" // register Microsoft provider
)
