package cli

import (
	"fmt"
	"io"

	"github.com/planetargon/harvest-tui/internal/config"
	"github.com/spf13/cobra"
)

// authStatus is the JSON shape returned by `auth status --json`.
type authStatus struct {
	ActiveSource string            `json:"active_source"` // "keyring", "file", or "none"
	Keyring      authSourceDetails `json:"keyring"`
	File         authSourceDetails `json:"file"`
}

type authSourceDetails struct {
	Present   bool   `json:"present"`
	AccountID string `json:"account_id,omitempty"`
	Path      string `json:"path,omitempty"` // only set for the file source
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
	}

	path, _ := config.ConfigFilePath()
	status.File.Path = path
	if cfg, err := config.LoadFromFile(); err == nil {
		status.File.Present = true
		status.File.AccountID = cfg.Harvest.AccountID
		if status.ActiveSource == "none" {
			status.ActiveSource = "file"
		}
	} else {
		status.File.Error = err.Error()
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
	fmt.Fprintf(tw, "file\t%t\t%s\t%s\n",
		status.File.Present,
		valueOrDash(status.File.AccountID),
		fileDetail(status),
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

func fileDetail(status authStatus) string {
	if status.File.Present {
		marker := status.File.Path
		if status.ActiveSource == "file" {
			marker = "active: " + marker
		}
		return marker
	}
	return status.File.Path
}
