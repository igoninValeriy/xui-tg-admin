package models

import (
	"crypto/rand"
	"encoding/hex"
)

// Client is the universal client payload accepted by POST /panel/api/clients/add
// and POST /panel/api/clients/update/{email} on 3x-ui 3.7.0. Per-protocol
// secrets the panel can mint itself (uuid, password, subId) may be left empty.
type Client struct {
	// ID is the client UUID. Left empty, the panel generates one.
	ID     string `json:"id,omitempty"`
	Email  string `json:"email"`
	Enable bool   `json:"enable"`
	// Flow is a VLESS flow control mode; empty for REALITY/XHTTP inbounds.
	Flow string `json:"flow,omitempty"`
	// TotalGB is the traffic quota in bytes despite the name; 0 means unlimited.
	TotalGB int `json:"totalGB"`
	// LimitIP caps concurrent source IPs; 0 means no limit.
	LimitIP int `json:"limitIp"`
	// ExpiryTime is a Unix timestamp in milliseconds; 0 means unlimited.
	ExpiryTime int64 `json:"expiryTime"`
	// TgID is a numeric Telegram user ID. Sending a string here makes the panel
	// fail with "cannot unmarshal string into Go struct field .tgId of type int64".
	TgID int64 `json:"tgId"`
	// SubID names the client's subscription. Left empty, the panel generates one.
	SubID   string `json:"subId,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// ClientTraffic is the traffic record attached to a client by
// GET /panel/api/clients/list and GET /panel/api/clients/get/{email}.
type ClientTraffic struct {
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	Total      int64  `json:"total"`
	ExpiryTime int64  `json:"expiryTime"`
	// LastOnline is a Unix timestamp in milliseconds; 0 means never seen.
	LastOnline int64 `json:"lastOnline"`
}

// ClientRecord is one entry of GET /panel/api/clients/list: the client itself
// plus the inbounds it is attached to and its traffic counters. Since 3.7.0 a
// client is a first-class entity shared across inbounds rather than a row
// nested inside one inbound's settings.
type ClientRecord struct {
	ID         int           `json:"id"`
	Email      string        `json:"email"`
	SubID      string        `json:"subId"`
	UUID       string        `json:"uuid"`
	Flow       string        `json:"flow"`
	LimitIP    int           `json:"limitIp"`
	TotalGB    int64         `json:"totalGB"`
	ExpiryTime int64         `json:"expiryTime"`
	Enable     bool          `json:"enable"`
	TgID       int64         `json:"tgId"`
	Comment    string        `json:"comment"`
	CreatedAt  int64         `json:"createdAt"`
	UpdatedAt  int64         `json:"updatedAt"`
	InboundIDs []int         `json:"inboundIds"`
	Traffic    ClientTraffic `json:"traffic"`
}

// subIDBytes is the entropy of a generated subscription ID. 16 bytes render as
// the 32 hex characters the subscription front end expects as a file name.
const subIDBytes = 16

// GenerateSubID generates a random subscription ID as 32 hex characters.
func GenerateSubID() string {
	buf := make([]byte, subIDBytes)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand on a supported platform does not fail; a caller that
		// still hits this gets an empty ID and the panel mints its own.
		return ""
	}
	return hex.EncodeToString(buf)
}
