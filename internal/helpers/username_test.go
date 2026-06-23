package helpers

import "testing"

func TestExtractBaseUsername(t *testing.T) {
	cases := map[string]string{
		"john-1":         "john",
		"john_doe-2":     "john_doe",
		"john":           "john",
		"a-b-c-3":        "a-b-c",
		"user-add1-1":    "user-add1",
		"user-add1":      "user-add1", // "add1" is not purely numeric
		"":               "",
		"name-007":       "name",
		"trailing-dash-": "trailing-dash-",
	}

	for input, want := range cases {
		if got := ExtractBaseUsername(input); got != want {
			t.Errorf("ExtractBaseUsername(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsEmailMatchingBaseUsername(t *testing.T) {
	cases := []struct {
		email, base string
		want        bool
	}{
		{"john-1", "john", true},
		{"john-2", "john", true},
		{"john", "john", true},
		{"jane-1", "john", false},
		{"user-add1-1", "user-add1", true},
	}

	for _, tc := range cases {
		if got := IsEmailMatchingBaseUsername(tc.email, tc.base); got != tc.want {
			t.Errorf("IsEmailMatchingBaseUsername(%q, %q) = %v, want %v", tc.email, tc.base, got, tc.want)
		}
	}
}

func TestFormatEmailWithInboundNumber(t *testing.T) {
	if got := FormatEmailWithInboundNumber("john", 1); got != "john-1" {
		t.Errorf("got %q, want %q", got, "john-1")
	}
	if got := FormatEmailWithInboundNumber("a-b", 12); got != "a-b-12" {
		t.Errorf("got %q, want %q", got, "a-b-12")
	}
}

func TestExtractFormatRoundTrip(t *testing.T) {
	base := "john_doe"
	for i := 1; i <= 5; i++ {
		email := FormatEmailWithInboundNumber(base, i)
		if got := ExtractBaseUsername(email); got != base {
			t.Errorf("round trip for %q: got %q, want %q", email, got, base)
		}
	}
}

func TestIsNumeric(t *testing.T) {
	cases := map[string]bool{
		"123": true,
		"0":   true,
		"":    false,
		"12a": false,
		"-1":  false,
		" 1":  false,
	}
	for input, want := range cases {
		if got := IsNumeric(input); got != want {
			t.Errorf("IsNumeric(%q) = %v, want %v", input, got, want)
		}
	}
}
