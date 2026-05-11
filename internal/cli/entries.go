package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// dateFormat is the YYYY-MM-DD format Harvest expects for spent_date.
const dateFormat = "2006-01-02"

func newEntriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entries",
		Short: "Manage Harvest time entries",
	}
	cmd.AddCommand(
		newEntriesListCommand(),
		newEntriesCreateCommand(),
		newEntriesEditCommand(),
		newEntriesDeleteCommand(),
		newEntriesStartCommand(),
		newEntriesStopCommand(),
	)
	return cmd
}

// parseEntryID parses an entry ID positional argument.
func parseEntryID(arg string) (int, error) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid entry ID %q: must be an integer", arg)
	}
	return id, nil
}

// parseDate validates a YYYY-MM-DD string. An empty input returns "" (caller decides default).
func parseDate(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if _, err := time.Parse(dateFormat, s); err != nil {
		return "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD", s)
	}
	return s, nil
}

// todayDate returns today's date in YYYY-MM-DD format.
func todayDate() string {
	return time.Now().Format(dateFormat)
}
