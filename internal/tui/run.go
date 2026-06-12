package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jc00ke/harvest/internal/config"
	"github.com/jc00ke/harvest/internal/harvest"
	"github.com/jc00ke/harvest/internal/state"
)

// Run loads configuration and state, validates Harvest credentials, and runs
// the TUI program. It returns the process exit code.
func Run() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	appState, err := state.Load()
	if err != nil {
		fmt.Printf("Error loading state: %v\n", err)
		return 1
	}

	harvestClient := harvest.NewClient(cfg.Harvest.AccountID, cfg.Harvest.AccessToken)

	// Validate authentication before starting the TUI
	user, err := harvestClient.ValidateAuth()
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		fmt.Println("Please check your Harvest credentials. Run `harvest auth login` to update them.")
		fmt.Printf("\nTo get started, set up your Harvest API credentials:\n%s\n", config.SetupInstructionsURL)
		return 1
	}

	fmt.Printf("Welcome, %s!\n", user.FirstName+" "+user.LastName)
	fmt.Printf("Starting Harvest TUI...\n")

	model := NewModel(cfg, harvestClient, appState, user)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	// Save state on exit
	if err := appState.Save(); err != nil {
		fmt.Printf("Warning: Could not save state: %v\n", err)
	}
	return 0
}
