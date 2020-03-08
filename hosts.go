package main

import (
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

	go scanner.Scan()

	timeout := time.NewTicker(time.Duration(c.Timeout) * time.Second)

	func() {
		for {
			fmt.Printf("found %d hosts...\r", len(scanner.Hosts))

			time.Sleep(1 * time.Second)

			select {
			case <-timeout.C:
				scanner.Stop()
				return
			case <-quit:
				scanner.Stop()
				return
			case <-scanner.Done:

				return
			default:
			}
		}
	}()

	// Clear line
	fmt.Printf("%c[2K\r", 27)

	if scanner.Error != nil {
		fmt.Printf("\n\r")
		return scanner.Error
	}

	fmt.Printf("\nFound %d hosts:   \n", len(scanner.Hosts))

	for _, h := range scanner.Hosts {
		fmt.Printf("  [IP: %s] \t [MAC: %s] \n", h.IP, h.MAC)
	}

	return nil
}
