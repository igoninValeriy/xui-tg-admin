package models

import "encoding/json"

// RawJSON holds a JSON fragment that 3x-ui returns as a nested object since
// panel 3.7.0, but that older panels (and the panel's own write path) encode as
// a JSON string. It unmarshals from either form and always yields the raw JSON
// text, so callers can json.Unmarshal it directly.
type RawJSON string

// UnmarshalJSON accepts both a JSON-encoded string and a raw JSON value.
func (r *RawJSON) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*r = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*r = RawJSON(s)
		return nil
	}
	*r = RawJSON(data)
	return nil
}

// MarshalJSON writes the fragment back as raw JSON.
func (r RawJSON) MarshalJSON() ([]byte, error) {
	if r == "" {
		return []byte("null"), nil
	}
	return []byte(r), nil
}

// Inbound represents an X-ray inbound configuration as returned by
// GET /panel/api/inbounds/list.
type Inbound struct {
	ID             int          `json:"id"`
	Up             int64        `json:"up"`
	Down           int64        `json:"down"`
	Total          int64        `json:"total"`
	Remark         string       `json:"remark"`
	Enable         bool         `json:"enable"`
	ExpiryTime     int64        `json:"expiryTime"`
	ClientStats    []ClientStat `json:"clientStats"`
	Listen         string       `json:"listen"`
	Port           int          `json:"port"`
	Protocol       string       `json:"protocol"`
	Tag            string       `json:"tag"`
	Settings       RawJSON      `json:"settings"`
	StreamSettings RawJSON      `json:"streamSettings"`
	Sniffing       RawJSON      `json:"sniffing"`
}

// ClientStat represents traffic statistics for a client on one inbound.
type ClientStat struct {
	ID         int    `json:"id"`
	InboundID  int    `json:"inboundId"`
	Enable     bool   `json:"enable"`
	Email      string `json:"email"`
	UUID       string `json:"uuid"`
	SubID      string `json:"subId"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	ExpiryTime int64  `json:"expiryTime"`
	Total      int64  `json:"total"`
	Reset      int64  `json:"reset"`
	// LastOnline is a Unix timestamp in milliseconds; 0 means never seen.
	LastOnline int64 `json:"lastOnline"`
}

// InboundSettings represents the parsed settings of an inbound.
type InboundSettings struct {
	Clients []InboundClient `json:"clients"`
}

// InboundClient represents a client entry inside inbound settings.
type InboundClient struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`
	SubID      string `json:"subId"`
	// TgID is a numeric Telegram user ID. The panel rejects a string here with
	// "cannot unmarshal string into Go struct field .tgId of type int64".
	TgID int64 `json:"tgId"`
}
