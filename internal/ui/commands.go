package ui

import (
	"strings"
)

type CommandType int

const (
	CommandUnknown CommandType = iota
	CommandQuit
	CommandPATs
	CommandPR
	CommandHelp
)

type Command struct {
	Type CommandType
	Args []string
}

func ParseCommand(input string) Command {
	input = strings.TrimSpace(input)

	if !strings.HasPrefix(input, ":") {
		return Command{Type: CommandUnknown}
	}

	input = strings.TrimPrefix(input, ":")
	parts := strings.Fields(input)

	if len(parts) == 0 {
		return Command{Type: CommandUnknown}
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "q", "quit":
		return Command{Type: CommandQuit, Args: args}
	case "p", "pats":
		return Command{Type: CommandPATs, Args: args}
	case "pr", "prs":
		return Command{Type: CommandPR, Args: args}
	case "h", "help":
		return Command{Type: CommandHelp, Args: args}
	default:
		return Command{Type: CommandUnknown, Args: args}
	}
}
