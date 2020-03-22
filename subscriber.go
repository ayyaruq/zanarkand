package zanarkand

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
)

// Subscriber describes the interface for individual Frame segment subscribers.
type Subscriber interface {
	Subscribe(*Sniffer)
	Close()
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

// Subscribe starts the GameEventSubscriber.
func (g *GameEventSubscriber) Subscribe(s *Sniffer) error {
	if !s.Active {
		go s.Start()
	}

	for {
		frame, err := s.NextFrame()
		if err != nil {
			return fmt.Errorf("error retrieving next frame: %w", err)
		}

		// Setup our Message reader
		r := bufio.NewReader(bytes.NewReader(frame.Body))
		if frame.Compressed {
			z, err := zlib.NewReader(bytes.NewReader(frame.Body))
			if err != nil {
				return fmt.Errorf("error creating ZLIB decoder: %w", err)
			}

			defer z.Close()

			r.Reset(z)
		}

		for i := 0; i < int(frame.Count); i++ {
			header := new(GenericHeader)
			err := header.Decode(r)
			if err != nil {
				return ErrDecodingFailure{Err: err}
			}

			if header.Segment == GameEvent {
				msg := new(GameEventMessage)
				msg.Decode(r)

				switch frame.Direction() {
				case FrameIngress:
					g.IngressEvents <- msg

				case FrameEgress:
					g.EgressEvents <- msg

				default:
					return ErrDecodingFailure{Err: fmt.Errorf("unexpected frame direction")}
				}
			}
		}

		// We're done with the current frame,
		// if Sniffer is stopped then exit and
		// user can start a new subscriber routine.
		if !s.Active {
			return nil
		}
	}
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

// Subscribe starts the KeepaliveSubscriber.
func (k *KeepaliveSubscriber) Subscribe(s *Sniffer) error {
	if !s.Active {
		go s.Start()
	}

	for {
		frame, err := s.NextFrame()
		if err != nil {
			return fmt.Errorf("error retrieving next frame: %s", err)
		}

		// Setup our Message reader
		r := bufio.NewReader(bytes.NewReader(frame.Body))
		if frame.Compressed {
			z, err := zlib.NewReader(bytes.NewReader(frame.Body))
			if err != nil {
				return fmt.Errorf("error creating ZLIB decoder: %w", err)
			}

			defer z.Close()

			r.Reset(z)
		}

		for i := 0; i < int(frame.Count); i++ {
			header := new(GenericHeader)
			err := header.Decode(r)
			if err != nil {
				return ErrDecodingFailure{Err: err}
			}

			if header.Segment == ServerPing || header.Segment == ServerPong {
				msg := new(KeepaliveMessage)
				msg.Decode(r)

				k.Events <- msg
			}
		}

		if !s.Active {
			return nil
		}
	}
}

// Close will stop a sniffer, drain the channel, then close the channel.
func (k *KeepaliveSubscriber) Close(s *Sniffer) {
	s.Stop()
	close(k.Events)
}
