package server

import (
	"testing"
	"time"
)

func resetAbuse(users ...int64) {
	abuse.Lock()
	for _, u := range users {
		delete(abuse.events, u)
		delete(abuse.suspended, u)
	}
	abuse.Unlock()
}

func TestAbuseRecordTripsAtThreshold(t *testing.T) {
	const u = int64(991001)
	resetAbuse(u)
	now := time.Now()
	// authz.forbidden weighs 2; three of them reach the threshold of 6, so the
	// third crosses it.
	if _, trip := abuseRecord(u, "authz.forbidden", now); trip {
		t.Fatal("tripped on the first signal")
	}
	if _, trip := abuseRecord(u, "authz.forbidden", now.Add(time.Second)); trip {
		t.Fatal("tripped on the second signal")
	}
	if s, trip := abuseRecord(u, "authz.forbidden", now.Add(2*time.Second)); !trip || s < abuseThreshold {
		t.Fatalf("should trip on the third signal: score=%d trip=%v", s, trip)
	}
	// Dedupe: another signal within the window must not re-trip.
	if _, trip := abuseRecord(u, "authz.forbidden", now.Add(3*time.Second)); trip {
		t.Fatal("re-tripped within the window")
	}
}

func TestAbuseRecordDecaysOutsideWindow(t *testing.T) {
	const u = int64(991002)
	resetAbuse(u)
	now := time.Now()
	abuseRecord(u, "scope.blocked", now) // weight 3
	// After the window the old event decays; a lone weight-3 signal is below the
	// threshold, so no trip.
	if s, trip := abuseRecord(u, "scope.blocked", now.Add(abuseWindow+time.Minute)); trip || s != 3 {
		t.Fatalf("stale event should have decayed: score=%d trip=%v", s, trip)
	}
}

func TestAbuseRecordIsPerUser(t *testing.T) {
	a, b := int64(991003), int64(991004)
	resetAbuse(a, b)
	now := time.Now()
	abuseRecord(a, "scope.blocked", now)
	abuseRecord(a, "scope.blocked", now)
	if _, trip := abuseRecord(b, "scope.blocked", now); trip {
		t.Fatal("user b tripped from user a's signals")
	}
}

func TestAbuseAutoSuspendDefaultsOff(t *testing.T) {
	t.Setenv("NEXUS_ABUSE_AUTOSUSPEND", "")
	if abuseAutoSuspendEnabled() {
		t.Fatal("auto-suspend must default to OFF (shadow mode)")
	}
	t.Setenv("NEXUS_ABUSE_AUTOSUSPEND", "true")
	if !abuseAutoSuspendEnabled() {
		t.Fatal("auto-suspend should be ON when set to true")
	}
	t.Setenv("NEXUS_ABUSE_AUTOSUSPEND", "1")
	if !abuseAutoSuspendEnabled() {
		t.Fatal("auto-suspend should be ON when set to 1")
	}
}
