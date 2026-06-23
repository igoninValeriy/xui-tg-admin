package validation

import (
	"strings"
	"testing"

	"xui-tg-admin/internal/constants"
)

func TestValidateUsername(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"too short", "ab", true},
		{"min length", "abc", false},
		{"with underscore and digits", "john_doe123", false},
		{"with dash", "my-vpn-2024", false},
		{"invalid char", "ab!", true},
		{"with space", "a b c", true},
		{"slash rejected", "a/b/c", true},
		{"empty", "", true},
		{"max length", strings.Repeat("a", constants.MaxUsernameLength), false},
		{"over max length", strings.Repeat("a", constants.MaxUsernameLength+1), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUsername(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	cases := []struct {
		input    string
		wantDays int
		wantErr  bool
	}{
		{"30", 30, false},
		{"1", 1, false},
		{"3650", 3650, false},
		{"0", 0, true},
		{"-5", 0, true},
		{"3651", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tc := range cases {
		days, err := ValidateDuration(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateDuration(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && days != tc.wantDays {
			t.Errorf("ValidateDuration(%q) = %d, want %d", tc.input, days, tc.wantDays)
		}
	}
}
