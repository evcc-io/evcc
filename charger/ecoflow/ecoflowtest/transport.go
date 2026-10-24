// Package ecoflowtest provides a fake MQTT transport and captured PowerPulse
// frames for testing code built on package ecoflow without a broker.
package ecoflowtest

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"google.golang.org/protobuf/encoding/protowire"
)

// Message is one publish recorded by the fake transport.
type Message struct {
	Topic   string
	Qos     byte
	Payload []byte
}

// Transport is a fake ecoflow.Transport. Feed delivers a payload to the
// handler subscribed to a topic; Published returns what was sent.
type Transport struct {
	mu        sync.Mutex
	handlers  map[string]func([]byte)
	qos       map[string]byte
	published []Message
	onPublish func(Message)
}

// NewTransport creates an empty fake transport.
func NewTransport() *Transport {
	return &Transport{
		handlers: make(map[string]func([]byte)),
		qos:      make(map[string]byte),
	}
}

// Subscribe implements ecoflow.Transport.
func (t *Transport) Subscribe(topic string, qos byte, handler func([]byte)) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers[topic] = handler
	t.qos[topic] = qos
	return nil
}

// Publish implements ecoflow.Transport.
func (t *Transport) Publish(_ context.Context, topic string, qos byte, payload []byte) error {
	msg := Message{Topic: topic, Qos: qos, Payload: payload}

	t.mu.Lock()
	t.published = append(t.published, msg)
	hook := t.onPublish
	t.mu.Unlock()

	if hook != nil {
		hook(msg)
	}
	return nil
}

// OnPublish registers fn to run after every publish, outside the lock and
// before Publish returns. Use it to answer a write with telemetry, the way
// the wallbox reacts to a command.
func (t *Transport) OnPublish(fn func(Message)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onPublish = fn
}

// Feed delivers payload to the handler subscribed to topic, synchronously.
// It returns an error when nothing is subscribed to topic.
func (t *Transport) Feed(topic string, payload []byte) error {
	t.mu.Lock()
	handler, ok := t.handlers[topic]
	t.mu.Unlock()

	if !ok {
		return fmt.Errorf("no subscription for %s", topic)
	}
	handler(payload)
	return nil
}

// Subscriptions returns the subscribed topics with their QoS.
func (t *Transport) Subscriptions() map[string]byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	res := make(map[string]byte, len(t.qos))
	for k, v := range t.qos {
		res[k] = v
	}
	return res
}

// Published returns all recorded publishes in order.
func (t *Transport) Published() []Message {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Message(nil), t.published...)
}

// Serial and Gateway are the masked serials used in the captured frames.
const (
	Serial  = "C376000000000001"
	Gateway = "HJ35000000000001"
)

//go:embed testdata/heartbeats.json
var heartbeatsJSON []byte

//go:embed testdata/writes.json
var writesJSON []byte

//go:embed testdata/settings.json
var settingsJSON []byte

// Heartbeat returns a captured 2/33 MQTT payload from 2026-10-23 or 2026-10-24
// as it arrived on the telemetry topic (XOR-masked, serials replaced by Serial
// and Gateway). Names are state<N>-<1p|3p>, e.g. "state3-1p"; available states
// are 1, 2, 3, 5 and 6. "state3-3p" charges at 16 A, "state3-3p-15a" at 15 A
// (10.25 kW, 15.1 A per phase), "state3-1p" at 10 A on L1 only (2.2 kW).
// "state2-3p-fixed" follows a written 3p (field 21 = 0) and "state6-1p" a stop
// in 1p (field 21 = 1), both from 2026-10-24.
func Heartbeat(name string) []byte {
	return payload(heartbeatsJSON, "heartbeat", name)
}

// Settings returns a captured 241/44 settings report as it arrived on the
// telemetry topic (XOR-masked, serials replaced by Serial and Gateway).
// "max10", "max15" and "max16" report that maximum current with phase mode
// auto; "phase1p", "phaseauto" (2026-10-23) and "phase3p" (2026-10-24) are the
// first reports showing the new phase mode after the phase writes. "no-params" is the shorter 241/44
// variant without the settings block, which the wallbox interleaves with the
// full one.
func Settings(name string) []byte {
	return payload(settingsJSON, "settings", name)
}

// payload looks up a hex-encoded fixture by name.
func payload(data []byte, kind, name string) []byte {
	var fixtures map[string]string
	if err := json.Unmarshal(data, &fixtures); err != nil {
		panic(err)
	}
	s, ok := fixtures[name]
	if !ok {
		panic("unknown " + kind + " fixture: " + name)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// HeartbeatWithState builds a minimal device 2/33 payload carrying only the
// given state, masked like a real device frame. Use it for states that were
// not observed on hardware.
func HeartbeatWithState(state uint64) []byte {
	const seq = 0x1234

	pdata := protowire.AppendTag(nil, 1, protowire.VarintType)
	pdata = protowire.AppendVarint(pdata, state)
	for i := range pdata {
		pdata[i] ^= seq & 0xff
	}

	h := protowire.AppendTag(nil, 1, protowire.BytesType)
	h = protowire.AppendBytes(h, pdata)
	for _, f := range [][2]uint64{{2, 2}, {3, 32}, {6, 1}, {8, 2}, {9, 33}, {10, uint64(len(pdata))}, {14, seq}} {
		h = protowire.AppendTag(h, protowire.Number(f[0]), protowire.VarintType)
		h = protowire.AppendVarint(h, f[1])
	}

	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(out, h)
}

// Write returns a write that the wallbox executed, with serials and user ID
// masked (user ID 1000000000000000001). Names are "stop", "start",
// "max-current-10", "max-current-16", "phase-1p" and "phase-auto", published by
// the probe on 2026-10-23, and "phase-3p", published by this package on 2026-10-24. The captured seq is replaced by seq, so the frame can be
// compared with what a fresh connection sends; every other byte is as captured.
func Write(name string, seq uint64) Message {
	var fixtures map[string]struct {
		Topic string
		Qos   byte
		Hex   string
	}
	if err := json.Unmarshal(writesJSON, &fixtures); err != nil {
		panic(err)
	}
	f, ok := fixtures[name]
	if !ok {
		panic("unknown write fixture: " + name)
	}
	b, err := hex.DecodeString(f.Hex)
	if err != nil {
		panic(err)
	}
	return Message{Topic: f.Topic, Qos: f.Qos, Payload: replaceSeq(b, seq)}
}

// replaceSeq re-encodes a single-header payload with header field 14 (seq)
// set to seq, keeping all other fields in their captured order.
func replaceSeq(payload []byte, seq uint64) []byte {
	num, typ, n := protowire.ConsumeTag(payload)
	if num != 1 || typ != protowire.BytesType {
		panic("payload has no header")
	}
	in, m := protowire.ConsumeBytes(payload[n:])
	if m < 0 || n+m != len(payload) {
		panic("payload must carry exactly one header")
	}

	var h []byte
	for len(in) > 0 {
		num, typ, n := protowire.ConsumeTag(in)
		size := protowire.ConsumeFieldValue(num, typ, in[n:])
		if n < 0 || size < 0 {
			panic("malformed header")
		}
		if num == 14 && typ == protowire.VarintType {
			h = protowire.AppendTag(h, 14, protowire.VarintType)
			h = protowire.AppendVarint(h, seq)
		} else {
			h = append(h, in[:n+size]...)
		}
		in = in[n+size:]
	}

	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(out, h)
}
