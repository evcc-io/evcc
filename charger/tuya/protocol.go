package tuya

// Tuya local LAN protocol (TCP port 6668), version 3.5.
// Reference implementation: https://github.com/jasonacox/tinytuya

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
)

const (
	cmdSessKeyNegStart  = 0x03
	cmdSessKeyNegResp   = 0x04
	cmdSessKeyNegFinish = 0x05
	cmdHeartbeat        = 0x09
	cmdControlNew       = 0x0d
	cmdDpQueryNew       = 0x10
)

const (
	prefix = 0x00006699
	suffix = 0x00009966

	headerLen  = 18
	maxPayload = 4096
)

// commands sent without the version header
var noVersionHeader = []uint32{cmdDpQueryNew, cmdHeartbeat, cmdSessKeyNegStart, cmdSessKeyNegResp, cmdSessKeyNegFinish}

var versionHeader = append([]byte("3.5"), make([]byte, 12)...)

type message struct {
	seq     uint32
	cmd     uint32
	payload []byte
}

type codec struct {
	localKey   []byte
	sessionKey []byte
}

func newCodec(localKey []byte) (*codec, error) {
	if len(localKey) != 16 {
		return nil, fmt.Errorf("invalid local key length: %d", len(localKey))
	}

	return &codec{localKey: localKey}, nil
}

func (c *codec) key() []byte {
	if c.sessionKey != nil {
		return c.sessionKey
	}
	return c.localKey
}

// encode encrypts the payload into a frame
func (c *codec) encode(seq, cmd uint32, payload []byte) ([]byte, error) {
	if !slices.Contains(noVersionHeader, cmd) {
		payload = append(slices.Clone(versionHeader), payload...)
	}

	gcm, err := newGCM(c.key())
	if err != nil {
		return nil, err
	}

	b := binary.BigEndian.AppendUint32(nil, prefix)
	b = binary.BigEndian.AppendUint16(b, 0)
	b = binary.BigEndian.AppendUint32(b, seq)
	b = binary.BigEndian.AppendUint32(b, cmd)
	b = binary.BigEndian.AppendUint32(b, uint32(gcm.NonceSize()+len(payload)+gcm.Overhead()))

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	aad := slices.Clone(b[4:])
	b = append(b, nonce...)
	b = gcm.Seal(b, nonce, payload, aad)

	return binary.BigEndian.AppendUint32(b, suffix), nil
}

// decode verifies and decrypts a frame, returning the plain payload without retcode and version header
func (c *codec) decode(frame []byte) (message, error) {
	gcm, err := newGCM(c.key())
	if err != nil {
		return message{}, err
	}

	if len(frame) < headerLen+gcm.NonceSize()+gcm.Overhead()+4 {
		return message{}, errors.New("frame too short")
	}

	length := int(binary.BigEndian.Uint32(frame[14:]))
	if len(frame) != headerLen+length+4 || length < gcm.NonceSize()+gcm.Overhead() {
		return message{}, errors.New("invalid frame length")
	}

	body := frame[headerLen : headerLen+length]
	payload, err := gcm.Open(nil, body[:gcm.NonceSize()], body[gcm.NonceSize():], frame[4:headerLen])
	if err != nil {
		return message{}, fmt.Errorf("decrypt: %w", err)
	}

	return message{
		seq:     binary.BigEndian.Uint32(frame[6:]),
		cmd:     binary.BigEndian.Uint32(frame[10:]),
		payload: bytes.TrimPrefix(stripRetcode(payload), versionHeader),
	}, nil
}

// stripRetcode removes the optional 4 byte return code which devices prepend to responses but not to all pushed messages
func stripRetcode(b []byte) []byte {
	if len(b) >= 4 && binary.BigEndian.Uint32(b)&0xFFFFFF00 == 0 {
		return b[4:]
	}
	return b
}

// readFrame reads the next complete frame, skipping garbage before a valid prefix
func readFrame(r *bufio.Reader) ([]byte, error) {
	var p uint32
	for p != prefix {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p = p<<8 | uint32(b)
	}

	header := make([]byte, headerLen)
	binary.BigEndian.PutUint32(header, prefix)
	if _, err := io.ReadFull(r, header[4:]); err != nil {
		return nil, err
	}

	remaining := int(binary.BigEndian.Uint32(header[14:])) + 4
	if remaining > maxPayload {
		return nil, fmt.Errorf("frame too large: %d", remaining)
	}

	frame := make([]byte, headerLen+remaining)
	copy(frame, header)
	if _, err := io.ReadFull(r, frame[headerLen:]); err != nil {
		return nil, err
	}

	return frame, nil
}

// deriveSessionKey derives the session key from both nonces
func (c *codec) deriveSessionKey(localNonce, remoteNonce []byte) ([]byte, error) {
	xor := make([]byte, 16)
	for i := range xor {
		xor[i] = localNonce[i] ^ remoteNonce[i]
	}

	gcm, err := newGCM(c.localKey)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nil, localNonce[:gcm.NonceSize()], xor, nil)[:16], nil
}

func hmacSum(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
