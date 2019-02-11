package zanarkand

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
)

// reassembledChan is a byte channel to receive the length of a full frame
var reassembledChan = make(chan []byte)

// tcpStreamFactory implements tcpassembly.StreamFactory.
type frameStreamFactory struct{}

// tcpStream handles decoding TCP packets
type frameStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
}

// New implements StreamFactory.New(), acting as a Factory for each new Flow.
func (f *frameStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	fStream := &frameStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
	}

	// Start the Stream or prepare to clench.
	go fStream.run()

	// ReaderStream implements tcpassembly.Stream so return a pointer to it.
	return &fStream.r
}

// Run the stream, quickly.
func (f *frameStream) run() {
	var reader = bufio.NewReaderSize(&f.r, 128*1024)

	for {
		// Skip to start of a frame
		err := discardUntilValid(reader)
		if err != nil {
			fmt.Errorf("Error syncing Frame start position: %s", err)
			return
		}

		// Grab the synced header bytes so we can make sure we have enough data
		header, err := reader.Peek(40)
		if err != nil {
			fmt.Errorf("Can't peek into header bytes from buffer: %s", err)
			return
		}

		// Make a buffer for the full Frame size
		length := binary.LittleEndian.Uint32(header[24:28])
		data := make([]byte, int(length))
		count, err := reader.Read(data)
		if err != nil {
			fmt.Errorf("Can't read %d bytes from buffer: %s", length, err)
			return
		}

		if count != int(length) {
			fmt.Errorf("Read less data than expected: %d < %d", count, length)
			return
		}

		reassembledChan <- data
	}

}

// Sniffer is a representation of a packet source, filter, and destination.
type Sniffer struct {
	// Sniffer State
	active        chan bool
	stateNotifier chan bool
	state         bool

	// Packet Assembler
	factory   tcpassembly.StreamFactory
	pool      *tcpassembly.StreamPool
	assembler *tcpassembly.Assembler

	filter string
	Source *gopacket.PacketSource
}

// NewSniffer creates a Sniffer instance.
func NewSniffer(fileName string, ifDevice string) (*Sniffer, error) {
	// Setup Packet Assembler
	streamFactory := new(frameStreamFactory)
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)
	assembler.AssemblerOptions.MaxBufferedPagesPerConnection = 16
	assembler.AssemblerOptions.MaxBufferedPagesTotal = 16

	// Setup state tracker
	stateController := make(chan bool, 1)
	stateNotifier := make(chan bool, 1)

	// Setup handle for device or file
	var handle *pcap.Handle
	var err error

	if fileName != "" {
		handle, err = pcap.OpenOffline(fileName)
	} else if ifDevice != "" {
		handle, err = pcap.OpenLive(ifDevice, 1600, true, pcap.BlockForever)
	} else {
		return nil, errors.New("capture handle: no device or file provided")
	}

	if err != nil {
		return nil, fmt.Errorf("Unabe to open capture handle: %s", err)
	}

	err = handle.SetBPFFilter("tcp portrange 54992-54994 or tcp portrange 55006-55007 or tcp portrange 55021-55040 or tcp portrange 55296-55551")
	if err != nil {
		return nil, fmt.Errorf("Unable to setup BPF filter: %s", err)
	}

	s := &Sniffer{
		factory:   streamFactory,
		pool:      streamPool,
		assembler: assembler,

		// Setup PacketSource
		active:        stateController,
		stateNotifier: stateNotifier,
		state:         false,
		Source:        gopacket.NewPacketSource(handle, handle.LinkType()),
	}

	return s, nil
}

// Start an initialised Sniffer.
func (s *Sniffer) Start() {
	s.state = true

	var packet gopacket.Packet
	var err error

	for {
		select {
		default:
			packet, err = s.Source.NextPacket()

			// Nil Packet means end of a PCAP file
			if packet == nil {
				return
			}

			if err != nil {
				fmt.Errorf("Error decoding packet: %s", err)
				continue
			}

			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				fmt.Println("Unusable Packet, something is not right")
				continue
			}

			tcp := packet.TransportLayer().(*layers.TCP)
			s.assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

		case <-s.active:
			s.stateNotifier <- false
			return
		}

	}
}

// Stop a running Sniffer.
func (s *Sniffer) Stop() int {
	// Stop reading more packets
	s.active <- false

	// Flush the assembler buffer
	closed := s.assembler.FlushAll()

	// Set state condition for Active()
	s.state = <-s.stateNotifier

	return closed
}

// Active returns the state of a Sniffer.
func (s *Sniffer) Active() bool {
	return s.state
}

// NextFrame returns the next decoded Frame read by the Sniffer.
func (s *Sniffer) NextFrame() (*Frame, error) {
	data := <-reassembledChan

	// Setup our Frame
	header := buildFrameHeader(data)

	if int(header.Length) != len(data) {
		return nil, fmt.Errorf("Data length %d does not match Frame header length %d", len(data), header.Length)
	}

	// Collect the payload
	payload, err := header.buildFrameData(data)
	if err != nil {
		return nil, err
	}

	// Create our base Frame
	frame := new(Frame)
	frame.Header = header

	// Set the data offset
	var offset uint32

	for i := 0; i < int(header.Count); i++ {
		segment := buildMessageHeader(payload[offset:])

		// Do we have a full Message?
		length := segment.Header.Length
		if int(length) > len(payload) {
			return frame, fmt.Errorf("Message is %d bytes larger than available in Frame", int(length)-len(payload))
		}

		// Sanity check
		if segment.Reserved != messageReservedMagic {
			return frame, fmt.Errorf("Message magic mismatch! %X is not: %X", segment.Reserved, messageReservedMagic)
		}

		// Init our message
		message := Message{}
		message.Header = segment
		message.Body = payload[offset:length]

		// Bump offset for next Message
		offset += length

		frame.Messages = append(frame.Messages, message)
	}

	return frame, nil
}
