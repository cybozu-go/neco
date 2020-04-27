package cke

import (
	"context"
	"fmt"
)

// Operator is the interface for operations
type Operator interface {
	// Name returns the operation name.
	Name() string
	// NextCommand returns the next command or nil if completed.
	NextCommand() Commander
	// Targets returns the ip which will be affected by the operation
	Targets() []string
}

// Commander is a single step to proceed an operation
type Commander interface {
	// Run executes the command
	Run(ctx context.Context, inf Infrastructure, leaderKey string) error
	// Command returns the command information
	Command() Command
}

// Command represents some command
type Command struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

// String implements fmt.Stringer
func (c Command) String() string {
	return fmt.Sprintf("%s %s", c.Name, c.Target)
}
