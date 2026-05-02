package zanarkand

import (
	"bufio"
	"context"
	"fmt"
)

// GameEventSubscriber is a Subscriber for GameEvent segments.
type GameEventSubscriber struct {
	IngressEvents chan *GameEventMessage
	EgressEvents  chan *GameEventMessage
	opcodes       map[uint16]struct{}
}

// NewGameEventSubscriber returns a Subscriber handle with channels for inbound and outbound GameEventMessages.
func NewGameEventSubscriber(opts ...GameEventOption) *GameEventSubscriber {
	cfg := gameEventConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &GameEventSubscriber{
		IngressEvents: make(chan *GameEventMessage),
		EgressEvents:  make(chan *GameEventMessage),
		opcodes:       cfg.opcodes,
	}
}

// Subscribe starts the GameEventSubscriber. It blocks until the context is cancelled,
// the Sniffer is stopped, or an error occurs. If the Sniffer is not already running,
// it will be started in a goroutine.
func (g *GameEventSubscriber) Subscribe(ctx context.Context, s *Sniffer) error {
	if !s.IsActive() {
		go s.Start(ctx)
	}

	return s.ProcessFrames(func(frame *Frame, header *GenericHeader, r *bufio.Reader) error {
		if header.Segment != GameEvent {
			return nil
		}

		msg := new(GameEventMessage)
		if err := msg.Decode(r); err != nil {
			return ErrDecodingFailure{Err: err}
		}

		if len(g.opcodes) > 0 {
			if _, ok := g.opcodes[msg.Opcode]; !ok {
				return nil
			}
		}

		switch frame.Direction() {
		case FrameIngress:
			g.IngressEvents <- msg
		case FrameEgress:
			g.EgressEvents <- msg
		default:
			return ErrDecodingFailure{Err: fmt.Errorf("unexpected frame direction")}
		}
		return nil
	})
}

// Close will stop a sniffer, drain the channels, then close the channels.
func (g *GameEventSubscriber) Close(s *Sniffer) {
	s.Stop()
	close(g.IngressEvents)
	close(g.EgressEvents)
}

