package bot

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"telegram-git-pusher/config"
	"telegram-git-pusher/internal/archive"
	"telegram-git-pusher/internal/git"

	tele "gopkg.in/telebot.v3"
)

// BotHandler handles Telegram commands and events.
type BotHandler struct {
	bot        *tele.Bot
	cfg        *config.Config
	gitService *git.GitService
	sessions   *SessionManager
}

// NewBotHandler creates a new BotHandler.
func NewBotHandler(b *tele.Bot, cfg *config.Config, gitSvc *git.GitService, sm *SessionManager) *BotHandler {
	return &BotHandler{
		bot:        b,
		cfg:        cfg,
		gitService: gitSvc,
		sessions:   sm,
	}
}

// RegisterHandlers registers all Telegram commands and media handlers.
func (h *BotHandler) RegisterHandlers() {
	h.bot.Handle("/start", h.handleHelp)
	h.bot.Handle("/help", h.handleHelp)
	h.bot.Handle("/status", h.handleStatus)
	h.bot.Handle("/setrepo", h.handleSetRepo)
	h.bot.Handle("/setbranch", h.handleSetBranch)
	h.bot.Handle("/setkey", h.handleSetKey)
	h.bot.Handle("/clearkey", h.handleClearKey)
	h.bot.Handle("/settoken", h.handleSetToken)
	h.bot.Handle("/cleartoken", h.handleClearToken)

	// Document handler for zip files
	h.bot.Handle(tele.OnDocument, h.handleDocument)
}

// Security middleware check
func (h *BotHandler) isAuthorized(c tele.Context) bool {
	sender := c.Sender()
	if sender == nil {
		return false
	}

	username := sender.Username
	userID := sender.ID

	allowed := h.cfg.IsUserAllowed(username, userID)
	if !allowed {
		c.Send(fmt.Sprintf("⛔ Access Denied. User @%s (ID: %d) is not authorized.", username, userID))
	}
	return allowed
}

func (h *BotHandler) handleHelp(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}

	msg := `🤖 *Telegram Git Pusher Bot*

Send a *.zip archive* containing your source code to automatically update and push changes to your Git repository!

*How to use:*
1️⃣ *Option A (Caption)*: Send a ZIP file with caption:
   ` + "`git@github.com:username/repository.git main \"Commit message\"`" + `

2️⃣ *Option B (Saved Settings)*: Set default repo & branch once, then just send ZIP files:
   • /setrepo ` + "`git@github.com:user/repo.git`" + `
   • /setbranch ` + "`main`" + `
   • /setkey ` + "`<paste SSH private key>`" + ` (for private repos)
   • /settoken ` + "`<paste Personal Access Token>`" + ` (for HTTPS)

3️⃣ *Option C (In-ZIP Config)*: Include a ` + "`.tg-git.yaml`" + ` manifest in the zip root.

*Available Commands:*
• /status - Show active defaults & configuration
• /setrepo <url> - Set default repository URL
• /setbranch <name> - Set default target branch
• /setkey - Register SSH Private Key for private repos
• /clearkey - Clear saved SSH Key
• /settoken <token> - Register HTTPS Access Token
• /cleartoken - Clear saved Token
• /help - Show this message`

	return c.Send(msg, tele.ModeMarkdown)
}

