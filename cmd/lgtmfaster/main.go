package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/storage"
	"github.com/johanforsgren/lgtmfaster/internal/ui"
)

func main() {
	debugLog := flag.String("debug", "", "Enable debug logging to specified file")
	flag.Parse()

	if *debugLog != "" {
		if err := logger.Init(*debugLog); err != nil {
			fmt.Printf("Error initializing logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Close()
		logger.Log("Application starting with debug logging enabled")
	} else {
		logger.EnsureInit()
		logger.Log("Application starting (in-memory logging only)")
	}

	// Enable global HTTP request/response logging
	// This wraps http.DefaultTransport to log all HTTP traffic
	// http.DefaultTransport = common.NewLoggingTransport(http.DefaultTransport)
	// logger.Log("HTTP request logging enabled globally")

	repo, err := storage.NewLocalRepository()
	if err != nil {
		logger.LogError("INIT", "storage", err)
		fmt.Printf("Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	logger.Log("Storage initialized successfully")

	model := ui.NewModel(repo)
	p := tea.NewProgram(model, tea.WithAltScreen())

	logger.Log("Starting TUI application")
	if _, err := p.Run(); err != nil {
		logger.LogError("RUN", "application", err)
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	logger.Log("Application exiting normally")
}
