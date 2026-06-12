package main

import (
	"os"

	"github.com/jc00ke/harvest/internal/cli"
	"github.com/jc00ke/harvest/internal/tui"
)

func main() {
	if args := os.Args[1:]; wantsUI(args) {
		os.Exit(tui.Run(wantsDemo(args)))
	}
	os.Exit(cli.Execute())
}

// wantsUI reports whether the first argument requests the TUI. The flag is
// handled here rather than in cobra because cobra parses -ui as the combined
// shorthands -u -i.
func wantsUI(args []string) bool {
	return len(args) > 0 && (args[0] == "-ui" || args[0] == "--ui")
}

// wantsDemo reports whether the TUI should run in demo mode, backed by an
// in-process fake Harvest API instead of real credentials.
func wantsDemo(args []string) bool {
	if !wantsUI(args) {
		return false
	}
	for _, arg := range args[1:] {
		if arg == "-demo" || arg == "--demo" {
			return true
		}
	}
	return false
}
