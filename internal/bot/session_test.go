package bot

import (
	"path/filepath"
	"testing"
)

func TestSQLiteSessionManager(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_sessions.db")

	sm, err := NewSessionManager(dbPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create SessionManager: %v", err)
	}
	defer sm.Close()

	userID := int64(123456789)

	// Test default session
	sess := sm.Get(userID)
	if sess.Branch != "main" {
		t.Errorf("Expected default branch main, got %s", sess.Branch)
	}
	if sess.RepoURL != "" {
		t.Errorf("Expected empty repo URL, got %s", sess.RepoURL)
	}

	// Update session properties
	sm.SetRepo(userID, "git@github.com:test/repo.git")
	sm.SetBranch(userID, "dev")
	sm.SetSSHKey(userID, "-----BEGIN OPENSSH PRIVATE KEY-----\ntestkey\n-----END OPENSSH PRIVATE KEY-----")
	sm.SetToken(userID, "ghp_testtoken123")
	sm.SetAuthor(userID, "John Doe", "john@example.com")

	// Retrieve updated session
	sessUpdated := sm.Get(userID)
	if sessUpdated.RepoURL != "git@github.com:test/repo.git" {
		t.Errorf("Unexpected repo URL: %s", sessUpdated.RepoURL)
	}
	if sessUpdated.Branch != "dev" {
		t.Errorf("Unexpected branch: %s", sessUpdated.Branch)
	}
	if sessUpdated.Token != "ghp_testtoken123" {
		t.Errorf("Unexpected token: %s", sessUpdated.Token)
	}
	if sessUpdated.AuthorName != "John Doe" || sessUpdated.AuthorEmail != "john@example.com" {
		t.Errorf("Unexpected author: %s <%s>", sessUpdated.AuthorName, sessUpdated.AuthorEmail)
	}

	// Test persistence across database reopening
	sm.Close()

	smReopened, err := NewSessionManager(dbPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen SessionManager: %v", err)
	}
	defer smReopened.Close()

	sessPersisted := smReopened.Get(userID)
	if sessPersisted.RepoURL != "git@github.com:test/repo.git" {
		t.Errorf("Persisted repo URL mismatch: %s", sessPersisted.RepoURL)
	}
	if sessPersisted.Branch != "dev" {
		t.Errorf("Persisted branch mismatch: %s", sessPersisted.Branch)
	}
}
