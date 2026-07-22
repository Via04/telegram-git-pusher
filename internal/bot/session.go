package bot

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// UserSession stores per-user configuration defaults.
type UserSession struct {
	UserID      int64
	RepoURL     string
	Branch      string
	SSHKeyPEM   string
	Token       string
	AuthorName  string
	AuthorEmail string
	UpdatedAt   time.Time
}

// SessionManager manages user sessions persisted in SQLite.
type SessionManager struct {
	db      *sql.DB
	tempDir string
	mu      sync.Mutex
}

// NewSessionManager initializes SQLite database and session manager.
func NewSessionManager(dbPath, tempDir string) (*SessionManager, error) {
	// Ensure directory for db exists
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open SQLite database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Enable WAL mode and busy timeout for concurrent safety and performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set sqlite pragma: %w", err)
	}

	// Create user_sessions table if not exists
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS user_sessions (
		user_id INTEGER PRIMARY KEY,
		repo_url TEXT DEFAULT '',
		branch TEXT DEFAULT 'main',
		ssh_key_pem TEXT DEFAULT '',
		token TEXT DEFAULT '',
		author_name TEXT DEFAULT '',
		author_email TEXT DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create user_sessions table: %w", err)
	}

	return &SessionManager{
		db:      db,
		tempDir: tempDir,
	}, nil
}

// Close closes the database connection.
func (sm *SessionManager) Close() error {
	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}

// Get fetches the session for a user ID from SQLite (creating default record if absent).
func (sm *SessionManager) Get(userID int64) *UserSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	query := `SELECT user_id, repo_url, branch, ssh_key_pem, token, author_name, author_email, updated_at FROM user_sessions WHERE user_id = ?`
	row := sm.db.QueryRow(query, userID)

	var s UserSession
	var updatedAtStr string
	err := row.Scan(&s.UserID, &s.RepoURL, &s.Branch, &s.SSHKeyPEM, &s.Token, &s.AuthorName, &s.AuthorEmail, &updatedAtStr)
	if err == sql.ErrNoRows {
		// Create default record
		insertSQL := `INSERT INTO user_sessions (user_id, repo_url, branch, ssh_key_pem, token, author_name, author_email, updated_at)
					  VALUES (?, '', 'main', '', '', '', '', CURRENT_TIMESTAMP)`
		_, _ = sm.db.Exec(insertSQL, userID)
		return &UserSession{
			UserID: userID,
			Branch: "main",
		}
	} else if err != nil {
		// Fallback in case of database read error
		return &UserSession{
			UserID: userID,
			Branch: "main",
		}
	}

	if s.Branch == "" {
		s.Branch = "main"
	}
	return &s
}

// SetRepo sets the repo URL for a user ID in SQLite.
func (sm *SessionManager) SetRepo(userID int64, repoURL string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	upsertSQL := `
	INSERT INTO user_sessions (user_id, repo_url, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(user_id) DO UPDATE SET
		repo_url = excluded.repo_url,
		updated_at = CURRENT_TIMESTAMP;`
	_, _ = sm.db.Exec(upsertSQL, userID, repoURL)
}

// SetBranch sets the default branch for a user ID in SQLite.
func (sm *SessionManager) SetBranch(userID int64, branch string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	upsertSQL := `
	INSERT INTO user_sessions (user_id, branch, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(user_id) DO UPDATE SET
		branch = excluded.branch,
		updated_at = CURRENT_TIMESTAMP;`
	_, _ = sm.db.Exec(upsertSQL, userID, branch)
}

// SetSSHKey sets the SSH private key PEM content for a user ID in SQLite.
func (sm *SessionManager) SetSSHKey(userID int64, pemKey string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	upsertSQL := `
	INSERT INTO user_sessions (user_id, ssh_key_pem, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(user_id) DO UPDATE SET
		ssh_key_pem = excluded.ssh_key_pem,
		updated_at = CURRENT_TIMESTAMP;`
	_, _ = sm.db.Exec(upsertSQL, userID, pemKey)
}

// SetToken sets the HTTPS token for a user ID in SQLite.
func (sm *SessionManager) SetToken(userID int64, token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	upsertSQL := `
	INSERT INTO user_sessions (user_id, token, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(user_id) DO UPDATE SET
		token = excluded.token,
		updated_at = CURRENT_TIMESTAMP;`
	_, _ = sm.db.Exec(upsertSQL, userID, token)
}

// SetAuthor sets the Git author name and email in SQLite.
func (sm *SessionManager) SetAuthor(userID int64, name, email string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	upsertSQL := `
	INSERT INTO user_sessions (user_id, author_name, author_email, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(user_id) DO UPDATE SET
		author_name = excluded.author_name,
		author_email = excluded.author_email,
		updated_at = CURRENT_TIMESTAMP;`
	_, _ = sm.db.Exec(upsertSQL, userID, name, email)
}

// SaveSSHKeyToFile writes the user's SSH key PEM to a temp file and returns its path.
func (sm *SessionManager) SaveSSHKeyToFile(userID int64) (string, error) {
	s := sm.Get(userID)
	if s.SSHKeyPEM == "" {
		return "", nil
	}

	keysDir := filepath.Join(sm.tempDir, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return "", err
	}

	normalizedPEM := NormalizeSSHKeyPEM(s.SSHKeyPEM)
	keyPath := filepath.Join(keysDir, fmt.Sprintf("%d_id_rsa", userID))
	if absPath, err := filepath.Abs(keyPath); err == nil {
		keyPath = absPath
	}
	// Write with strict 0600 permissions required by SSH
	if err := os.WriteFile(keyPath, []byte(normalizedPEM), 0600); err != nil {
		return "", err
	}
	return keyPath, nil
}

// NormalizeSSHKeyPEM cleans and formats SSH private key strings to ensure valid OpenSSH PEM format.
func NormalizeSSHKeyPEM(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Standardize line endings
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	headers := []string{
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"-----BEGIN RSA PRIVATE KEY-----",
		"-----BEGIN PRIVATE KEY-----",
		"-----BEGIN EC PRIVATE KEY-----",
		"-----BEGIN DSA PRIVATE KEY-----",
	}
	footers := []string{
		"-----END OPENSSH PRIVATE KEY-----",
		"-----END RSA PRIVATE KEY-----",
		"-----END PRIVATE KEY-----",
		"-----END EC PRIVATE KEY-----",
		"-----END DSA PRIVATE KEY-----",
	}

	var header, footer string
	for _, h := range headers {
		if strings.Contains(raw, h) {
			header = h
			break
		}
	}
	for _, f := range footers {
		if strings.Contains(raw, f) {
			footer = f
			break
		}
	}

	if header != "" && footer != "" {
		startIdx := strings.Index(raw, header) + len(header)
		endIdx := strings.Index(raw, footer)
		if startIdx < endIdx {
			body := strings.TrimSpace(raw[startIdx:endIdx])
			// Replace any spaces inside body with newlines
			bodyFields := strings.Fields(body)
			bodyClean := strings.Join(bodyFields, "\n")
			return header + "\n" + bodyClean + "\n" + footer + "\n"
		}
	}

	// Fallback cleanup
	lines := strings.Split(raw, "\n")
	var cleaned []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			cleaned = append(cleaned, t)
		}
	}
	return strings.Join(cleaned, "\n") + "\n"
}
