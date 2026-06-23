package models

import (
	"strings"
	"testing"
)

func TestGenerateSubID(t *testing.T) {
	id := GenerateSubID()

	if id == "" {
		t.Fatal("GenerateSubID returned empty string")
	}
	if len(id) > 16 {
		t.Errorf("GenerateSubID length = %d, want <= 16", len(id))
	}
	for _, ch := range []string{"=", "+", "/"} {
		if strings.Contains(id, ch) {
			t.Errorf("GenerateSubID %q contains forbidden char %q", id, ch)
		}
	}
}

func TestGenerateSubIDUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		id := GenerateSubID()
		if _, dup := seen[id]; dup {
			t.Fatalf("GenerateSubID produced a duplicate: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestClientToDictionary(t *testing.T) {
	expiry := int64(123456)
	flow := "xtls"
	c := Client{
		ID:          "john-1",
		Enable:      true,
		Flow:        &flow,
		Email:       "john-1",
		TotalGB:     0,
		LimitIP:     0,
		ExpiryTime:  &expiry,
		Fingerprint: "fp",
		TgID:        "42",
		SubID:       "sub",
	}

	dict := c.ToDictionary()

	if dict["email"] != "john-1" {
		t.Errorf("email = %v, want john-1", dict["email"])
	}
	if dict["flow"] != "xtls" {
		t.Errorf("flow = %v, want xtls", dict["flow"])
	}
	if dict["expiryTime"] != expiry {
		t.Errorf("expiryTime = %v, want %d", dict["expiryTime"], expiry)
	}

	// Optional fields must be omitted when nil
	noOpt := Client{ID: "x", Email: "x"}
	d2 := noOpt.ToDictionary()
	if _, ok := d2["flow"]; ok {
		t.Error("flow should be omitted when nil")
	}
	if _, ok := d2["expiryTime"]; ok {
		t.Error("expiryTime should be omitted when nil")
	}
}
