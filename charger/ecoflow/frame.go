// Ported from shuette42/ecoflow-energy-ha (MIT), see the package documentation.

package ecoflow

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// EcoFlow header field numbers.
const (
	fPdata   = 1
	fSrc     = 2
	fDest    = 3
	fDSrc    = 4
	fDDest   = 5
	fEncType = 6
	fCheck   = 7
	fCmdFunc = 8
	fCmdID   = 9
	fDataLen = 10
	fNeedAck = 11
	fSeq     = 14
	fVersion = 16
	fPayVer  = 17
	fFrom    = 23
	fDevSN   = 25
)

// srcApp marks a frame the app itself sent. Such frames are never XOR-masked,
// even when enc_type is 1.
const srcApp = 32

// descriptorMarker is the device address of the PowerPulse behind the gateway.
const descriptorMarker = 215

// field is one decoded protobuf field occurrence.
type field struct {
	Num  int
	Wire protowire.Type
	V    uint64 // varint, fixed32 and fixed64 payloads
	B    []byte // length-delimited payload
}

// Header is one EcoFlow message header, i.e. one element of the repeated
// field 1 of the outer message.
type Header struct {
	fields []field
}

// Varint returns the first varint value for num.
func (h Header) Varint(num int) (uint64, bool) {
	for _, f := range h.fields {
		if f.Num == num && f.Wire == protowire.VarintType {
			return f.V, true
		}
	}
	return 0, false
}

// Bytes returns every length-delimited payload for num, in wire order.
func (h Header) Bytes(num int) [][]byte {
	var out [][]byte
	for _, f := range h.fields {
		if f.Num == num && f.Wire == protowire.BytesType {
			out = append(out, f.B)
		}
	}
	return out
}

// Command reports the (cmd_func, cmd_id) tuple. Frames without both carry no
// command, which is legal: the get-all request has neither.
func (h Header) Command() (uint64, uint64, bool) {
	fn, ok1 := h.Varint(fCmdFunc)
	id, ok2 := h.Varint(fCmdID)
	return fn, id, ok1 && ok2
}

// Pdata returns the payloads of field 1, XOR-unmasked when the header says so.
// Field 6 is enc_type; field 11 is need_ack and must not be confused with it.
func (h Header) Pdata() [][]byte {
	raw := h.Bytes(fPdata)
	enc, hasEnc := h.Varint(fEncType)
	seq, hasSeq := h.Varint(fSeq)
	src, _ := h.Varint(fSrc)

	if !hasEnc || enc != 1 || !hasSeq || src == srcApp {
		return raw
	}

	out := make([][]byte, len(raw))
	for i, p := range raw {
		out[i] = xorMask(p, byte(seq))
	}
	return out
}

// xorMask applies EcoFlow's pdata masking. It is self-inverse.
func xorMask(b []byte, key byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		out[i] = c ^ key
	}
	return out
}

// ParseMessage decodes one MQTT payload into its headers. Field 1 of the outer
// message is repeated, so a payload may carry several frames.
func ParseMessage(payload []byte) ([]Header, error) {
	outer, err := parseFields(payload)
	if err != nil {
		return nil, err
	}

	var out []Header
	for _, f := range outer {
		if f.Num != 1 || f.Wire != protowire.BytesType {
			continue
		}
		inner, err := parseFields(f.B)
		if err != nil {
			return nil, fmt.Errorf("header: %w", err)
		}
		out = append(out, Header{fields: inner})
	}
	return out, nil
}

// parseFields decodes a flat protobuf message. Unknown wire types are skipped.
func parseFields(b []byte) ([]field, error) {
	var out []field
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("tag: %w", protowire.ParseError(n))
		}
		b = b[n:]

		f := field{Num: int(num), Wire: typ}
		switch typ {
		case protowire.VarintType:
			f.V, n = protowire.ConsumeVarint(b)
		case protowire.Fixed32Type:
			var v uint32
			v, n = protowire.ConsumeFixed32(b)
			f.V = uint64(v)
		case protowire.Fixed64Type:
			f.V, n = protowire.ConsumeFixed64(b)
		case protowire.BytesType:
			f.B, n = protowire.ConsumeBytes(b)
		default:
			n = protowire.ConsumeFieldValue(num, typ, b)
		}
		if n < 0 {
			return nil, fmt.Errorf("field %d: %w", num, protowire.ParseError(n))
		}
		b = b[n:]
		out = append(out, f)
	}
	return out, nil
}

