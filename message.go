package zanarkand

import (
	"encoding/binary"
	"fmt"
	"time"
)

var messageHeaderLength = 32
var messageReservedMagic uint16 = 0x0014

// Segment types separate messages into their relevant field maps.
// Session/Encryption types are not implemented due to them largely only
// containing the player ID, or information that should be kept hidden due
// to privacy concerns. While this may inconvenience debugging, this project
// will not facilitate capturing player login details.
const (
	SessionInit = 1
	SessionRecv = 2

	GameEvent = 3

	ServerPing = 7
	ServerPong = 8

	EncryptInit = 9
	EncryptRecv = 10
)

// MessageHeader provides the metadata for an FFXIV IPC.
// Bytes 14:15 are padding.
// Data pulled from Sapphire's `Network/CommonNetwork.h`
type MessageHeader interface {
	GetLength() uint32
	GetSource() uint32
	GetTarget() uint32
	GetSegment() uint16
	String() string
	ToMap() map[string]interface{}
}

// RawMessage is a generic FFXIV IPC container.
type RawMessage struct {
	Header MessageHeader
	Body   []byte
}

func buildMessageHeader(p []byte) MessageHeader {
	// Build the Message Headers
	segment := SegmentHeader{}
	segment.Length = binary.LittleEndian.Uint32(p[0:4])
	segment.SourceActor = binary.LittleEndian.Uint32(p[4:8])
	segment.TargetActor = binary.LittleEndian.Uint32(p[8:12])
	segment.SegmentType = binary.LittleEndian.Uint16(p[12:14])

	// Are we in a Game IPC?
	switch segment.SegmentType {
	case GameEvent:
		header := GameEventHeader{SegmentHeader: segment}
		header.buildMessageHeader(p)
		return header
	default:
		return segment
	}
}

// Length returns the length of the Message payload.
func (m *RawMessage) Length() uint32 {
	return m.Header.GetLength()
}

// Segment returns the Segment ID for the Message.
func (m *RawMessage) Segment() uint16 {
	return m.Header.GetSegment()
}

// Source returns the Message source actor ID.
func (m *RawMessage) Source() uint32 {
	return m.Header.GetSource()
}

// Target returns the Message target actor ID.
func (m *RawMessage) Target() uint32 {
	return m.Header.GetTarget()
}

/* Methods for Segment Headers */

// SegmentHeader is a generic header that all segments have
type SegmentHeader struct {
	Length      uint32 // [0:3] - always 0x18 (24)
	SourceActor uint32 // [4:7] - always 0
	TargetActor uint32 // [8:11] - always 0
	SegmentType uint16 // [12:13] - 7 or 8
	padding     uint16 // [14:15]
}

// GetLength returns the Segment length.
func (h SegmentHeader) GetLength() uint32 {
	return h.Length
}

// GetSegment returns the Segment type.
func (h SegmentHeader) GetSegment() uint16 {
	return h.SegmentType
}

// GetSource returns the source actor for the message.
func (h SegmentHeader) GetSource() uint32 {
	return h.SourceActor
}

// GetTarget returns the target actor for the message.
func (h SegmentHeader) GetTarget() uint32 {
	return h.TargetActor
}

// String presents a message header in a string format.
func (h SegmentHeader) String() string {
	return fmt.Sprintf("Segment - length: %d, source: %d, target: %d, segment: %d\n",
		h.Length, h.SourceActor, h.TargetActor, h.SegmentType)
}

// ToMap presents a message header as a hash.
func (h SegmentHeader) ToMap() map[string]interface{} {
	data := make(map[string]interface{})

	data["length"] = h.Length
	data["source"] = h.SourceActor
	data["target"] = h.TargetActor
	data["segment"] = h.SegmentType

	return data
}

/* Methods for IPC Messages */

// GameEventHeader is a sub-header for the data block of a GameEvent Message
// Bytes [20:21], [28:31] are padding
// Data pulled from Sapphire's `Network/CommonNetwork.h`
type GameEventHeader struct {
	SegmentHeader
	Reserved  uint16    // [16:17] - always 0x1400
	Opcode    uint16    // [18:19] - message context identifier, the "opcode"
	ServerID  uint16    // [22:23]
	Timestamp time.Time // [24:27]
}

func (h *GameEventHeader) buildMessageHeader(p []byte) {
	h.Reserved = binary.LittleEndian.Uint16(p[16:18])
	h.Opcode = binary.LittleEndian.Uint16(p[18:20])
	h.ServerID = binary.LittleEndian.Uint16(p[22:24])
	h.Timestamp = time.Unix(int64(binary.LittleEndian.Uint32(p[24:28])), 0)
}

// GetLength prints the length of the payload.
func (h GameEventHeader) GetLength() uint32 {
	return h.Length
}

// GetSegment returns the Segment type for the message.
func (h GameEventHeader) GetSegment() uint16 {
	return h.SegmentType
}

// GetSource returns the source actor for the message.
func (h GameEventHeader) GetSource() uint32 {
	return h.SourceActor
}

// GetTarget returns the target actor for the message.
func (h GameEventHeader) GetTarget() uint32 {
	return h.TargetActor
}

// String prints a Segment and IPC Message specific headers.
func (h GameEventHeader) String() string {
	segment := h.SegmentHeader.String()
	return fmt.Sprintf(segment+"Message - server: %v, opcode: 0x%X, timestamp: %v\n", h.ServerID, h.Opcode, h.Timestamp)
}

// ToMap returns a map of Segment and IPC Message specific headers.
func (h GameEventHeader) ToMap() map[string]interface{} {
	data := h.SegmentHeader.ToMap()

	data["opcode"] = h.Opcode
	data["server"] = h.ServerID
	data["timestamp"] = h.Timestamp

	return data
}
