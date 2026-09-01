# 🚀 X-UI Telegram Admin Bot

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Telegram](https://img.shields.io/badge/Telegram-Bot-blue.svg)
![X-Ray](https://img.shields.io/badge/X--Ray-Panel-orange.svg)

**Modern Telegram bot for managing X-UI panel with role-based access and advanced features**

[🚀 Quick Start](#quick-start) • [📋 Features](#features) • [⚙️ Installation](#installation) • [🔧 Configuration](#configuration) • [📖 Usage](#usage)

</div>

---

## 🎯 What is this?

**X-UI Telegram Admin Bot** is a modern solution for managing VPN servers through Telegram. The bot provides full control over the X-UI panel directly from the messenger with an intuitive interface and role-based access system.

### 🌟 Key advantages

- **🔐 Access control**: Admin-only (everyone else is denied)
- **📱 User-friendly interface**: Intuitive buttons and menus with proper HTML formatting
- **⚡ Fast operation**: Session caching and optimized requests
- **🔄 Automation**: Bulk operations and automatic management
- **📊 Monitoring**: Real-time traffic statistics and connection status
- **🔒 Security**: Access control verification and data validation
- **🎯 Smart navigation**: Universal button command handling with emoji support
- **🏗️ Modern architecture**: Clean modular structure with dependency injection

---

## 📋 Features

### 👑 Administrator
- ✅ **User creation** with expiration time settings (including infinite duration)
- 🔄 **Traffic management** (reset individual or all users)
- 👥 **Online users view** with real-time connection status
- 📊 **Detailed usage statistics** with aggregated data
- 🗑️ **User deletion** with confirmation dialogs
- 🔗 **QR code generation** for configurations and for subscription links
- 🖥 **Node status** of the host: channel, REALITY donor, watchdog, services, uptime, memory, disk
- 📡 **Subscription request statistics** straight from the host's own script
- ⚙️ **Bulk operations** (reset traffic for all users)
- 🎯 **Smart navigation** with universal return buttons

Only Telegram IDs listed in `TG_ADMIN_IDS` may use the bot; everyone else is denied.

---

## 🚀 Quick Start

### Requirements
- **Docker** and **Docker Compose**
- **X-UI panel** with API access
- **Telegram Bot Token**

### ⚡ Super Quick Start (Using Pre-built Image)

**No git clone needed! Just 3 commands:**

```bash
# 1. Create project directory
mkdir xui-tg-admin && cd xui-tg-admin

# 2. Download docker-compose.yml
curl -o docker-compose.yml https://raw.githubusercontent.com/d3kause/xui-tg-admin/main/docker-compose.yml

# 3. Edit configuration and start
nano docker-compose.yml  # Edit your settings
docker-compose up -d
```

### 🔧 Manual Docker Compose Setup

Create `docker-compose.yml`:

```yaml
services:
  x-ui-tg-go:
    image: ghcr.io/d3kause/xui-tg-admin:latest
    container_name: x-ui-tg-go
    restart: unless-stopped
    environment:
      # Replace with your actual values
      - TG_TOKEN=1234567890:YOUR_BOT_TOKEN_FROM_BOTFATHER
      - TG_ADMIN_IDS=123456789,987654321
      - XRAY_API_URL=http://localhost:2053/secretpath
      - XRAY_USER=admin
      - XRAY_PASSWORD=your_xui_panel_password
      # Optional: replaces the username/password login when set
      # - XRAY_API_TOKEN=
      - XRAY_SUB_URL_PREFIX=https://sub.example.com
      - LOG_LEVEL=info
    volumes:
      - ./data:/root/data
```

Then run:
```bash
docker-compose up -d
```

### 🛠️ Development Setup (From Source)

```bash
# 1. Clone and build
git clone https://github.com/d3kause/xui-tg-admin.git
cd xui-tg-admin
go mod download
go build -o xui-tg-admin ./cmd/bot

# 2. Set environment variables
export TG_TOKEN=your_telegram_bot_token
export TG_ADMIN_IDS=123456789,987654321
export XRAY_API_URL=http://localhost:2053/secretpath
export XRAY_USER=admin
export XRAY_PASSWORD=password123
export XRAY_SUB_URL_PREFIX=https://sub.example.com

# 3. Run
./xui-tg-admin
```

---

## ⚙️ Configuration

### 🔑 Required Configuration

Replace these values in your `docker-compose.yml`:

| Parameter | Description | Example |
|-----------|-------------|---------|
| `TG_TOKEN` | Get from @BotFather | `1234567890:ABCdef_your_token` |
| `TG_ADMIN_IDS` | Your Telegram ID(s) | `123456789,987654321` |
| `XRAY_API_URL` | 3x-ui panel base URL, including its secret web base path, no trailing slash | `http://localhost:2053/secretpath` |
| `XRAY_USER` | Panel username | `admin` |
| `XRAY_PASSWORD` | Panel password | `your_secure_password` |
| `XRAY_API_TOKEN` | *Optional.* Panel API token (Settings → Security → API Token). Replaces user/password and skips the CSRF handshake. A full-admin credential — guard it like the password. | `` |
| `XRAY_SUB_URL_PREFIX` | Base URL of the subscription front end. A user's link is `{prefix}/{subId}`. | `https://sub.example.com` |

### 📝 How to get required values

1. **Telegram Bot Token**:
   - Message @BotFather in Telegram
   - Send `/newbot` and follow instructions
   - Copy the token

2. **Your Telegram ID**:
   - Message @userinfobot in Telegram
   - Send any message to get your ID

3. **X-UI Panel Settings**:
   - Ensure X-UI panel is running
   - Use your admin credentials
   - Replace `YOUR_SERVER_IP` with your actual server IP

---

## 📖 Usage

### 🎮 Administrator interface

#### Main menu
```
┌─────────────────────────┐
│    🏠 Main Menu         │
├─────────────────────────┤
│  👤 Add Member  │ 🟢 Online │
│  ✏️ Edit Member │ 📈 Detailed│
│  🖥 Node Status │ 📡 Subs   │
│  🔄 Reset Network Usage │
└─────────────────────────┘
```

#### User management
```
┌─────────────────────────┐
│  👤 vasya_pupkin        │
├─────────────────────────┤
│  🔗 View Config │ 📱 Sub QR │
│  🔄 Reset │ 🗑️ Delete   │
│  ↩️ Return to Main Menu │
└─────────────────────────┘
```

### 📱 Administrator commands

| Command | Description | Example |
|---------|-------------|---------|
| `/start` | Start the bot | `/start` |
| `Add Member` | Add user | Creates user with expiration settings |
| `Edit Member` | Edit user | View, reset traffic, delete |
| `Online Members` | Online users | Active connections plus a last-seen list for everyone else |
| `Node Status` | Host state | Channel, REALITY donor, watchdog, services, uptime, memory, disk — with an inline **Refresh** button that redraws the same message |
| `Subscriptions` | Subscription requests | Output of the host's `vpn-sub-stats` script, verbatim |
| `Detailed Usage` | Detailed statistics | Traffic by users and inbounds |
| `Reset Network Usage` | Reset all traffic | Bulk operation with confirmation |

### 🔄 Workflow

1. **User creation**:
   ```
   Add Member → Enter name → Choose duration (∞ Infinite available) → ✅ Done!
   ```

2. **User management**:
   ```
   Edit Member → Select user → Action → Result
   ```

3. **Monitoring**:
   ```
   Online Members → Currently connected, then when everyone else was last seen
   Detailed Usage → Traffic statistics with aggregation
   Node Status → Host health, refreshed in place
   Subscriptions → Who fetched their subscription, when and from where
   ```

3. **Handing a user their subscription**:
   ```
   Edit Member → Select user → 📱 Subscription QR
   ```
   The QR encodes the **subscription URL**, not a config link, and the caption
   says so: a link imported as a plain config never updates itself.

### 🎯 Smart Navigation

The bot features universal button handling:
- **↩️ Return to Main Menu** - Works from any state
- **∞ Infinite** - For unlimited duration subscriptions
- **✅ Confirm** - For confirmation dialogs
- **❌ Cancel** - For cancellation

---

## 🔌 3x-ui panel API

The bot targets the **3x-ui 3.7.0** panel API. Two things differ from the legacy
x-ui API and both are load-bearing.

**1. Unsafe requests need a CSRF token.** A bare `POST /login` answers `403` with
an empty body. The client fetches the panel page, reads the token out of
`<meta name="csrf-token" content="...">`, refreshes it via `GET /csrf-token`
once the session exists, and replays it in the `X-CSRF-Token` header alongside
the `3x-ui` session cookie. Setting `XRAY_API_TOKEN` skips all of this — the
panel short-circuits CSRF for Bearer callers.

**2. A client is a first-class entity, not a row inside one inbound.** It is
created once with the list of inbounds it should attach to, and is addressed by
its email everywhere afterwards. The old scheme of one client per inbound named
`username-1`, `username-2`, … is gone; a user is now a single record. Clients
created by earlier versions keep their suffixed emails and are still grouped
into one user by their shared `subId`.

### Endpoints used

| Purpose | Endpoint |
|---|---|
| Log in | `POST {base}/login` |
| Refresh CSRF token | `GET {base}/csrf-token` |
| List inbounds | `GET {base}/panel/api/inbounds/list` |
| List clients | `GET {base}/panel/api/clients/list` |
| Read one client | `GET {base}/panel/api/clients/get/{email}` |
| Share links for a client | `GET {base}/panel/api/clients/links/{email}` |
| Create a client | `POST {base}/panel/api/clients/add` |
| Delete a client | `POST {base}/panel/api/clients/del/{email}` |
| Reset one client's traffic | `POST {base}/panel/api/clients/resetTraffic/{email}` |
| Reset every client's traffic | `POST {base}/panel/api/clients/resetAllTraffics` |
| Who is connected now | `POST {base}/panel/api/clients/onlines` |
| When each client was last seen | `POST {base}/panel/api/clients/lastOnline` |

The panel serves its own OpenAPI description at
`GET {base}/panel/api/openapi.json` (session required), with a browsable UI at
`{base}/panel/api-docs`. That spec is the authority on request and response
shapes.

### Gotchas worth knowing

- **`tgId` is an `int64`, not a string.** Sending a string fails with
  `json: cannot unmarshal string into Go struct field .tgId of type int64`.
- **`settings`, `streamSettings` and `sniffing` are nested objects since 3.7.0**,
  where older panels returned them as JSON-encoded strings. `models.RawJSON`
  accepts either form.
- **An unauthenticated `/panel/api/*` request answers `404`, not `401`.** The
  client therefore treats `401`, `403` and `404` alike as a possibly expired
  session and retries once after re-authenticating.
- **A `SUCCESS` response is not proof the mutation landed.** Adds and deletes are
  verified with a follow-up read rather than trusted.
- **Subscriptions need not come from the panel.** `XRAY_SUB_URL_PREFIX` is
  whatever front end serves them; a user's URL is that prefix joined with the
  client's `subId`. If the front end builds its links on a schedule, a freshly
  created user's subscription URL only resolves once it has refreshed — so the
  bot shows the panel's direct share link too, and puts that in the QR code
  because it works immediately.

---

## 🖥 Host integration

The bot runs on the same box as the VPN stack, so the node screens call the
tooling that already owns each answer instead of reimplementing it. That keeps
one source of truth and means a change on the server needs no change here.

| Screen | Reads |
|---|---|
| Channel | `ss -ltn` (looks for a listener on `443`) |
| REALITY donor | `/etc/vpn-dest` |
| Watchdog | last line of `/var/log/vpn-watchdog.log` |
| Services | `systemctl is-active x-ui vpn-watchdog.timer fail2ban` |
| Uptime / memory / disk | `uptime -p`, `free -m`, `df -h /` |
| Subscriptions | `/usr/local/bin/vpn-sub-stats` |

Every path and unit name is declared once in `internal/constants` and grouped
into `services.NodeTargets`. A probe that cannot be run is shown as **no data**;
it never aborts the screen and never silently reads as "fine". These commands
must exist on the host, so the node screens are meaningless in a container that
does not share it — the rest of the bot works regardless.

---

## 🏗️ Architecture

### 📁 Project structure

```
xui-tg-admin/
├── 📂 cmd/bot/           # Application entry point
│   └── main.go           # Main application file
├── 📂 internal/          # Internal logic
│   ├── 📂 commands/      # Command constants
│   ├── 📂 config/        # Configuration and loading
│   ├── 📂 constants/     # Application constants
│   ├── 📂 handlers/      # Telegram handlers
│   │   ├── admin.go                   # Admin: dispatch, start, trusted delegation
│   │   ├── admin_members.go           # Admin: user create/edit/delete
│   │   ├── admin_traffic.go           # Admin: online/usage/traffic reset
│   │   ├── admin_client_operations.go # Admin: client creation across inbounds
│   │   ├── admin_trusted.go           # Admin: trusted user management
│   │   ├── base.go                    # Shared handler helpers
│   │   ├── factory.go                 # Handler factory (by access type)
│   │   └── trusted.go                 # Trusted user handler
│   ├── 📂 helpers/       # Helper functions
│   ├── 📂 models/        # Data models
│   ├── 📂 permissions/   # Access control system
│   ├── 📂 services/      # Business logic
│   └── 📂 validation/    # Data validation
├── 📂 pkg/               # Reusable packages
│   ├── 📂 telegrambot/   # Telegram bot
│   └── 📂 xrayclient/    # X-UI API client
└── 📄 Configuration files
```

### 🔧 Main components

- **`handlers/`** - Telegram message handlers with role system and smart button handling
- **`services/`** - Business logic and X-UI API integration
- **`xrayclient/`** - HTTP client for X-UI API with session management
- **`permissions/`** - Role and access control system
- **`commands/`** - Centralized command constants
- **`models/`** - Data structures for clients, inbounds, and states
- **`config/`** - Configuration loading and validation

### 🎯 Key Architecture Features

- **Modular structure**: Clear separation of concerns between components
- **Role-based system**: Different handlers for different user types
- **State management**: User state tracking in conversations
- **Session caching**: Optimized X-UI API requests
- **Universal button handling**: Single system for all emoji buttons
- **Dependency injection**: Clean testable architecture

---

## 🛠️ Development

A `Makefile` wraps the common tasks:

```bash
make build   # build the bot binary
make run     # run the bot
make test    # go test ./...
make lint    # golangci-lint run ./...
make fmt     # gofmt -w
make vet     # go vet ./...
```

### 🔨 Building

```bash
# Development build
go build -o xui-tg-admin ./cmd/bot

# Production build
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xui-tg-admin ./cmd/bot
```

### 🧪 Testing & linting

```bash
# Run tests (with race detector, as CI does)
go test -race ./...

# Static analysis (see .golangci.yml)
golangci-lint run ./...
```

CI runs build, `go vet`, `gofmt` check, tests and `golangci-lint` on every pull request
(see `.github/workflows/ci.yml`).

### 📝 Logging

```bash
# Log levels
LOG_LEVEL=debug  # Detailed logs
LOG_LEVEL=info   # Information messages
LOG_LEVEL=warn   # Warnings only
LOG_LEVEL=error  # Errors only
```

### 🐛 Debugging

Key logging points:
- X-UI API authentication
- Client creation/deletion
- API request errors
- User states
- Command and button handling

---

## 🆕 Recent Updates

### ✅ Fixed Issues
- **Smart button handling**: Universal command extraction from emoji buttons
- **HTML formatting**: Proper `<b>` tags rendering in all messages
- **Navigation**: Return to Main Menu works from any state
- **User experience**: Improved error messages and confirmation dialogs

### 🎯 Key Improvements
- **Universal button processing**: Single function handles all emoji buttons
- **Better error handling**: More informative error messages
- **Consistent UI**: All messages use proper HTML formatting
- **Robust navigation**: Return buttons work reliably across all states
- **Optimized architecture**: Clear separation of responsibilities between components

---

## 🔧 Docker

### 📦 Pre-built Docker Image

The easiest way to run the bot is using the pre-built Docker image:

```bash
# Pull and run directly
docker run -d \
  --name xui-tg-go \
  --restart unless-stopped \
  -e TG_TOKEN="YOUR_BOT_TOKEN" \
  -e TG_ADMIN_IDS="YOUR_TELEGRAM_ID" \
  -e XRAY_API_URL="http://localhost:2053/secretpath" \
  -e XRAY_USER="admin" \
  -e XRAY_PASSWORD="your_password" \
  -e XRAY_SUB_URL_PREFIX="https://sub.example.com" \
  -e LOG_LEVEL="info" \
  -v $(pwd)/data:/root/data \
  ghcr.io/d3kause/xui-tg-admin:latest
```

### 🔄 Updates

```bash
# Update to latest version
docker-compose pull
docker-compose up -d

# View logs
docker-compose logs -f
```

### 🛠️ Build from source

```yaml
services:
  x-ui-tg-go:
    build: .  # Build from local source
    container_name: x-ui-tg-go
    restart: unless-stopped
    environment:
      - TG_TOKEN=your_token
      # ... other variables
```

---

## 🤝 Contributing

We welcome contributions to the project!

### 📋 How to help

1. 🍴 Fork the repository
2. 🌿 Create a branch for new feature
3. 💾 Commit your changes
4. 🔀 Create a Pull Request

### 📝 Code standards

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Add tests for new functionality
- Update documentation when changing API

---

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [X-UI](https://github.com/vaxilu/x-ui) - Excellent X-Ray management panel
- [Telegram Bot API](https://core.telegram.org/bots/api) - Telegram Bot API
- [Go](https://golang.org/) - Go programming language
- [Telebot](https://gopkg.in/telebot.v3) - Telegram Bot framework for Go

---

<div align="center">

**⭐ If you liked the project, give it a star!**

[🚀 Start using](#quick-start) • [📖 Documentation](#usage) • [🐛 Report bug](https://github.com/d3kause/xui-tg-admin/issues)

</div>
