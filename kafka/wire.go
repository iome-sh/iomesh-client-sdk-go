package kafka

import (
	"encoding/binary"
	"io"
)

const (
	apiProduce      int16 = 0
	apiMetadata     int16 = 3
	apiApiVersions  int16 = 18
	errNone         int16 = 0
	errUnknownTopic int16 = 3
)

type byteReader struct {
	b []byte
	i int
}

func newByteReader(b []byte) *byteReader {
	return &byteReader{b: b}
}

func (r *byteReader) remaining() int {
	return len(r.b) - r.i
}

func (r *byteReader) readInt8() (int8, error) {
	if r.remaining() < 1 {
		return 0, io.ErrUnexpectedEOF
	}
	v := int8(r.b[r.i])
	r.i++
	return v, nil
}

func (r *byteReader) readInt16() (int16, error) {
	if r.remaining() < 2 {
		return 0, io.ErrUnexpectedEOF
	}
	v := int16(binary.BigEndian.Uint16(r.b[r.i:]))
	r.i += 2
	return v, nil
}

func (r *byteReader) readInt32() (int32, error) {
	if r.remaining() < 4 {
		return 0, io.ErrUnexpectedEOF
	}
	v := int32(binary.BigEndian.Uint32(r.b[r.i:]))
	r.i += 4
	return v, nil
}

func (r *byteReader) readInt64() (int64, error) {
	if r.remaining() < 8 {
		return 0, io.ErrUnexpectedEOF
	}
	v := int64(binary.BigEndian.Uint64(r.b[r.i:]))
	r.i += 8
	return v, nil
}

func (r *byteReader) readBytes() ([]byte, error) {
	n, err := r.readInt32()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, nil
	}
	if int(n) > r.remaining() {
		return nil, io.ErrUnexpectedEOF
	}
	out := make([]byte, n)
	copy(out, r.b[r.i:r.i+int(n)])
	r.i += int(n)
	return out, nil
}

func (r *byteReader) readString() (string, error) {
	n, err := r.readInt16()
	if err != nil {
		return "", err
	}
	if n < 0 {
		return "", nil
	}
	if int(n) > r.remaining() {
		return "", io.ErrUnexpectedEOF
	}
	s := string(r.b[r.i : r.i+int(n)])
	r.i += int(n)
	return s, nil
}

type byteWriter struct {
	b []byte
}

func newByteWriter(capacity int) *byteWriter {
	return &byteWriter{b: make([]byte, 0, capacity)}
}

func (w *byteWriter) bytes() []byte {
	return w.b
}

func (w *byteWriter) writeInt8(v int8) {
	w.b = append(w.b, byte(v))
}

func (w *byteWriter) writeInt16(v int16) {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], uint16(v))
	w.b = append(w.b, buf[:]...)
}

func (w *byteWriter) writeInt32(v int32) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(v))
	w.b = append(w.b, buf[:]...)
}

func (w *byteWriter) writeInt64(v int64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(v))
	w.b = append(w.b, buf[:]...)
}

func (w *byteWriter) writeBytes(v []byte) {
	w.writeInt32(int32(len(v)))
	w.b = append(w.b, v...)
}

func (w *byteWriter) writeString(s string) {
	w.writeInt16(int16(len(s)))
	w.b = append(w.b, s...)
}

type requestHeader struct {
	apiKey        int16
	apiVersion    int16
	correlationID int32
	clientID      string
}

func parseRequestHeader(body []byte) (requestHeader, []byte, error) {
	r := newByteReader(body)
	var h requestHeader
	var err error
	if h.apiKey, err = r.readInt16(); err != nil {
		return h, nil, err
	}
	if h.apiVersion, err = r.readInt16(); err != nil {
		return h, nil, err
	}
	if h.correlationID, err = r.readInt32(); err != nil {
		return h, nil, err
	}
	if h.clientID, err = r.readString(); err != nil {
		return h, nil, err
	}
	return h, body[r.i:], nil
}

func encodeResponse(correlationID int32, payload []byte) []byte {
	w := newByteWriter(len(payload) + 8)
	w.writeInt32(int32(4 + len(payload)))
	w.writeInt32(correlationID)
	w.b = append(w.b, payload...)
	return w.bytes()
}

// decodeMessageSetV1 extracts record values from a legacy message set (magic 0/1).
func decodeMessageSetV1(messageSet []byte) ([][]byte, error) {
	r := newByteReader(messageSet)
	var values [][]byte
	for r.remaining() > 0 {
		if _, err := r.readInt64(); err != nil {
			return nil, err
		}
		msgSize, err := r.readInt32()
		if err != nil {
			return nil, err
		}
		if int(msgSize) > r.remaining() {
			return nil, io.ErrUnexpectedEOF
		}
		msgBytes := r.b[r.i : r.i+int(msgSize)]
		r.i += int(msgSize)
		msg, err := DecodeMessage(msgBytes)
		if err != nil {
			return nil, err
		}
		values = append(values, msg.Value)
	}
	return values, nil
}

func encodeMessageSetV1(value []byte) []byte {
	raw := EncodeMessage(nil, value)
	w := newByteWriter(len(raw) + 16)
	w.writeInt64(0)
	w.writeInt32(int32(len(raw)))
	w.b = append(w.b, raw...)
	return w.bytes()
}