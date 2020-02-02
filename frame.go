package zanarkand

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

var frameHeaderLength = 40
var frameMagicLE uint64 = 0xE2465DFF41A05252

// Frame is an FFXIV bundled message encapsulation layer.
// Currently, bytes 4:7, 8:15, 32, and 34:39 are unknown.
type Frame struct {
	Magic      uint64    `json:"-"`              // [0:8] - mainly to verify magic bytes
	Timestamp  time.Time `json:"-"`              // [16:24] - timestamp in milliseconds since epoch
	Length     uint32    `json:"length"`         // [24:28]
	Connection uint16    `json:"connectionType"` // [28:30] - 0 lobby, 1 zone, 2 chat
	Count      uint16    `json:"count"`          // [30:32]
	reserved1  byte      // [32]
	Compressed bool      `json:"compressed"` // [33] UINT8 bool tho
	reserved2  uint32    // [34:38]
	reserved3  uint16    // [38:40]
	Body       []byte    `json:"-"`
}

// Decode a frame from byte data
func (f *Frame) Decode(p []byte) {
	// Keep the magic alive
	f.Magic = binary.LittleEndian.Uint64(p[0:8])

	// Time in Go is a bit weird, this basically turns it into an int64
	msec := time.Duration(binary.LittleEndian.Uint64(p[16:24])) * time.Millisecond
	f.Timestamp = time.Unix(0, 0).Add(msec)

	// Remaining fields
	f.Length = binary.LittleEndian.Uint32(p[24:28])
	f.Connection = binary.LittleEndian.Uint16(p[28:30])
	f.Compressed = p[33] != 0
	f.Count = binary.LittleEndian.Uint16(p[30:32])

	f.Body = p[frameHeaderLength:f.Length]
}

// String provides a string representation of a frame header.
func (f *Frame) String() string {
	return fmt.Sprintf("Frame - magic: 0x%X, timestamp: %v, length: %v, count: %v, compressed: %t, connection: %v",
		f.Magic, f.Timestamp.Unix(), f.Length, f.Count, f.Compressed, f.Connection)
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

// MarshalJSON provides an override for timestamp handling for encoding/JSON
func (f *Frame) MarshalJSON() ([]byte, error) {
	type Alias Frame
	data := make([]int, len(f.Body))
	for i, b := range f.Body {
		data[i] = int(b)
	}

	return json.Marshal(&struct {
		Data      []int `json:"data"`
		Timestamp int32 `json:"timestamp"`
		*Alias
	}{
		Data:      data,
		Timestamp: int32(f.Timestamp.Unix()),
		Alias:     (*Alias)(f),
	})
}

func validateMagic(header []byte) bool {
	magic := binary.LittleEndian.Uint64(header)

	if magic != frameMagicLE {
		return false
	}

	return true
}
