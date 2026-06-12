package cli

import (
	"github.com/jc00ke/harvest/internal/harvest"
	"github.com/spf13/cobra"
)

func newProjectsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List Harvest projects and their tasks",
	}
	cmd.AddCommand(newProjectsListCommand())
	return cmd
}

func newProjectsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active projects with their available tasks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := authedClient()
			if err != nil {
				return err
			}
			projects, err := client.FetchProjects()
			if err != nil {
				return err
			}
			assignments, err := client.FetchTaskAssignments()
			if err != nil {
				return err
			}
			return renderProjects(out(cmd), harvest.AggregateProjectsWithTasks(projects, assignments))
		},
	}
}
