# Telegram Git Pusher Bot (Go)

A high-performance, lightweight Telegram Bot written in Go that allows you to push code updates to Git repositories directly via Telegram `.zip` archive uploads.

This tool is designed for restricted network environments (e.g., work networks where GitHub or personal servers are blocked but Telegram is accessible). You can send a `.zip` archive containing updated source code to your bot running on an unblocked server, and it will automatically clone the repository, replace the files, stage changes, commit, and push the new version to your Git repository (handling branches and private repository credentials).

---

## 🌟 Key Features

- 📦 **Zip Archive Ingestion**: Extract code directly into the target git working tree with file replacement.
- 🌿 **Full Branch Support**: Target existing branches or create new branches dynamically.
- 🗄️ **Zero-Leak SQLite Persistence**: Uses a pure Go SQLite database (`modernc.org/sqlite`) to store per-user configuration defaults (repo URL, branch, SSH keys, tokens) with zero memory leaks across process restarts.
- 🔑 **Private Repository Support**:
  - **SSH Keys**: Upload `.pem` key files directly or use `/setkey`. Keys are stored per-user in SQLite and formatted automatically with valid OpenSSH line endings.
  - **HTTPS Tokens**: Set Personal Access Token (`/settoken`) for token-authenticated HTTPS pushes.
- ⚙️ **Flexible Parameter Passing**:
  1. **Direct File Caption**: Attach your `.zip` with your commit message as caption (`/setrepo` configured).
  2. **Key-Value Format**: `repo=... branch=... msg="My commit"`
  3. **Positional Caption**: `git@github.com:user/repo.git main "My commit message"`
  4. **In-ZIP Manifest**: Include a `.tg-git.yaml` configuration inside your `.zip` archive root.
- 🐳 **Docker & Docker Compose**: Includes multi-stage `Dockerfile` and `docker-compose.yml` for instant production deployment.
- 🛡️ **Security & Reliability**: Restricted user access list (`ALLOWED_USERS`), Zip Slip path traversal protection, non-interactive Git Credential Manager settings, and absolute SSH path resolution.
- 🧪 **Dry Run Mode**: Test locally (`DRY_RUN=true`) to clone, unzip, stage, and commit locally without executing `git push`.

---

## 📁 Project Structure

```
telegram-git-pusher/
├── cmd/
│   └── bot/
│       └── main.go           # Application entry point
├── config/
│   └── config.go             # Configuration loader & permission check
├── internal/
│   ├── archive/
│   │   ├── extractor.go      # Zip extractor with Zip Slip & manifest support
│   │   └── extractor_test.go # Extractor unit tests
│   ├── bot/
│   │   ├── handler.go        # Telegram commands & document handlers
│   │   ├── handler_test.go   # Caption parsing unit tests
│   │   ├── session.go        # SQLite database session manager
│   │   └── session_test.go   # SQLite session unit tests
│   └── git/
│       ├── git.go            # Git clone, checkout, stage, commit, push service
│       └── git_test.go       # Git operations unit tests
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── README.md
└── go.mod
```

---

## 🚀 Quick Start

### 1. Prerequisites
- Go 1.20+
- `git` CLI installed on server
- Telegram Bot Token from [@BotFather](https://t.me/BotFather)

### 2. Running Locally

```bash
# Set environment variables
export TELEGRAM_BOT_TOKEN="your_bot_token_from_botfather"
export ALLOWED_USERS="your_telegram_username"

# Run tests
go test -v ./...

# Run Bot
go run cmd/bot/main.go
```

Or pass flags directly:
```bash
go run cmd/bot/main.go -token "YOUR_TELEGRAM_TOKEN" -dry-run=false
```

---

## 🐳 Docker Deployment

### Using Docker Compose (Recommended)

1. Create a `.env` file on your server:
   ```bash
   TELEGRAM_BOT_TOKEN=123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ
   ALLOWED_USERS=your_telegram_username
   DRY_RUN=false
   ```

2. Start the container:
   ```bash
   docker compose up -d --build
   ```

---

## 🤖 Telegram Bot Usage Guide

### Method 1: Set Defaults Once & Send Zip File (Recommended)
1. Set default repository URL:
   ```text
   /setrepo git@github.com:username/my-repo.git
   ```
2. Set default branch (optional, defaults to `main`):
   ```text
   /setbranch main
   ```
3. Register your SSH key for private repos by **uploading your `bot_key` file directly to Telegram with caption `/setkey`**.
4. Simply attach your `.zip` archive in Telegram and type your **commit message as caption**:
   ```text
   Fix authentication bug and update database pool
   ```

### Method 2: Explicit Key-Value Caption
Attach your `.zip` archive with caption:
```text
repo=git@github.com:username/my-repo.git branch=dev msg="Fixed API bug"
```

### Method 3: In-ZIP Manifest
Include a `.tg-git.yaml` file in the root of your `.zip` archive:
```yaml
repo: git@github.com:username/my-repo.git
branch: main
message: "Implement new feature"
```

---

## 🔒 Security & User Isolation

- Each Telegram user's credentials, SSH keys, HTTPS tokens, default repositories, and branches are isolated by their unique Telegram `user_id` in SQLite.
- Temporary SSH keys are generated per job with strict `0600` file permissions and deleted immediately after git operations complete.
- Path traversal (Zip-Slip) attacks are blocked during extraction.