func (h *BotHandler) handleStatus(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}

	sess := h.sessions.Get(c.Sender().ID)

	repo := sess.RepoURL
	if repo == "" {
		repo = "*(Not set)*"
	}

	branch := sess.Branch
	if branch == "" {
		branch = "main"
	}

	hasKey := "❌ No key set"
	if sess.SSHKeyPEM != "" {
		hasKey = "✅ SSH Private Key configured"
	} else if h.cfg.DefaultSSHKeyPath != "" {
		hasKey = fmt.Sprintf("✅ Server default key (%s)", h.cfg.DefaultSSHKeyPath)
	}

	hasToken := "❌ No token set"
	if sess.Token != "" {
		hasToken = "✅ Access Token configured"
	}

	dryRunStatus := "❌ Disabled (live push)"
	if h.cfg.DryRun {
		dryRunStatus = "⚠️ Enabled (local commit only, push skipped)"
	}

	msg := fmt.Sprintf("📊 *Current User Settings*\n\n"+
		"• *Repo URL:* %s\n"+
		"• *Default Branch:* `%s`\n"+
		"• *SSH Authentication:* %s\n"+
		"• *HTTPS Token:* %s\n"+
		"• *Dry Run Mode:* %s\n\n"+
		"_Use /setrepo, /setbranch, /setkey, or /settoken to change defaults._",
		repo, branch, hasKey, hasToken, dryRunStatus)

	return c.Send(msg, tele.ModeMarkdown)
}

func (h *BotHandler) handleSetRepo(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}

	repoURL := strings.TrimSpace(c.Data())
	if repoURL == "" {
		return c.Send("❌ Please specify repo URL. Usage: `/setrepo git@github.com:user/repo.git`", tele.ModeMarkdown)
	}

	h.sessions.SetRepo(c.Sender().ID, repoURL)
	return c.Send(fmt.Sprintf("✅ Default repository updated to: `%s`", repoURL), tele.ModeMarkdown)
}

func (h *BotHandler) handleSetBranch(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}

	branch := strings.TrimSpace(c.Data())
	if branch == "" {
		return c.Send("❌ Please specify branch name. Usage: `/setbranch main`", tele.ModeMarkdown)
	}

	h.sessions.SetBranch(c.Sender().ID, branch)
	return c.Send(fmt.Sprintf("✅ Default branch updated to: `%s`", branch), tele.ModeMarkdown)
}

func (h *BotHandler) handleSetKey(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}

	keyPEM := strings.TrimSpace(c.Data())
	if keyPEM == "" {
		return c.Send("🔑 To set SSH Key, send: `/setkey <PEM content>` or upload your key file with caption `/setkey`", tele.ModeMarkdown)
	}

	h.sessions.SetSSHKey(c.Sender().ID, keyPEM)
	return c.Send("✅ SSH Private Key saved for your session.", tele.ModeMarkdown)
}

func (h *BotHandler) handleClearKey(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}
	h.sessions.SetSSHKey(c.Sender().ID, "")
	return c.Send("🗑 SSH Private Key cleared.", tele.ModeMarkdown)
}

func (h *BotHandler) handleSetToken(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}

	token := strings.TrimSpace(c.Data())
	if token == "" {
		return c.Send("🔑 Please specify HTTPS Token. Usage: `/settoken ghp_xxxx...`", tele.ModeMarkdown)
	}

	h.sessions.SetToken(c.Sender().ID, token)
	return c.Send("✅ Personal Access Token saved for your session.", tele.ModeMarkdown)
}

func (h *BotHandler) handleClearToken(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}
	h.sessions.SetToken(c.Sender().ID, "")
	return c.Send("🗑 HTTPS Token cleared.", tele.ModeMarkdown)
}

