package config

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

// keyringService is the OS keyring service name.
const keyringService = "harvest"

// keyringUser is the fixed account identifier for the single keyring entry that
// holds the Harvest account ID and access token as a JSON-encoded blob.
const keyringUser = "default"

// LoadFromKeyring returns credentials from the OS keyring. Returns
// keyring.ErrNotFound when no entry exists, or another error if the keyring
// itself is unavailable or the stored value is malformed.
func LoadFromKeyring() (*Config, error) {
	raw, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return nil, err
	}

	var hc HarvestConfig
	if err := json.Unmarshal([]byte(raw), &hc); err != nil {
		return nil, fmt.Errorf("malformed keyring entry: %w", err)
	}

	cfg := &Config{Harvest: hc}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid keyring credentials: %w", err)
	}
	return cfg, nil
}

// StoreCredentialsInKeyring writes credentials to the OS keyring after validating them.
func StoreCredentialsInKeyring(hc HarvestConfig) error {
	if err := (&Config{Harvest: hc}).Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(hc)
	if err != nil {
		return fmt.Errorf("could not encode credentials: %w", err)
	}
	return keyring.Set(keyringService, keyringUser, string(data))
}

// DeleteCredentialsFromKeyring removes credentials from the OS keyring. Returns
// keyring.ErrNotFound if no entry existed.
func DeleteCredentialsFromKeyring() error {
	return keyring.Delete(keyringService, keyringUser)
}
