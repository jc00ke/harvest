package main

import (
	"os"

	"github.com/planetargon/harvest-tui/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
