// Package xrayclient is the HTTP client for the 3x-ui panel API (tested against
// panel 3.7.0).
//
// Two things changed from the legacy x-ui API this bot was originally written
// against, and both are load-bearing here:
//
//  1. Unsafe requests need a CSRF token. A bare POST /login answers 403 with an
//     empty body. The token is minted in the panel's HTML as
//     <meta name="csrf-token" content="..."> and refreshed by GET /csrf-token;
//     it travels in the X-CSRF-Token header alongside the "3x-ui" session cookie.
//     A Bearer API token skips CSRF entirely — the middleware short-circuits it.
//
//  2. A client is a first-class entity, not a row nested inside one inbound.
//     It is created once via POST /panel/api/clients/add with the inbound IDs it
//     should attach to, and is addressed by email everywhere afterwards.
package xrayclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"

	"xui-tg-admin/internal/config"
	"xui-tg-admin/internal/constants"
	"xui-tg-admin/internal/models"
)

// csrfMetaRe extracts the CSRF token the panel embeds in every rendered page.
var csrfMetaRe = regexp.MustCompile(`name="csrf-token"\s+content="([^"]+)"`)

// ErrClientNotFound is returned when the panel has no client with the requested
// email. It lets callers tell "absent" apart from "the call failed".
var ErrClientNotFound = errors.New("client not found")

// Client is an HTTP client for one 3x-ui panel.
type Client struct {
	httpClient   *resty.Client
	serverConfig config.ServerConfig
	logger       *logrus.Logger

	// mu guards csrfToken and authenticated. The cookie jar inside httpClient
	// is safe for concurrent use on its own.
	mu            sync.Mutex
	csrfToken     string
	authenticated bool
}

// apiResponse is the envelope every panel endpoint wraps its payload in.
type apiResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

// NewClient creates a new 3x-ui API client.
func NewClient(serverConfig config.ServerConfig, logger *logrus.Logger) *Client {
	httpClient := resty.New().
		SetTimeout(constants.DefaultTimeout * time.Second).
		SetRetryCount(constants.DefaultRetryCount).
		SetRetryWaitTime(constants.DefaultRetryWaitTime * time.Second).
		SetRetryMaxWaitTime(constants.DefaultRetryMaxWaitTime * time.Second).
		// The panel is normally reached over a private tunnel or a self-signed
		// certificate, so certificate verification would only break the bot.
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec

	return &Client{
		httpClient:   httpClient,
		serverConfig: serverConfig,
		logger:       logger,
	}
}

// baseURL returns the panel base URL without a trailing slash.
func (c *Client) baseURL() string {
	return strings.TrimRight(c.serverConfig.APIURL, "/")
}

// url builds an absolute panel URL from a path relative to the panel root.
func (c *Client) url(path string) string {
	return c.baseURL() + "/" + strings.TrimLeft(path, "/")
}

// usesToken reports whether the client authenticates with a Bearer API token
// instead of a username/password session.
func (c *Client) usesToken() bool {
	return c.serverConfig.APIToken != ""
}

