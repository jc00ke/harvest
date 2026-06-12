package config

import (
	"errors"
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

		if got, want := err.Error(), "account_id is required.\n\nTo get started, set up your Harvest API credentials:\n"+SetupInstructionsURL; got != want {
			t.Errorf("error=%q, want=%q", got, want)
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

		if got, want := err.Error(), "access_token is required.\n\nTo get started, set up your Harvest API credentials:\n"+SetupInstructionsURL; got != want {
			t.Errorf("error=%q, want=%q", got, want)
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

	t.Run("given no keyring entry when loaded then returns helpful error mentioning auth login", func(t *testing.T) {
		resetKeyring(t)

		_, err := Load()
		if err == nil {
			t.Fatal("expected error for missing credentials")
		}

		if got, want := err.Error(), "could not load credentials. Run `harvest-cli auth login` to store credentials in your OS keyring.\n\nTo get started, set up your Harvest API credentials:\n"+SetupInstructionsURL; got != want {
			t.Errorf("error=%q, want=%q", got, want)
		}
	})

	t.Run("given credentials stored in keyring when loaded then returns those credentials", func(t *testing.T) {
		resetKeyring(t)

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

		if got, want := config.Harvest.AccountID, "keyring-account"; got != want {
			t.Errorf("account_id=%s, want=%s", got, want)
		}
		if got, want := config.Harvest.AccessToken, "keyring-token"; got != want {
			t.Errorf("access_token=%s, want=%s", got, want)
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
