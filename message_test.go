package zanarkand

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
	"testing"
	"time"
)

var compressedGameEventBlob = []byte{
	0x78, 0x9C, 0x33, 0x60, 0x60, 0x60, 0x78, 0x64,
	0x18, 0x21, 0x60, 0x71, 0x27, 0x43, 0x80, 0x19,
	0xC8, 0x16, 0x61, 0x50, 0x65, 0x64, 0x60, 0x60,
	0x66, 0x28, 0xC8, 0x31, 0x8B, 0x03, 0x72, 0x19,
	0xE2, 0x7C, 0x18, 0x19, 0x04, 0xD4, 0x19, 0x18,
	0x6E, 0x31, 0xD5, 0xDD, 0xFD, 0x5F, 0xCF, 0xC0,
	0x00, 0x00, 0xCD, 0xED, 0x09, 0x7F,
}

var lalafellLengthGameEventBlob = []byte{
	0x30, 0x00, 0x00, 0x00, 0xE2, 0x31, 0x58, 0x10,
	0x38, 0xDC, 0x68, 0x10, 0x03, 0x00, 0x00, 0x00,
}

var decompressedKeepaliveBlob = []byte{
	0x18, 0x00, 0x00, 0x00, // Length
	0x01, 0x02, 0x03, 0x04, // Source Actor ID
	0x05, 0x06, 0x07, 0x08, // Target Actor ID
	0x08, 0x00, 0x00, 0x00, // Segment and padding
	0x15, 0xCD, 0x5B, 0x07, // Data
	0x42, 0xe0, 0x89, 0x58,
}

var lalafellLengthKeepaliveBlob = []byte{
	0x18, 0x00, 0x00, 0x00, // Length
	0x01, 0x02, 0x03, 0x04, // Source Actor ID
	0x05, 0x06, 0x07, 0x08, // Target Actor ID
}

func TestHeaderDecode(t *testing.T) {
	reader := bufio.NewReader(bytes.NewReader(decompressedKeepaliveBlob))

	header := GenericHeader{}
	err := header.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	if header.Length != 24 {
		t.Errorf("Expected message length 24, got %v", header.Length)
	}

	if header.SourceActor != 0x04030201 {
		t.Errorf("Expected source actor 0x04030201, got %v", header.SourceActor)
	}

	if header.TargetActor != 0x08070605 {
		t.Errorf("Expected target actor 0x08070605, got %v", header.TargetActor)
	}

	if header.Segment != ServerPong {
		t.Errorf("Expected Keepalive response segment (8), got %v", header.Segment)
	}

	reader.Reset(bytes.NewReader(lalafellLengthKeepaliveBlob))
	shortHeader := GenericHeader{}
	err = shortHeader.Decode(reader)

	// This is fucking dumb, for some reason errors.As() doesn't work here
	typedErr, ok := err.(ErrNotEnoughData)
	if !ok {
		t.Errorf(err.Error())
	}

	if err.Error() != "Not enough data: Expected 16 bytes but received 12: EOF" {
		t.Errorf("Unexpected ErrNotEnoughData string! Expected 'Not enough data: Expected 16 bytes but received 12: EOF', got %s",
			err.Error())
	}

	if typedErr.Unwrap() != io.EOF {
		t.Errorf("Expected io.EOF, received %v", typedErr.Unwrap())
	}
}

func TestHeaderStringer(t *testing.T) {
	var sentinel = "Segment - size: 24, source: 67305985, target: 134678021, segment: 8\n"
	reader := bufio.NewReader(bytes.NewReader(decompressedKeepaliveBlob))

	message := GenericHeader{}
	err := message.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	stringy := message.String()

	if stringy != sentinel {
		t.Errorf("Unexpected string, got %s, expected %s", stringy, sentinel)
	}
}

func TestGameEventDecode(t *testing.T) {
	z, _ := zlib.NewReader(bytes.NewReader(compressedGameEventBlob))
	reader := bufio.NewReader(z)

	message := new(GameEventMessage)
	err := message.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	if message.Length != 48 {
		t.Errorf("Expected message length 48, got %v", message.Length)
	}

	if message.SourceActor != 0x105831E2 {
		t.Errorf("Expected source actor 0x105831E2, got %v", message.SourceActor)
	}

	if message.TargetActor != 0x1068DC38 {
		t.Errorf("Expected target actor 0x1068DC38, got %v", message.TargetActor)
	}

	if message.Segment != GameEvent {
		t.Errorf("Expected GameEvent segment (3), got %v", message.Segment)
	}

	if message.Opcode != 0x125 {
		t.Errorf("Expected opcode 0x125, got %v", message.Opcode)
	}

	if message.Timestamp != time.Unix(int64(1580625008), int64(0)) {
		t.Errorf("Expected GameEvent timestamp to be 2020-02-02 06:30:08 GMT, got %v", message.Timestamp.UnixNano())
	}

	reader.Reset(bytes.NewReader(lalafellLengthGameEventBlob))
	shortMessage := new(GameEventMessage)
	err = shortMessage.Decode(reader)

	typedErr, ok := err.(ErrNotEnoughData)
	if !ok {
		t.Errorf(err.Error())
	}

	if err.Error() != "Not enough data: Expected 48 bytes but received 16: EOF" {
		t.Errorf("Unexpected ErrNotEnoughData string! Expected 'Not enough data: Expected 48 bytes but received 16: EOF', got %s",
			err.Error())
	}

	if typedErr.Unwrap() != io.EOF {
		t.Errorf("Expected io.EOF, received %v", typedErr.Unwrap())
	}
}