// Login establishes a panel session: it fetches the login page for a CSRF
// token, posts the credentials, then refreshes the token for the authenticated
// session. With an API token configured it is a no-op.
func (c *Client) Login(ctx context.Context) error {
	if c.usesToken() {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.authenticated {
		return nil
	}
	return c.loginLocked(ctx)
}

// loginLocked performs the login handshake. The caller must hold c.mu.
func (c *Client) loginLocked(ctx context.Context) error {
	c.logger.Infof("Logging in to 3x-ui panel at %s", c.baseURL())

	// A fresh jar drops any stale "3x-ui" cookie so a re-login cannot be
	// answered from the expired session.
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}
	c.httpClient.SetCookieJar(jar)
	c.authenticated = false
	c.csrfToken = ""

	// Step 1: the panel page carries the CSRF token the login POST must replay.
	pageResp, err := c.httpClient.R().SetContext(ctx).Get(c.url("/"))
	if err != nil {
		return fmt.Errorf("failed to fetch panel page: %w", err)
	}
	token := ""
	if m := csrfMetaRe.FindSubmatch(pageResp.Body()); m != nil {
		token = string(m[1])
	}
	if token == "" {
		c.logger.Warn("No csrf-token meta tag on the panel page; login will likely be rejected with 403")
	}

	// Step 2: authenticate. The panel answers 200 with {"success":false} on bad
	// credentials and 403 with an empty body when the CSRF header is missing.
	loginResp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-CSRF-Token", token).
		SetBody(map[string]string{
			"username": c.serverConfig.User,
			"password": c.serverConfig.Password,
		}).
		Post(c.url("/login"))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	if loginResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("login failed with status code %d: %s",
			loginResp.StatusCode(), truncate(string(loginResp.Body())))
	}

	var parsed apiResponse
	if err := json.Unmarshal(loginResp.Body(), &parsed); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}
	if !parsed.Success {
		return fmt.Errorf("login failed: %s", parsed.Msg)
	}

	// Step 3: mint a token bound to the authenticated session. The bootstrap
	// token usually keeps working, so a failure here is not fatal.
	c.csrfToken = token
	if refreshed, rerr := c.fetchCSRFToken(ctx); rerr != nil {
		c.logger.Warnf("Failed to refresh CSRF token, reusing the bootstrap one: %v", rerr)
	} else if refreshed != "" {
		c.csrfToken = refreshed
	}

	c.authenticated = true
	c.logger.Info("Successfully logged in to the 3x-ui panel")
	return nil
}

// fetchCSRFToken asks the panel for a CSRF token for the current session.
func (c *Client) fetchCSRFToken(ctx context.Context) (string, error) {
	resp, err := c.httpClient.R().SetContext(ctx).Get(c.url("/csrf-token"))
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("csrf-token returned status %d", resp.StatusCode())
	}
	var parsed apiResponse
	if err := json.Unmarshal(resp.Body(), &parsed); err != nil {
		return "", err
	}
	var token string
	if err := json.Unmarshal(parsed.Obj, &token); err != nil {
		return "", err
	}
	return token, nil
}

// invalidateSession forgets the current session so the next call logs in again.
func (c *Client) invalidateSession() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authenticated = false
	c.csrfToken = ""
}

// call performs one authenticated panel API request and decodes the envelope's
// obj field into out (which may be nil to discard it).
//
// An unauthenticated request to /panel/api/* answers 404, not 401, so any of
// 401/403/404 is treated as a possibly expired session and retried once after
// a fresh login. The retry is bounded: the old code recursed without a limit
// and could spin on a permanent 401.
func (c *Client) call(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	for attempt := 0; attempt < 2; attempt++ {
		if err := c.Login(ctx); err != nil {
			return err
		}

		req := c.httpClient.R().SetContext(ctx)
		if c.usesToken() {
			req.SetHeader("Authorization", "Bearer "+c.serverConfig.APIToken)
		} else {
			c.mu.Lock()
			token := c.csrfToken
			c.mu.Unlock()
			req.SetHeader("X-CSRF-Token", token)
		}
		if body != nil {
			req.SetHeader("Content-Type", "application/json").SetBody(body)
		}

		resp, err := req.Execute(method, c.url(path))
		if err != nil {
			return fmt.Errorf("%s %s failed: %w", method, path, err)
		}

		status := resp.StatusCode()
		if status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound {
			if attempt == 0 && !c.usesToken() {
				c.logger.Warnf("%s %s returned %d; re-authenticating and retrying once", method, path, status)
				c.invalidateSession()
				continue
			}
			return fmt.Errorf("%s %s failed with status code %d: %s", method, path, status, truncate(string(resp.Body())))
		}
		if status != http.StatusOK {
			return fmt.Errorf("%s %s failed with status code %d: %s", method, path, status, truncate(string(resp.Body())))
		}
		if len(resp.Body()) == 0 {
			return fmt.Errorf("%s %s returned an empty body", method, path)
		}

		var parsed apiResponse
		if err := json.Unmarshal(resp.Body(), &parsed); err != nil {
			return fmt.Errorf("failed to parse %s %s response: %w (body: %s)",
				method, path, err, truncate(string(resp.Body())))
		}
		if !parsed.Success {
			return fmt.Errorf("%s %s failed: %s", method, path, parsed.Msg)
		}
		if out == nil || len(parsed.Obj) == 0 || string(parsed.Obj) == "null" {
			return nil
		}
		if err := json.Unmarshal(parsed.Obj, out); err != nil {
			return fmt.Errorf("failed to unmarshal %s %s payload: %w", method, path, err)
		}
		return nil
	}
	return fmt.Errorf("%s %s failed after re-authentication", method, path)
}

