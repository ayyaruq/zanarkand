package zanarkand

import (
	"bufio"
	"encoding/binary"
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
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
			// #nosec G104
			fmt.Errorf("error syncing Frame start position: %w", err)
			return
		}

		// Grab the synced header bytes so we can make sure we have enough data
		header, err := reader.Peek(frameHeaderLength)
		if err != nil {
			// #nosec G104
			fmt.Errorf("can't peek into header bytes from buffer: %w", err)
			return
		}

		// Make a buffer for the full Frame size
		length := binary.LittleEndian.Uint32(header[24:28])
		data := make([]byte, int(length))

		count, err := reader.Read(data)
		if err != nil {
			// #nosec G104
			fmt.Errorf("can't read %d bytes from buffer: %w", length, err)
			return
		}

		if count != int(length) {
			// #nosec G104
			fmt.Errorf("read less data than expected: %d < %d", count, length)
			return
		}

		reassembledChan <- reassembledPacket{Body: data, Flow: f.net}
	}
}
