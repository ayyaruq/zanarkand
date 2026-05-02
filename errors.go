package zanarkand

import (
	"fmt"
)

// ErrNotEnoughData occurs when the Length field is longer than the payload.
type ErrNotEnoughData struct {
	Expected int
	Received int
	Err      error
}

func (e ErrNotEnoughData) Error() string {
	return fmt.Sprintf("not enough data: expected %d bytes but received %d: %v", e.Expected, e.Received, e.Err)
}

func (e *ErrNotEnoughData) Unwrap() error { return e.Err }

// ErrDecodingFailure indicates an error during decoding a specific message type.
type ErrDecodingFailure struct {
	Err error
}

func (e ErrDecodingFailure) Error() string {
	return fmt.Sprintf("unable to decode message: %v", e.Err)
}

func (e *ErrDecodingFailure) Unwrap() error { return e.Err }

// ErrUnknownInput indicates the provided mode for a sniffer is not a known type.
type ErrUnknownInput struct {
	Err error
}

func (e ErrUnknownInput) Error() string {
	return fmt.Sprintf("unknown input type: %v", e.Err)
}

func (e *ErrUnknownInput) Unwrap() error { return e.Err }

// ErrReassemblyError indicates an error during TCP stream reassembly.
type ErrReassemblyError struct {
	Err error
}

func (e ErrReassemblyError) Error() string {
	return fmt.Sprintf("reassembly error: %v", e.Err)
}

func (e *ErrReassemblyError) Unwrap() error { return e.Err }
