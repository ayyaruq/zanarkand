package zanarkand

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
)

type Subscriber interface {
	Subscribe(*Sniffer)
}

type GameEventSubscriber struct {
	Events chan *GameEventMessage
}

func NewGameEventSubscriber() *GameEventSubscriber {
	return &GameEventSubscriber{
		Events: make(chan *GameEventMessage),
	}
}

func (g *GameEventSubscriber) Subscribe(s *Sniffer) error {
	if !s.Active() {
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
				return fmt.Errorf("error creating ZLIB decoder: %s", err)
			}

			defer z.Close()

			r.Reset(z)
		}

		for i := 0; i < int(frame.Count); i++ {
			header := new(GenericHeader)
			err := header.Decode(r)
			if err != nil {
				return fmt.Errorf("error decoding message header: %s", err)
			}

			if header.Segment == GameEvent {
				msg := new(GameEventMessage)
				msg.Decode(r)

				g.Events <- msg
			}
		}
	}
}

type KeepaliveSubscriber struct {
	Events chan *KeepaliveMessage
}

func NewKeepaliveSubscriber() *KeepaliveSubscriber {
	return &KeepaliveSubscriber{
		Events: make(chan *KeepaliveMessage),
	}
}

func (k *KeepaliveSubscriber) Subscribe(s *Sniffer) error {
	if !s.Active() {
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
				return fmt.Errorf("error creating ZLIB decoder: %s", err)
			}

			defer z.Close()

			r.Reset(z)
		}

		for i := 0; i < int(frame.Count); i++ {
			header := new(GenericHeader)
			err := header.Decode(r)
			if err != nil {
				return fmt.Errorf("error decoding message header: %s", err)
			}

			if (header.Segment == ServerPing || header.Segment == ServerPong) {
				msg := new(KeepaliveMessage)
				msg.Decode(r)

				k.Events <- msg
			}
		}
	}
}
