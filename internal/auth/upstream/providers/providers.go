// Package providers imports every social-SSO provider package so that their
// init() functions register them in the upstream registry. Import for side
// effects from the binary entrypoint (and from tests that need the registry
// populated).
package providers

import (
	_ "github.com/watzon/cypra/internal/auth/upstream/apple"
	_ "github.com/watzon/cypra/internal/auth/upstream/discord"
	_ "github.com/watzon/cypra/internal/auth/upstream/github"
	_ "github.com/watzon/cypra/internal/auth/upstream/google"
	_ "github.com/watzon/cypra/internal/auth/upstream/microsoft"
)
