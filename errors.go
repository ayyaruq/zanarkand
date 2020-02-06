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
	return fmt.Sprintf("Not enough data: Expected %d bytes but received %d: %v", e.Expected, e.Received, e.Err)
}

func (e *ErrNotEnoughData) Unwrap() error { return e.Err }

// ErrDecodingFailure indicates an error during decoding a specific message type.
type ErrDecodingFailure struct {
	Err error
}

func (e ErrDecodingFailure) Error() string {
	return fmt.Sprintf("Unable to decode message: %v", e.Err)
}

func (e *ErrDecodingFailure) Unwrap() error { return e.Err }

// ErrUnknownInput indicates the provided mode for a sniffer is not a known type.
type ErrUnknownInput struct {
	Err error
}

func (e *ErrUnknownInput) Unwrap() error { return e.Err }
