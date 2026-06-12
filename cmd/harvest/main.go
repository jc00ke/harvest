package main

import (
	"os"

	"github.com/jc00ke/harvest/internal/cli"
	"github.com/jc00ke/harvest/internal/tui"
)

func main() {
	if wantsUI(os.Args[1:]) {
		os.Exit(tui.Run())
	}
	os.Exit(cli.Execute())
}

// wantsUI reports whether the first argument requests the TUI. The flag is
// handled here rather than in cobra because cobra parses -ui as the combined
// shorthands -u -i.
func wantsUI(args []string) bool {
	return len(args) > 0 && (args[0] == "-ui" || args[0] == "--ui")
}
