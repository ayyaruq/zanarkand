package zanarkand

import (
	"encoding/binary"
	"fmt"
	"time"
)

var messageHeaderLength = 32
var messageReservedMagic uint16 = 0x0014

// Segment types are as follows:
// Lobby client/server session handshake: 1+2
// Game IPC: 3
// Keepalive ping/pong: 7+8
// Lobby encryption init and handshake: 9+10
const (
	SessionInit = 1
	SessionRecv = 2

	GameEvent = 3

	ServerPing = 7
	ServerPong = 8

	EncryptInit = 9
	EncryptRecv = 10
)

// GenericHeader provides the metadata for an FFXIV IPC.
// Bytes 14:15 are padding.
// Data pulled from Sapphire's `Network/CommonNetwork.h`
type GenericHeader struct {
	Length      uint32 // [0:3]
	SourceActor uint32 // [4:7]
	TargetActor uint32 // [8:11]
	SegmentType uint16 // [12:13]
}

// MessageHeader is a sub-header for the data block of a GameEvent Message
// Bytes [20:21], [28:31] are padding
// Data pulled from Sapphire's `Network/CommonNetwork.h`
type MessageHeader struct {
	Header    GenericHeader // [0:15]
	Reserved  uint16        // [16:17] - always 0x1400
	Opcode    uint16        // [18:19] - message context identifier, the "opcode"
	ServerID  uint16        // [22:23]
	Timestamp time.Time     // [24:27]
}

// Message is a generic FFXIV IPC container.
type Message struct {
	Header MessageHeader
	Body   []byte
}

// ToMap presents a message header as a hash.
func (h *MessageHeader) ToMap() map[string]interface{} {
	data := make(map[string]interface{})

	data["timestamp"] = h.Timestamp
	data["length"] = h.Header.Length
	data["source"] = h.Header.SourceActor
	data["target"] = h.Header.TargetActor
	data["segment"] = h.Header.SegmentType
	data["opcode"] = h.Opcode
	data["server"] = h.ServerID

	return data
}

// ToString presents a message header in a string format.
func (h *MessageHeader) ToString() string {
	return fmt.Sprintf("  Message - timestamp: %v, length: %v, opcode: %v, source: %v, target: %v, segment: %v, server: %v",
		h.Timestamp, h.Header.Length, h.Opcode, h.Header.SourceActor, h.Header.TargetActor, h.Header.SegmentType, h.ServerID)
}

func buildMessageHeader(p []byte) MessageHeader {
	// Build the Message Headers
	generic := GenericHeader{}

	generic.Length = binary.LittleEndian.Uint32(p[0:4])
	generic.SourceActor = binary.LittleEndian.Uint32(p[4:8])
	generic.TargetActor = binary.LittleEndian.Uint32(p[8:12])
	generic.SegmentType = binary.LittleEndian.Uint16(p[12:14])

	// Are we in a Game IPC?
	if generic.SegmentType == GameEvent {
		header := MessageHeader{}

		header.Header = generic
		header.Reserved = binary.LittleEndian.Uint16(p[16:18])
		header.Opcode = binary.LittleEndian.Uint16(p[18:20])
		header.ServerID = binary.LittleEndian.Uint16(p[22:24])
		header.Timestamp = time.Unix(int64(binary.LittleEndian.Uint32(p[24:28])), 0)

		return header
	}

	return MessageHeader{Header: generic}
}
