package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

// expiryBuffer is how far before actual expiry we consider a token expired,
// to avoid races with in-flight requests.
const expiryBuffer = 30 * time.Second

// refreshTimeout bounds the OAuth2 token refresh HTTP request (matches code-exchange timeout in oauth.go).
const refreshTimeout = 30 * time.Second

// TokenPath returns the full path to token.json.
func TokenPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// LoadToken reads the stored OAuth2 token from disk.
func LoadToken() (*oauth2.Token, error) {
	path, err := TokenPath()
	if err != nil {
		return nil, err
	}
	return LoadTokenFrom(path)
}

// LoadTokenFrom reads an OAuth2 token from the given path.
func LoadTokenFrom(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no stored token found at %s. Run 'sheets-mcp auth' to authenticate", path)
		}
		return nil, fmt.Errorf("read token file: %w", err)
	}

	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse token file %s: %w", path, err)
	}

	return &tok, nil
}

// SaveToken writes an OAuth2 token to disk with 0600 permissions.
func SaveToken(tok *oauth2.Token) error {
	path, err := TokenPath()
	if err != nil {
		return err
	}
	return SaveTokenTo(path, tok)
}

// SaveTokenTo writes an OAuth2 token to the given path with 0600 permissions.
func SaveTokenTo(path string, tok *oauth2.Token) error {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	return nil
}

// DeleteToken removes the stored token file.
func DeleteToken() error {
	path, err := TokenPath()
	if err != nil {
		return err
	}
	return DeleteTokenAt(path)
}

// DeleteTokenAt removes the token file at the given path.
func DeleteTokenAt(path string) error {
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete token file: %w", err)
	}
	return nil
}

// TokenValid reports whether the token has a non-expired access token.
func TokenValid(tok *oauth2.Token) bool {
	if tok == nil || tok.AccessToken == "" {
		return false
	}
	if tok.Expiry.IsZero() {
		return true
	}
	return time.Now().Add(expiryBuffer).Before(tok.Expiry)
}

// RefreshIfNeeded checks the token's expiry and refreshes it using the refresh
// token if needed. Returns the (possibly refreshed) token. If the token was
// refreshed, it is saved to disk at the default path.
func RefreshIfNeeded(cfg *oauth2.Config, tok *oauth2.Token) (*oauth2.Token, error) {
	path, err := TokenPath()
	if err != nil {
		return nil, err
	}
	return RefreshIfNeededTo(cfg, tok, path)
}

// RefreshIfNeededTo checks the token's expiry and refreshes it if needed,
// saving the result to the given path.
func RefreshIfNeededTo(cfg *oauth2.Config, tok *oauth2.Token, savePath string) (*oauth2.Token, error) {
	if TokenValid(tok) {
		return tok, nil
	}

	if tok.RefreshToken == "" {
		return nil, fmt.Errorf("access token expired and no refresh token available. Run 'sheets-mcp auth' to re-authenticate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()
	src := cfg.TokenSource(ctx, tok)
	newTok, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token failed (token may be revoked). Run 'sheets-mcp auth' to re-authenticate: %w", err)
	}

	if err := SaveTokenTo(savePath, newTok); err != nil {
		return nil, fmt.Errorf("save refreshed token: %w", err)
	}

	return newTok, nil
}
