// Package auth stores Bitbucket credentials in the operating system keyring,
// falling back to a private file when no keyring is available.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zalando/go-keyring"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
)

const keyringService = "bk"

// Credential is a token and the HTTP authentication scheme it belongs to.
type Credential struct {
	Token    string `json:"token"`
	Username string `json:"username,omitempty"`
	Method   string `json:"method"`
}

// Store persists credentials per host.
type Store struct {
	dir string
}

// NewStore returns a credential store rooted at the config directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// FilePath returns the fallback credential file path.
func (s *Store) FilePath() string {
	return filepath.Join(s.dir, "credentials.json")
}

// Get returns the credential for hostname and where it came from ("keyring"
// or "file"). It returns an empty credential when none exists.
func (s *Store) Get(hostname string) (Credential, string, error) {
	if os.Getenv("BK_FORCE_FILE_STORE") == "" && keyringAvailable() {
		raw, err := keyring.Get(keyringService, hostname)
		if err == nil {
			var c Credential
			if err := json.Unmarshal([]byte(raw), &c); err != nil {
				return Credential{}, "", fmt.Errorf("corrupt keyring entry for %s: %w", hostname, err)
			}
			return c, "keyring", nil
		}
		if err != keyring.ErrNotFound {
			// Fall through to the file store on keyring errors such as a
			// missing or locked D-Bus session.
		}
	}
	creds, err := s.loadFile()
	if err != nil {
		return Credential{}, "", err
	}
	if c, ok := creds[hostname]; ok {
		return c, "file", nil
	}
	return Credential{}, "", nil
}

// Set stores the credential and reports which backend was used.
func (s *Store) Set(hostname string, c Credential) (string, error) {
	if c.Method == "" {
		c.Method = config.AuthBearer
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	if os.Getenv("BK_FORCE_FILE_STORE") == "" && keyringAvailable() {
		if err := keyring.Set(keyringService, hostname, string(raw)); err == nil {
			return "keyring", nil
		}
	}
	if err := s.setFile(hostname, c); err != nil {
		return "", err
	}
	return "file", nil
}

// Delete removes the credential for hostname from both backends.
func (s *Store) Delete(hostname string) error {
	if keyringAvailable() {
		_ = keyring.Delete(keyringService, hostname)
	}
	creds, err := s.loadFile()
	if err != nil {
		return err
	}
	if _, ok := creds[hostname]; ok {
		delete(creds, hostname)
		return s.saveFile(creds)
	}
	return nil
}

func keyringAvailable() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	// A missing session bus means there is no Secret Service to talk to.
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" && os.Getenv("XDG_RUNTIME_DIR") == "" {
		return false
	}
	return true
}

func (s *Store) loadFile() (map[string]Credential, error) {
	creds := map[string]Credential{}
	path := s.FilePath()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return creds, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, &creds); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return creds, nil
}

func (s *Store) saveFile(creds map[string]Credential) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, ".credentials-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.FilePath())
}

func (s *Store) setFile(hostname string, c Credential) error {
	creds, err := s.loadFile()
	if err != nil {
		return err
	}
	creds[hostname] = c
	return s.saveFile(creds)
}

// Redact returns a token with its middle hidden, for display.
func Redact(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
