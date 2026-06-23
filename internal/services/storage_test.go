package services

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func newTestStorage(t *testing.T) (*StorageService, string) {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	path := filepath.Join(t.TempDir(), "data.json")
	return NewStorageService(path, logger), path
}

func TestStorageTrustedUsers(t *testing.T) {
	s, _ := newTestStorage(t)

	if err := s.AddTrusted(100, "alice"); err != nil {
		t.Fatalf("AddTrusted: %v", err)
	}
	if !s.IsTrusted(100) {
		t.Error("IsTrusted(100) = false, want true")
	}
	if s.IsTrusted(200) {
		t.Error("IsTrusted(200) = true, want false")
	}

	ok, id := s.IsTrustedByUsername("alice")
	if !ok || id != 100 {
		t.Errorf("IsTrustedByUsername(alice) = (%v, %d), want (true, 100)", ok, id)
	}

	// Adding the same telegram ID must not duplicate
	if err := s.AddTrusted(100, "alice"); err != nil {
		t.Fatalf("AddTrusted (dup): %v", err)
	}
	if got := len(s.GetTrustedUsers()); got != 1 {
		t.Errorf("GetTrustedUsers len = %d, want 1", got)
	}

	if err := s.UpdateTrustedUserTelegramID("alice", 999); err != nil {
		t.Fatalf("UpdateTrustedUserTelegramID: %v", err)
	}
	if !s.IsTrusted(999) {
		t.Error("IsTrusted(999) = false after update, want true")
	}

	if err := s.RemoveTrusted(999); err != nil {
		t.Fatalf("RemoveTrusted: %v", err)
	}
	if s.IsTrusted(999) {
		t.Error("IsTrusted(999) = true after removal, want false")
	}
	if got := len(s.GetTrustedUsers()); got != 0 {
		t.Errorf("GetTrustedUsers len = %d after removal, want 0", got)
	}
}

func TestStorageVpnAccounts(t *testing.T) {
	s, _ := newTestStorage(t)

	if err := s.AddVpnAccount("acc1", 100); err != nil {
		t.Fatalf("AddVpnAccount: %v", err)
	}
	if err := s.AddVpnAccount("acc2", 100); err != nil {
		t.Fatalf("AddVpnAccount: %v", err)
	}
	if err := s.AddVpnAccount("other", 200); err != nil {
		t.Fatalf("AddVpnAccount: %v", err)
	}

	if got := s.GetUserAccountCount(100); got != 2 {
		t.Errorf("GetUserAccountCount(100) = %d, want 2", got)
	}

	accounts := s.GetUserAccounts(100)
	if len(accounts) != 2 {
		t.Fatalf("GetUserAccounts(100) len = %d, want 2", len(accounts))
	}

	// Removing with the wrong owner must be a no-op
	if err := s.RemoveVpnAccount(accounts[0].ID, 200); err != nil {
		t.Fatalf("RemoveVpnAccount (wrong owner): %v", err)
	}
	if got := s.GetUserAccountCount(100); got != 2 {
		t.Errorf("GetUserAccountCount(100) = %d after wrong-owner removal, want 2", got)
	}

	// Removing with the right owner works
	if err := s.RemoveVpnAccount(accounts[0].ID, 100); err != nil {
		t.Fatalf("RemoveVpnAccount: %v", err)
	}
	if got := s.GetUserAccountCount(100); got != 1 {
		t.Errorf("GetUserAccountCount(100) = %d after removal, want 1", got)
	}
}

func TestStoragePersistence(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	path := filepath.Join(t.TempDir(), "data.json")

	s1 := NewStorageService(path, logger)
	if err := s1.AddTrusted(7, "bob"); err != nil {
		t.Fatalf("AddTrusted: %v", err)
	}
	if err := s1.AddVpnAccount("bob-add1", 7); err != nil {
		t.Fatalf("AddVpnAccount: %v", err)
	}

	// A fresh service reading the same file must see the persisted data
	s2 := NewStorageService(path, logger)
	if !s2.IsTrusted(7) {
		t.Error("reloaded storage: IsTrusted(7) = false, want true")
	}
	if got := s2.GetUserAccountCount(7); got != 1 {
		t.Errorf("reloaded storage: GetUserAccountCount(7) = %d, want 1", got)
	}
}
