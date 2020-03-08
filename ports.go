package main

import (
	"fmt"
)

type portsCommand struct {
}

// Execute will run the command
func (c *portsCommand) Execute(_ []string) error {
	fmt.Printf("Scanning Ports..\n")
	return nil
}
