package zanarkand

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

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
	_, ok := err.(ErrNotEnoughData)
	if !ok {
		t.Errorf(err.Error())
	}
}

func TestHeaderStringer(t *testing.T) {
	var sentinel = "Segment - length: 24, source: 67305985, target: 134678021, segment: 8\n"
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
}

func TestKeepaliveMarshal(t *testing.T) {
	var sentinel = `{"length":24,"sourceActorID":67305985,"targetActorID":134678021,"segmentType":8,"ID":123456789}`
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
	var sentinel = "Segment - length: 24, source: 67305985, target: 134678021, segment: 8\nMessage - ID: 123456789, timestamp: 1485430850\n"
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
