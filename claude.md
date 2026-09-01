# 🤖 Claude AI Agent Guide — X-UI Telegram Admin Bot

## 📋 Project Overview

**X-UI Telegram Admin Bot** is a Go application that manages a 3x-ui VPN panel
(tested against panel 3.7.0) through a Telegram bot. Access is admin-only.

### 🎯 Purpose
Automate VPN user management from Telegram: admins create/delete users, reset
traffic, and monitor connections and usage.

---

## 🏗️ Architecture

Flow: **Telegram update → admin check → handler → services → X-UI HTTP client**.

I/O lives at the edges (`pkg/telegrambot`, `pkg/xrayclient`, `services`), and the
cancellable request `context.Context` is threaded from `Bot.Start` all the way down
to the X-UI HTTP calls (no `context.Background()` inside handlers).

```
xui-tg-admin/
├── cmd/bot/main.go              # Entry point: config, services, signal-based shutdown
├── internal/
│   ├── commands/                # Telegram command/button string constants
│   ├── config/                  # Env-based configuration loading & validation
│   ├── constants/               # Numeric/format constants (limits, timeouts, …)
│   ├── handlers/                # Telegram message handlers
│   │   ├── base.go              # BaseHandler: shared send/keyboard helpers
│   │   ├── factory.go           # Builds a handler for a given access type
│   │   ├── admin.go             # AdminHandler: dispatch, /start, command routing
│   │   ├── admin_members.go     # Admin: user create / edit / delete flows
│   │   ├── admin_traffic.go     # Admin: online list, usage reports, traffic resets
│   │   ├── admin_node.go        # Admin: node status, subscription stats, subscription QR
│   │   └── admin_client_operations.go # Admin: client creation across inbounds
│   ├── helpers/                 # Pure helpers: username, traffic, subscription formatting
│   ├── models/                  # Data models (Client, Inbound, MemberInfo, state)
│   ├── permissions/             # Access control (Admin / None)
│   ├── services/                # XrayService, UserStateService, QRService, NodeService
│   └── validation/              # Username/duration validation
└── pkg/
    ├── telegrambot/bot.go       # Bot wiring, middleware, update routing
    └── xrayclient/client.go     # HTTP client for the X-UI API
```

---

## 🔐 Roles & Permissions (`internal/permissions`)

Two access types — **no Trusted/Demo/User roles exist**:

- **`Admin`** — Telegram IDs listed in `TG_ADMIN_IDS`. Full access.
- **`None`** — everyone else; the bot refuses to serve them.

`PermissionController.GetAccessType(userID)` returns `Admin` for listed IDs and
`None` otherwise.

---

## 🧭 Handlers & State

`HandlerFactory.CreateHandler(accessType)` returns the right handler. Each handler
embeds `BaseHandler` and dispatches on the user's `ConversationState`.

Command/button handlers are stored in a
`map[string]func(context.Context, telebot.Context) error`; button text is mapped to a
command by `getButtonCommand` (strips the emoji prefix).

`ConversationState` values (`internal/models/userstate.go`):
`Default`, `AwaitingInputUserName`, `AwaitingDuration`, `AwaitSelectUserName`,
`AwaitMemberAction`, `AwaitConfirmMemberDeletion`,
`AwaitConfirmResetUsersNetworkUsage`, `AwaitUsageReportChoice`.

