package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStageCommitPushDryRun(t *testing.T) {
	// Ensure git is installed
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not found in PATH")
	}

	tempDir := t.TempDir()

	// Initialize a local git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, out)
	}

	// Create initial commit on main
	initialFile := filepath.Join(tempDir, "README.md")
	os.WriteFile(initialFile, []byte("# Test Repo"), 0644)

	exec.Command("git", "-C", tempDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", tempDir, "add", "-A").Run()
	exec.Command("git", "-C", tempDir, "commit", "-m", "Initial commit").Run()

	// Modify file and add a new file to simulate zip extract replacement
	os.WriteFile(initialFile, []byte("# Updated Test Repo"), 0644)
	os.WriteFile(filepath.Join(tempDir, "new_file.go"), []byte("package main"), 0644)

	// Run StageCommitPush in DryRun mode
	gitSvc := NewGitService("Bot User", "bot@example.com", true)
	res, err := gitSvc.StageCommitPush(tempDir, "main", "Updated via test", "Bot User", "bot@example.com", "")

	if err != nil {
		t.Fatalf("StageCommitPush failed: %v", err)
	}

	if res.FilesStaged != 2 {
		t.Errorf("Expected 2 staged files, got %d", res.FilesStaged)
	}

	if res.CommitHash == "" || res.CommitHash == "unknown" {
		t.Errorf("Expected valid commit hash, got %s", res.CommitHash)
	}

	if res.Pushed != false {
		t.Errorf("Expected Pushed=false in dry run mode")
	}
}

func TestFormatRepoURL(t *testing.T) {
	url := "https://github.com/user/repo.git"
	token := "ghp_secret_token_123"

	formatted := FormatRepoURL(url, token)
	expected := "https://x-access-token:ghp_secret_token_123@github.com/user/repo.git"

	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}

	sshURL := "git@github.com:user/repo.git"
	if FormatRepoURL(sshURL, token) != sshURL {
		t.Errorf("SSH URL should remain unchanged")
	}

	rawURL := "github.com/Via04/abap-autoempl.git"
	expectedSSH := "git@github.com:Via04/abap-autoempl.git"
	if FormatRepoURL(rawURL, "") != expectedSSH {
		t.Errorf("Expected raw URL to be converted to %s, got %s", expectedSSH, FormatRepoURL(rawURL, ""))
	}
}
