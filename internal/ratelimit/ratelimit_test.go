package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/ratelimit"
)

func TestLimiterRejectsAfterThreshold(t *testing.T) {
	harness := dbtest.New(t)
	limiter := ratelimit.Limiter{DB: harness.SQL}
	for range 2 {
		ok, err := limiter.Allow(context.Background(), nil, "login", "127.0.0.1", 2, time.Minute)
		if err != nil || !ok {
			t.Fatalf("allow = %v err=%v, want true", ok, err)
		}
	}
	ok, err := limiter.Allow(context.Background(), nil, "login", "127.0.0.1", 2, time.Minute)
	if err != nil {
		t.Fatalf("third allow: %v", err)
	}
	if ok {
		t.Fatal("third request unexpectedly allowed")
	}
}
