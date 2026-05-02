package zanarkand

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"runtime/trace"
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

	dataCh chan reassembledPacket
	errCh  chan error
	ctx    context.Context
	cancel context.CancelFunc

	factory   tcpassembly.StreamFactory
	pool      *tcpassembly.StreamPool
	assembler *tcpassembly.Assembler

	Source *gopacket.PacketSource
}

// Option configures a Sniffer.
type Option func(*snifferConfig)

type snifferConfig struct {
	dataBufSize int
	errBufSize  int
}

// Default buffer sizes
const (
	defaultDataBufSize = 200
	defaultErrBufSize  = 1
)

// WithDataBufferSize sets the buffer size for the frame data channel.
// This controls how many reassembled frames can be queued before the
// reassembler goroutines block. The default is 200.
func WithDataBufferSize(n int) Option {
	return func(c *snifferConfig) { c.dataBufSize = n }
}

// WithErrorBufferSize sets the buffer size for the error channel.
// Errors are dropped if the buffer is full. The default is 1.
func WithErrorBufferSize(n int) Option {
	return func(c *snifferConfig) { c.errBufSize = n }
}

// NewSniffer creates a Sniffer instance.
func NewSniffer(mode, src string) (*Sniffer, error) {
	dataCh := make(chan reassembledPacket, 200)
	errCh := make(chan error, 1)
	streamFactory := &frameStreamFactory{dataCh: dataCh, errCh: errCh}
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
		dataCh:    dataCh,
		errCh:     errCh,
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

// Errors returns a channel that receives reassembler errors.
// These errors indicate problems during TCP stream reassembly,
// such as lost frames or malformed data. The channel is buffered
// and will drop errors if not consumed.
func (s *Sniffer) Errors() <-chan error {
	return s.errCh
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

// StartTrace begins runtime/trace profiling, writing to w.
// Call StopTrace to finish. Useful for diagnosing performance issues
// such as channel buffer exhaustion or slow frame decoding.
func StartTrace(w io.Writer) error {
	return trace.Start(w)
}

// StopTrace stops runtime/trace profiling.
func StopTrace() {
	trace.Stop()
}

// NextFrame returns the next decoded Frame read by the Sniffer.
func (s *Sniffer) NextFrame() (*Frame, error) {
	select {
	case data := <-s.dataCh:
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

// FrameHandler is called by ProcessFrames for each message in a frame.
type FrameHandler func(frame *Frame, header *GenericHeader, r *bufio.Reader) error

// ProcessFrames iterates over frames and calls fn for each message in each frame.
// It handles decompression and reader setup. It blocks until the Sniffer is stopped,
// or an error occurs.
func (s *Sniffer) ProcessFrames(fn FrameHandler) error {
	var zpool sync.Pool

	for {
		frame, err := s.NextFrame()
		if err != nil {
			return fmt.Errorf("error retrieving next frame: %w", err)
		}

		var r *bufio.Reader
		var z io.ReadCloser

		// Setup our Message reader
		if frame.Compression == FrameCompressionZlib {
			z = zpool.Get().(io.ReadCloser)
			if z != nil {
				err = z.(zlib.Resetter).Reset(bytes.NewReader(frame.Body), nil)
				if err != nil {
					return fmt.Errorf("error resetting ZLIB decoder: %w", err)
				}
			} else {
				z, err = zlib.NewReader(bytes.NewReader(frame.Body))
				if err != nil {
					return fmt.Errorf("error creating ZLIB decoder: %w", err)
				}
			}
			r = bufio.NewReader(z)
		} else {
			r = bufio.NewReader(bytes.NewReader(frame.Body))
		}

		for i := 0; i < int(frame.Count); i++ {
			header := new(GenericHeader)

			err := header.Decode(r)
			if err != nil {
				if z != nil {
					z.Close()
				}
				return ErrDecodingFailure{Err: err}
			}

			if err := fn(frame, header, r); err != nil {
				if z != nil {
					z.Close()
				}
				return err
			}
		}

		// Return the zlib reader to the pool for reuse
		if z != nil {
			zpool.Put(z)
		}

		// We're done with the current frame, if Sniffer is stopped then exit,
		// allowing user to start a new subscriber routine.
		if !s.IsActive() {
			return nil
		}
	}
}
