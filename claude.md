# 🤖 Claude AI Agent Guide — X-UI Telegram Admin Bot

## 📋 Project Overview

**X-UI Telegram Admin Bot** is a Go application that manages an X-UI (3x-ui) VPN
panel through a Telegram bot with a role-based access system.

### 🎯 Purpose
Automate VPN user management from Telegram: admins create/delete users, monitor
traffic and online status; trusted users self-serve a small number of accounts.

---

## 🏗️ Architecture

Flow: **Telegram update → permission check → handler (by role) → services → X-UI HTTP client**.

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
│   │   ├── admin.go             # AdminHandler: dispatch, /start, trusted delegation
│   │   ├── admin_members.go     # Admin: user create / edit / delete flows
│   │   ├── admin_traffic.go     # Admin: online list, usage reports, traffic resets
│   │   ├── admin_client_operations.go # Admin: client creation across inbounds
│   │   ├── admin_trusted.go     # Admin: grant/revoke trusted users
│   │   └── trusted.go           # TrustedHandler: self-service VPN accounts
│   ├── helpers/                 # Pure helpers: username, traffic, subscription formatting
│   ├── models/                  # Data models (Client, Inbound, MemberInfo, state, trusted)
│   ├── permissions/             # Access control (Admin / Trusted / None)
│   ├── services/                # XrayService, UserStateService, QRService, StorageService
│   └── validation/              # Username/duration validation
└── pkg/
    ├── telegrambot/bot.go       # Bot wiring, middleware, update routing
    └── xrayclient/client.go     # HTTP client for the X-UI API
```

---

## 🔐 Roles & Permissions (`internal/permissions`)

Three access types — **no Demo/User roles exist**:

- **`Admin`** — Telegram IDs listed in `TG_ADMIN_IDS`. Full access.
- **`Trusted`** — users stored in `data.json` (added by an admin via `Add Trusted`).
  May create up to `constants.MaxTrustedAccounts` (3) of their own VPN accounts.
- **`None`** — everyone else; the bot refuses to serve them.

`PermissionController.GetAccessType(userID)` resolves the role. Trusted users are
first added by `@username` (with a pseudo Telegram ID derived from the username
hash) and reconciled to their real Telegram ID on their first message
(`Bot.checkAndUpdateTrustedUser`).

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
`AwaitConfirmResetUsersNetworkUsage`, `StateAwaitingTrustedUsername`.

State is stored in-memory in `UserStateService` (a `go-cache` with a 30-minute TTL).

---

## 🗄️ Storage (`internal/services/storage.go`)

`StorageService` persists trusted users and their VPN accounts to a JSON file
(`data.json` by default), written atomically (temp file + rename) and guarded by a
`sync.RWMutex`. It holds `TrustedUser` and `VpnAccount` records.

---

## 🔧 X-UI API (`pkg/xrayclient`)

`XRAY_API_URL` is the panel **base URL** (no `/api` suffix). The client calls:

- `POST {base}/login` — authenticate (session cookie is cached)
- `GET  {base}/xui/API/inbounds` — list inbounds
- `POST {base}/xui/API/inbounds/addClient`
- `POST {base}/xui/API/inbounds/{id}/delClient/{uuid}`
- `POST {base}/xui/API/inbounds/{id}/resetClientTraffic/{email}`
- `POST {base}/xui/API/inbounds/onlines`

Data conventions: one user maps to several clients (`username-1`, `username-2`, …,
one per enabled inbound) sharing a common `SubID`; traffic is unlimited
(`TotalGB: 0`); `ExpiryTime` is a Unix timestamp in milliseconds (`0` = unlimited).

---

## ⚙️ Configuration (env)

```env
TG_TOKEN=your_telegram_bot_token
TG_ADMIN_IDS=123456789,987654321
XRAY_USER=admin
XRAY_PASSWORD=password123
XRAY_API_URL=http://localhost:54321          # base URL, NO /api suffix
XRAY_SUB_URL_PREFIX=http://localhost:54321/sub
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
- Pure logic (helpers, validation, models, storage) is unit-tested — add tests
  when changing it.
- Thread `context.Context` through to X-UI calls; don't introduce
  `context.Background()` inside handlers.
- Magic numbers belong in `internal/constants`.
