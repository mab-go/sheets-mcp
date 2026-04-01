package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFrom_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{
		"client_id": "test-id",
		"client_secret": "test-secret",
		"default_spreadsheet": "abc123",
		"allowed_spreadsheets": ["abc123", "def456"]
	}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ClientID != "test-id" {
		t.Errorf("ClientID = %q, want %q", cfg.ClientID, "test-id")
	}
	if cfg.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %q, want %q", cfg.ClientSecret, "test-secret")
	}
	if cfg.DefaultSpreadsheet != "abc123" {
		t.Errorf("DefaultSpreadsheet = %q, want %q", cfg.DefaultSpreadsheet, "abc123")
	}
	if len(cfg.AllowedSpreadsheets) != 2 {
		t.Errorf("AllowedSpreadsheets len = %d, want 2", len(cfg.AllowedSpreadsheets))
	}
}

func TestLoadConfigFrom_MinimalValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"client_id": "id", "client_secret": "secret"}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DefaultSpreadsheet != "" {
		t.Errorf("DefaultSpreadsheet = %q, want empty", cfg.DefaultSpreadsheet)
	}
	if cfg.AllowedSpreadsheets != nil {
		t.Errorf("AllowedSpreadsheets = %v, want nil", cfg.AllowedSpreadsheets)
	}
}

func TestLoadConfigFrom_MissingFile(t *testing.T) {
	_, err := LoadConfigFrom("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfigFrom_MissingClientID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"client_secret": "secret"}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfigFrom(path)
	if err == nil {
		t.Fatal("expected error for missing client_id")
	}
}

func TestLoadConfigFrom_MissingClientSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"client_id": "id"}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfigFrom(path)
	if err == nil {
		t.Fatal("expected error for missing client_secret")
	}
}

func TestLoadConfigFrom_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfigFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadConfigFrom_EmptyCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"client_id": "", "client_secret": ""}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfigFrom(path)
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}
