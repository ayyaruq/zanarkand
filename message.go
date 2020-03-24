//lint:file-ignore U1000 Ignore unused struct members as they're part of the payload and users may want them
package zanarkand

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

var gameEventMessageHeaderLength = 32

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

// GenericMessage is an interface for other Message types to make the Framer generic.
type GenericMessage interface {
	Decode(*bufio.Reader) error
	IsMessage()
	MarshalJSON() ([]byte, error)
	String() string
}

// GenericHeader provides the metadata for an FFXIV IPC.
// 0:16 provide the generic header, other types have additional header fields.
// Data pulled from Sapphire's `Network/CommonNetwork.h`
type GenericHeader struct {
	Length      uint32 `json:"size"`          // [0:4]
	SourceActor uint32 `json:"sourceActorID"` // [4:8]
	TargetActor uint32 `json:"targetActorID"` // [8:12]
	Segment     uint16 `json:"segmentType"`   // [12:14]
	padding     uint16 // [14:16]
}

// Decode a GenericHeader from a byte array.
func (m *GenericHeader) Decode(r *bufio.Reader) error {
	data, err := r.Peek(16)
	lengthBytes := len(data)

	if err != nil {
		return ErrNotEnoughData{Expected: 16, Received: lengthBytes, Err: err}
	}

	m.Length = binary.LittleEndian.Uint32(data[0:4])
	m.SourceActor = binary.LittleEndian.Uint32(data[4:8])
	m.TargetActor = binary.LittleEndian.Uint32(data[8:12])
	m.Segment = binary.LittleEndian.Uint16(data[12:14])
	m.padding = binary.LittleEndian.Uint16(data[14:16])

	return nil
}

// String is a stringer for the GenericHeader of a Message.
func (m *GenericHeader) String() string {
	return fmt.Sprintf("Segment - size: %d, source: %d, target: %d, segment: %d\n",
		m.Length, m.SourceActor, m.TargetActor, m.Segment)
}

// GameEventMessage is a pre-type casted GameEventHeader and body.
type GameEventMessage struct {
	GenericHeader
	reserved  uint16    // [16:18] - always 0x1400
	Opcode    uint16    `json:"opcode"` // [18:20] - message context identifier, the "opcode"
	padding2  uint16    // [20:22]
	ServerID  uint16    `json:"serverID"` // [22:24]
	Timestamp time.Time `json:"-"`        // [24:28]
	padding3  uint32    // [28:32]
	Body      []byte    `json:"-"`
}

// IsMessage confirms a GameEventMessage is a Message.
func (GameEventMessage) IsMessage() {}

// Decode turns a byte payload into a real GameEventMessage.
func (m *GameEventMessage) Decode(r *bufio.Reader) error {
	header := GenericHeader{}
	err := header.Decode(r)
	if err != nil {
		return err
	}

	length := int(header.Length)
	data, err := r.Peek(length)
	lengthBytes := len(data)

	if err != nil {
		return ErrNotEnoughData{Expected: length, Received: lengthBytes, Err: err}
	}

	defer func() {
		// Regardless of what we read, the buffer is treated like we got everything
		_, _ = r.Discard(lengthBytes)
	}()

	m.GenericHeader = header
	m.reserved = binary.LittleEndian.Uint16(data[16:18])
	m.Opcode = binary.LittleEndian.Uint16(data[18:20])
	m.ServerID = binary.LittleEndian.Uint16(data[22:24])
	m.Timestamp = time.Unix(int64(binary.LittleEndian.Uint32(data[24:28])), 0)
	m.Body = data[gameEventMessageHeaderLength:length]

	return nil
}

// MarshalJSON provides an override for timestamp handling for encoding/JSON
func (m *GameEventMessage) MarshalJSON() ([]byte, error) {
	type Alias GameEventMessage
	data := make([]int, len(m.Body))
	for i, b := range m.Body {
		data[i] = int(b)
	}

	return json.Marshal(&struct {
		Data      []int `json:"data"`
		Timestamp int32 `json:"timestamp"`
		*Alias
	}{
		Data:      data,
		Timestamp: int32(m.Timestamp.Unix()),
		Alias:     (*Alias)(m),
	})
}

// String prints a Segment and IPC Message specific headers.
func (m GameEventMessage) String() string {
	return fmt.Sprintf(m.GenericHeader.String()+"Message - server: %v, opcode: 0x%X, timestamp: %v\n",
		m.ServerID, m.Opcode, m.Timestamp.Unix())
}

// KeepaliveMessage is a representation of ping/pong requests.
type KeepaliveMessage struct {
	GenericHeader
	ID        uint32    `json:"ID"` // [16:20]
	Timestamp time.Time `json:"-"`  // [20:24]
}

// IsMessage confirms a KeepaliveMessage is a Message.
func (KeepaliveMessage) IsMessage() {}

// Decode turns a byte payload into a real KeepaliveMessage.
func (m *KeepaliveMessage) Decode(r *bufio.Reader) error {
	header := GenericHeader{}
	err := header.Decode(r)
	if err != nil {
		return err
	}

	length := int(header.Length)
	data, err := r.Peek(length)
	lengthBytes := len(data)

	if err != nil {
		return ErrNotEnoughData{Expected: length, Received: lengthBytes, Err: err}
	}

	defer func() {
		// Regardless of what we read, the buffer is treated like we got everything
		_, _ = r.Discard(lengthBytes)
	}()

	m.GenericHeader = header
	m.ID = binary.LittleEndian.Uint32(data[16:20])
	m.Timestamp = time.Unix(int64(binary.LittleEndian.Uint32(data[20:24])), 0)

	return nil
}

// MarshalJSON provides an override for timestamp handling for encoding/JSON
func (m *KeepaliveMessage) MarshalJSON() ([]byte, error) {
	type Alias KeepaliveMessage
	return json.Marshal(&struct {
		Timestamp int32 `json:"timestamp"`
		*Alias
	}{
		Timestamp: int32(m.Timestamp.Unix()),
		Alias:     (*Alias)(m),
	})
}

// String prints the Segment header and Keepalive Message.
func (m *KeepaliveMessage) String() string {
	return fmt.Sprintf(m.GenericHeader.String()+"Message - ID: %d, timestamp: %v\n", m.ID, m.Timestamp.Unix())
}
