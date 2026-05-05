// Package botmitigation verifies bot-challenge tokens before auth flows.
package botmitigation

import "context"

//revive:disable:exported

type Verifier interface {
	Verify(ctx context.Context, token string) error
}

type NoopVerifier struct{}

func (NoopVerifier) Verify(context.Context, string) error { return nil }
