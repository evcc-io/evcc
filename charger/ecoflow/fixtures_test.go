package ecoflow

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"

	"google.golang.org/protobuf/encoding/protowire"
)

// testGateway is the masked PowerOcean serial of the captured frames.
const testGateway = "HJ35000000000001"

//go:embed testdata/heartbeats.json
var heartbeatsJSON []byte

//go:embed testdata/writes.json
var writesJSON []byte

// fixtureHeartbeat returns a captured 2/33 payload as it arrived on the
// telemetry topic, XOR-masked and with serials replaced.
func fixtureHeartbeat(name string) []byte {
	var fixtures map[string]string
	if err := json.Unmarshal(heartbeatsJSON, &fixtures); err != nil {
		panic(err)
	}
	b, err := hex.DecodeString(fixtures[name])
	if err != nil || len(b) == 0 {
		panic("unknown heartbeat fixture: " + name)
	}
	return b
}

type fixtureMessage struct {
	Topic   string
	Payload []byte
}

// fixtureWrite returns a write the wallbox executed, with the captured seq
// replaced by seq so it compares with a freshly built frame.
func fixtureWrite(name string, seq uint64) fixtureMessage {
	var fixtures map[string]struct{ Topic, Hex string }
	if err := json.Unmarshal(writesJSON, &fixtures); err != nil {
		panic(err)
	}
	b, err := hex.DecodeString(fixtures[name].Hex)
	if err != nil || len(b) == 0 {
		panic("unknown write fixture: " + name)
	}
	return fixtureMessage{Topic: fixtures[name].Topic, Payload: replaceSeq(b, seq)}
}

// replaceSeq re-encodes a single-header payload with header field 14 set to seq.
func replaceSeq(payload []byte, seq uint64) []byte {
	_, _, n := protowire.ConsumeTag(payload)
	in, _ := protowire.ConsumeBytes(payload[n:])

	var h []byte
	for len(in) > 0 {
		num, typ, n := protowire.ConsumeTag(in)
		size := protowire.ConsumeFieldValue(num, typ, in[n:])
		if n < 0 || size < 0 {
			panic("malformed header")
		}
		if num == fSeq && typ == protowire.VarintType {
			h = protowire.AppendTag(h, fSeq, protowire.VarintType)
			h = protowire.AppendVarint(h, seq)
		} else {
			h = append(h, in[:n+size]...)
		}
		in = in[n+size:]
	}

	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(out, h)
}
