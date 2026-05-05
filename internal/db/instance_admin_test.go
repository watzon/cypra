package db_test

import (
	"context"
	"testing"

	"github.com/watzon/cypra/internal/db"
)

func TestContextAsInstanceAdmin(t *testing.T) {
	ctx := db.ContextAsInstanceAdmin(context.Background())
	if !db.IsInstanceAdminContext(ctx) {
		t.Fatal("context was not marked instance-admin")
	}
}
