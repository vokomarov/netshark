package main

import (
	"fmt"
)

type PortsCommand struct {
}

func (c *PortsCommand) Execute(_ []string) error {
	fmt.Printf("Scanning Ports..\n")
	return nil
}
