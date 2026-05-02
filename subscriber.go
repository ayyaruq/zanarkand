package zanarkand

import (
	"context"
)

// Subscriber describes the interface for individual Frame segment subscribers.
type Subscriber interface {
	Subscribe(ctx context.Context, s *Sniffer) error
	Close(s *Sniffer)
}

// GameEventOption configures a GameEventSubscriber or GameEventHandler.
type GameEventOption func(*gameEventConfig)

type gameEventConfig struct {
	opcodes map[uint16]struct{}
}

// WithOpcodes filters GameEventMessages to only those matching the given opcodes.
// If no opcodes are specified, all GameEvent messages are delivered.
func WithOpcodes(opcodes ...uint16) GameEventOption {
	return func(c *gameEventConfig) {
		c.opcodes = make(map[uint16]struct{}, len(opcodes))
		for _, op := range opcodes {
			c.opcodes[op] = struct{}{}
		}
	}
}
