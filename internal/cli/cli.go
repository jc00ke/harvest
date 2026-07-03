// Package cli implements the harvest command-line interface backed by the
// same internal/harvest API client that powers the TUI.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

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
		newInvoiceCommand(),
	)

	return root
}

// Execute runs the root command and returns its exit code. Ctrl-C or
// SIGTERM cancels the command's context so in-flight API requests abort.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := NewRootCommand().ExecuteContext(ctx); err != nil {
		// Cobra already prints the error via SilenceErrors=false.
		return 1
	}
	return 0
}

// newAPIClient constructs a Harvest API client. It is a variable so tests
// can point commands at a stub server.
var newAPIClient = harvest.NewClient

// authedClient loads credentials from the OS keyring, constructs a Harvest
// client, and validates the credentials. It is invoked from each command's
// RunE so that --help works without requiring credentials.
func authedClient(ctx context.Context) (*harvest.Client, *harvest.User, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	client := newAPIClient(cfg.Harvest.AccountID, cfg.Harvest.AccessToken)
	user, err := client.ValidateAuth(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("authentication failed: %w", err)
	}
	return client, user, nil
}

// out returns the writer commands should write user-facing output to.
func out(cmd *cobra.Command) io.Writer {
	return cmd.OutOrStdout()
}