// handleDocument processes uploaded ZIP archives or SSH key files
func (h *BotHandler) handleDocument(c tele.Context) error {
	if !h.isAuthorized(c) {
		return nil
	}

	doc := c.Message().Document
	if doc == nil {
		return nil
	}

	caption := strings.TrimSpace(c.Message().Caption)

	// If document uploaded with /setkey caption, handle saving SSH key file!
	if strings.HasPrefix(caption, "/setkey") {
		fileReader, err := h.bot.File(&doc.File)
		if err != nil {
			return c.Send("❌ Failed to download SSH key file.")
		}
		defer fileReader.Close()

		keyBytes, err := io.ReadAll(fileReader)
		if err != nil {
			return c.Send("❌ Failed to read SSH key file content.")
		}

		keyContent := NormalizeSSHKeyPEM(string(keyBytes))
		h.sessions.SetSSHKey(c.Sender().ID, keyContent)
		return c.Send("✅ SSH Private Key saved successfully from file upload!", tele.ModeMarkdown)
	}

	// Ensure document is a ZIP file
	if !strings.HasSuffix(strings.ToLower(doc.FileName), ".zip") {
		return c.Send("⚠️ Please send a **.zip** file archive containing your code, or upload your SSH private key file with caption `/setkey`.", tele.ModeMarkdown)
	}

	userID := c.Sender().ID
	sess := h.sessions.Get(userID)

	// Step 1: Parse parameters from caption or user defaults
	caption = strings.TrimSpace(c.Message().Caption)
	repoURL, branch, commitMsg := parseCaption(caption, sess.RepoURL, sess.Branch)

	// Progress notification message
	statusMsg, err := h.bot.Send(c.Recipient(), "⏳ *Step 1/5:* Downloading archive...", tele.ModeMarkdown)
	if err != nil {
		log.Printf("Error sending status message: %v", err)
	}

	updateStatus := func(text string) {
		if statusMsg != nil {
			h.bot.Edit(statusMsg, text, tele.ModeMarkdown)
		}
	}

	// Create temporary workspace for this job
	jobID := fmt.Sprintf("job_%d_%d", userID, time.Now().UnixNano())
	workDir := filepath.Join(h.cfg.WorkDir, jobID)
	zipPath := filepath.Join(workDir, "archive.zip")
	cloneDir := filepath.Join(workDir, "repo")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		updateStatus("❌ Failed to create work directory on server.")
		return fmt.Errorf("mkdir failed: %w", err)
	}
	defer os.RemoveAll(workDir) // Clean up temp workspace after processing

	// Download file from Telegram
	if err := h.bot.Download(&doc.File, zipPath); err != nil {
		updateStatus(fmt.Sprintf("❌ Failed to download archive: %v", err))
		return err
	}

	// Step 2: Extract Zip
	updateStatus("📦 *Step 2/5:* Extracting zip archive...")
	extractResult, err := archive.ExtractZip(zipPath, filepath.Join(workDir, "extracted"))
	if err != nil {
		updateStatus(fmt.Sprintf("❌ Failed to extract zip file: %v", err))
		return err
	}

	// Override params if manifest exists inside zip
	if extractResult.Manifest != nil {
		m := extractResult.Manifest
		if m.Repo != "" {
			repoURL = m.Repo
		}
		if m.Branch != "" {
			branch = m.Branch
		}
		if m.Message != "" {
			commitMsg = m.Message
		}
		if m.SSHKey != "" {
			h.sessions.SetSSHKey(userID, m.SSHKey)
		}
		if m.Token != "" {
			h.sessions.SetToken(userID, m.Token)
		}
	}

	// Verify we have a repository URL
	if repoURL == "" {
		updateStatus("❌ **Missing Repository URL!**\n\n" +
			"Please specify the repo URL in the document caption:\n" +
			"`git@github.com:user/repo.git main \"commit message\"`\n\n" +
			"Or set a default repo using `/setrepo <repo_url>` first.")
		return nil
	}

	if branch == "" {
		branch = "main"
	}

	// Step 3: Determine SSH key path or token
	sshKeyPath := ""
	if sess.SSHKeyPEM != "" {
		keyPath, err := h.sessions.SaveSSHKeyToFile(userID)
		if err == nil {
			sshKeyPath = keyPath
			defer os.Remove(keyPath)
		}
	} else if h.cfg.DefaultSSHKeyPath != "" {
		sshKeyPath = h.cfg.DefaultSSHKeyPath
	}

	token := sess.Token

	// Step 4: Clone repo
	updateStatus(fmt.Sprintf("⚙️ *Step 3/5:* Cloning branch `%s` from `%s`...", branch, sanitizeURL(repoURL)))
	err = h.gitService.CloneOrPrepare(repoURL, branch, cloneDir, sshKeyPath, token)
	if err != nil {
		updateStatus(fmt.Sprintf("❌ **Git Clone Failed:**\n`%v`", err))
		return nil
	}

	// Step 5: Copy extracted files with replacement
	updateStatus("🔄 *Step 4/5:* Replacing files in repository...")
	extractedDir := filepath.Join(workDir, "extracted")
	err = copyDirReplace(extractedDir, cloneDir)
	if err != nil {
		updateStatus(fmt.Sprintf("❌ **File Copy Failed:**\n`%v`", err))
		return nil
	}

	// Step 6: Commit and Push
	updateStatus("🚀 *Step 5/5:* Committing and pushing changes to remote...")
	gitRes, err := h.gitService.StageCommitPush(
		cloneDir,
		branch,
		commitMsg,
		sess.AuthorName,
		sess.AuthorEmail,
		sshKeyPath,
	)

	if err != nil {
		updateStatus(fmt.Sprintf("❌ **Git Operation Failed:**\n`%v`", err))
		return nil
	}

	if gitRes.FilesStaged == 0 {
		updateStatus("ℹ️ *No changes detected.* Repository is already up to date with the sent archive.")
		return nil
	}

	pushStatusMsg := "🚀 Pushed to remote"
	if h.cfg.DryRun || !gitRes.Pushed {
		pushStatusMsg = "🔒 Committed locally (Dry Run / Push Disabled)"
	}

	// Success Summary Message
	finalMsg := fmt.Sprintf("✅ *Operation Successful!*\n\n"+
		"• *Repository:* `%s`\n"+
		"• *Branch:* `%s`\n"+
		"• *Commit Hash:* `%s`\n"+
		"• *Files Updated:* %d\n"+
		"• *Commit Message:* %s\n"+
		"• *Status:* %s\n",
		sanitizeURL(repoURL),
		gitRes.Branch,
		gitRes.CommitHash,
		gitRes.FilesStaged,
		commitMsg,
		pushStatusMsg,
	)

	updateStatus(finalMsg)
	return nil
}

