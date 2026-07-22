package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration settings.
type Config struct {
	TelegramToken     string
	AllowedUsers      []string
	WorkDir           string
	DBPath            string
	DefaultSSHKeyPath string
	DefaultGitName    string
	DefaultGitEmail   string
	DryRun            bool
}

// LoadConfig loads configuration from environment variables with sensible defaults.
func LoadConfig() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		// We allow empty token for dry-run testing / command line flags,
		// but will check before starting the bot.
	}

	allowedUsersStr := os.Getenv("ALLOWED_USERS")
	var allowedUsers []string
	if allowedUsersStr != "" {
		for _, u := range strings.Split(allowedUsersStr, ",") {
			trimmed := strings.TrimSpace(u)
			if trimmed != "" {
				allowedUsers = append(allowedUsers, strings.TrimPrefix(trimmed, "@"))
			}
		}
	}

	workDir := os.Getenv("WORK_DIR")
	if workDir == "" {
		workDir = "./tmp_repos"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./bot.db"
	}

	dryRun, _ := strconv.ParseBool(os.Getenv("DRY_RUN"))

	gitName := os.Getenv("DEFAULT_GIT_NAME")
	if gitName == "" {
		gitName = "Telegram Bot Pusher"
	}

	gitEmail := os.Getenv("DEFAULT_GIT_EMAIL")
	if gitEmail == "" {
		gitEmail = "telegram-bot@local.dev"
	}

	cfg := &Config{
		TelegramToken:     token,
		AllowedUsers:      allowedUsers,
		WorkDir:           workDir,
		DBPath:            dbPath,
		DefaultSSHKeyPath: os.Getenv("DEFAULT_SSH_KEY_PATH"),
		DefaultGitName:    gitName,
		DefaultGitEmail:   gitEmail,
		DryRun:            dryRun,
	}

	return cfg, nil
}

// IsUserAllowed checks if a given Telegram username or string representation of user ID is authorized.
func (c *Config) IsUserAllowed(username string, userID int64) bool {
	if len(c.AllowedUsers) == 0 {
		// If no allowed users set, allow everyone (or warning mode)
		return true
	}

	strID := fmt.Sprintf("%d", userID)
	for _, allowed := range c.AllowedUsers {
		if strings.EqualFold(allowed, username) || allowed == strID {
			return true
		}
	}
	return false
}
