package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

var parser *flags.Parser

type scanCommand struct {
	HostsCommand hostsCommand `command:"hosts" description:"Scan all available neighbor hosts of current local network"`
	PortsCommand portsCommand `command:"ports" description:"Scan open ports on a host"`
}

func registerCommands(parser *flags.Parser) {
	_, _ = parser.AddCommand(
		"scan",
		"Network scanner",
		"Perform scanning over network",
		&scanCommand{},
	)
}

func init() {
	parser = flags.NewParser(nil, flags.HelpFlag|flags.PassDoubleDash)
	registerCommands(parser)
}

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
			return
		}

		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
		return
	}

	os.Exit(0)
}