// descriptor identifies the PowerPulse inside a message addressed to the
// gateway: device address 215 plus the 16 ASCII bytes of the wallbox serial.
// It is built from the serial; on 2026-10-23 the descriptor read from 241/44
// was byte-identical to the built one on every run.
func descriptor(serial string) []byte {
	out := protowire.AppendTag(nil, 1, protowire.VarintType)
	out = protowire.AppendVarint(out, descriptorMarker)
	out = protowire.AppendTag(out, 2, protowire.BytesType)
	return protowire.AppendBytes(out, []byte(serial))
}

// envelope wraps a pdata body in the header the app uses for writes. The field
// order reproduces captured app frames byte for byte; enc_type stays absent
// because outgoing bodies are never masked. devSN is omitted when empty: the
// app sends it on 241/102 but not on 241/100.
func envelope(pdata []byte, cmdFunc, cmdID, seq uint64, devSN string) []byte {
	h := protowire.AppendTag(nil, fPdata, protowire.BytesType)
	h = protowire.AppendBytes(h, pdata)

	put := func(num protowire.Number, v uint64) {
		h = protowire.AppendTag(h, num, protowire.VarintType)
		h = protowire.AppendVarint(h, v)
	}
	put(fSrc, srcApp)
	put(fDest, 96)
	put(fDSrc, 1)
	put(fDDest, 1)
	put(fCheck, 3)
	put(fCmdFunc, cmdFunc)
	put(fCmdID, cmdID)
	put(fDataLen, uint64(len(pdata)))
	put(fNeedAck, 1)
	put(fSeq, seq)
	put(fVersion, 3)
	put(fPayVer, 1)

	h = protowire.AppendTag(h, fFrom, protowire.BytesType)
	h = protowire.AppendBytes(h, []byte("ios"))

	if devSN != "" {
		h = protowire.AppendTag(h, fDevSN, protowire.BytesType)
		h = protowire.AppendBytes(h, []byte(devSN))
	}

	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(out, h)
}

// BuildPhaseSet returns a 241/102 settings write selecting the phase mode.
// mode is 0 = auto, 1 = one phase, 2 = three phases.
func BuildPhaseSet(serial, gateway string, mode, seq uint64) []byte {
	settings := protowire.AppendTag(nil, 5, protowire.VarintType)
	settings = protowire.AppendVarint(settings, mode)
	return envelope(settingsPdata(serial, settings), 241, 102, seq, gateway)
}

// BuildMaxCurrentSet returns a 241/102 settings write for the maximum output
// current. raw is in tenths of an ampere, so 16 A is 160.
func BuildMaxCurrentSet(serial, gateway string, raw, seq uint64) []byte {
	settings := protowire.AppendTag(nil, 3, protowire.VarintType)
	settings = protowire.AppendVarint(settings, raw)
	return envelope(settingsPdata(serial, settings), 241, 102, seq, gateway)
}

// settingsPdata assembles descriptor plus the field 4 settings sub-message.
func settingsPdata(serial string, settings []byte) []byte {
	p := protowire.AppendTag(nil, 1, protowire.BytesType)
	p = protowire.AppendBytes(p, descriptor(serial))
	p = protowire.AppendTag(p, 4, protowire.BytesType)
	return protowire.AppendBytes(p, settings)
}

// BuildStartStop returns a 241/100 command. selector is 1 to stop, 2 to start.
func BuildStartStop(serial string, selector, seq uint64) []byte {
	p := protowire.AppendTag(nil, 1, protowire.BytesType)
	p = protowire.AppendBytes(p, descriptor(serial))
	p = protowire.AppendTag(p, 2, protowire.VarintType)
	p = protowire.AppendVarint(p, selector)
	return envelope(p, 241, 100, seq, "")
}

// BuildGetAll returns the app's protobuf "get everything" request. The wallbox
// only pushes on change, so this is what gets a first snapshot after connect.
func BuildGetAll(seq uint64) []byte {
	h := protowire.AppendTag(nil, fSrc, protowire.VarintType)
	h = protowire.AppendVarint(h, srcApp)
	h = protowire.AppendTag(h, fDest, protowire.VarintType)
	h = protowire.AppendVarint(h, srcApp)
	h = protowire.AppendTag(h, fSeq, protowire.VarintType)
	h = protowire.AppendVarint(h, seq)
	h = protowire.AppendTag(h, fFrom, protowire.BytesType)
	h = protowire.AppendBytes(h, []byte("app"))

	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(out, h)
}
