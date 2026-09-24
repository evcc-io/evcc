package tuya

// Tuya local LAN protocol (TCP port 6668), versions 3.3, 3.4 and 3.5.
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
	"hash/crc32"
	"io"
	"slices"
)

const (
	cmdSessKeyNegStart  = 0x03
	cmdSessKeyNegResp   = 0x04
	cmdSessKeyNegFinish = 0x05
	cmdControl          = 0x07
	cmdHeartbeat        = 0x09
	cmdDpQuery          = 0x0a
	cmdControlNew       = 0x0d
	cmdDpQueryNew       = 0x10
)

const (
	prefix55AA = 0x000055AA
	suffix55AA = 0x0000AA55
	prefix6699 = 0x00006699
	suffix6699 = 0x00009966

	maxPayload = 4096
)

// commands sent without the version header
var noVersionHeader = []uint32{cmdDpQuery, cmdDpQueryNew, cmdHeartbeat, cmdSessKeyNegStart, cmdSessKeyNegResp, cmdSessKeyNegFinish}

type message struct {
	seq     uint32
	cmd     uint32
	payload []byte
}

type codec struct {
	version    string
	localKey   []byte
	sessionKey []byte
}

func newCodec(version string, localKey []byte) (*codec, error) {
	if len(localKey) != 16 {
		return nil, fmt.Errorf("invalid local key length: %d", len(localKey))
	}

	switch version {
	case "3.3", "3.4", "3.5":
	default:
		return nil, fmt.Errorf("unsupported protocol version: %s", version)
	}

	return &codec{version: version, localKey: localKey}, nil
}

func (c *codec) key() []byte {
	if c.sessionKey != nil {
		return c.sessionKey
	}
	return c.localKey
}

func (c *codec) versionHeader() []byte {
	return append([]byte(c.version), make([]byte, 12)...)
}

func (c *codec) encode(seq, cmd uint32, payload []byte) ([]byte, error) {
	header := !slices.Contains(noVersionHeader, cmd)

	switch c.version {
	case "3.5":
		if header {
			payload = append(c.versionHeader(), payload...)
		}
		return pack6699(c.key(), seq, cmd, payload)

	case "3.4":
		if header {
			payload = append(c.versionHeader(), payload...)
		}
		enc, err := ecbEncrypt(c.key(), pkcs7Pad(payload))
		if err != nil {
			return nil, err
		}
		return pack55AA(c.key(), seq, cmd, enc), nil

	default:
		enc, err := ecbEncrypt(c.localKey, pkcs7Pad(payload))
		if err != nil {
			return nil, err
		}
		if header {
			enc = append(c.versionHeader(), enc...)
		}
		return pack55AA(nil, seq, cmd, enc), nil
	}
}

// decode verifies and decrypts a frame, returning the plain payload without retcode and version header
func (c *codec) decode(frame []byte) (message, error) {
	var (
		msg message
		err error
	)

	if c.version == "3.5" {
		if msg, err = unpack6699(c.key(), frame); err != nil {
			return msg, err
		}
	} else {
		var mac []byte
		if c.version == "3.4" {
			mac = c.key()
		}
		if msg, err = unpack55AA(mac, frame); err != nil {
			return msg, err
		}
	}

	msg.payload = stripRetcode(msg.payload)

	switch c.version {
	case "3.3":
		msg.payload = bytes.TrimPrefix(msg.payload, c.versionHeader())
		if msg.payload, err = ecbDecrypt(c.localKey, msg.payload); err != nil {
			return msg, err
		}
	case "3.4":
		if msg.payload, err = ecbDecrypt(c.key(), msg.payload); err != nil {
			return msg, err
		}
	}

	if c.version != "3.3" {
		msg.payload = bytes.TrimPrefix(msg.payload, c.versionHeader())
	}

	return msg, nil
}

// stripRetcode removes the optional 4 byte return code which devices prepend to responses but not to all pushed messages
func stripRetcode(b []byte) []byte {
	if len(b) >= 4 && binary.BigEndian.Uint32(b)&0xFFFFFF00 == 0 {
		return b[4:]
	}
	return b
}

func pack55AA(mac []byte, seq, cmd uint32, payload []byte) []byte {
	trailer := 4
	if mac != nil {
		trailer = sha256.Size
	}

	b := binary.BigEndian.AppendUint32(nil, prefix55AA)
	b = binary.BigEndian.AppendUint32(b, seq)
	b = binary.BigEndian.AppendUint32(b, cmd)
	b = binary.BigEndian.AppendUint32(b, uint32(len(payload)+trailer+4))
	b = append(b, payload...)

	if mac != nil {
		h := hmac.New(sha256.New, mac)
		h.Write(b)
		b = h.Sum(b)
	} else {
		b = binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(b))
	}

	return binary.BigEndian.AppendUint32(b, suffix55AA)
}

