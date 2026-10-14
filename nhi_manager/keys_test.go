package nhi_manager

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistentKeysSurviveReloadAndRejectCorruption(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "secrets")
	first, err := loadManagerKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadManagerKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.ca.CACertPEM(), second.ca.CACertPEM()) || !bytes.Equal(first.jwt, second.jwt) || !bytes.Equal(first.audit.PublicKeyBytes(), second.audit.PublicKeyBytes()) {
		t.Fatal("keys changed after reload")
	}
	token, err := IssueJWT(first.jwt, &NHI{AgentID: "test", DID: NewDID()}, "", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyJWT(second.jwt, token); err != nil {
		t.Fatal("persisted JWT key rejected original token")
	}
	path := filepath.Join(dir, "manager-keys.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("key bundle permissions are not private")
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadManagerKeys(dir); err == nil {
		t.Fatal("public key bundle accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadManagerKeys(dir); err == nil {
		t.Fatal("corrupt key bundle accepted")
	}
	unchanged, _ := os.ReadFile(path)
	if !bytes.Equal(unchanged, []byte(`{"version":1}`)) {
		t.Fatal("corrupt keys were silently replaced")
	}
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(dir, linked); err != nil {
		t.Fatal(err)
	}
	if _, err := loadManagerKeys(linked); err == nil {
		t.Fatal("symlink key directory accepted")
	}
}

func TestConcurrentKeyInitializationUsesOneCompleteBundle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "secrets")
	type result struct {
		keys *managerKeys
		err  error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() { keys, err := loadManagerKeys(dir); results <- result{keys, err} }()
	}
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatal("concurrent key initialization failed", first.err, second.err)
	}
	if !bytes.Equal(first.keys.jwt, second.keys.jwt) || !bytes.Equal(first.keys.audit.PublicKeyBytes(), second.keys.audit.PublicKeyBytes()) || !bytes.Equal(first.keys.ca.CACertPEM(), second.keys.ca.CACertPEM()) {
		t.Fatal("initializers received different key bundles")
	}
}
