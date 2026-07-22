package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitService manages git commands and repo operations.
type GitService struct {
	DefaultName  string
	DefaultEmail string
	DryRun       bool
}

// NewGitService creates a new GitService instance.
func NewGitService(defaultName, defaultEmail string, dryRun bool) *GitService {
	return &GitService{
		DefaultName:  defaultName,
		DefaultEmail: defaultEmail,
		DryRun:       dryRun,
	}
}

// GitResult holds outputs from git operations.
type GitResult struct {
	CommitHash  string
	Branch      string
	Status      string
	FilesStaged int
	Output      string
	Pushed      bool
}

// CloneOrPrepare clones the specified git repo into targetDir and checks out the requested branch.
func (g *GitService) CloneOrPrepare(repoURL, branch, targetDir, sshKeyPath, token string) error {
	finalURL := FormatRepoURL(repoURL, token)

	// Prepare env vars for SSH if key provided
	env := os.Environ()
	if sshKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", filepath.ToSlash(sshKeyPath))
		env = append(env, "GIT_SSH_COMMAND="+sshCmd)
	}

	// First try: clone specific branch directly
	args := []string{"clone", "--branch", branch, finalURL, targetDir}
	_, err := runCmdWithEnv(targetDir, env, "git", args...)
	if err == nil {
		return nil
	}

	// If clone with specific branch failed (e.g. branch doesn't exist yet on remote), clone default branch
	argsDefault := []string{"clone", finalURL, targetDir}
	if _, errDefault := runCmdWithEnv("", env, "git", argsDefault...); errDefault != nil {
		return fmt.Errorf("failed to clone repository %s: %w", repoURL, errDefault)
	}

	// Now attempt checkout branch, or create it if missing
	coArgs := []string{"checkout", branch}
	if _, errCo := runCmdWithEnv(targetDir, env, "git", coArgs...); errCo != nil {
		// Create and checkout new branch
		cobArgs := []string{"checkout", "-b", branch}
		if _, errCob := runCmdWithEnv(targetDir, env, "git", cobArgs...); errCob != nil {
			return fmt.Errorf("failed to create/checkout branch %s: %w", branch, errCob)
		}
	}

	return nil
}

// StageCommitPush stages all changes, commits with message, and pushes to remote branch.
func (g *GitService) StageCommitPush(targetDir, branch, commitMsg, authorName, authorEmail, sshKeyPath string) (*GitResult, error) {
	if authorName == "" {
		authorName = g.DefaultName
	}
	if authorEmail == "" {
		authorEmail = g.DefaultEmail
	}
	if commitMsg == "" {
		commitMsg = "Update repository files via Telegram Bot"
	}

	env := os.Environ()
	if sshKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", filepath.ToSlash(sshKeyPath))
		env = append(env, "GIT_SSH_COMMAND="+sshCmd)
	}

	// Set git local user configs
	if _, err := runCmdWithEnv(targetDir, env, "git", "config", "user.name", authorName); err != nil {
		return nil, fmt.Errorf("git config user.name failed: %w", err)
	}
	if _, err := runCmdWithEnv(targetDir, env, "git", "config", "user.email", authorEmail); err != nil {
		return nil, fmt.Errorf("git config user.email failed: %w", err)
	}

	// Stage all changes
	if _, err := runCmdWithEnv(targetDir, env, "git", "add", "-A"); err != nil {
		return nil, fmt.Errorf("git add failed: %w", err)
	}

	// Check status
	statusOutput, err := runCmdWithEnv(targetDir, env, "git", "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	statusLines := strings.Split(strings.TrimSpace(statusOutput), "\n")
	stagedCount := 0
	if statusOutput != "" {
		stagedCount = len(statusLines)
	}

	if stagedCount == 0 {
		return &GitResult{
			Branch:      branch,
			Status:      "No changes detected in working tree.",
			FilesStaged: 0,
			Pushed:      false,
		}, nil
	}

	// Commit changes
	commitOut, err := runCmdWithEnv(targetDir, env, "git", "commit", "-m", commitMsg)
	if err != nil {
		return nil, fmt.Errorf("git commit failed: %w (output: %s)", err, commitOut)
	}

	// Get latest commit hash
	hashOut, err := runCmdWithEnv(targetDir, env, "git", "rev-parse", "--short", "HEAD")
	commitHash := strings.TrimSpace(hashOut)
	if err != nil {
		commitHash = "unknown"
	}

	// Push if not dry run
	pushed := false
	var pushOutput string
	if !g.DryRun {
		// Attempt push to origin <branch>
		pushOut, errPush := runCmdWithEnv(targetDir, env, "git", "push", "origin", branch)
		if errPush != nil {
			// Try pushing with -u if upstream is not set
			pushOutUpstream, errUp := runCmdWithEnv(targetDir, env, "git", "push", "-u", "origin", branch)
			if errUp != nil {
				return nil, fmt.Errorf("git push failed: %w (output: %s)", errUp, pushOutUpstream)
			}
			pushOutput = pushOutUpstream
		} else {
			pushOutput = pushOut
		}
		pushed = true
	} else {
		pushOutput = "DRY RUN MODE: Commit created locally. Push to remote skipped."
	}

	return &GitResult{
		CommitHash:  commitHash,
		Branch:      branch,
		Status:      "Changes committed successfully",
		FilesStaged: stagedCount,
		Output:      pushOutput,
		Pushed:      pushed,
	}, nil
}

// FormatRepoURL handles HTTPS credentials insertion if token is provided.
func FormatRepoURL(repoURL, token string) string {
	if token == "" || !strings.HasPrefix(repoURL, "https://") {
		return repoURL
	}

	// Transform https://github.com/owner/repo.git -> https://x-access-token:TOKEN@github.com/owner/repo.git
	trimmed := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://x-access-token:%s@%s", token, trimmed)
}

func runCmdWithEnv(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// Disable interactive credential prompts and GUI dialogs
	env = append(env,
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=never",
		"GIT_ASKPASS=",
	)
	cmd.Env = env

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	output := outBuf.String() + "\n" + errBuf.String()
	if err != nil {
		return strings.TrimSpace(output), fmt.Errorf("command %s %v failed: %w (output: %s)", name, args, err, output)
	}
	return strings.TrimSpace(output), nil
}
