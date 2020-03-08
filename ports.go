package main

import (
	"fmt"
)

type portsCommand struct {
}

func (c *portsCommand) Execute(_ []string) error {
	fmt.Printf("Scanning Ports..\n")
	return nil
}