func unpack55AA(mac []byte, frame []byte) (message, error) {
	trailer := 4
	if mac != nil {
		trailer = sha256.Size
	}

	if len(frame) < 16+trailer+4 {
		return message{}, errors.New("frame too short")
	}

	length := int(binary.BigEndian.Uint32(frame[12:]))
	if len(frame) != 16+length || length < trailer+4 {
		return message{}, errors.New("invalid frame length")
	}

	data, sum := frame[:len(frame)-trailer-4], frame[len(frame)-trailer-4:len(frame)-4]

	if mac != nil {
		h := hmac.New(sha256.New, mac)
		h.Write(data)
		if !hmac.Equal(h.Sum(nil), sum) {
			return message{}, errors.New("hmac mismatch")
		}
	} else if crc32.ChecksumIEEE(data) != binary.BigEndian.Uint32(sum) {
		return message{}, errors.New("crc mismatch")
	}

	return message{
		seq:     binary.BigEndian.Uint32(frame[4:]),
		cmd:     binary.BigEndian.Uint32(frame[8:]),
		payload: slices.Clone(data[16:]),
	}, nil
}

func pack6699(key []byte, seq, cmd uint32, payload []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	b := binary.BigEndian.AppendUint32(nil, prefix6699)
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

	return binary.BigEndian.AppendUint32(b, suffix6699), nil
}

func unpack6699(key []byte, frame []byte) (message, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return message{}, err
	}

	if len(frame) < 18+gcm.NonceSize()+gcm.Overhead()+4 {
		return message{}, errors.New("frame too short")
	}

	length := int(binary.BigEndian.Uint32(frame[14:]))
	if len(frame) != 18+length+4 || length < gcm.NonceSize()+gcm.Overhead() {
		return message{}, errors.New("invalid frame length")
	}

	body := frame[18 : 18+length]
	payload, err := gcm.Open(nil, body[:gcm.NonceSize()], body[gcm.NonceSize():], frame[4:18])
	if err != nil {
		return message{}, fmt.Errorf("decrypt: %w", err)
	}

	return message{
		seq:     binary.BigEndian.Uint32(frame[6:]),
		cmd:     binary.BigEndian.Uint32(frame[10:]),
		payload: payload,
	}, nil
}

// readFrame reads the next complete frame, skipping garbage before a valid prefix
func readFrame(r *bufio.Reader) ([]byte, error) {
	var prefix uint32
	for prefix != prefix55AA && prefix != prefix6699 {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		prefix = prefix<<8 | uint32(b)
	}

	var header []byte
	var remaining int

	if prefix == prefix55AA {
		header = make([]byte, 16)
		if _, err := io.ReadFull(r, header[4:]); err != nil {
			return nil, err
		}
		remaining = int(binary.BigEndian.Uint32(header[12:]))
	} else {
		header = make([]byte, 18)
		if _, err := io.ReadFull(r, header[4:]); err != nil {
			return nil, err
		}
		remaining = int(binary.BigEndian.Uint32(header[14:])) + 4
	}

	if remaining > maxPayload {
		return nil, fmt.Errorf("frame too large: %d", remaining)
	}

	binary.BigEndian.PutUint32(header, prefix)
	frame := append(header, make([]byte, remaining)...)
	if _, err := io.ReadFull(r, frame[len(header):]); err != nil {
		return nil, err
	}

	return frame, nil
}

// deriveSessionKey derives the 3.4/3.5 session key from both nonces
func (c *codec) deriveSessionKey(localNonce, remoteNonce []byte) ([]byte, error) {
	xor := make([]byte, 16)
	for i := range xor {
		xor[i] = localNonce[i] ^ remoteNonce[i]
	}

	if c.version == "3.4" {
		return ecbEncrypt(c.localKey, xor)
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

func ecbEncrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, errors.New("invalid block size")
	}

	res := make([]byte, len(data))
	for i := 0; i < len(data); i += aes.BlockSize {
		block.Encrypt(res[i:], data[i:])
	}
	return res, nil
}

func ecbDecrypt(key, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("invalid ciphertext length: %d", len(data))
	}

	res := make([]byte, len(data))
	for i := 0; i < len(data); i += aes.BlockSize {
		block.Decrypt(res[i:], data[i:])
	}
	return pkcs7Unpad(res)
}

func pkcs7Pad(data []byte) []byte {
	n := aes.BlockSize - len(data)%aes.BlockSize
	return append(slices.Clone(data), bytes.Repeat([]byte{byte(n)}, n)...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	n := int(data[len(data)-1])
	if n < 1 || n > aes.BlockSize || n > len(data) {
		return nil, errors.New("invalid padding")
	}
	return data[:len(data)-n], nil
}
