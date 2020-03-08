package host

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type Scanner struct {
	mu     sync.RWMutex
	unique map[string]bool
	Hosts  []*Host
	stop   chan struct{}
	Done   chan struct{}
	Error  error
}

func NewScanner() *Scanner {
	return &Scanner{
		mu:     sync.RWMutex{},
		stop:   make(chan struct{}),
		unique: make(map[string]bool),
		Hosts:  make([]*Host, 0),
		Done:   make(chan struct{}),
	}
}

func (s *Scanner) Stop() {
	s.stop <- struct{}{}
	close(s.stop)
	<-s.Done
}

func (s *Scanner) finish(err error) {
	if err != nil {
		s.Error = err
	}

	s.Done <- struct{}{}
	close(s.Done)
}

func (s *Scanner) HasHost(host *Host) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.unique[host.ID()]; ok {
		return true
	}

	return false
}

// Push new detected Host to the list of all detected
func (s *Scanner) AddHost(host *Host) *Scanner {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Hosts = append(s.Hosts, host)
	s.unique[host.ID()] = true

	return s
}

// Detect system interfaces and go over each one to detect IP addresses
// and read/write ARP packets
// Blocked until every interfaces unable to write packets or stop call
func (s *Scanner) Scan() {
	interfaces, err := net.Interfaces()
	if err != nil {
		s.finish(err)
		return
	}

	var wg sync.WaitGroup

	for i := range interfaces {
		wg.Add(1)

		go func(iface net.Interface) {
			defer wg.Done()

			if err := s.scanInterface(&iface); err != nil {
				s.finish(fmt.Errorf("interface [%v] error: %w", iface.Name, err))
				return
			}
		}(interfaces[i])
	}

	// Wait for all interfaces' scans to complete.  They'll try to run
	// forever, but will stop on an error, so if we get past this Wait
	// it means all attempts to write have failed.
	wg.Wait()
}

// Scans an individual interface's local network for machines using ARP requests/replies.
//
// Loops forever, sending packets out regularly.  It returns an error if
// it's ever unable to write a packet.
func (s *Scanner) scanInterface(iface *net.Interface) error {
	// We just look for IPv4 addresses, so try to find if the interface has one.
	var addr *net.IPNet

	if addresses, err := iface.Addrs(); err != nil {
		return err
	} else {
		for _, a := range addresses {
			if IPNet, ok := a.(*net.IPNet); ok {
				if IPv4 := IPNet.IP.To4(); IPv4 != nil {
					addr = &net.IPNet{
						IP:   IPv4,
						Mask: IPNet.Mask[len(IPNet.Mask)-4:],
					}

					break
				}
			}
		}
	}

	// Sanity-check that the interface has a good address.
	if addr == nil {
		return nil
	} else if addr.IP[0] == 127 {
		return nil
	} else if addr.Mask[0] != 0xff || addr.Mask[1] != 0xff {
		return nil
	}

	// Open up a pcap handle for packet reads/writes.
	handle, err := pcap.OpenLive(iface.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()

	// Start up a goroutine to read in packet data.
	go s.listenARP(handle, iface)

	for {
		// Write our scan packets out to the handle.
		if err := writeARP(handle, iface, addr); err != nil {
			return fmt.Errorf("error writing packets: %w", err)
		}

		// We don't know exactly how long it'll take for packets to be
		// sent back to us, but 10 seconds should be more than enough
		// time ;)
		time.Sleep(10 * time.Second)
	}
}

// Watches a handle for incoming ARP responses we might care about.
// Push new Host once any correct response received
// Work until 'stop' is closed.
func (s *Scanner) listenARP(handle *pcap.Handle, iface *net.Interface) {
	src := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
	in := src.Packets()

	for {
		var packet gopacket.Packet

		select {
		case <-s.stop:
			s.finish(nil)
			return
		case packet = <-in:
			arpLayer := packet.Layer(layers.LayerTypeARP)

			if arpLayer == nil {
				continue
			}

			arp := arpLayer.(*layers.ARP)

			if arp.Operation != layers.ARPReply || bytes.Equal([]byte(iface.HardwareAddr), arp.SourceHwAddress) {
				// This is a packet I sent.
				continue
			}

			// Note:  we might get some packets here that aren't responses to ones we've sent,
			// if for example someone else sends US an ARP request.  Doesn't much matter, though...
			// all information is good information :)
			host := Host{
				IP:  fmt.Sprintf("%v", net.IP(arp.SourceProtAddress)),
				MAC: fmt.Sprintf("%v", net.HardwareAddr(arp.SourceHwAddress)),
			}

			if !s.HasHost(&host) {
				s.AddHost(&host)
			}
		}
	}
}

// writeARP writes an ARP request for each address on our local network to the
// pcap handle.
func writeARP(handle *pcap.Handle, iface *net.Interface, addr *net.IPNet) error {
	// Set up all the layers' fields we can.

	eth := layers.Ethernet{
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(iface.HardwareAddr),
		SourceProtAddress: []byte(addr.IP),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
	}

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()

	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	// Send one packet for every address.
	for _, ip := range ips(addr) {
		arp.DstProtAddress = ip

		if err := gopacket.SerializeLayers(buf, opts, &eth, &arp); err != nil {
			return err
		}

		if err := handle.WritePacketData(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

// ips is a simple and not very good method for getting all IPv4 addresses from a
// net.IPNet.  It returns all IPs it can over the channel it sends back, closing
// the channel when done.
func ips(n *net.IPNet) (out []net.IP) {
	num := binary.BigEndian.Uint32([]byte(n.IP))
	mask := binary.BigEndian.Uint32([]byte(n.Mask))
	num &= mask

	for mask < 0xffffffff {
		var buf [4]byte

		binary.BigEndian.PutUint32(buf[:], num)
		out = append(out, net.IP(buf[:]))
		mask++
		num++
	}

	return
}
