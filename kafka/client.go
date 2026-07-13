package kafka

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

// Client speaks the Aion Kafka protocol subset for tests and pilots.
type Client struct {
	addr string
	seq  atomic.Int32

	mu   sync.Mutex
	conn net.Conn
}

// NewClient returns a protocol client for addr.
func NewClient(addr string) *Client {
	return &Client{addr: addr}
}

// Close closes any persistent connection.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) connOrDial(ctx context.Context) (net.Conn, error) {
	if c.conn != nil {
		return c.conn, nil
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return conn, nil
}

// Produce sends one record to topic/partition and returns the broker offset.
func (c *Client) Produce(ctx context.Context, topic string, partition int32, key, value []byte) (int64, error) {
	if c == nil || c.addr == "" {
		return 0, fmt.Errorf("kafka: client addr not configured")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.connOrDial(ctx)
	if err != nil {
		return 0, err
	}

	recordSet := encodeMessageSetV1(value)
	var body bytes.Buffer
	_ = WriteInt16(&body, apiProduce)
	_ = WriteInt16(&body, 1)
	corr := c.seq.Add(1)
	_ = WriteInt32(&body, corr)
	_ = WriteString(&body, "aion-kafka-client")
	_ = WriteInt16(&body, 1)
	_ = WriteInt32(&body, 5000)
	_ = WriteInt32(&body, 1)
	_ = WriteString(&body, topic)
	_ = WriteInt32(&body, 1)
	_ = WriteInt32(&body, partition)
	_ = WriteBytes(&body, recordSet)

	if err := WriteFrame(conn, body.Bytes()); err != nil {
		_ = c.Close()
		return 0, err
	}
	resp, err := ReadFrame(conn)
	if err != nil {
		_ = c.Close()
		return 0, err
	}
	r := newByteReader(resp)
	gotCorr, err := r.readInt32()
	if err != nil {
		return 0, err
	}
	if gotCorr != corr {
		return 0, fmt.Errorf("kafka: correlation mismatch %d != %d", gotCorr, corr)
	}
	topicCount, err := r.readInt32()
	if err != nil {
		return 0, err
	}
	for i := int32(0); i < topicCount; i++ {
		gotTopic, err := r.readString()
		if err != nil {
			return 0, err
		}
		if gotTopic != topic {
			partCount, _ := r.readInt32()
			for j := int32(0); j < partCount; j++ {
				_, _ = r.readInt32()
				_, _ = r.readInt16()
				_, _ = r.readInt64()
			}
			continue
		}
		partCount, err := r.readInt32()
		if err != nil {
			return 0, err
		}
		for j := int32(0); j < partCount; j++ {
			gotPart, err := r.readInt32()
			if err != nil {
				return 0, err
			}
			code, err := r.readInt16()
			if err != nil {
				return 0, err
			}
			offset, err := r.readInt64()
			if err != nil {
				return 0, err
			}
			if gotPart == partition {
				if code != errNone {
					return 0, fmt.Errorf("kafka: produce error code %d", code)
				}
				return offset, nil
			}
		}
	}
	_ = key
	return 0, fmt.Errorf("kafka: topic %q partition %d missing from response", topic, partition)
}
