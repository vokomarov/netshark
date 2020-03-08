package host

import (
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

	if scanner.started == true {
		t.Errorf("Scanner has wrong started flag")
	}

	if scanner.stopped == true {
		t.Errorf("Scanner has wrong stopped flag")
	}

	if scanner.Hosts == nil {
		t.Errorf("Host storage is not initialised")
	}

	if scanner.Done == nil {
		t.Errorf("Done channel is not created")
	}

	if scanner.stop == nil {
		t.Errorf("Stop channel is not created")
	}
}

func TestScanner_StopEmpty(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	scanner.Stop()

	ok := true
	select {
	case _, ok = <-scanner.stop:
	default:
	}

	if ok {
		t.Errorf("stop channel must be closed once Stop method called")
	}
}

func TestScanner_StopStartedStopped(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	go scanner.Scan()
	scanner.Stop()

	ok := true
	select {
	case _, ok = <-scanner.stop:
	default:
	}

	if ok {
		t.Errorf("stop channel must be closed once Stop method called")
	}

	ok = true
	select {
	case _, ok = <-scanner.Done:
	default:
	}
	if ok {
		t.Errorf("done channel must be closed once Stop method finished")
	}

	scanner.Stop()

	ok = true
	select {
	case _, ok = <-scanner.stop:
	default:
	}
	if ok {
		t.Errorf("stop channel must be closed once Stop method called")
	}

	ok = true
	select {
	case _, ok = <-scanner.Done:
	default:
	}
	if ok {
		t.Errorf("done channel must be closed once Stop method finished")
	}
}

func TestScanner_StopWorking(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	// simulate fake scanner
	go func(s *Scanner) {
		s.started = true

		select {
		case <-s.stop:
			s.finish(nil)
		}
	}(scanner)

	time.Sleep(1 * time.Second)

	scanner.Stop()

	ok := true
	select {
	case _, ok = <-scanner.stop:
	default:
	}

	if ok {
		t.Errorf("stop channel must be closed once Stop method called")
	}

	ok = true
	select {
	case _, ok = <-scanner.Done:
	default:
	}
	if ok {
		t.Errorf("done channel must be closed once Stop method finished")
	}
}

func TestScanner_AddHost(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	host := Host{
		IP:  "127.0.0.1",
		MAC: "ff:ff:ff:ff:ff:ff",
	}

	scanner.AddHost(&host)

	if len(scanner.Hosts) != 1 {
		t.Errorf("Host is not added")
	}

	if host.ID() != scanner.Hosts[0].ID() {
		t.Errorf("Host is added but changed")
	}

	if host.IP != scanner.Hosts[0].IP {
		t.Errorf("Host is added but changed IP")
	}

	if host.MAC != scanner.Hosts[0].MAC {
		t.Errorf("Host is added but changed MAC")
	}

	if len(scanner.unique) != 1 {
		t.Errorf("Host is not registered to unique registry")
	}

	if _, ok := scanner.unique[host.ID()]; !ok {
		t.Errorf("Host is not registered to unique registry")
	}
}

func TestScanner_HasHost(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Errorf("Scanner instance is empty")
		return
	}

	host := Host{
		IP:  "127.0.0.1",
		MAC: "ff:ff:ff:ff:ff:ff",
	}

	if scanner.HasHost(&host) {
		t.Errorf("Host is wrongly detected as already added")
	}

	scanner.AddHost(&host)

	if !scanner.HasHost(&host) {
		t.Errorf("Host is not registered as already addded")
	}
}
