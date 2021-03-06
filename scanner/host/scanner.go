package host

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// Scanner provide container for control local network scanning
// process and checking results
type Scanner struct {
	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
	found      chan *Host
	unique     map[string]bool
	Error      error
}

// NewScanner will initialise new instance of Scanner
func NewScanner() *Scanner {
	ctx, cancelFunc := context.WithCancel(context.Background())

	return &Scanner{
		mu:         sync.RWMutex{},
		ctx:        ctx,
		cancelFunc: cancelFunc,
		found:      make(chan *Host),
		unique:     make(map[string]bool),
	}
}

// Ctx wrap given context and return new with cancel func
func (s *Scanner) Ctx(ctx context.Context) (context.Context, context.CancelFunc) {
	s.ctx, s.cancelFunc = context.WithCancel(ctx)

	return s.ctx, s.cancelFunc
}

// Hosts will return a read only channel to receive found Host
func (s *Scanner) Hosts() <-chan *Host {
	return s.found
}

func (s *Scanner) fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Error = err

	if s.ctx.Err() == nil {
		s.cancelFunc()
	}
}

func (s *Scanner) foundHost(host *Host) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ctx.Err() != nil {
		return false
	}

	if _, ok := s.unique[host.ID()]; !ok {
		s.unique[host.ID()] = true
		s.found <- host
	}

	return true
}

// Scan will detect system interfaces and go over each one to detect
// IP addresses to read/write ARP packets
// Blocked until every interfaces unable to write packets or stop call
// so typically should be run as a goroutine
func (s *Scanner) Scan() {
	interfaces, err := net.Interfaces()
	if err != nil {
		s.fail(err)
		return
	}

	var wg sync.WaitGroup

	for i := range interfaces {
		wg.Add(1)

		go func(iface net.Interface) {
			defer wg.Done()

			if err := s.scanInterface(&iface); err != nil {
				s.fail(fmt.Errorf("interface [%v] error: %w", iface.Name, err))
				return
			}
		}(interfaces[i])
	}

	wg.Wait()

	close(s.found)
}

// Scans an individual interface's local network for machines using ARP requests/replies.
//
// Loops forever, sending packets out regularly.  It returns an error if
// it's ever unable to write a packet.
func (s *Scanner) scanInterface(iface *net.Interface) error {
	// We just look for IPv4 addresses, so try to find if the interface has one.
	var addr *net.IPNet

	addresses, err := iface.Addrs()
	if err != nil {
		return err
	}

	for _, a := range addresses {
		if IPNet, ok := a.(*net.IPNet); ok {
			IPv4 := IPNet.IP.To4()

			if IPv4 == nil {
				continue
			}

			addr = &net.IPNet{
				IP:   IPv4,
				Mask: IPNet.Mask[len(IPNet.Mask)-4:],
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

		timeout := time.NewTicker(10 * time.Second)

		select {
		case <-timeout.C:
			continue
		case <-s.ctx.Done():
			return nil
		}
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
		case <-s.ctx.Done():
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
			if !s.foundHost(&Host{
				IP:  fmt.Sprintf("%v", net.IP(arp.SourceProtAddress)),
				MAC: fmt.Sprintf("%v", net.HardwareAddr(arp.SourceHwAddress)),
			}) {
				return
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
// the channel when fail.
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