func TestGameEventMarshal(t *testing.T) {
	var sentinel = `{"data":[94,76,1,0,16,39,0,0,218,2,126,221,255,127,0,0],"timestamp":1580625008,"size":48,"sourceActorID":274215394,"targetActorID":275307576,"segmentType":3,"opcode":293,"serverID":3}`
	z, _ := zlib.NewReader(bytes.NewReader(compressedGameEventBlob))
	reader := bufio.NewReader(z)

	message := new(GameEventMessage)
	err := message.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	serialised, err := json.Marshal(message)
	if err != nil {
		t.Error(err)
	}

	if string(serialised) != sentinel {
		t.Errorf("Unexpected encoding, got %s, expected %s", string(serialised), sentinel)
	}
}

func TestGameEventStringer(t *testing.T) {
	var sentinel = "Segment - size: 48, source: 274215394, target: 275307576, segment: 3\nMessage - server: 3, opcode: 0x125, timestamp: 1580625008\n"
	z, _ := zlib.NewReader(bytes.NewReader(compressedGameEventBlob))
	reader := bufio.NewReader(z)

	message := new(GameEventMessage)
	err := message.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	stringy := message.String()

	if stringy != sentinel {
		t.Errorf("Unexpected string, got %s, expected %s", stringy, sentinel)
	}
}

func TestKeepaliveDecode(t *testing.T) {
	reader := bufio.NewReader(bytes.NewReader(decompressedKeepaliveBlob))

	message := KeepaliveMessage{}
	err := message.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	if message.Length != 24 {
		t.Errorf("Expected message length 24, got %v", message.Length)
	}

	if message.SourceActor != 0x04030201 {
		t.Errorf("Expected source actor 0x04030201, got %v", message.SourceActor)
	}

	if message.TargetActor != 0x08070605 {
		t.Errorf("Expected target actor 0x08070605, got %v", message.TargetActor)
	}

	if message.Segment != ServerPong {
		t.Errorf("Expected Keepalive response segment (8), got %v", message.Segment)
	}

	if message.ID != 123456789 {
		t.Errorf("Expected Keepalive ID 123456789, got %v", message.ID)
	}

	if message.Timestamp != time.Unix(int64(1485430850), int64(0)) {
		t.Errorf("Expected Keepalive timestamp to be 2017-01-26 11:40:50 GMT, got %v", message.Timestamp.UnixNano())
	}

	reader.Reset(bytes.NewReader(lalafellLengthKeepaliveBlob))
	shortMessage := new(KeepaliveMessage)
	err = shortMessage.Decode(reader)

	typedErr, ok := err.(ErrNotEnoughData)
	if !ok {
		t.Errorf(err.Error())
	}

	if err.Error() != "Not enough data: Expected 16 bytes but received 12: EOF" {
		t.Errorf("Unexpected ErrNotEnoughData string! Expected 'Not enough data: Expected 16 bytes but received 12: EOF', got %s",
			err.Error())
	}

	if typedErr.Unwrap() != io.EOF {
		t.Errorf("Expected io.EOF, received %v", typedErr.Unwrap())
	}
}

func TestKeepaliveMarshal(t *testing.T) {
	var sentinel = `{"size":24,"sourceActorID":67305985,"targetActorID":134678021,"segmentType":8,"ID":123456789}`
	reader := bufio.NewReader(bytes.NewReader(decompressedKeepaliveBlob))

	message := KeepaliveMessage{}
	err := message.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	serialised, err := json.Marshal(message)
	if err != nil {
		t.Error(err)
	}

	if string(serialised) != sentinel {
		t.Errorf("Unexpected encoding, got %s, expected %s", string(serialised), sentinel)
	}
}

func TestKeepaliveStringer(t *testing.T) {
	var sentinel = "Segment - size: 24, source: 67305985, target: 134678021, segment: 8\nMessage - ID: 123456789, timestamp: 1485430850\n"
	reader := bufio.NewReader(bytes.NewReader(decompressedKeepaliveBlob))

	message := KeepaliveMessage{}
	err := message.Decode(reader)
	if err != nil {
		t.Errorf(err.Error())
	}

	stringy := message.String()

	if stringy != sentinel {
		t.Errorf("Unexpected string, got %s, expected %s", stringy, sentinel)
	}
}
