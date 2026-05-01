package zanarkand

import (
	"fmt"
	"io"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/gopacket/gopacket/tcpassembly"

	"github.com/ayyaruq/zanarkand/devices"
)

// Sniffer is a representation of a packet source, filter, and destination.
type Sniffer struct {
	// Sniffer State
	Active   bool
	Status   string
	notifier chan bool

	// Packet Assembler
	factory   tcpassembly.StreamFactory
	pool      *tcpassembly.StreamPool
	assembler *tcpassembly.Assembler

	Source *gopacket.PacketSource
}

// NewSniffer creates a Sniffer instance.
func NewSniffer(mode, src string) (*Sniffer, error) {
	// Setup Packet Assembler
	streamFactory := new(frameStreamFactory)
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)
	assembler.AssemblerOptions.MaxBufferedPagesPerConnection = 32
	assembler.AssemblerOptions.MaxBufferedPagesTotal = 192 // 32 for each of the Client/Server pairs for Lobby, Chat, and Zone

	// Setup state tracker
	stateNotifier := make(chan bool, 1)

	// Setup handle and filter
	var err error
	var handle devices.DeviceHandle

	var filter = "tcp portrange 54992-54994 or tcp portrange 55006-55007 or tcp portrange 55021-55040 or tcp portrange 55296-55551"

	if src == "" {
		return nil, fmt.Errorf("capture handle: no source provided")
	}

	switch mode {
	case "file":
		handle, err = devices.OpenFile(src, filter)

	case "pcap":
		handle, err = devices.OpenPcap(src, filter, pcap.BlockForever)

	case "pfring":
		handle, err = devices.OpenPFRing(src, filter, 1600, pcap.BlockForever)

	case "afpacket":
		handle, err = devices.OpenAFPacket(src, filter, 25, pcap.BlockForever)

	default:
		err = ErrUnknownInput{Err: fmt.Errorf("unknown input type: %s", mode)}
	}

	if err != nil {
		return nil, fmt.Errorf("capture handle: %w", err)
	}

	s := &Sniffer{
		factory:   streamFactory,
		pool:      streamPool,
		assembler: assembler,

		Active:   false,
		Status:   "stopped",
		notifier: stateNotifier,

		Source: gopacket.NewPacketSource(handle, handle.LinkType()),
	}

	return s, nil
}

// Start an initialised Sniffer.
func (s *Sniffer) Start() error {
	s.notifier <- true
	s.Active = <-s.notifier
	s.Status = "started"

	packets := s.Source.Packets()
	ticker := time.NewTicker(3 * time.Second)

	defer ticker.Stop()

	for s.Active {
		select {
		case state := <-s.notifier:
			// Set state condition and loop control, if state is false, we're stopped
			s.Active = state
			if !state {
				s.Status = "stopped"
				s.assembler.FlushAll()

				return nil
			}

			s.Status = "running"

		case packet := <-packets:
			// Nil Packet means end of a PCAP file
			if packet == nil {
				s.Status = "finished"
				return io.EOF
			}

			// Kinda weird, just skip this packet
			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				continue
			}

			tcp := packet.TransportLayer().(*layers.TCP)
			s.assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

		case t := <-ticker.C:
			s.assembler.FlushWithOptions(tcpassembly.FlushOptions{CloseAll: false, T: t.Add(-3 * time.Second)})
		}
	}

	return nil
}

// Stop a running Sniffer.
func (s *Sniffer) Stop() {
	s.notifier <- false
}

// NextFrame returns the next decoded Frame read by the Sniffer.
func (s *Sniffer) NextFrame() (*Frame, error) {
	data := <-reassembledChan

	// Setup our Frame
	frame := new(Frame)

	if err := frame.Decode(data.Body); err != nil {
		return nil, err
	}

	if int(frame.Length) != len(data.Body) {
		return nil, ErrNotEnoughData{Expected: len(data.Body), Received: int(frame.Length)}
	}

	// Add our flow data
	frame.meta.Flow = data.Flow

	return frame, nil
}
