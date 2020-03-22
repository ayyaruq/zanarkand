package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var EventIDs = map[string]uint32{
	"CraftState": 0xA0001,
}

type EventPlayHeader struct {
	ActorID    uint64
	EventID    uint32
	Scene      uint16
	Pad1       uint16
	Flags      uint32
	P1         uint32
	ParamCount byte
	Pad2       [3]byte
	P2         uint32
}

type EventPlay32 struct {
	EventPlayHeader
	Data EventPlay32Data
}

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
		err := binary.Read(bytes.NewReader(data[headerSize:]), binary.LittleEndian, &craftState)
		if err != nil {
			return err
		}
		e.Data = craftState

	default:
		event := new(GenericEventPlay32Data)
		err := binary.Read(bytes.NewReader(data[headerSize:]), binary.LittleEndian, &event)
		if err != nil {
			return err
		}
		e.Data = event
	}

	return nil
}

type EventPlay32Data interface {
	isEventPlay32Data()
}

type GenericEventPlay32Data [32]uint32

func (GenericEventPlay32Data) isEventPlay32Data() {}
