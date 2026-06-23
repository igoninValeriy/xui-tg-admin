package helpers

import (
	"testing"

	"xui-tg-admin/internal/models"
)

func TestBuildEmailToSubID(t *testing.T) {
	inbounds := []models.Inbound{
		{Settings: `{"clients":[{"email":"john_doe-1","subId":"SUB1"}]}`},
		{Settings: `{"clients":[{"email":"john_doe-2","subId":"SUB1"}]}`},
		{Settings: `{"clients":[{"email":"other-1","subId":"SUB2"}]}`},
		{Settings: ``},          // no settings — skipped
		{Settings: `{bad json`}, // malformed — skipped
	}

	m := BuildEmailToSubID(inbounds)

	if m["john_doe-1"] != "SUB1" || m["john_doe-2"] != "SUB1" {
		t.Errorf("expected john_doe-* -> SUB1, got %v", m)
	}
	if m["other-1"] != "SUB2" {
		t.Errorf("expected other-1 -> SUB2, got %q", m["other-1"])
	}
}

func TestUserGroupKey(t *testing.T) {
	m := map[string]string{
		"john_doe-1": "SUB1",
		"john_doe-2": "SUB1",
	}

	// Same SubID across protocols groups together
	if got := UserGroupKey("john_doe-1", m); got != "SUB1" {
		t.Errorf("UserGroupKey(john_doe-1) = %q, want SUB1", got)
	}
	if UserGroupKey("john_doe-1", m) != UserGroupKey("john_doe-2", m) {
		t.Error("clients sharing a SubID must share a group key")
	}

	// Fallback to base username preserves the full name (no cut at '_')
	if got := UserGroupKey("alice_smith-1", m); got != "alice_smith" {
		t.Errorf("UserGroupKey(alice_smith-1) = %q, want alice_smith (no truncation)", got)
	}
}
