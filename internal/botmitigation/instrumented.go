package botmitigation

import (
	"context"
	"sync/atomic"
)

var checksOK atomic.Uint64

var checksFail atomic.Uint64

// InstrumentedVerifier wraps a verifier and records Prometheus-style counters.
type InstrumentedVerifier struct {
	Name string
	Next Verifier
}

// Verify delegates to the wrapped verifier and records the outcome.
func (v InstrumentedVerifier) Verify(ctx context.Context, token string) error {
	next := v.Next
	if next == nil {
		next = NoopVerifier{}
	}
	if err := next.Verify(ctx, token); err != nil {
		checksFail.Add(1)
		return err
	}
	checksOK.Add(1)
	return nil
}

// Metrics returns the in-process bot-mitigation check counters.
func Metrics() (ok uint64, fail uint64) { return checksOK.Load(), checksFail.Load() }
