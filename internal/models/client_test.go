package models

import (
	"encoding/json"
	"regexp"
	"testing"
)

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestGenerateSubID(t *testing.T) {
	id := GenerateSubID()

	if !hex32.MatchString(id) {
		t.Errorf("GenerateSubID() = %q, want 32 lowercase hex characters", id)
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

// TestClientMarshalTgIDIsNumber pins the field that breaks the panel: 3x-ui
// decodes tgId into an int64 and rejects a JSON string with
// "cannot unmarshal string into Go struct field .tgId of type int64".
func TestClientMarshalTgIDIsNumber(t *testing.T) {
	c := Client{
		Email:      "john",
		Enable:     true,
		ExpiryTime: 123456,
		TgID:       42,
		SubID:      "0123456789abcdef0123456789abcdef",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := string(raw["tgId"]); got != "42" {
		t.Errorf("tgId = %s, want the bare number 42", got)
	}
	if got := string(raw["expiryTime"]); got != "123456" {
		t.Errorf("expiryTime = %s, want 123456", got)
	}
	if _, ok := raw["email"]; !ok {
		t.Error("email must always be sent")
	}
	if _, ok := raw["totalGB"]; !ok {
		t.Error("totalGB must always be sent, including the 0 that means unlimited")
	}
}

// TestClientMarshalOmitsGeneratedSecrets checks that the fields the panel can
// mint itself are left out when empty rather than sent as "".
func TestClientMarshalOmitsGeneratedSecrets(t *testing.T) {
	data, err := json.Marshal(Client{Email: "john", Enable: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, field := range []string{"id", "subId", "flow", "comment"} {
		if _, ok := raw[field]; ok {
			t.Errorf("%q should be omitted when empty so the panel generates it", field)
		}
	}
}
