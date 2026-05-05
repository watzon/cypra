// Package fuzz contains the reusable tenant-isolation fuzzer harness that later
// phases use to enroll HTTP handlers.
package fuzz

//revive:disable:exported

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/db"
)

type Outcome struct {
	Rows int
	Err  error
}

type Handler func(ctx context.Context) Outcome

type Harness struct {
	ProtectedTenant uuid.UUID
	OtherTenant     uuid.UUID
}

func (h Harness) AssertIsolated(ctx context.Context, handler Handler) error {
	if h.ProtectedTenant == uuid.Nil || h.OtherTenant == uuid.Nil {
		return fmt.Errorf("tenant-isolation fuzzer requires two tenants")
	}
	mutated := db.ContextWithTenant(ctx, h.OtherTenant)
	outcome := handler(mutated)
	if outcome.Err != nil {
		return nil
	}
	if outcome.Rows != 0 {
		return fmt.Errorf("tenant-isolation violation: handler returned %d cross-tenant rows", outcome.Rows)
	}
	return nil
}
