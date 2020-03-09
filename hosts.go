package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/vokomarov/netshark/scanner/host"
)

type hostsCommand struct {
	Timeout int `short:"t" long:"timeout" default:"5" description:"Timeout in seconds to wait for ARP responses."`
}

// Execute will run the command
func (c *hostsCommand) Execute(_ []string) error {
	fmt.Printf("Scanning Hosts..\n")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	scanner := host.NewScanner()
	_, stopFunc := scanner.Ctx(context.Background())

	go scanner.Scan()

	go func() {
		timeout := time.NewTicker(time.Duration(c.Timeout) * time.Second)

		select {
		case <-timeout.C:
			stopFunc()
			return
		case <-quit:
			stopFunc()
			return
		}
	}()

	for h := range scanner.Hosts() {
		fmt.Printf("  [IP: %s] \t [MAC: %s] \n", h.IP, h.MAC)
	}

	return scanner.Error
}
