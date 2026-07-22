# Telegram Git Pusher Bot (Go)

A Telegram Bot written in Go that allows you to push code updates to Git repositories directly through Telegram.

This tool is designed for restricted network environments (e.g., work networks where GitHub or personal servers are blocked but Telegram is accessible). You can send a `.zip` archive containing updated source code to your bot running on an unblocked server, and it will clone the repository, replace the files, stage changes, commit, and push the new version to your Git repository (handling branches and private repo authentication).

---

## 🌟 Key Features

- 📦 **Zip Archive Ingestion**: Extract code directly into the target git working tree with replacement.
- 🌿 **Full Branch Support**: Target existing branches or create new branches dynamically.
- 🔑 **Private Repository Support**:
  - **SSH Keys**: Set session SSH key (`/setkey`) or use server default SSH key.
  - **HTTPS Tokens**: Set Personal Access Token (`/settoken`) for HTTPS authentication.
- ⚙️ **Flexible Parameter Passing**:
  1. **Telegram File Captions**: `<repo_url> [branch] [commit message]` or `repo=... branch=... msg=...`.
  2. **In-ZIP Manifest**: Include a `.tg-git.yaml` configuration inside your `.zip` archive root.
  3. **Saved User Defaults**: Set default repository URL (`/setrepo`) and branch (`/setbranch`) per chat session.
- 🛡️ **Security**: Restricted user access list (`ALLOWED_USERS`) and Zip Slip vulnerability protection.
- 🧪 **Dry Run Mode**: Test locally (`DRY_RUN=true`) to verify local git clone/commit without pushing to remote.

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
│   │   └── session.go        # User session manager
│   └── git/
│       ├── git.go            # Git clone, checkout, stage, commit, push service
│       └── git_test.go       # Git operations unit tests
├── .env.example
├── README.md
└── go.mod
```

---

## 🚀 Quick Start

### 1. Prerequisites
- Go 1.20+
- `git` CLI installed on server
- Telegram Bot Token from [@BotFather](https://t.me/BotFather)

### 2. Installation & Running

```bash
# Clone or navigate to directory
cd C:\dev\Bots\telegram-git-pusher

# Run tests
go test ./...

# Set environment variables
export TELEGRAM_BOT_TOKEN="your_bot_token_from_botfather"
export ALLOWED_USERS="your_telegram_username"

# Build & Run
go run cmd/bot/main.go
```

Or pass flags directly:
```bash
go run cmd/bot/main.go -token "YOUR_TELEGRAM_TOKEN" -dry-run=false
```

---

## 🤖 Telegram Bot Usage Guide

### Method 1: File Caption (Recommended)
Attach your `.zip` archive in Telegram and set the caption to:

```text
git@github.com:username/my-repo.git main "Update feature X"
```

Or key-value format:
```text
repo=git@github.com:username/my-repo.git branch=dev msg="Fixed bug in API"
```

### Method 2: Set Defaults Once
Use commands to configure default settings:
- `/setrepo git@github.com:username/my-repo.git`
- `/setbranch dev`
- `/setkey` (paste your SSH Private Key for private repos)
- `/settoken` (paste Personal Access Token for HTTPS repos)

Then simply send any `.zip` archive to the bot!

### Method 3: Manifest inside ZIP archive
Include a `.tg-git.yaml` file in the root of your `.zip` archive:

```yaml
repo: git@github.com:username/my-repo.git
branch: feature/auth
message: "Implement JWT authentication"
author_name: "Developer"
author_email: "dev@company.com"
```

---

## 🔒 Private Repository Setup

### SSH Key Authentication
1. Send `/setkey` followed by your SSH Private Key PEM string (or upload `.pem` key file).
2. The bot uses `GIT_SSH_COMMAND` to authenticate via SSH key without requiring interactive prompts.

### HTTPS Personal Access Token (PAT)
1. Send `/settoken ghp_yourPersonalAccessTokenHere`.
2. The bot automatically formats HTTPS URLs with token credentials (`https://x-access-token:TOKEN@github.com/...`).

---

## 🛠️ Local Dry Run Testing

To test git cloning and committing locally without pushing to remote:

```bash
go run cmd/bot/main.go -token "YOUR_TOKEN" -dry-run=true
```

In dry run mode, the bot clones, unzips, replaces files, stages, and commits locally, returning the commit hash to Telegram without executing `git push`.
