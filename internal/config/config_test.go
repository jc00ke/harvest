package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// resetKeyring installs a fresh in-memory keyring mock for the duration of t.
// Without this, tests would touch the developer's real OS keyring.
func resetKeyring(t *testing.T) {
	t.Helper()
	keyring.MockInit()
}

func TestConfig(t *testing.T) {
	t.Run("given a valid config file when loaded then returns config with correct values", func(t *testing.T) {
		resetKeyring(t)
		tempDir := t.TempDir()

		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })
		os.Setenv("HOME", tempDir)

		configDir := filepath.Join(tempDir, ".config", "harvest-tui")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		configContent := `[harvest]
account_id = "12345"
access_token = "abc123def456"
`
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config.Harvest.AccountID != "12345" {
			t.Errorf("expected account_id 12345, got %s", config.Harvest.AccountID)
		}
		if config.Harvest.AccessToken != "abc123def456" {
			t.Errorf("expected access_token abc123def456, got %s", config.Harvest.AccessToken)
		}
	})

	t.Run("given a config with missing account_id when validated then returns error", func(t *testing.T) {
		config := &Config{
			Harvest: HarvestConfig{
				AccountID:   "",
				AccessToken: "abc123def456",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Fatal("expected error for missing account_id")
		}

		expectedMsg := "account_id is required.\n\nTo get started, set up your Harvest API credentials:\n" + SetupInstructionsURL
		if err.Error() != expectedMsg {
			t.Errorf("expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("given a config with missing access_token when validated then returns error", func(t *testing.T) {
		config := &Config{
			Harvest: HarvestConfig{
				AccountID:   "12345",
				AccessToken: "",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Fatal("expected error for missing access_token")
		}

		expectedMsg := "access_token is required.\n\nTo get started, set up your Harvest API credentials:\n" + SetupInstructionsURL
		if err.Error() != expectedMsg {
			t.Errorf("expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("given a valid config when validated then returns no error", func(t *testing.T) {
		config := &Config{
			Harvest: HarvestConfig{
				AccountID:   "12345",
				AccessToken: "abc123def456",
			},
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("expected no error for valid config, got %v", err)
		}
	})

	t.Run("given no keyring entry and missing config file when loaded then returns helpful error mentioning auth login", func(t *testing.T) {
		resetKeyring(t)
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })
		os.Setenv("HOME", tempDir)

		_, err := Load()
		if err == nil {
			t.Fatal("expected error for missing credentials")
		}

		expectedPath := filepath.Join(tempDir, ".config", "harvest-tui", "config.toml")
		expectedMsg := "could not load credentials. Run `harvest-cli auth login` to store credentials in your OS keyring, or create " + expectedPath + " with your Harvest credentials.\n\nTo get started, set up your Harvest API credentials:\n" + SetupInstructionsURL
		if err.Error() != expectedMsg {
			t.Errorf("expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("given malformed config file when loaded then returns parse error", func(t *testing.T) {
		resetKeyring(t)
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })
		os.Setenv("HOME", tempDir)

		configDir := filepath.Join(tempDir, ".config", "harvest-tui")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		malformedContent := `[harvest
account_id = "12345"
access_token = "abc123"
`
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(malformedContent), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Load()
		if err == nil {
			t.Fatal("expected error for malformed config file")
		}

		if err.Error()[:28] != "could not parse config file:" {
			t.Errorf("expected parse error, got '%s'", err.Error())
		}
	})

	t.Run("given config file missing harvest section when loaded then returns validation error", func(t *testing.T) {
		resetKeyring(t)
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })
		os.Setenv("HOME", tempDir)

		configDir := filepath.Join(tempDir, ".config", "harvest-tui")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		emptyContent := `[other]
setting = "value"
`
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(emptyContent), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Load()
		if err == nil {
			t.Fatal("expected error for config missing harvest fields")
		}

		if err.Error() != "invalid config: account_id is required.\n\nTo get started, set up your Harvest API credentials:\n"+SetupInstructionsURL {
			t.Errorf("expected account_id required error, got '%s'", err.Error())
		}
	})

	t.Run("given credentials stored in keyring when loaded then returns those credentials", func(t *testing.T) {
		resetKeyring(t)
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })
		os.Setenv("HOME", tempDir)

		err := StoreCredentialsInKeyring(HarvestConfig{
			AccountID:   "keyring-account",
			AccessToken: "keyring-token",
		})
		if err != nil {
			t.Fatalf("StoreCredentialsInKeyring: %v", err)
		}

		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config.Harvest.AccountID != "keyring-account" {
			t.Errorf("expected account_id keyring-account, got %s", config.Harvest.AccountID)
		}
		if config.Harvest.AccessToken != "keyring-token" {
			t.Errorf("expected access_token keyring-token, got %s", config.Harvest.AccessToken)
		}
	})

	t.Run("given credentials in both keyring and file when loaded then keyring takes precedence", func(t *testing.T) {
		resetKeyring(t)
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })
		os.Setenv("HOME", tempDir)

		configDir := filepath.Join(tempDir, ".config", "harvest-tui")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		fileContent := `[harvest]
account_id = "file-account"
access_token = "file-token"
`
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(fileContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := StoreCredentialsInKeyring(HarvestConfig{
			AccountID:   "keyring-account",
			AccessToken: "keyring-token",
		})
		if err != nil {
			t.Fatal(err)
		}

		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config.Harvest.AccountID != "keyring-account" {
			t.Errorf("expected keyring to win for account_id, got %s", config.Harvest.AccountID)
		}
		if config.Harvest.AccessToken != "keyring-token" {
			t.Errorf("expected keyring to win for access_token, got %s", config.Harvest.AccessToken)
		}
	})

	t.Run("given empty keyring and valid file when loaded then file is used", func(t *testing.T) {
		resetKeyring(t)
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })
		os.Setenv("HOME", tempDir)

		configDir := filepath.Join(tempDir, ".config", "harvest-tui")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		fileContent := `[harvest]
account_id = "file-account"
access_token = "file-token"
`
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(fileContent), 0644); err != nil {
			t.Fatal(err)
		}

		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config.Harvest.AccountID != "file-account" {
			t.Errorf("expected file account_id, got %s", config.Harvest.AccountID)
		}
	})

	t.Run("given no keyring entry when LoadFromKeyring called then returns ErrNotFound", func(t *testing.T) {
		resetKeyring(t)
		_, err := LoadFromKeyring()
		if !errors.Is(err, keyring.ErrNotFound) {
			t.Errorf("expected keyring.ErrNotFound, got %v", err)
		}
	})

	t.Run("given credentials in keyring when DeleteCredentialsFromKeyring called then entry is removed", func(t *testing.T) {
		resetKeyring(t)
		if err := StoreCredentialsInKeyring(HarvestConfig{AccountID: "a", AccessToken: "b"}); err != nil {
			t.Fatal(err)
		}
		if err := DeleteCredentialsFromKeyring(); err != nil {
			t.Fatalf("DeleteCredentialsFromKeyring: %v", err)
		}
		if _, err := LoadFromKeyring(); !errors.Is(err, keyring.ErrNotFound) {
			t.Errorf("expected keyring.ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("given invalid credentials when StoreCredentialsInKeyring called then returns validation error without writing", func(t *testing.T) {
		resetKeyring(t)
		err := StoreCredentialsInKeyring(HarvestConfig{AccountID: "", AccessToken: "tok"})
		if err == nil {
			t.Fatal("expected validation error for empty account_id")
		}
		if _, err := LoadFromKeyring(); !errors.Is(err, keyring.ErrNotFound) {
			t.Errorf("expected nothing stored, got %v", err)
		}
	})
}
