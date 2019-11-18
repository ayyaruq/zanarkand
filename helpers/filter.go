package helpers

import (
	"fmt"

	"github.com/ayyaruq/zanarkand"
)

// FilterOpcodes takes a list of Messages and Opcodes and returns a list of matching Messages.
func FilterOpcodes(m []zanarkand.RawMessage, opcodes []int) []zanarkand.RawMessage {
	var flist []zanarkand.RawMessage

	for _, message := range m {
		header, ok := message.Header.(zanarkand.GameEventHeader)
		if ok {
			for _, opcode := range opcodes {
				if header.Opcode == uint16(opcode) {
					flist = append(flist, message)
					break
				} else {
					fmt.Printf("opcode %X did not match %X\n", header.Opcode, uint16(opcode))
				}
			}
		}
		fmt.Printf("length of filtered list so far %d\n", len(flist))
	}

	return flist
}

// FilterSegments takes a list of Messages and Segment IDs and returns a list of matching Messages.
func FilterSegments(m []zanarkand.RawMessage, segments []int) []zanarkand.RawMessage {
	var flist []zanarkand.RawMessage

	for _, message := range m {
		for _, segment := range segments {
			if message.Segment() == uint16(segment) {
				flist = append(flist, message)
				break
			}
		}
	}

	return flist
}
