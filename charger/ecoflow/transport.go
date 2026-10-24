package ecoflow

import (
	"errors"
	"slices"
	"sync"
)

// ErrNotConnected is returned by a publish while the connection is down.
var ErrNotConnected = errors.New("not connected")

var errClosed = errors.New("connection closed")

type subscription struct {
	topic   string
	qos     byte
	handler func([]byte)
}

// switchTransport is the stable Transport of a Connection. It replays the
// subscriptions onto each new underlying transport, so devices survive a reconnect.
type switchTransport struct {
	mu      sync.Mutex
	current Transport // nil while disconnected
	gen     uint64    // counts attach and detach
	subs    []subscription
	closed  bool
}

func newSwitchTransport(t Transport) *switchTransport {
	return &switchTransport{current: t}
}

// generation identifies the attached transport.
func (s *switchTransport) generation() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gen
}

// attach makes t the underlying transport and subscribes it to all topics.
func (s *switchTransport) attach(t Transport) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		closeTransport(t)
		return errClosed
	}

	s.current = t
	s.gen++

	for _, sub := range s.subs {
		if err := t.Subscribe(sub.topic, sub.qos, sub.handler); err != nil {
			return err
		}
	}

	return nil
}

// detach drops the underlying transport, closing it when it can be closed.
func (s *switchTransport) detach() {
	s.mu.Lock()
	t := s.current
	s.current = nil
	s.gen++
	s.mu.Unlock()

	closeTransport(t)
}

// close detaches for good: later attaches are refused.
func (s *switchTransport) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	s.detach()
}

func closeTransport(t Transport) {
	if closer, ok := t.(interface{ Close() }); ok {
		closer.Close()
	}
}

// Subscribe implements Transport. While disconnected the subscription is only
// remembered for the next attach.
func (s *switchTransport) Subscribe(topic string, qos byte, handler func([]byte)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// a re-created device replaces its predecessor's handler
	s.subs = slices.DeleteFunc(s.subs, func(sub subscription) bool { return sub.topic == topic })
	s.subs = append(s.subs, subscription{topic: topic, qos: qos, handler: handler})

	if s.current == nil {
		return nil
	}
	return s.current.Subscribe(topic, qos, handler)
}

// Publish implements Transport.
func (s *switchTransport) Publish(topic string, qos byte, payload []byte) error {
	s.mu.Lock()
	t := s.current
	s.mu.Unlock()

	if t == nil {
		return ErrNotConnected
	}
	return t.Publish(topic, qos, payload)
}