// parseCaption extracts repo, branch, commit message from caption or defaults
func parseCaption(caption, defaultRepo, defaultBranch string) (repo, branch, message string) {
	repo = defaultRepo
	branch = defaultBranch
	message = "Update code via Telegram Bot"

	if caption == "" {
		return
	}

	// Check key=value format: repo=... branch=... msg=...
	if strings.Contains(caption, "=") {
		kvMap := parseKeyValueCaption(caption)
		if r, ok := kvMap["repo"]; ok && r != "" {
			repo = r
		}
		if b, ok := kvMap["branch"]; ok && b != "" {
			branch = b
		}
		if m, ok := kvMap["msg"]; ok && m != "" {
			message = m
		}
		if m, ok := kvMap["message"]; ok && m != "" {
			message = m
		}
		return
	}

	// Positional format: <repo_url> [branch] [commit message]
	// Example: git@github.com:user/repo.git feature/login "Fix login bug"
	re := regexp.MustCompile(`[^\s"']+|"[^"]*"|'[^']*'`)
	tokens := re.FindAllString(caption, -1)

	for i := range tokens {
		tokens[i] = strings.Trim(tokens[i], "\"'")
	}

	if len(tokens) >= 1 && tokens[0] != "" {
		repo = tokens[0]
	}
	if len(tokens) >= 2 && tokens[1] != "" {
		branch = tokens[1]
	}
	if len(tokens) >= 3 && tokens[2] != "" {
		message = strings.Join(tokens[2:], " ")
	}

	return
}

func parseKeyValueCaption(caption string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Fields(caption)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			k := strings.ToLower(parts[0])
			v := strings.Trim(parts[1], "\"")
			result[k] = v
		}
	}
	return result
}

// copyDirReplace copies all files from src to dst, replacing existing files, ignoring .git
func copyDirReplace(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip .git folder
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy file with overwrite
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func sanitizeURL(url string) string {
	// Hide tokens if present in HTTPS URLs
	re := regexp.MustCompile(`https://[^@]+@`)
	return re.ReplaceAllString(url, "https://***@")
}
