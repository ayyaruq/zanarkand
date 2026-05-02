package zanarkand

import (
	"bufio"
	"context"
)

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

// KeepaliveCallback is a function called for each decoded KeepaliveMessage.
type KeepaliveCallback func(msg *KeepaliveMessage)

// KeepaliveHandler delivers KeepaliveMessages via a callback function
// instead of channels.
type KeepaliveHandler struct {
	callback KeepaliveCallback
}

// NewKeepaliveHandler returns a subscriber that calls fn for each
// decoded KeepaliveMessage.
func NewKeepaliveHandler(fn KeepaliveCallback) *KeepaliveHandler {
	return &KeepaliveHandler{
		callback: fn,
	}
}

// Subscribe starts the KeepaliveHandler. It blocks until the context is cancelled,
// the Sniffer is stopped, or an error occurs. If the Sniffer is not already running,
// it will be started in a goroutine.
func (k *KeepaliveHandler) Subscribe(ctx context.Context, s *Sniffer) error {
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

		k.callback(msg)
		return nil
	})
}

// Close stops the sniffer.
func (k *KeepaliveHandler) Close(s *Sniffer) {
	s.Stop()
}
