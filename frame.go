//lint:file-ignore U1000 Ignore unused struct members as they're part of the payload and users may want them
package zanarkand

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/gopacket/gopacket"
)

const frameHeaderLength = 40
const frameMagicLE uint64 = 0xE2465DFF41A05252

var (
	privateBlock10  = mustParseCIDR("10.0.0.0/8")
	privateBlock172 = mustParseCIDR("172.16.0.0/12")
	privateBlock192 = mustParseCIDR("192.168.0.0/16")
)

func mustParseCIDR(s string) *net.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return ipnet
}

const (
	FrameCompressionNone  = 0
	FrameCompressionZlib  = 1
	FrameCompressionOodle = 2
)

type Compressor uint8

// FrameIngress is an inbound Frame.
// FrameEgress is an outbound Frame.
const (
	FrameIngress FlowDirection = 1
	FrameEgress  FlowDirection = 2
)

// FlowDirection indicates the flow being inbound or outbound.
type FlowDirection int

// Frame is an FFXIV bundled message encapsulation layer.
// Currently, bytes 4:7, 8:15, 32, and 34:39 are unknown.
type Frame struct {
	Magic       uint64     `json:"-"`              // [0:8] - mainly to verify magic bytes
	Timestamp   time.Time  `json:"-"`              // [16:24] - timestamp in milliseconds since epoch
	Length      uint32     `json:"size"`           // [24:28]
	Connection  uint16     `json:"connectionType"` // [28:30] - 0 lobby, 1 zone, 2 chat
	Count       uint16     `json:"count"`          // [30:32]
	reserved1   byte       // [32]
	Compression Compressor `json:"compression"` // [33] UINT8 - 0 none, 1 zlib, 2 untrained oodle, 3 trained oodle?
	reserved2   uint32     // [34:38]
	reserved3   uint16     // [38:40]
	Body        []byte     `json:"-"`

	meta FrameMeta
}

func (c Compressor) String() string {
	switch c {
	case FrameCompressionNone:
		return "None"
	case FrameCompressionZlib:
		return "ZLib"
	case FrameCompressionOodle:
		return "Oodle"
	default:
		return "Unknown"
	}
}

// FrameMeta represents metadata from the IP and TCP layers on the Frame.
type FrameMeta struct {
	Flow gopacket.Flow
}

// Decode a frame from byte data
func (f *Frame) Decode(p []byte) error {
	if len(p) < frameHeaderLength {
		return ErrNotEnoughData{Expected: frameHeaderLength, Received: len(p)}
	}

	// Keep the magic alive
	f.Magic = binary.LittleEndian.Uint64(p[0:8])

	// Time in Go is a bit weird, this basically turns it into an int64
	msec := time.Duration(binary.LittleEndian.Uint64(p[16:24])) * time.Millisecond
	f.Timestamp = time.Unix(0, 0).Add(msec)

	// Remaining fields
	f.Length = binary.LittleEndian.Uint32(p[24:28])
	f.Connection = binary.LittleEndian.Uint16(p[28:30])
	f.Compression = Compressor(p[33])
	f.Count = binary.LittleEndian.Uint16(p[30:32])

	f.Body = p[frameHeaderLength:f.Length]

	return nil
}

// Direction outputs if the Frame is inbound or outbound.
func (f *Frame) Direction() FlowDirection {
	src, dst := f.meta.Flow.Endpoints()
	srcIP := net.ParseIP(src.String())
	dstIP := net.ParseIP(dst.String())

	// Check for inbound first since that's the majority
	if isPrivate(dstIP) && !isPrivate(srcIP) {
		return FrameIngress
	}

	// Next up, outbound
	if isPrivate(srcIP) && !isPrivate(dstIP) {
		return FrameEgress
	}

	// If we get here, wtf is up with the src and dst
	return 0
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

// Meta returns the frame metadata, a gopacket.Flow
// this allows the user to determine if a Frame is inbound or outbound.
func (f *Frame) Meta() *FrameMeta {
	return &f.meta
}

// String provides a string representation of a frame header.
func (f *Frame) String() string {
	return fmt.Sprintf("Frame - magic: 0x%X, timestamp: %v, size: %v, count: %v, compression: %s, connection: %v",
		f.Magic, f.Timestamp.Unix(), f.Length, f.Count, f.Compression.String(), f.Connection)
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

func isPrivate(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	return privateBlock10.Contains(ip) || privateBlock172.Contains(ip) || privateBlock192.Contains(ip)
}

func validateMagic(header []byte) bool {
	magic := binary.LittleEndian.Uint64(header)

	return magic == frameMagicLE
}
