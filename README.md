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
- 🔗 **QR code generation** for configurations
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
      - XRAY_USER=admin
      - XRAY_PASSWORD=your_xui_panel_password
      - XRAY_API_URL=http://localhost:54321
      - XRAY_SUB_URL_PREFIX=http://YOUR_SERVER_IP:54321/sub
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
export XRAY_USER=admin
export XRAY_PASSWORD=password123
export XRAY_API_URL=http://localhost:8080
export XRAY_SUB_URL_PREFIX=http://localhost:8080/sub

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
| `XRAY_USER` | X-UI panel username | `admin` |
| `XRAY_PASSWORD` | X-UI panel password | `your_secure_password` |
| `XRAY_API_URL` | X-UI panel base URL (no `/api` suffix) | `http://localhost:54321` |
| `XRAY_SUB_URL_PREFIX` | Subscription URL prefix | `http://YOUR_SERVER_IP:54321/sub` |

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
│  🔄 Reset Network Usage │
└─────────────────────────┘
```

#### User management
```
┌─────────────────────────┐
│  👤 vasya_pupkin        │
├─────────────────────────┤
│  🔗 View Config         │
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
| `Online Members` | Online users | List of active connections |
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
   Online Members → Active list with real-time status
   Detailed Usage → Traffic statistics with aggregation
   ```

### 🎯 Smart Navigation

The bot features universal button handling:
- **↩️ Return to Main Menu** - Works from any state
- **∞ Infinite** - For unlimited duration subscriptions
- **✅ Confirm** - For confirmation dialogs
- **❌ Cancel** - For cancellation

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
  -e XRAY_USER="admin" \
  -e XRAY_PASSWORD="your_password" \
  -e XRAY_API_URL="http://localhost:54321" \
  -e XRAY_SUB_URL_PREFIX="http://YOUR_SERVER_IP:54321/sub" \
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
