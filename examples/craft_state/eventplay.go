package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// EventIDs is a map of the different Event subtypes.
var EventIDs = map[string]uint32{
	"CraftState": 0xA0001,
}

// EventPlayHeader is the shared header for the different EventPlay Messages.
type EventPlayHeader struct {
	ActorID    uint64
	EventID    uint32
	Scene      uint16
	Pad1       uint16 `json:"-"`
	Flags      uint32
	P1         uint32
	ParamCount byte
	Pad2       [3]byte `json:"-"`
	P2         uint32
}

// EventPlay32 is the 32-byte variant of the EventPlay Messages.
type EventPlay32 struct {
	EventPlayHeader
	Data EventPlay32Data
}

// UnmarshalBytes will take a raw binary slice from a Message and unmarshal it into an EventPlay32 struct.
func (e *EventPlay32) UnmarshalBytes(data []byte) error {
	headerSize := binary.Size(e.EventPlayHeader)
	if len(data) != (headerSize + 128) {
		return fmt.Errorf("unexpected length: received %d, expected %d", len(data), headerSize+128)
	}

	err := binary.Read(bytes.NewReader(data[:headerSize]), binary.LittleEndian, &e.EventPlayHeader)
	if err != nil {
		return err
	}

	switch e.EventID {
	case EventIDs["CraftState"]:
		craftState := new(CraftState)
		err := binary.Read(bytes.NewReader(data[headerSize:]), binary.LittleEndian, craftState)
		if err != nil {
			return err
		}
		e.Data = craftState

	default:
		event := new(GenericEventPlay32Data)
		err := binary.Read(bytes.NewReader(data[headerSize:]), binary.LittleEndian, event)
		if err != nil {
			return err
		}
		e.Data = event
	}

	return nil
}

// EventPlay32Data is the interface for EventPlay32 subtypes.
type EventPlay32Data interface {
	isEventPlay32Data()
}

// GenericEventPlay32Data is the implementation for EventPlay32 subtypes that don't have explicit handling.
type GenericEventPlay32Data [32]uint32

func (GenericEventPlay32Data) isEventPlay32Data() {}
