package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jc00ke/harvest/internal/harvest"
	"github.com/spf13/cobra"
)

func newInvoiceCommand() *cobra.Command {
	var month string
	var project string
	var clientName string
	var billableOnly bool
	cmd := &cobra.Command{
		Use:   "invoice",
		Short: "Summarize a month's hours per person for a project or client",
		Long: "Summarize a month's hours per person for a project or client.\n\n" +
			"--project and --client accept either an ID or a name. Seeing other\n" +
			"people's hours requires a Harvest admin or manager account; for\n" +
			"regular members the Harvest API only returns their own entries.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if month == "" {
				month = time.Now().Format("2006-01")
			}
			from, to, err := parseMonth(month)
			if err != nil {
				return err
			}
			api, _, err := authedClient(cmd.Context())
			if err != nil {
				return err
			}
			filter, err := resolveInvoiceFilter(cmd.Context(), api, project, clientName)
			if err != nil {
				return err
			}
			entries, err := api.FetchTeamTimeEntries(cmd.Context(), from, to, filter)
			if err != nil {
				return err
			}
			if billableOnly {
				billable := entries[:0]
				for _, e := range entries {
					if e.IsBillable {
						billable = append(billable, e)
					}
				}
				entries = billable
			}
			return renderInvoice(out(cmd), summarizeInvoice(entries))
		},
	}
	cmd.Flags().StringVar(&month, "month", "", "Month in YYYY-MM format (default: current month)")
	cmd.Flags().StringVar(&project, "project", "", "Project ID or name to invoice")
	cmd.Flags().StringVar(&clientName, "client", "", "Client ID or name to invoice")
	cmd.Flags().BoolVar(&billableOnly, "billable-only", false, "Include only billable hours")
	cmd.MarkFlagsOneRequired("project", "client")
	cmd.MarkFlagsMutuallyExclusive("project", "client")
	return cmd
}

// resolveInvoiceFilter turns the --project or --client flag value into a
// time-entry filter. Numeric values are used as IDs directly; anything else
// is matched case-insensitively against project or client names.
func resolveInvoiceFilter(ctx context.Context, api *harvest.Client, project, client string) (harvest.TeamTimeEntriesFilter, error) {
	arg := project
	if client != "" {
		arg = client
	}
	if id, err := strconv.Atoi(arg); err == nil {
		if client != "" {
			return harvest.TeamTimeEntriesFilter{ClientID: id}, nil
		}
		return harvest.TeamTimeEntriesFilter{ProjectID: id}, nil
	}

	projects, err := api.FetchProjects(ctx)
	if err != nil {
		return harvest.TeamTimeEntriesFilter{}, err
	}
	if client != "" {
		id := 0
		for _, p := range projects {
			if !strings.EqualFold(p.Client.Name, client) {
				continue
			}
			if id != 0 && id != p.Client.ID {
				return harvest.TeamTimeEntriesFilter{}, fmt.Errorf("client name %q is ambiguous; use the client ID", client)
			}
			id = p.Client.ID
		}
		if id == 0 {
			return harvest.TeamTimeEntriesFilter{}, fmt.Errorf("no client matching %q found", client)
		}
		return harvest.TeamTimeEntriesFilter{ClientID: id}, nil
	}
	id := 0
	for _, p := range projects {
		if !strings.EqualFold(p.Name, project) {
			continue
		}
		if id != 0 {
			return harvest.TeamTimeEntriesFilter{}, fmt.Errorf("project name %q is ambiguous; use the project ID", project)
		}
		id = p.ID
	}
	if id == 0 {
		return harvest.TeamTimeEntriesFilter{}, fmt.Errorf("no project matching %q found", project)
	}
	return harvest.TeamTimeEntriesFilter{ProjectID: id}, nil
}
