package handlers

import (
	"testing"

	"xui-tg-admin/internal/commands"
)

func TestParseRevokeTrustedCallback(t *testing.T) {
	id, err := ParseRevokeTrustedCallback("revoke_trusted_12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 12345 {
		t.Errorf("got %d, want 12345", id)
	}

	if _, err := ParseRevokeTrustedCallback("invalid_prefix_1"); err == nil {
		t.Error("expected error for invalid prefix, got nil")
	}
	if _, err := ParseRevokeTrustedCallback("revoke_trusted_notanumber"); err == nil {
		t.Error("expected error for non-numeric id, got nil")
	}
}

func TestParseRemoveVpnCallback(t *testing.T) {
	id, err := parseRemoveVpnCallback("remove_vpn_7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 7 {
		t.Errorf("got %d, want 7", id)
	}

	if _, err := parseRemoveVpnCallback("nope_1"); err == nil {
		t.Error("expected error for invalid prefix, got nil")
	}
}

func TestGeneratePseudoTelegramID(t *testing.T) {
	a := generatePseudoTelegramID("alice")
	b := generatePseudoTelegramID("alice")
	c := generatePseudoTelegramID("bob")

	if a != b {
		t.Errorf("generatePseudoTelegramID not deterministic: %d != %d", a, b)
	}
	if a == c {
		t.Errorf("generatePseudoTelegramID collided for different inputs: %d", a)
	}
	if a <= 0 {
		t.Errorf("generatePseudoTelegramID must be positive, got %d", a)
	}
}

func TestCalculateExpiryTime(t *testing.T) {
	expiry, err := calculateExpiryTime(commands.Infinite)
	if err != nil {
		t.Fatalf("unexpected error for infinite: %v", err)
	}
	if expiry != 0 {
		t.Errorf("infinite expiry = %d, want 0", expiry)
	}

	expiry, err = calculateExpiryTime("30")
	if err != nil {
		t.Fatalf("unexpected error for 30 days: %v", err)
	}
	if expiry <= 0 {
		t.Errorf("30-day expiry = %d, want > 0", expiry)
	}

	if _, err := calculateExpiryTime("garbage"); err == nil {
		t.Error("expected error for invalid duration, got nil")
	}
}
