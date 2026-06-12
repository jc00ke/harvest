// Package cli implements the harvest command-line interface backed by the
// same internal/harvest API client that powers the TUI.
package cli

import (
	"fmt"
	"io"

	"github.com/jc00ke/harvest/internal/config"
	"github.com/jc00ke/harvest/internal/harvest"
	"github.com/spf13/cobra"
)

// outputJSON is set by the --json persistent flag on the root command.
var outputJSON bool

// NewRootCommand builds the harvest command tree. Exported so tests can
// exercise it without going through main.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "harvest",
		Short:         "Command-line client for the Harvest API",
		Long:          "harvest is a CLI for managing Harvest time entries, projects, and tasks.\nRun `harvest -ui` to launch the interactive TUI instead.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output results as JSON instead of a table")

	root.AddCommand(
		newAuthCommand(),
		newMeCommand(),
		newEntriesCommand(),
		newProjectsCommand(),
	)

	return root
}

// Execute runs the root command and returns its exit code.
func Execute() int {
	if err := NewRootCommand().Execute(); err != nil {
		// Cobra already prints the error via SilenceErrors=false.
		return 1
	}
	return 0
}

// authedClient loads config (env vars first, then file), constructs a Harvest
// client, and validates the credentials. It is invoked from each command's
// RunE so that --help works without requiring credentials.
func authedClient() (*harvest.Client, *harvest.User, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	client := harvest.NewClient(cfg.Harvest.AccountID, cfg.Harvest.AccessToken)
	user, err := client.ValidateAuth()
	if err != nil {
		return nil, nil, fmt.Errorf("authentication failed: %w", err)
	}
	return client, user, nil
}

// out returns the writer commands should write user-facing output to.
func out(cmd *cobra.Command) io.Writer {
	return cmd.OutOrStdout()
}
