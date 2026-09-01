package xrayclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"xui-tg-admin/internal/config"
	"xui-tg-admin/internal/models"
)

// fakePanel is a stand-in for 3x-ui 3.7.0 that enforces the two rules the real
// panel enforces: unsafe calls need the CSRF header, and API calls need the
// session cookie (an unauthenticated /panel/api/* request answers 404, not 401).
type fakePanel struct {
	mu sync.Mutex

	sessionCookie string
	csrfToken     string

	logins   int
	requests []string

	// handlers maps a path to the JSON payload placed in the envelope's obj.
	handlers map[string]func(r *http.Request) (interface{}, bool, string)

	// expireAfter drops the session once this many API calls have been served,
	// so a re-login can be exercised.
	expireAfter int
	apiCalls    int
}

func newFakePanel() *fakePanel {
	return &fakePanel{
		sessionCookie: "session-value",
		csrfToken:     "csrf-value",
		handlers:      map[string]func(r *http.Request) (interface{}, bool, string){},
		expireAfter:   -1,
	}
}

func (f *fakePanel) handle(path string, fn func(r *http.Request) (interface{}, bool, string)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[path] = fn
}

func (f *fakePanel) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *fakePanel) writeEnvelope(w http.ResponseWriter, obj interface{}, success bool, msg string) {
	payload := map[string]interface{}{"success": success, "msg": msg}
	if obj != nil {
		payload["obj"] = obj
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (f *fakePanel) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie("3x-ui")
	return err == nil && cookie.Value == f.sessionCookie
}

func (f *fakePanel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()

	switch r.URL.Path {
	case "/":
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><head><meta name="csrf-token" content="`+f.csrfToken+`"></head></html>`)
		return

	case "/login":
		// The real panel answers 403 with an empty body when the header is absent.
		if r.Header.Get("X-CSRF-Token") != f.csrfToken {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var creds map[string]string
		_ = json.NewDecoder(r.Body).Decode(&creds)
		if creds["username"] != "admin" || creds["password"] != "secret" {
			f.writeEnvelope(w, nil, false, "Invalid username or password")
			return
		}
		f.mu.Lock()
		f.logins++
		f.apiCalls = 0
		f.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: f.sessionCookie, Path: "/"})
		f.writeEnvelope(w, nil, true, "")
		return

	case "/csrf-token":
		f.writeEnvelope(w, f.csrfToken, true, "")
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/panel/api/") {
		http.NotFound(w, r)
		return
	}

	// Bearer callers skip both the cookie and the CSRF header.
	bearer := r.Header.Get("Authorization") == "Bearer token-value"
	if !bearer {
		if !f.authenticated(r) {
			http.NotFound(w, r)
			return
		}
		f.mu.Lock()
		f.apiCalls++
		expired := f.expireAfter >= 0 && f.apiCalls > f.expireAfter
		f.mu.Unlock()
		if expired {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-CSRF-Token") != f.csrfToken {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	f.mu.Lock()
	handler, ok := f.handlers[r.URL.Path]
	f.mu.Unlock()
	if !ok {
		f.writeEnvelope(w, nil, false, "record not found")
		return
	}
	obj, success, msg := handler(r)
	f.writeEnvelope(w, obj, success, msg)
}

func newTestClient(t *testing.T, panel *fakePanel, token string) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(panel)
	t.Cleanup(server.Close)

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	return NewClient(config.ServerConfig{
		User:     "admin",
		Password: "secret",
		APIToken: token,
		APIURL:   server.URL,
	}, logger), server
}

func TestLoginPerformsCSRFHandshake(t *testing.T) {
	panel := newFakePanel()
	client, _ := newTestClient(t, panel, "")

	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}

	calls := panel.calls()
	if len(calls) < 2 || calls[0] != "GET /" || calls[1] != "POST /login" {
		t.Fatalf("expected the page fetch to precede the login, got %v", calls)
	}
	if panel.logins != 1 {
		t.Errorf("logins = %d, want 1", panel.logins)
	}

	// A second Login must reuse the cached session.
	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("second Login: %v", err)
	}
	if panel.logins != 1 {
		t.Errorf("logins after reuse = %d, want 1", panel.logins)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	panel := newFakePanel()
	client, _ := newTestClient(t, panel, "")
	client.serverConfig.Password = "wrong"

	err := client.Login(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Invalid username or password") {
		t.Fatalf("expected a credentials error, got %v", err)
	}
}

func TestGetInboundsParsesNestedSettings(t *testing.T) {
	panel := newFakePanel()
	panel.handle("/panel/api/inbounds/list", func(*http.Request) (interface{}, bool, string) {
		return []map[string]interface{}{{
			"id":       1,
			"remark":   "reality",
			"enable":   true,
			"protocol": "vless",
			// 3.7.0 nests these rather than encoding them as strings.
			"settings": map[string]interface{}{
				"clients": []map[string]interface{}{
					{"email": "john", "subId": "abc", "tgId": 42, "enable": true},
				},
			},
			"clientStats": []map[string]interface{}{
				{"id": 1, "inboundId": 1, "email": "john", "up": 10, "down": 20, "lastOnline": 1788265376002},
			},
		}}, true, ""
	})
	client, _ := newTestClient(t, panel, "")

	inbounds, err := client.GetInbounds(context.Background())
	if err != nil {
		t.Fatalf("GetInbounds: %v", err)
	}
	if len(inbounds) != 1 {
		t.Fatalf("got %d inbounds, want 1", len(inbounds))
	}

	var settings models.InboundSettings
	if err := json.Unmarshal([]byte(inbounds[0].Settings), &settings); err != nil {
		t.Fatalf("settings are not parseable JSON: %v", err)
	}
	if len(settings.Clients) != 1 || settings.Clients[0].TgID != 42 {
		t.Errorf("clients = %+v, want one client with tgId 42", settings.Clients)
	}
	if inbounds[0].ClientStats[0].LastOnline != 1788265376002 {
		t.Errorf("lastOnline = %d, want 1788265376002", inbounds[0].ClientStats[0].LastOnline)
	}
}

func TestAddClientSendsClientAndInboundIDs(t *testing.T) {
	panel := newFakePanel()

	var body struct {
		Client     map[string]json.RawMessage `json:"client"`
		InboundIDs []int                      `json:"inboundIds"`
	}
	panel.handle("/panel/api/clients/add", func(r *http.Request) (interface{}, bool, string) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		return nil, true, ""
	})
	panel.handle("/panel/api/clients/get/john", func(*http.Request) (interface{}, bool, string) {
		return map[string]interface{}{
			"client":     map[string]interface{}{"email": "john", "subId": "abc"},
			"inboundIds": []int{1, 2},
		}, true, ""
	})
	client, _ := newTestClient(t, panel, "")

	err := client.AddClient(context.Background(), models.Client{
		Email: "john", Enable: true, TgID: 42, SubID: "abc",
	}, []int{1, 2})
	if err != nil {
		t.Fatalf("AddClient: %v", err)
	}

	if got := string(body.Client["tgId"]); got != "42" {
		t.Errorf("tgId sent as %s, want the bare number 42", got)
	}
	if len(body.InboundIDs) != 2 || body.InboundIDs[0] != 1 || body.InboundIDs[1] != 2 {
		t.Errorf("inboundIds = %v, want [1 2]", body.InboundIDs)
	}

	// The add must be verified by reading the client back.
	found := false
	for _, call := range panel.calls() {
		if call == "GET /panel/api/clients/get/john" {
			found = true
		}
	}
	if !found {
		t.Error("AddClient must read the client back instead of trusting the SUCCESS response")
	}
}

func TestAddClientFailsWhenNotReadBack(t *testing.T) {
	panel := newFakePanel()
	// The panel claims success but the client never appears — the failure mode
	// that made blind trust in the response unsafe.
	panel.handle("/panel/api/clients/add", func(*http.Request) (interface{}, bool, string) {
		return nil, true, ""
	})
	client, _ := newTestClient(t, panel, "")

	err := client.AddClient(context.Background(), models.Client{Email: "john"}, []int{1})
	if err == nil {
		t.Fatal("expected an error when the created client cannot be read back")
	}
	if !strings.Contains(err.Error(), "could not be read back") {
		t.Errorf("error = %v, want it to name the failed verification", err)
	}
}

func TestAddClientRequiresInbounds(t *testing.T) {
	panel := newFakePanel()
	client, _ := newTestClient(t, panel, "")

	if err := client.AddClient(context.Background(), models.Client{Email: "john"}, nil); err == nil {
		t.Fatal("expected an error when no inbound IDs are given")
	}
}

func TestDeleteClientDetectsPhantomSuccess(t *testing.T) {
	panel := newFakePanel()
	panel.handle("/panel/api/clients/del/john", func(*http.Request) (interface{}, bool, string) {
		return nil, true, "" // reports SUCCESS...
	})
	panel.handle("/panel/api/clients/get/john", func(*http.Request) (interface{}, bool, string) {
		return map[string]interface{}{"client": map[string]interface{}{"email": "john"}}, true, ""
	}) // ...but the client is still there
	client, _ := newTestClient(t, panel, "")

	err := client.DeleteClient(context.Background(), "john")
	if err == nil || !strings.Contains(err.Error(), "still exists") {
		t.Fatalf("expected the phantom delete to be caught, got %v", err)
	}
}

func TestDeleteClientSucceedsWhenGone(t *testing.T) {
	panel := newFakePanel()
	panel.handle("/panel/api/clients/del/john", func(*http.Request) (interface{}, bool, string) {
		return nil, true, ""
	})
	// No handler for clients/get/john -> the panel answers "record not found".
	client, _ := newTestClient(t, panel, "")

	if err := client.DeleteClient(context.Background(), "john"); err != nil {
		t.Fatalf("DeleteClient: %v", err)
	}
}

func TestGetClientReportsNotFound(t *testing.T) {
	panel := newFakePanel()
	client, _ := newTestClient(t, panel, "")

	_, err := client.GetClient(context.Background(), "ghost")
	if !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("error = %v, want ErrClientNotFound", err)
	}
}

func TestOnlineAndLastOnline(t *testing.T) {
	panel := newFakePanel()
	panel.handle("/panel/api/clients/onlines", func(*http.Request) (interface{}, bool, string) {
		return []string{"john"}, true, ""
	})
	panel.handle("/panel/api/clients/lastOnline", func(*http.Request) (interface{}, bool, string) {
		return map[string]int64{"john": 1788265376002, "jane": 1788265000000}, true, ""
	})
	client, _ := newTestClient(t, panel, "")

	online, err := client.GetOnlineUsers(context.Background())
	if err != nil {
		t.Fatalf("GetOnlineUsers: %v", err)
	}
	if len(online) != 1 || online[0] != "john" {
		t.Errorf("online = %v, want [john]", online)
	}

	last, err := client.GetLastOnline(context.Background())
	if err != nil {
		t.Fatalf("GetLastOnline: %v", err)
	}
	if last["jane"] != 1788265000000 {
		t.Errorf("lastOnline[jane] = %d, want 1788265000000", last["jane"])
	}
}

func TestGetClientLinks(t *testing.T) {
	panel := newFakePanel()
	panel.handle("/panel/api/clients/links/john", func(*http.Request) (interface{}, bool, string) {
		return []string{"vless://uuid@example:443?type=xhttp#john"}, true, ""
	})
	client, _ := newTestClient(t, panel, "")

	links, err := client.GetClientLinks(context.Background(), "john")
	if err != nil {
		t.Fatalf("GetClientLinks: %v", err)
	}
	if len(links) != 1 || !strings.HasPrefix(links[0], "vless://") {
		t.Errorf("links = %v, want one vless:// URL", links)
	}
}

// TestReAuthenticatesOnce covers the expired-session path: /panel/api/* answers
// 404 rather than 401 when the cookie is stale, so 404 must trigger exactly one
// re-login and retry — not the unbounded recursion the old client used.
func TestReAuthenticatesOnce(t *testing.T) {
	panel := newFakePanel()
	panel.expireAfter = 1
	panel.handle("/panel/api/inbounds/list", func(*http.Request) (interface{}, bool, string) {
		return []map[string]interface{}{{"id": 1}}, true, ""
	})
	client, _ := newTestClient(t, panel, "")

	if _, err := client.GetInbounds(context.Background()); err != nil {
		t.Fatalf("first GetInbounds: %v", err)
	}
	// The session is now stale; this call must recover on its own.
	if _, err := client.GetInbounds(context.Background()); err != nil {
		t.Fatalf("GetInbounds after expiry: %v", err)
	}
	if panel.logins != 2 {
		t.Errorf("logins = %d, want 2 (one initial, one refresh)", panel.logins)
	}
}

func TestReAuthenticationGivesUpAfterOneRetry(t *testing.T) {
	panel := newFakePanel()
	panel.expireAfter = 0 // every API call sees a stale session
	client, _ := newTestClient(t, panel, "")

	if _, err := client.GetInbounds(context.Background()); err == nil {
		t.Fatal("expected a permanent failure rather than an endless retry loop")
	}
	if panel.logins > 2 {
		t.Errorf("logins = %d, want at most 2 — the retry must be bounded", panel.logins)
	}
}

// TestBearerTokenSkipsLogin covers the documented short-circuit: an API token
// authenticates on its own and needs neither a session nor a CSRF token.
func TestBearerTokenSkipsLogin(t *testing.T) {
	panel := newFakePanel()
	panel.handle("/panel/api/inbounds/list", func(*http.Request) (interface{}, bool, string) {
		return []map[string]interface{}{{"id": 1}}, true, ""
	})
	client, _ := newTestClient(t, panel, "token-value")

	if _, err := client.GetInbounds(context.Background()); err != nil {
		t.Fatalf("GetInbounds with a token: %v", err)
	}
	if panel.logins != 0 {
		t.Errorf("logins = %d, want 0 — a token must not trigger a login", panel.logins)
	}
	for _, call := range panel.calls() {
		if call == "POST /login" || call == "GET /" {
			t.Errorf("unexpected session handshake call %q when using a token", call)
		}
	}
}

func TestRemoveClientsSucceedsIfAnyDeleted(t *testing.T) {
	panel := newFakePanel()
	panel.handle("/panel/api/clients/del/john", func(*http.Request) (interface{}, bool, string) {
		return nil, true, ""
	})
	// "jane" has no delete handler -> the panel reports a failure for her.
	client, _ := newTestClient(t, panel, "")

	if err := client.RemoveClients(context.Background(), []string{"john", "jane"}); err != nil {
		t.Fatalf("RemoveClients: %v", err)
	}

	if err := client.RemoveClients(context.Background(), []string{"jane"}); err == nil {
		t.Fatal("expected an error when nothing could be deleted")
	}
	if err := client.RemoveClients(context.Background(), nil); err == nil {
		t.Fatal("expected an error for an empty email list")
	}
}
