// TamaLLM - A Tamagotchi-like terminal game with an LLM brain.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/danielmerja/TamaLLM/internal/storage"
	"github.com/danielmerja/TamaLLM/internal/tui"
)

func main() {
	// Parse command line flags
	startNew := flag.Bool("new", false, "Start a new game (ignore existing save)")
	modelName := flag.String("model", "", "Override Ollama model name")
	hostURL := flag.String("host", "", "Override Ollama host URL")
	tickMs := flag.Int("tick-ms", 1000, "Simulation tick interval in milliseconds")
	noLLM := flag.Bool("no-llm", false, "Run without LLM (use canned messages)")
	autoMode := flag.Bool("auto", false, "Start with LLM auto mode enabled")
	flag.Parse()

	// Get configuration from environment/flags
	config := tui.DefaultConfig()

	// Apply environment variables
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		config.LLMConfig.Host = host
	}
	if model := os.Getenv("OLLAMA_MODEL"); model != "" {
		config.LLMConfig.Model = model
	}
	if timeout := os.Getenv("OLLAMA_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.LLMConfig.Timeout = d
		}
	}
	if think := os.Getenv("OLLAMA_THINK"); think != "" {
		config.LLMConfig.Think = think
	}

	// Apply CLI flags (override env)
	if *hostURL != "" {
		config.LLMConfig.Host = *hostURL
	}
	if *modelName != "" {
		config.LLMConfig.Model = *modelName
	}
	if *tickMs > 0 {
		config.TickInterval = time.Duration(*tickMs) * time.Millisecond
	}
	if *noLLM {
		config.LLMEnabled = false
	}
	if *autoMode && !*noLLM {
		config.LLMAutoMode = true
	}
	config.StartNew = *startNew

	// Initialize storage
	store, err := storage.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	// Delete existing save if starting new
	if *startNew {
		_ = store.Delete()
	}

	// Create and run the TUI
	model := tui.New(config, store)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TamaLLM: %v\n", err)
		os.Exit(1)
	}
}
