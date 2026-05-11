package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/planetargon/harvest-tui/internal/config"
	"github.com/planetargon/harvest-tui/internal/harvest"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthLoginCommand() *cobra.Command {
	var accountIDFlag, accessTokenFlag string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Harvest and store credentials in the OS keyring",
		Long: `Prompts for a Harvest Account ID and Personal Access Token, validates them
against the Harvest API, and stores them securely in the OS keyring (Keychain
on macOS, Secret Service on Linux, Credential Manager on Windows).

Credentials may also be supplied non-interactively via --account-id and
--access-token; note that --access-token via flag will appear in shell history.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			accountID, accessToken, err := collectCredentials(cmd, accountIDFlag, accessTokenFlag)
			if err != nil {
				return err
			}

			client := harvest.NewClient(accountID, accessToken)
			user, err := client.ValidateAuth()
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			if err := config.StoreCredentialsInKeyring(config.HarvestConfig{
				AccountID:   accountID,
				AccessToken: accessToken,
			}); err != nil {
				return fmt.Errorf("could not store credentials: %w", err)
			}

			return renderMessage(out(cmd), fmt.Sprintf("Logged in as %s %s (%s)", user.FirstName, user.LastName, user.Email))
		},
	}
	cmd.Flags().StringVar(&accountIDFlag, "account-id", "", "Harvest Account ID (skip the interactive prompt)")
	cmd.Flags().StringVar(&accessTokenFlag, "access-token", "", "Harvest Personal Access Token (visible in shell history)")
	return cmd
}

// collectCredentials returns the account ID and access token, sourced from
// flags first, then interactive prompts for whatever is missing.
func collectCredentials(cmd *cobra.Command, accountIDFlag, accessTokenFlag string) (string, string, error) {
	accountID := strings.TrimSpace(accountIDFlag)
	accessToken := accessTokenFlag

	if accountID == "" {
		v, err := promptLine(cmd, "Account ID: ")
		if err != nil {
			return "", "", err
		}
		accountID = strings.TrimSpace(v)
	}
	if accountID == "" {
		return "", "", fmt.Errorf("account ID is required")
	}

	if accessToken == "" {
		v, err := promptSecret(cmd, "Access Token: ")
		if err != nil {
			return "", "", err
		}
		accessToken = strings.TrimSpace(v)
	}
	if accessToken == "" {
		return "", "", fmt.Errorf("access token is required")
	}

	return accountID, accessToken, nil
}

// promptLine writes prompt to the command's stdout and reads a line from stdin.
func promptLine(cmd *cobra.Command, prompt string) (string, error) {
	if _, err := fmt.Fprint(out(cmd), prompt); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}

// promptSecret writes prompt to stdout and reads a line from stdin with input
// hidden if stdin is a terminal. Falls back to visible input otherwise.
func promptSecret(cmd *cobra.Command, prompt string) (string, error) {
	if _, err := fmt.Fprint(out(cmd), prompt); err != nil {
		return "", err
	}
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(out(cmd)) // ReadPassword swallows the trailing newline
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	// Non-TTY (piped input, etc.) — read a line visibly.
	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}
