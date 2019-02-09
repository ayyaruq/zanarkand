package zanarkand

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"time"
)

var frameHeaderLength = 40

var frameMagicLE uint64 = 0xE2465DFF41A05252

// FrameHeader is the metadata around an FFXIV frame.
// Currently, bytes 4:7, 8:15, 32, and 34:39 are unknown.
type FrameHeader struct {
	Magic      uint64    // [0:8] - mainly to verify magic bytes
	Timestamp  time.Time // [16:23] - timestamp in milliseconds since epoch
	Length     uint32    // [24:27]
	Connection uint16    // [28:29] - 0 lobby, 1 zone, 2 chat
	Count      uint16    // [30:31]
	Compressed bool      // [33] UINT8 bool tho
}

// Frame is an FFXIV bundled message encapsulation layer.
type Frame struct {
	Header   FrameHeader
	Messages []Message
}

// ToMap provides a hash representation of a frame header.
func (h *FrameHeader) ToMap() map[string]interface{} {
	data := make(map[string]interface{})

	data["count"] = h.Count
	data["compressed"] = h.Compressed
	data["connection"] = h.Connection
	data["length"] = h.Length
	data["magic"] = h.Magic
	data["timestamp"] = h.Timestamp

	return data
}

// ToString provides a string representation of a frame header.
func (h *FrameHeader) ToString() string {
	return fmt.Sprintf("Frame - magic: 0x%X, timestamp: %v, length: %v, count: %v, compressed: %t, connection: %v",
		h.Magic, h.Timestamp, h.Length, h.Count, h.Compressed, h.Connection)
}

func (h *FrameHeader) buildFrameData(p []byte) ([]byte, error) {
	if h.Compressed {
		// ZLIB a dumb and needs to read from a fixed size buffer or it just dies in the butt
		buf := bytes.NewReader(p[frameHeaderLength:h.Length])
		z, err := zlib.NewReader(buf)
		if err != nil {
			return nil, fmt.Errorf("Error creating ZLIB decoder: %s", err)
		}

		defer z.Close()

		body, err := ioutil.ReadAll(z)
		if err != nil {
			return nil, fmt.Errorf("Error decoding ZLIB data: %s", err)
		}

		return body, nil
	}

	// Compression decoder returns so no need for else, just send back the raw body and EOF
	return p[frameHeaderLength:], nil
}

func buildFrameHeader(p []byte) FrameHeader {
	// Build the Frame Header
	header := FrameHeader{}

	// Keep the magic
	header.Magic = binary.LittleEndian.Uint64(p[0:8])

	// Time in Go is a bit weird, this basically turns it into an int64
	msec := time.Duration(binary.LittleEndian.Uint64(p[16:24])) * time.Millisecond
	header.Timestamp = time.Unix(0, 0).Add(msec)

	// Remaining fields
	header.Length = binary.LittleEndian.Uint32(p[24:28])
	header.Connection = binary.LittleEndian.Uint16(p[28:30])
	header.Compressed = p[33] != 0
	header.Count = binary.LittleEndian.Uint16(p[30:32])

	return header
}

func discardUntilValid(r *bufio.Reader) error {
	for {
		header, err := r.Peek(8)
		if err != nil {
			return err
		}

		if validateMagic(header) {
			return nil
		}

		_, _ = r.Discard(1)
	}
}

func validateMagic(header []byte) bool {
	magic := binary.LittleEndian.Uint64(header)

	if magic != frameMagicLE {
		return false
	}

	return true
}
