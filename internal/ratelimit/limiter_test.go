package ratelimit_test

import (
	"testing"
	"time"

	"github.com/religiosa1/auth_server/internal/ratelimit"
)

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestLimiter_NotBlockedBeforeThreshold(t *testing.T) {
	now := time.Now()
	l := ratelimit.NewWithClock(ratelimit.Config{MaxAttempts: 10, Window: time.Minute}, fixedClock(now))

	for i := 0; i < 9; i++ {
		l.RecordFailure("1.2.3.4")
	}
	if l.IsBlocked("1.2.3.4") {
		t.Fatal("expected not blocked after 9 failures (threshold is 10)")
	}
}

func TestLimiter_BlockedAtThreshold(t *testing.T) {
	now := time.Now()
	l := ratelimit.NewWithClock(ratelimit.Config{MaxAttempts: 10, Window: time.Minute}, fixedClock(now))

	for i := 0; i < 10; i++ {
		l.RecordFailure("1.2.3.4")
	}
	if !l.IsBlocked("1.2.3.4") {
		t.Fatal("expected blocked after 10 failures")
	}
}

func TestLimiter_WindowResetUnblocks(t *testing.T) {
	var current time.Time
	clock := func() time.Time { return current }
	window := 10 * time.Minute

	l := ratelimit.NewWithClock(ratelimit.Config{MaxAttempts: 3, Window: window}, clock)

	for i := 0; i < 3; i++ {
		l.RecordFailure("1.2.3.4")
	}
	if !l.IsBlocked("1.2.3.4") {
		t.Fatal("expected blocked before window expiry")
	}

	current = current.Add(window + time.Second)
	if l.IsBlocked("1.2.3.4") {
		t.Fatal("expected unblocked after window reset")
	}
}

func TestLimiter_DifferentIPsAreIndependent(t *testing.T) {
	now := time.Now()
	l := ratelimit.NewWithClock(ratelimit.Config{MaxAttempts: 3, Window: time.Minute}, fixedClock(now))

	for i := 0; i < 3; i++ {
		l.RecordFailure("1.2.3.4")
	}
	if l.IsBlocked("5.6.7.8") {
		t.Fatal("blocking 1.2.3.4 should not affect 5.6.7.8")
	}
}

func TestLimiter_PortStripping(t *testing.T) {
	now := time.Now()
	l := ratelimit.NewWithClock(ratelimit.Config{MaxAttempts: 3, Window: time.Minute}, fixedClock(now))

	l.RecordFailure("1.2.3.4:5678")
	l.RecordFailure("1.2.3.4")
	l.RecordFailure("1.2.3.4:9999")

	if !l.IsBlocked("1.2.3.4") {
		t.Fatal("port variants should map to the same bucket")
	}
}

func TestLimiter_IPv6WithPort(t *testing.T) {
	now := time.Now()
	l := ratelimit.NewWithClock(ratelimit.Config{MaxAttempts: 2, Window: time.Minute}, fixedClock(now))

	l.RecordFailure("[::1]:8080")
	l.RecordFailure("::1")

	if !l.IsBlocked("::1") {
		t.Fatal("IPv6 with port should map to the same bucket as bare IPv6")
	}
}