// GetInbounds lists the inbounds with their per-client traffic counters.
func (c *Client) GetInbounds(ctx context.Context) ([]models.Inbound, error) {
	var inbounds []models.Inbound
	if err := c.call(ctx, http.MethodGet, "/panel/api/inbounds/list", nil, &inbounds); err != nil {
		return nil, err
	}
	return inbounds, nil
}

// GetClients lists every client with its attached inbound IDs and traffic
// record. This is the 3.7.0 view of users; the per-inbound settings blob is
// only a projection of it.
func (c *Client) GetClients(ctx context.Context) ([]models.ClientRecord, error) {
	var clients []models.ClientRecord
	if err := c.call(ctx, http.MethodGet, "/panel/api/clients/list", nil, &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

// clientDetail is the payload of GET /panel/api/clients/get/{email}.
type clientDetail struct {
	Client     models.ClientRecord `json:"client"`
	InboundIDs []int               `json:"inboundIds"`
}

// GetClient fetches one client by email. It returns ErrClientNotFound when the
// panel has no such client.
func (c *Client) GetClient(ctx context.Context, email string) (*models.ClientRecord, error) {
	var detail clientDetail
	err := c.call(ctx, http.MethodGet, "/panel/api/clients/get/"+url.PathEscape(email), nil, &detail)
	if err != nil {
		// The panel reports an unknown email as a failed call, not as an empty
		// result, so the message is the only signal available.
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrClientNotFound, email)
		}
		return nil, err
	}
	if detail.Client.Email == "" {
		return nil, fmt.Errorf("%w: %s", ErrClientNotFound, email)
	}
	record := detail.Client
	if len(record.InboundIDs) == 0 {
		record.InboundIDs = detail.InboundIDs
	}
	return &record, nil
}

// AddClient creates a client and attaches it to the given inbounds in one call,
// then reads it back: the panel has been seen to answer SUCCESS for a mutation
// that did not land, so the result is verified rather than trusted.
func (c *Client) AddClient(ctx context.Context, client models.Client, inboundIDs []int) error {
	if len(inboundIDs) == 0 {
		return errors.New("at least one inbound ID is required to add a client")
	}

	body := map[string]interface{}{
		"client":     client,
		"inboundIds": inboundIDs,
	}
	c.logger.Infof("Adding client %s to inbounds %v", client.Email, inboundIDs)
	if err := c.call(ctx, http.MethodPost, "/panel/api/clients/add", body, nil); err != nil {
		return err
	}

	created, err := c.GetClient(ctx, client.Email)
	if err != nil {
		return fmt.Errorf("client %s was reported as added but could not be read back: %w", client.Email, err)
	}
	c.logger.Infof("Successfully added client %s (inbounds %v)", created.Email, created.InboundIDs)
	return nil
}