Inline-button presses bypass the state machine: `AdminHandler.Handle` routes any
update carrying a callback to `handleCallback`, which dispatches on the callback
data (telebot's `\f<unique>[|payload]`). The Node Status screen's **Refresh**
button uses this to redraw its own message via `editMessageText` instead of
sending a new one.

State is stored in-memory in `UserStateService` (a `go-cache` with a 30-minute TTL).
The bot keeps no persistent storage.

---

## 🔧 3x-ui panel API (`pkg/xrayclient`)

Targets **3x-ui 3.7.0**. `XRAY_API_URL` is the panel **base URL including its
secret web base path**, with no trailing slash.

**CSRF is mandatory.** A bare `POST /login` answers `403` with an empty body.
The client fetches the panel page, reads the token from
`<meta name="csrf-token" content="...">`, refreshes it via `GET /csrf-token`
once the session exists, and replays it in `X-CSRF-Token` alongside the `3x-ui`
session cookie. `XRAY_API_TOKEN` skips the whole handshake — the panel
short-circuits CSRF for Bearer callers.

Endpoints:

- `POST {base}/login` — authenticate (needs the CSRF header)
- `GET  {base}/csrf-token` — mint a token for the current session
- `GET  {base}/panel/api/inbounds/list`
- `GET  {base}/panel/api/clients/list`
- `GET  {base}/panel/api/clients/get/{email}`
- `GET  {base}/panel/api/clients/links/{email}` — ready-made share URLs
- `POST {base}/panel/api/clients/add` — `{"client": {...}, "inboundIds": [1]}`
- `POST {base}/panel/api/clients/del/{email}`
- `POST {base}/panel/api/clients/resetTraffic/{email}`
- `POST {base}/panel/api/clients/resetAllTraffics`
- `POST {base}/panel/api/clients/onlines` — connected right now
- `POST {base}/panel/api/clients/lastOnline` — email → last-seen Unix ms

The panel serves its own spec at `GET {base}/panel/api/openapi.json` (session
required; UI at `{base}/panel/api-docs`). **Check request and response shapes
against that spec rather than guessing.**

Data conventions:

- A **client is a first-class entity**, not a row nested in one inbound. One user
  is one client attached to a list of inbounds. The legacy scheme of one client
  per inbound named `username-1`, `username-2`, … is gone; clients created that
  way still work and are grouped into one user by their shared `SubID`.
- `tgId` is an **`int64`**. A string fails with
  `json: cannot unmarshal string into Go struct field .tgId of type int64`.
- `settings`, `streamSettings` and `sniffing` come back as **nested objects**
  since 3.7.0, where older panels returned JSON-encoded strings.
  `models.RawJSON` decodes either form into the raw JSON text.
- An unauthenticated `/panel/api/*` request answers **`404`, not `401`**, so the
  client treats `401`/`403`/`404` alike as a possibly stale session and retries
  **once** after re-authenticating.
- **A `SUCCESS` response is not proof the mutation landed.** Adds and deletes are
  verified with a follow-up read.
- Traffic is unlimited (`TotalGB: 0`); `ExpiryTime` is Unix **milliseconds**
  (`0` = unlimited).
- Subscriptions are served by whatever `XRAY_SUB_URL_PREFIX` points at — not
  necessarily the panel. A user's URL is `{prefix}/{subId}`.

## 🖥 Host integration (`internal/services/node.go`)

The bot shares a host with the VPN stack, so the node screens **call the tooling
that already owns each answer** rather than reimplementing it — one source of
truth, and a server-side change needs no change here.

| Screen field | Source |
|---|---|
| Channel | `ss -ltn`, looking for a listener on `443` |
| REALITY donor | `/etc/vpn-dest` |
| Watchdog | last line of `/var/log/vpn-watchdog.log` |
| Services | `systemctl is-active x-ui vpn-watchdog.timer fail2ban` |
| Uptime / memory / disk | `uptime -p`, `free -m`, `df -h /` |
| Subscription stats | `/usr/local/bin/vpn-sub-stats` |

Rules that hold this together:

- Paths, unit names and timeouts live in `internal/constants` and are grouped
  into `services.NodeTargets` (`DefaultNodeTargets()`); nothing is spelled out
  in a handler.
- `NodeService` is pure I/O: it runs commands with `exec.CommandContext` under
  the request context plus a timeout. All parsing and formatting is in
  `internal/helpers/node.go` and unit-tested.
- **A failed probe is shown as "no data".** `NodeService.Status` never returns an
  error and never lets one dead command blank the screen, and a probe whose
  command could not run reports `ProbeUnknown` rather than "not listening".
- Values read from the host are HTML-escaped before they reach Telegram, and the
  subscription script's output is stripped of ANSI sequences and trimmed to fit
  a message.
- The subscription QR encodes `{XRAY_SUB_URL_PREFIX}/{subId}` — a subscription
  link, not a config link, because `subId` is deliberately the secret
  subscription filename. Its caption says to add it **as a subscription**: a
  config imported by hand never refreshes itself.

---

## ⚙️ Configuration (env)

```env
TG_TOKEN=your_telegram_bot_token
TG_ADMIN_IDS=123456789,987654321
XRAY_API_URL=http://localhost:2053/secretpath  # panel base URL incl. web base path
XRAY_USER=admin
XRAY_PASSWORD=password123
XRAY_API_TOKEN=                                # optional; replaces user/password
XRAY_SUB_URL_PREFIX=https://sub.example.com    # link is {prefix}/{subId}
LOG_LEVEL=error
```

---

## 🛠️ Development

```bash
make build   # build binary
make run     # run the bot
make test    # go test ./...
make lint    # golangci-lint run ./...   (config: .golangci.yml)
make fmt     # gofmt -w
make vet     # go vet ./...
```

CI (`.github/workflows/ci.yml`) runs build, `go vet`, a `gofmt` check, tests
(`-race`) and `golangci-lint` on every pull request. `docker-build-push.yml`
builds and publishes the image on pushes to `main`.

### Conventions
- Format with `gofmt`; keep `golangci-lint` (govet, staticcheck, errcheck,
  ineffassign, unused, misspell, unconvert, gosimple) green.
- Comments and user-facing strings are in English; admin messages use Telegram
  **HTML** parse mode (`<b>…</b>`), sent via `BaseHandler.sendTextMessage`.
- Pure logic (helpers, validation, models) is unit-tested — add tests
  when changing it.
- Thread `context.Context` through to X-UI calls; don't introduce
  `context.Background()` inside handlers.
- Magic numbers belong in `internal/constants`.
