package models

import (
	"encoding/json"
	"testing"
)

// TestRawJSONAcceptsBothForms covers the shape change in panel 3.7.0:
// /panel/api/inbounds/list now returns settings as a nested object, while older
// panels returned it as a JSON-encoded string. Both must decode to the same
// raw JSON text.
func TestRawJSONAcceptsBothForms(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"nested object (3.7.0)", `{"settings":{"clients":[{"email":"john"}]}}`, `{"clients":[{"email":"john"}]}`},
		{"encoded string (legacy)", `{"settings":"{\"clients\":[{\"email\":\"john\"}]}"}`, `{"clients":[{"email":"john"}]}`},
		{"null", `{"settings":null}`, ``},
		{"absent", `{}`, ``},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var inbound Inbound
			if err := json.Unmarshal([]byte(tc.in), &inbound); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if string(inbound.Settings) != tc.want {
				t.Errorf("Settings = %q, want %q", inbound.Settings, tc.want)
			}
			if tc.want == "" {
				return
			}
			// Whatever the wire form was, the result must be parseable JSON.
			var settings InboundSettings
			if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
				t.Fatalf("settings are not valid JSON: %v", err)
			}
			if len(settings.Clients) != 1 || settings.Clients[0].Email != "john" {
				t.Errorf("clients = %+v, want one client john", settings.Clients)
			}
		})
	}
}

// TestInboundClientTgIDIsNumeric guards the int64 tgId the panel returns.
func TestInboundClientTgIDIsNumeric(t *testing.T) {
	var settings InboundSettings
	if err := json.Unmarshal([]byte(`{"clients":[{"email":"john","tgId":123456789,"subId":"abc"}]}`), &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if settings.Clients[0].TgID != 123456789 {
		t.Errorf("TgID = %d, want 123456789", settings.Clients[0].TgID)
	}
}

func TestRawJSONRoundTrip(t *testing.T) {
	in := Inbound{Settings: RawJSON(`{"clients":[]}`)}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Inbound
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Settings != in.Settings {
		t.Errorf("round trip = %q, want %q", out.Settings, in.Settings)
	}
}
