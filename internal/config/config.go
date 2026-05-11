package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const SetupInstructionsURL = "https://github.com/planetargon/harvest-tui?tab=readme-ov-file#getting-harvest-api-credentials"

type Config struct {
	Harvest HarvestConfig `toml:"harvest"`
}

type HarvestConfig struct {
	AccountID   string `toml:"account_id" json:"account_id"`
	AccessToken string `toml:"access_token" json:"access_token"`
}

// Load returns Harvest credentials, preferring the OS keyring and falling back
// to the TOML config file if the keyring has no entry (or is unavailable).
func Load() (*Config, error) {
	if cfg, err := LoadFromKeyring(); err == nil {
		return cfg, nil
	}
	return LoadFromFile()
}

// LoadFromFile reads credentials from the TOML config file at ~/.config/harvest-tui/config.toml.
func LoadFromFile() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("could not determine config path: %w", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("could not load credentials. Run `harvest-cli auth login` to store credentials in your OS keyring, or create %s with your Harvest credentials.\n\nTo get started, set up your Harvest API credentials:\n%s", configPath, SetupInstructionsURL)
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
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

// ConfigFilePath returns the location of the TOML config file.
func ConfigFilePath() (string, error) {
	return getConfigPath()
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "harvest-tui", "config.toml"), nil
}
