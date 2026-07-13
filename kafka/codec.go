package kafka

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

var crc32CTable = crc32.MakeTable(crc32.Castagnoli)

// RequestHeader is the common Kafka request header after the frame size.
type RequestHeader struct {
	APIKey        int16
	APIVersion    int16
	CorrelationID int32
	ClientID      string
}

// ReadInt8 reads one signed byte.
func ReadInt8(r io.Reader) (int8, error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int8(buf[0]), nil
}

// WriteInt8 writes one signed byte.
func WriteInt8(w io.Writer, v int8) error {
	return writeFull(w, []byte{byte(v)})
}

// ReadInt16 reads a big-endian int16.
func ReadInt16(r io.Reader) (int16, error) {
	var buf [2]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(buf[:])), nil
}

// WriteInt16 writes a big-endian int16.
func WriteInt16(w io.Writer, v int16) error {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], uint16(v))
	return writeFull(w, buf[:])
}

// ReadInt32 reads a big-endian int32.
func ReadInt32(r io.Reader) (int32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf[:])), nil
}

// WriteInt32 writes a big-endian int32.
func WriteInt32(w io.Writer, v int32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(v))
	return writeFull(w, buf[:])
}

// ReadInt64 reads a big-endian int64.
func ReadInt64(r io.Reader) (int64, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf[:])), nil
}

// WriteInt64 writes a big-endian int64.
func WriteInt64(w io.Writer, v int64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(v))
	return writeFull(w, buf[:])
}

// ReadString reads a Kafka STRING (int16 length prefix).
func ReadString(r io.Reader) (string, error) {
	n, err := ReadInt16(r)
	if err != nil {
		return "", err
	}
	if n < 0 {
		return "", fmt.Errorf("invalid string length %d", n)
	}
	if n == 0 {
		return "", nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// WriteString writes a Kafka STRING (int16 length prefix).
func WriteString(w io.Writer, s string) error {
	if len(s) > int(^uint16(0)>>1) {
		return fmt.Errorf("string too long")
	}
	if err := WriteInt16(w, int16(len(s))); err != nil {
		return err
	}
	if len(s) == 0 {
		return nil
	}
	return writeFull(w, []byte(s))
}

// ReadBytes reads a Kafka BYTES field (int32 length prefix, -1 is null).
func ReadBytes(r io.Reader) ([]byte, error) {
	n, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// WriteBytes writes a Kafka BYTES field (int32 length prefix, nil is -1).
func WriteBytes(w io.Writer, b []byte) error {
	if b == nil {
		return WriteInt32(w, -1)
	}
	if err := WriteInt32(w, int32(len(b))); err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return writeFull(w, b)
}

// ReadRequestHeader parses api_key, api_version, correlation_id, and client_id.
func ReadRequestHeader(r io.Reader) (RequestHeader, error) {
	var hdr RequestHeader
	var err error
	if hdr.APIKey, err = ReadInt16(r); err != nil {
		return hdr, err
	}
	if hdr.APIVersion, err = ReadInt16(r); err != nil {
		return hdr, err
	}
	if hdr.CorrelationID, err = ReadInt32(r); err != nil {
		return hdr, err
	}
	if hdr.ClientID, err = ReadString(r); err != nil {
		return hdr, err
	}
	return hdr, nil
}

// ReadFrame reads a 4-byte length-prefixed Kafka frame.
func ReadFrame(r io.Reader) ([]byte, error) {
	size, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	if size < 0 {
		return nil, fmt.Errorf("invalid frame size %d", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// WriteFrame writes payload with a leading 4-byte size.
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > int(^uint32(0)>>1) {
		return fmt.Errorf("frame too large")
	}
	if err := WriteInt32(w, int32(len(payload))); err != nil {
		return err
	}
	return writeFull(w, payload)
}

// Message is a legacy Kafka message (magic 0 or 1).
type Message struct {
	Magic      int8
	Attributes int8
	Timestamp  int64
	Key        []byte
	Value      []byte
}

// DecodeMessage parses one message and verifies CRC when present.
func DecodeMessage(data []byte) (Message, error) {
	if len(data) < 14 {
		return Message{}, fmt.Errorf("message too short")
	}
	storedCRC := binary.BigEndian.Uint32(data[0:4])
	magic := int8(data[4])
	attributes := int8(data[5])
	off := 6

	var timestamp int64
	switch magic {
	case 0:
		timestamp = 0
	case 1:
		if len(data) < off+8 {
			return Message{}, fmt.Errorf("message v1 too short")
		}
		timestamp = int64(binary.BigEndian.Uint64(data[off : off+8]))
		off += 8
	default:
		return Message{}, fmt.Errorf("unsupported magic byte %d", magic)
	}

	key, n, err := readNullableBytesAt(data, off)
	if err != nil {
		return Message{}, err
	}
	off += n

	value, n, err := readNullableBytesAt(data, off)
	if err != nil {
		return Message{}, err
	}
	off += n
	if off != len(data) {
		return Message{}, fmt.Errorf("trailing message bytes")
	}

	crcData := data[4:len(data)]
	if crc32.Checksum(crcData, crc32CTable) != storedCRC {
		return Message{}, fmt.Errorf("crc mismatch")
	}

	return Message{
		Magic:      magic,
		Attributes: attributes,
		Timestamp:  timestamp,
		Key:        key,
		Value:      value,
	}, nil
}

// EncodeMessage builds a legacy Kafka message (magic 1, no compression).
func EncodeMessage(key, value []byte) []byte {
	body := make([]byte, 0, 32+len(key)+len(value))
	body = append(body, 1) // magic
	body = append(body, 0) // attributes
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], 0)
	body = append(body, ts[:]...)
	body = appendNullableBytes(body, key)
	body = appendNullableBytes(body, value)

	out := make([]byte, 4+len(body))
	crc := crc32.Checksum(body, crc32CTable)
	binary.BigEndian.PutUint32(out[0:4], crc)
	copy(out[4:], body)
	return out
}

func readNullableBytesAt(data []byte, off int) ([]byte, int, error) {
	if off+4 > len(data) {
		return nil, 0, fmt.Errorf("bytes length underflow")
	}
	n := int32(binary.BigEndian.Uint32(data[off : off+4]))
	if n < 0 {
		return nil, 4, nil
	}
	if off+4+int(n) > len(data) {
		return nil, 0, fmt.Errorf("bytes payload underflow")
	}
	if n == 0 {
		return []byte{}, 4, nil
	}
	buf := make([]byte, n)
	copy(buf, data[off+4:off+4+int(n)])
	return buf, 4 + int(n), nil
}

func appendNullableBytes(dst []byte, b []byte) []byte {
	if b == nil {
		return append(dst, 0xff, 0xff, 0xff, 0xff)
	}
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(b)))
	dst = append(dst, n[:]...)
	return append(dst, b...)
}

func writeFull(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}
