// Package auth handles OAuth2 authentication with Google, including config
// loading, token storage, and the browser-based consent flow.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration loaded from config.json.
type Config struct {
	ClientID            string   `json:"client_id"`
	ClientSecret        string   `json:"client_secret"`
	DefaultSpreadsheet  string   `json:"default_spreadsheet"`
	AllowedSpreadsheets []string `json:"allowed_spreadsheets"`
}

// configDir returns the sheets-mcp configuration directory path.
func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine config directory: %w", err)
	}
	return filepath.Join(base, "sheets-mcp"), nil
}

// ConfigPath returns the full path to config.json.
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig reads and validates the configuration file. It returns a clear
// error if client_id or client_secret are missing.
func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadConfigFrom(path)
}

// LoadConfigFrom reads and validates a configuration file at the given path.
func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found at %s. See docs/SETUP.md for instructions", path)
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("missing client_id and client_secret in config. See docs/SETUP.md for instructions")
	}

	return &cfg, nil
}
