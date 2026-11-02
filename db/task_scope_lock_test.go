package db

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestTaskScopeLock pins the client-engagement scope lock: once locked, agents and
// auto-scoping cannot widen the scope (only operator rows go in), and only
// operator rows authorize targets — rows that existed before the lock do not.
func TestTaskScopeLock(t *testing.T) {
	d, as, _ := testSetup(t)
	t.Cleanup(func() { d.Close() }) // registered first so it runs after the row cleanups

	task, err := d.CreateTask("scope lock", "goal", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.DeleteTask(task.ID) })
	stamp := time.Now().UnixNano()
	pre := fmt.Sprintf("pre%d.invalid", stamp)
	client := fmt.Sprintf("client%d.invalid", stamp)
	widened := fmt.Sprintf("widen%d.invalid", stamp)

	// Before the lock the agent may add scope (unchanged behavior).
	if _, err := as.AddAgentScope(task.ID, "root_domain", pre, "agent guess", "agent"); err != nil {
		t.Fatal(err)
	}
	if locked, err := d.TaskScopeLocked(task.ID); err != nil || locked {
		t.Fatalf("new task locked=%v err=%v", locked, err)
	}

	if err := d.LockTaskScope(task.ID, "portal request #test", "operator"); err != nil {
		t.Fatal(err)
	}
	if _, err := as.AddAgentScope(task.ID, "subdomain", "app."+client, "operator seed", "manual"); err != nil {
		t.Fatalf("operator row refused on a locked scope: %v", err)
	}
	if _, err := as.AddAgentScope(task.ID, "root_domain", widened, "agent widening", "agent"); !errors.Is(err, ErrScopeLocked) {
		t.Fatalf("agent widening a locked scope: err=%v, want ErrScopeLocked", err)
	}
	// Auto-scope from an inserted asset is silently dropped, not an error: asset
	// insertion must keep working, it just no longer grants authorization.
	if err := as.AddAutoScope(task.ID, "subdomain", "auto."+widened, "", ""); err != nil {
		t.Fatalf("auto-scope on a locked task errored: %v", err)
	}
	if err := as.AddAutoScope(task.ID, "ip", "", "", "192.0.2.77"); err != nil {
		t.Fatal(err)
	}

	locked, rules, err := d.LockedTaskScope(task.ID)
	if err != nil || !locked {
		t.Fatalf("LockedTaskScope locked=%v err=%v", locked, err)
	}
	if len(rules) != 1 || rules[0].Kind != "subdomain" || rules[0].Domain != "app."+client {
		t.Fatalf("authorizing rows=%+v, want only the operator subdomain", rules)
	}
	var widenedRows int
	if err := d.QueryRow(`SELECT count(*) FROM task_scope WHERE task_id=$1 AND (domain LIKE $2 OR net='192.0.2.77/32')`,
		task.ID, "%"+widened).Scan(&widenedRows); err != nil {
		t.Fatal(err)
	}
	if widenedRows != 0 {
		t.Fatalf("%d widening rows stored on a locked scope", widenedRows)
	}

	// Locking again keeps the original record; unlocking restores agent scoping.
	if err := d.LockTaskScope(task.ID, "second", "operator"); err != nil {
		t.Fatal(err)
	}
	if was, err := d.UnlockTaskScope(task.ID); err != nil || !was {
		t.Fatalf("unlock was=%v err=%v", was, err)
	}
	if was, _ := d.UnlockTaskScope(task.ID); was {
		t.Fatal("second unlock reported a lock")
	}
	if _, err := as.AddAgentScope(task.ID, "root_domain", widened, "after unlock", "agent"); err != nil {
		t.Fatalf("agent scope after unlock: %v", err)
	}
	if locked, rules, _ := d.LockedTaskScope(task.ID); locked || rules != nil {
		t.Fatalf("unlocked task reported locked=%v rules=%v", locked, rules)
	}
}
