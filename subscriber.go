package zanarkand

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"sync"
)

// Subscriber describes the interface for individual Frame segment subscribers.
type Subscriber interface {
	Subscribe(*Sniffer)
	Close()
}

type readerPool struct {
	body sync.Pool
	bare sync.Pool
	zlib sync.Pool
}

func newReaderPool() *readerPool {
	return &readerPool{
		body: sync.Pool{New: func() interface{} { return bytes.NewReader(nil) }},
		bare: sync.Pool{New: func() interface{} { return bufio.NewReader(nil) }},
		zlib: sync.Pool{},
	}
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

	pool := newReaderPool()

	for {
		frame, err := s.NextFrame()
		if err != nil {
			return fmt.Errorf("error retrieving next frame: %w", err)
		}

		// Setup our Message reader
		b := pool.body.Get().(*bytes.Reader)
		if b != nil {
			b.Reset(frame.Body)
		}

		r := pool.bare.Get().(*bufio.Reader)
		if r != nil {
			r.Reset(b)
		}

		z := pool.zlib.Get().(io.ReadCloser)
		if z != nil {
			err = z.(zlib.Resetter).Reset(b, nil)
			if err != nil {
				return fmt.Errorf("error resetting ZLIB decoder: %w", err)
			}
		} else {
			z, err = zlib.NewReader(b)
			if err != nil {
				return fmt.Errorf("error creating ZLIB decoder: %w", err)
			}
		}

		if frame.Compressed {
			r.Reset(z.(io.ReadCloser))
		}

		for i := 0; i < int(frame.Count); i++ {
			header := new(GenericHeader)

			err := header.Decode(r)
			if err != nil {
				return ErrDecodingFailure{Err: err}
			}

			if header.Segment == GameEvent {
				msg := new(GameEventMessage)

				err = msg.Decode(r)
				if err != nil {
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
			}
		}

		// Return our readers to the pool - the pool will get GC'd when the function exits
		pool.zlib.Put(z)
		pool.bare.Put(r)
		pool.body.Put(b)

		// We're done with the current frame, if Sniffer is stopped then exit,
		// allowing user to start a new subscriber routine.
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

	pool := newReaderPool()

	for {
		frame, err := s.NextFrame()
		if err != nil {
			return fmt.Errorf("error retrieving next frame: %s", err)
		}

		// Setup our Message reader
		b := pool.body.Get().(*bytes.Reader)
		if b != nil {
			b.Reset(frame.Body)
		}

		r := pool.bare.Get().(*bufio.Reader)
		if r != nil {
			r.Reset(b)
		}

		z := pool.zlib.Get().(io.ReadCloser)
		if z != nil {
			err = z.(zlib.Resetter).Reset(b, nil)
			if err != nil {
				return fmt.Errorf("error resetting ZLIB decoder: %w", err)
			}
		} else {
			z, err = zlib.NewReader(b)
			if err != nil {
				return fmt.Errorf("error creating ZLIB decoder: %w", err)
			}
		}

		if frame.Compressed {
			r.Reset(z.(io.ReadCloser))
		}

		for i := 0; i < int(frame.Count); i++ {
			header := new(GenericHeader)

			err := header.Decode(r)
			if err != nil {
				return ErrDecodingFailure{Err: err}
			}

			if header.Segment == ServerPing || header.Segment == ServerPong {
				msg := new(KeepaliveMessage)

				err = msg.Decode(r)
				if err != nil {
					return ErrDecodingFailure{Err: err}
				}

				k.Events <- msg
			}
		}

		// Return our readers to the pool
		pool.zlib.Put(z)
		pool.bare.Put(r)
		pool.body.Put(b)

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
