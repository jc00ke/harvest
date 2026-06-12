package config

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const SetupInstructionsURL = "https://github.com/jc00ke/harvest?tab=readme-ov-file#getting-harvest-api-credentials"

type Config struct {
	Harvest HarvestConfig
}

type HarvestConfig struct {
	AccountID   string `json:"account_id"`
	AccessToken string `json:"access_token"`
}

// Load returns Harvest credentials from the OS keyring.
func Load() (*Config, error) {
	cfg, err := LoadFromKeyring()
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, fmt.Errorf("could not load credentials. Run `harvest auth login` to store credentials in your OS keyring.\n\nTo get started, set up your Harvest API credentials:\n%s", SetupInstructionsURL)
		}
		return nil, fmt.Errorf("could not load credentials from keyring: %w", err)
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Harvest.AccountID == "" {
		return fmt.Errorf("account_id is required.\n\nTo get started, set up your Harvest API credentials:\n%s", SetupInstructionsURL)
	}
	if c.Harvest.AccessToken == "" {
		return fmt.Errorf("access_token is required.\n\nTo get started, set up your Harvest API credentials:\n%s", SetupInstructionsURL)
	}
	return nil
}
