package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	tok := &oauth2.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		Expiry:       time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	if err := SaveTokenTo(path, tok); err != nil {
		t.Fatalf("SaveTokenTo: %v", err)
	}

	// Verify file permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("token file permissions = %o, want 0600", perm)
	}

	loaded, err := LoadTokenFrom(path)
	if err != nil {
		t.Fatalf("LoadTokenFrom: %v", err)
	}

	if loaded.AccessToken != tok.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tok.AccessToken)
	}
	if loaded.RefreshToken != tok.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, tok.RefreshToken)
	}
}

func TestLoadTokenFrom_MissingFile(t *testing.T) {
	_, err := LoadTokenFrom("/nonexistent/token.json")
	if err == nil {
		t.Fatal("expected error for missing token file")
	}
}

func TestLoadTokenFrom_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	if err := os.WriteFile(path, []byte("{bad"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadTokenFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSaveTokenTo_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "token.json")

	tok := &oauth2.Token{AccessToken: "test"}
	if err := SaveTokenTo(path, tok); err != nil {
		t.Fatalf("SaveTokenTo with nested dirs: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("token file not created: %v", err)
	}
}

func TestDeleteTokenAt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := DeleteTokenAt(path); err != nil {
		t.Fatalf("DeleteTokenAt: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("token file still exists after delete")
	}
}

func TestDeleteTokenAt_NonExistent(t *testing.T) {
	// Deleting a non-existent file should not error.
	if err := DeleteTokenAt("/nonexistent/token.json"); err != nil {
		t.Fatalf("DeleteTokenAt non-existent: %v", err)
	}
}

func TestTokenValid(t *testing.T) {
	tests := []struct {
		name  string
		token *oauth2.Token
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  false,
		},
		{
			name:  "empty access token",
			token: &oauth2.Token{},
			want:  false,
		},
		{
			name: "expired token",
			token: &oauth2.Token{
				AccessToken: "test",
				Expiry:      time.Now().Add(-1 * time.Hour),
			},
			want: false,
		},
		{
			name: "expiring within buffer",
			token: &oauth2.Token{
				AccessToken: "test",
				Expiry:      time.Now().Add(10 * time.Second),
			},
			want: false,
		},
		{
			name: "valid token",
			token: &oauth2.Token{
				AccessToken: "test",
				Expiry:      time.Now().Add(1 * time.Hour),
			},
			want: true,
		},
		{
			name: "no expiry (zero time)",
			token: &oauth2.Token{
				AccessToken: "test",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TokenValid(tt.token); got != tt.want {
				t.Errorf("TokenValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
