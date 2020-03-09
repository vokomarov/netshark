package host

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewScanner(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	if scanner.unique == nil {
		t.Errorf("Unique host registry is not initialised")
	}

	if scanner.ctx == nil {
		t.Errorf("Context is not initialised")
	}

	if scanner.cancelFunc == nil {
		t.Errorf("Cancel func is not initialised")
	}

	if scanner.found == nil {
		t.Errorf("Found channel is not initialised")
	}

	if scanner.Error != nil {
		t.Errorf("Error is not empty")
	}
}

func TestScanner_Ctx(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	ctx, cancelFunc := scanner.Ctx(context.Background())
	if ctx == nil {
		t.Errorf("Wrapped context is empty")
		return
	}

	if cancelFunc == nil {
		t.Errorf("Cancel func is empty")
		return
	}

	if scanner.ctx != ctx {
		t.Errorf("Context is not equal")
	}

	if scanner.ctx.Err() != nil {
		t.Errorf("Context is already closed")
	}

	cancelFunc()

	if scanner.ctx.Err() == nil {
		t.Errorf("Context is not closed")
	}
}

func TestScanner_Hosts(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	channel := scanner.Hosts()
	if channel == nil {
		t.Errorf("Result hosts channel is empty")
	}

	host := Host{
		IP:  "127.0.0.1",
		MAC: "ff:ff:ff:ff:ff:ff",
	}

	go func() {
		for host := range channel {
			if host == nil {
				t.Errorf("Received nil host")
				return
			}

			if host.IP != "127.0.0.1" {
				t.Errorf("Received host IP is not same as input")
				return
			}

			if host.MAC != "ff:ff:ff:ff:ff:ff" {
				t.Errorf("Received host IP is not same as input")
				return
			}
		}
	}()

	scanner.foundHost(&host)
	close(scanner.found)
}

func TestScanner_Scan(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	_, stopFunc := scanner.Ctx(context.Background())

	go scanner.Scan()

	go func() {
		<-time.NewTicker(2 * time.Second).C
		stopFunc()
	}()

	hosts := make([]*Host, 0)

	for h := range scanner.Hosts() {
		hosts = append(hosts, h)
	}

	if len(scanner.unique) != len(hosts) {
		t.Errorf("Some of found hosts is not registered")
	}

	if scanner.foundHost(&Host{}) {
		t.Errorf("Registering host after stop scanning must be failed")
	}
}

func TestScanner_ScanFail(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	err := fmt.Errorf("test error")

	go scanner.Scan()

	go func() {
		<-time.NewTicker(2 * time.Second).C
		scanner.fail(err)
	}()

	for range scanner.Hosts() {
	}

	if scanner.Error != nil {
		if strings.Contains(scanner.Error.Error(), "permission") {
			t.Skipf("run tests with sudo to allow scan interface")
			return
		}

		if scanner.Error != err {
			t.Errorf("Error is not propagated")
		}
	}

}
