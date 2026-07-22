package bot

import (
	"os"
	"path/filepath"
	"sync"
)

// UserSession stores per-user configuration defaults.
type UserSession struct {
	RepoURL     string
	Branch      string
	SSHKeyPEM   string
	Token       string
	AuthorName  string
	AuthorEmail string
}

// SessionManager manages user sessions in-memory.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
	tempDir  string
}

// NewSessionManager creates a new SessionManager instance.
func NewSessionManager(tempDir string) *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*UserSession),
		tempDir:  tempDir,
	}
}

// Get returns the session for a user ID (creating empty one if needed).
func (sm *SessionManager) Get(userID int64) *UserSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[userID]
	if !ok {
		s = &UserSession{
			Branch: "main", // Default branch
		}
		sm.sessions[userID] = s
	}
	return s
}

// SetRepo sets the repo URL for a user ID.
func (sm *SessionManager) SetRepo(userID int64, repoURL string) {
	s := sm.Get(userID)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s.RepoURL = repoURL
}

// SetBranch sets the default branch for a user ID.
func (sm *SessionManager) SetBranch(userID int64, branch string) {
	s := sm.Get(userID)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s.Branch = branch
}

// SetSSHKey sets the SSH private key PEM content for a user ID.
func (sm *SessionManager) SetSSHKey(userID int64, pemKey string) {
	s := sm.Get(userID)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s.SSHKeyPEM = pemKey
}

// SetToken sets the HTTPS token for a user ID.
func (sm *SessionManager) SetToken(userID int64, token string) {
	s := sm.Get(userID)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s.Token = token
}

// SetAuthor sets the Git author name and email.
func (sm *SessionManager) SetAuthor(userID int64, name, email string) {
	s := sm.Get(userID)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s.AuthorName = name
	s.AuthorEmail = email
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

	keyPath := filepath.Join(keysDir, filepath.Clean(filepath.FromSlash(string(rune(userID))+"_id_rsa")))
	// Write with strict 0600 permissions required by SSH
	if err := os.WriteFile(keyPath, []byte(s.SSHKeyPEM), 0600); err != nil {
		return "", err
	}
	return keyPath, nil
}