// DeleteClient removes a client from every inbound it is attached to and drops
// its traffic record, then verifies it is really gone.
func (c *Client) DeleteClient(ctx context.Context, email string) error {
	path := "/panel/api/clients/del/" + url.PathEscape(email) + "?keepTraffic=0"
	if err := c.call(ctx, http.MethodPost, path, nil, nil); err != nil {
		return err
	}

	if _, err := c.GetClient(ctx, email); err == nil {
		return fmt.Errorf("panel reported client %s as deleted but it still exists", email)
	} else if !errors.Is(err, ErrClientNotFound) {
		c.logger.Warnf("Could not verify deletion of %s: %v", email, err)
	}
	c.logger.Infof("Successfully deleted client %s", email)
	return nil
}

// RemoveClients deletes every client whose email matches one of the given
// emails. It succeeds if at least one client was removed and reports the rest.
func (c *Client) RemoveClients(ctx context.Context, emails []string) error {
	if len(emails) == 0 {
		return errors.New("no emails given to remove")
	}

	var failures []string
	deleted := 0
	for _, email := range emails {
		if err := c.DeleteClient(ctx, email); err != nil {
			c.logger.Errorf("Failed to delete client %s: %v", email, err)
			failures = append(failures, fmt.Sprintf("%s: %v", email, err))
			continue
		}
		deleted++
	}

	if deleted == 0 {
		return fmt.Errorf("failed to delete any clients: %s", strings.Join(failures, "; "))
	}
	if len(failures) > 0 {
		c.logger.Warnf("Some deletions failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

// ResetClientTraffic zeroes a client's counters across every inbound it is
// attached to. Traffic is a property of the client, not of an inbound, so no
// inbound ID is involved.
func (c *Client) ResetClientTraffic(ctx context.Context, email string) error {
	c.logger.Debugf("Resetting traffic for client %s", email)
	return c.call(ctx, http.MethodPost, "/panel/api/clients/resetTraffic/"+url.PathEscape(email), nil, nil)
}

// ResetAllTraffics zeroes the counters of every client on the panel in one call.
func (c *Client) ResetAllTraffics(ctx context.Context) error {
	c.logger.Info("Resetting traffic counters for all clients")
	return c.call(ctx, http.MethodPost, "/panel/api/clients/resetAllTraffics", nil, nil)
}

// GetOnlineUsers returns the emails of clients seen within the panel's
// heartbeat window.
func (c *Client) GetOnlineUsers(ctx context.Context) ([]string, error) {
	var online []string
	if err := c.call(ctx, http.MethodPost, "/panel/api/clients/onlines", nil, &online); err != nil {
		return nil, err
	}
	return online, nil
}

// GetLastOnline maps each client email to when it was last seen, as a Unix
// timestamp in milliseconds. Clients that have never connected are absent.
func (c *Client) GetLastOnline(ctx context.Context) (map[string]int64, error) {
	lastOnline := make(map[string]int64)
	if err := c.call(ctx, http.MethodPost, "/panel/api/clients/lastOnline", nil, &lastOnline); err != nil {
		return nil, err
	}
	return lastOnline, nil
}

// GetClientLinks returns the ready-made share URLs (vless://, vmess://, …) for
// one client across all of its inbounds — the same strings the panel's Copy URL
// button yields, so the bot never has to assemble them itself.
func (c *Client) GetClientLinks(ctx context.Context, email string) ([]string, error) {
	var links []string
	err := c.call(ctx, http.MethodGet, "/panel/api/clients/links/"+url.PathEscape(email), nil, &links)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrClientNotFound, email)
		}
		return nil, err
	}
	return links, nil
}

// isNotFound reports whether a panel error means "no such record". The API has
// no machine-readable error code, so the message is all there is to go on.
func isNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such") ||
		strings.Contains(msg, "record not found")
}

// maxLoggedBody caps how much of a response body ends up in an error message.
const maxLoggedBody = 512

// truncate shortens a response body so error messages stay readable.
func truncate(s string) string {
	if len(s) <= maxLoggedBody {
		return s
	}
	return s[:maxLoggedBody] + "…"
}
