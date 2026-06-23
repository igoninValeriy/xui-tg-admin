package handlers

import (
	"testing"

	"xui-tg-admin/internal/commands"
)

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
