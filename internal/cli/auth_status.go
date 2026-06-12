package cli

import (
	"fmt"
	"io"

	"github.com/planetargon/harvest-tui/internal/config"
	"github.com/spf13/cobra"
)

// authStatus is the JSON shape returned by `auth status --json`.
type authStatus struct {
	ActiveSource string            `json:"active_source"` // "keyring" or "none"
	Keyring      authSourceDetails `json:"keyring"`
}

type authSourceDetails struct {
	Present   bool   `json:"present"`
	AccountID string `json:"account_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show where Harvest credentials are loaded from",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			status := buildAuthStatus()
			return renderAuthStatus(out(cmd), status)
		},
	}
}

func buildAuthStatus() authStatus {
	status := authStatus{ActiveSource: "none"}

	if cfg, err := config.LoadFromKeyring(); err == nil {
		status.Keyring = authSourceDetails{Present: true, AccountID: cfg.Harvest.AccountID}
		status.ActiveSource = "keyring"
	} else {
		status.Keyring.Error = err.Error()
	}

	return status
}

func renderAuthStatus(w io.Writer, status authStatus) error {
	if outputJSON {
		return renderJSON(w, status)
	}
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "SOURCE\tPRESENT\tACCOUNT_ID\tDETAIL")
	fmt.Fprintf(tw, "keyring\t%t\t%s\t%s\n",
		status.Keyring.Present,
		valueOrDash(status.Keyring.AccountID),
		activeMarker(status.ActiveSource == "keyring"),
	)
	if err := tw.Flush(); err != nil {
		return err
	}
	if status.ActiveSource == "none" {
		_, err := fmt.Fprintln(w, "\nNo credentials found. Run `harvest-cli auth login` to store credentials in your keyring.")
		return err
	}
	return nil
}

func valueOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func activeMarker(active bool) string {
	if active {
		return "active"
	}
	return ""
}
