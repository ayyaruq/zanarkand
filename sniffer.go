package zanarkand

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/gopacket/gopacket/tcpassembly"

	"github.com/ayyaruq/zanarkand/devices"
)

// SnifferState represents the current state of a Sniffer.
type SnifferState int

const (
	SnifferStopped SnifferState = iota
	SnifferRunning
	SnifferFinished
)

func (s SnifferState) String() string {
	switch s {
	case SnifferStopped:
		return "stopped"
	case SnifferRunning:
		return "running"
	case SnifferFinished:
		return "finished"
	default:
		return "unknown"
	}
}

// Sniffer is a representation of a packet source, filter, and destination.
type Sniffer struct {
	mu    sync.RWMutex
	state SnifferState

	ch     chan reassembledPacket
	ctx    context.Context
	cancel context.CancelFunc

	factory   tcpassembly.StreamFactory
	pool      *tcpassembly.StreamPool
	assembler *tcpassembly.Assembler

	Source *gopacket.PacketSource
}

// NewSniffer creates a Sniffer instance.
func NewSniffer(mode, src string) (*Sniffer, error) {
	ch := make(chan reassembledPacket, 200)
	streamFactory := &frameStreamFactory{ch: ch}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)
	assembler.AssemblerOptions.MaxBufferedPagesPerConnection = 32
	assembler.AssemblerOptions.MaxBufferedPagesTotal = 192 // 32 for each of the Client/Server pairs for Lobby, Chat, and Zone

	var err error
	var handle devices.DeviceHandle

	filter := "tcp portrange 54992-54994 or tcp portrange 55006-55007 or tcp portrange 55021-55040 or tcp portrange 55296-55551"

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

	return &Sniffer{
		factory:   streamFactory,
		pool:      streamPool,
		assembler: assembler,
		state:     SnifferStopped,
		ch:        ch,
		Source:    gopacket.NewPacketSource(handle, handle.LinkType()),
	}, nil
}

// IsActive reports whether the Sniffer is currently capturing.
func (s *Sniffer) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == SnifferRunning
}

// Status returns the current Sniffer state.
func (s *Sniffer) Status() SnifferState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Start an initialised Sniffer. It blocks until Stop is called or the context is cancelled.
// For file mode, it returns io.EOF when the file is exhausted.
func (s *Sniffer) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	defer s.cancel()

	s.mu.Lock()
	s.state = SnifferRunning
	s.mu.Unlock()

	packets := s.Source.Packets()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.mu.Lock()
			s.state = SnifferStopped
			s.mu.Unlock()
			s.assembler.FlushAll()
			return nil

		case packet := <-packets:
			// Nil Packet means end of a PCAP file
			if packet == nil {
				s.mu.Lock()
				s.state = SnifferFinished
				s.mu.Unlock()
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
}

// Stop a running Sniffer.
func (s *Sniffer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// NextFrame returns the next decoded Frame read by the Sniffer.
func (s *Sniffer) NextFrame() (*Frame, error) {
	select {
	case data := <-s.ch:
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

	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}
