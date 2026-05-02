package zanarkand

import (
	"bufio"
	"context"
	"fmt"
)

// Subscriber describes the interface for individual Frame segment subscribers.
type Subscriber interface {
	Subscribe(ctx context.Context, s *Sniffer) error
	Close(s *Sniffer)
}

// GameEventSubscriber is a Subscriber for GameEvent segments.
type GameEventSubscriber struct {
	IngressEvents chan *GameEventMessage
	EgressEvents  chan *GameEventMessage
}

// NewGameEventSubscriber returns a Subscriber handle with channels for inbound and outbound GameEventMessages.
func NewGameEventSubscriber() *GameEventSubscriber {
	return &GameEventSubscriber{
		IngressEvents: make(chan *GameEventMessage),
		EgressEvents:  make(chan *GameEventMessage),
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

// KeepaliveSubscriber is a Subscriber for Keepalive segments.
type KeepaliveSubscriber struct {
	Events chan *KeepaliveMessage
}

// NewKeepaliveSubscriber returns a Subscriber handle. As the traffic is minimal, this subscriber uses a single Event channel.
func NewKeepaliveSubscriber() *KeepaliveSubscriber {
	return &KeepaliveSubscriber{
		Events: make(chan *KeepaliveMessage),
	}
}

// Subscribe starts the KeepaliveSubscriber. It blocks until the context is cancelled,
// the Sniffer is stopped, or an error occurs. If the Sniffer is not already running,
// it will be started in a goroutine.
func (k *KeepaliveSubscriber) Subscribe(ctx context.Context, s *Sniffer) error {
	if !s.IsActive() {
		go s.Start(ctx)
	}

	return s.ProcessFrames(func(frame *Frame, header *GenericHeader, r *bufio.Reader) error {
		if header.Segment != ServerPing && header.Segment != ServerPong {
			return nil
		}

		msg := new(KeepaliveMessage)
		if err := msg.Decode(r); err != nil {
			return ErrDecodingFailure{Err: err}
		}

		k.Events <- msg
		return nil
	})
}

// Close will stop a sniffer, drain the channel, then close the channel.
func (k *KeepaliveSubscriber) Close(s *Sniffer) {
	s.Stop()
	close(k.Events)
}
