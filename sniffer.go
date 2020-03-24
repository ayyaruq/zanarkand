package zanarkand

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"

	"github.com/ayyaruq/zanarkand/devices"
)

// reassembledPacket is a frame payload with TCP metadata
type reassembledPacket struct {
	Body []byte
	Flow gopacket.Flow
}

// reassembledChan is a byte channel to receive the length of a full frame
var reassembledChan = make(chan reassembledPacket, 200)

// frameStreamFactory implements tcpassembly.StreamFactory
type frameStreamFactory struct{}

// frameStream handles decoding TCP packets
type frameStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
}

// New implements StreamFactory.New(), acting as a Factory for each new Flow.
func (f *frameStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	fs := &frameStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
	}

	// Start the Stream or prepare to clench
	go fs.run()

	// ReaderStream implements tcpassembly.Stream so return a pointer to it
	return &fs.r
}

// Run the stream, quickly
func (f *frameStream) run() {
	var reader = bufio.NewReaderSize(&f.r, 128*1024)

	for {
		// Skip to start of a frame
		err := discardUntilValid(reader)
		if err != nil {
			fmt.Errorf("error syncing Frame start position: %w", err)
			return
		}

		// Grab the synced header bytes so we can make sure we have enough data
		header, err := reader.Peek(frameHeaderLength)
		if err != nil {
			fmt.Errorf("can't peek into header bytes from buffer: %w", err)
			return
		}

		// Make a buffer for the full Frame size
		length := binary.LittleEndian.Uint32(header[24:28])
		data := make([]byte, int(length))
		count, err := reader.Read(data)
		if err != nil {
			fmt.Errorf("can't read %d bytes from buffer: %w", length, err)
			return
		}

		if count != int(length) {
			fmt.Errorf("read less data than expected: %d < %d", count, length)
			return
		}

		reassembledChan <- reassembledPacket{Body: data, Flow: f.net}
	}

}

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
func NewSniffer(mode string, src string) (*Sniffer, error) {
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
	case "afpacket":
		handle, err = devices.OpenAFPacket(src, filter, 25, pcap.BlockForever)

	case "file":
		handle, err = devices.OpenFile(src, filter)

	case "pcap":
		handle, err = devices.OpenPcap(src, filter, pcap.BlockForever)
	default:
		err = ErrDecodingFailure{Err: fmt.Errorf("unknown input type: %s", mode)}
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
	s.Active = <- s.notifier
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

	frame.Decode(data.Body)

	if int(frame.Length) != len(data.Body) {
		return nil, ErrNotEnoughData{Expected: len(data.Body), Received: int(frame.Length)}
	}

	// Add our flow data
	frame.meta.Flow = data.Flow

	return frame, nil
}
