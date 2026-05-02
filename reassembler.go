package zanarkand

import (
	"bufio"
	"encoding/binary"
	"fmt"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/tcpassembly"
	"github.com/gopacket/gopacket/tcpassembly/tcpreader"
)

// reassembledPacket is a frame payload with TCP metadata
type reassembledPacket struct {
	Body []byte
	Flow gopacket.Flow
}

// frameStreamFactory implements tcpassembly.StreamFactory
type frameStreamFactory struct {
	dataCh chan<- reassembledPacket
	errCh  chan<- error
}

// frameStream handles decoding TCP packets
type frameStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
	dataCh         chan<- reassembledPacket
	errCh          chan<- error
}

// New implements StreamFactory.New(), acting as a Factory for each new Flow.
func (f *frameStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	fs := &frameStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
		dataCh:    f.dataCh,
		errCh:     f.errCh,
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
			f.reportError(fmt.Errorf("error syncing Frame start position: %w", err))
			return
		}

		// Grab the synced header bytes so we can make sure we have enough data
		header, err := reader.Peek(frameHeaderLength)
		if err != nil {
			f.reportError(fmt.Errorf("can't peek into header bytes from buffer: %w", err))
			return
		}

		// Make a buffer for the full Frame size
		length := binary.LittleEndian.Uint32(header[24:28])
		data := make([]byte, int(length))

		count, err := reader.Read(data)
		if err != nil {
			f.reportError(fmt.Errorf("can't read %d bytes from buffer: %w", length, err))
			return
		}

		if count != int(length) {
			f.reportError(fmt.Errorf("read less data than expected: %d < %d", count, length))
			return
		}

		f.dataCh <- reassembledPacket{Body: data, Flow: f.net}
	}
}

func (f *frameStream) reportError(err error) {
	if f.errCh != nil {
		select {
		case f.errCh <- ErrReassemblyError{Err: err}:
		default:
		}
	}
}
