// Package iomeshclient is the Go HTTP client for the I/O Mesh broker (/v1 API).
//
// Wire headers: X-IOMesh-Tenant, X-IOMesh-Org (and related X-IOMesh-* ingress headers).
package iomeshclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Options configures the I/O Mesh HTTP client.
type Options struct {
	URL            string
	HTTPClient     *http.Client
	RequestTimeout time.Duration
}

const (
	tenantHeader    = "X-IOMesh-Tenant"
	orgHeader       = "X-IOMesh-Org"
	workspaceHeader = "X-IOMesh-Workspace"
)

// Client talks to an I/O Mesh broker over HTTP.
type Client struct {
	baseURL     string
	http        *http.Client
	timeout     time.Duration
	tenant      string
	org         string
	workspace   string
	bearerToken string
}

type connectOpts struct {
	tenant      string
	org         string
	workspace   string
	bearerToken string
}

// ConnectOpt configures optional client connection settings.
type ConnectOpt func(*connectOpts)

// WithTenant sets X-IOMesh-Tenant on all HTTP requests.
func WithTenant(tenant string) ConnectOpt {
	return func(o *connectOpts) {
		o.tenant = strings.TrimSpace(tenant)
	}
}

// WithBearerToken sets Authorization: Bearer on all HTTP requests.
func WithBearerToken(token string) ConnectOpt {
	return func(o *connectOpts) {
		o.bearerToken = strings.TrimSpace(token)
	}
}

// WithOrg sets X-IOMesh-Org on all HTTP requests (PlanGate metering attribution).
func WithOrg(orgID string) ConnectOpt {
	return func(o *connectOpts) {
		o.org = strings.TrimSpace(orgID)
	}
}

// WithWorkspace sets X-IOMesh-Workspace on all HTTP requests (multi-tenant metering / entitlements).
// Parity with iomesh-tui mesh client and aion WorkspaceHeader.
func WithWorkspace(workspaceID string) ConnectOpt {
	return func(o *connectOpts) {
		o.workspace = strings.TrimSpace(workspaceID)
	}
}

func applyConnectOpts(opts []ConnectOpt) connectOpts {
	var co connectOpts
	for _, opt := range opts {
		opt(&co)
	}
	return co
}

// Connect returns a client for the broker at base.URL. No network I/O is performed.
// URL must be absolute http or https (file:// and other schemes are rejected).
func Connect(base Options, opts ...ConnectOpt) (*Client, error) {
	raw := strings.TrimSpace(base.URL)
	if raw == "" {
		return nil, errors.New("iomeshclient: URL required")
	}
	raw = strings.TrimRight(raw, "/")
	if err := validateBrokerURL(raw); err != nil {
		return nil, err
	}

	httpClient := base.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	timeout := base.RequestTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	co := applyConnectOpts(opts)

	return &Client{
		baseURL:     raw,
		http:        httpClient,
		timeout:     timeout,
		tenant:      co.tenant,
		org:         co.org,
		workspace:   co.workspace,
		bearerToken: co.bearerToken,
	}, nil
}

// validateBrokerURL enforces absolute http(s) endpoints for the mesh broker.
// Loopback http is allowed for local development.
func validateBrokerURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("iomeshclient: invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("iomeshclient: unsupported URL scheme %q (want http or https)", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("iomeshclient: URL host required")
	}
	// Reject embedded credentials in the URL (prefer WithBearerToken).
	if u.User != nil {
		return errors.New("iomeshclient: URL must not contain userinfo; use WithBearerToken")
	}
	return nil
}

// PubAck is the response from a successful publish.
type PubAck struct {
	Stream    string
	Seq       uint64
	Subject   string
	Partition int
	Timestamp time.Time
}

// Publish appends a message to stream.
func (c *Client) Publish(ctx context.Context, stream, subject string, payload []byte, opts ...PublishOpt) (*PubAck, error) {
	po := applyPublishOpts(opts)

	body := publishRequest{
		Subject: subject,
		Payload: base64.StdEncoding.EncodeToString(payload),
	}
	if po.partitionKey != "" {
		body.PartitionKey = po.partitionKey
	}
	if po.setPartition {
		body.Partition = po.partition
	}

	var resp pubAckResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/streams/"+stream+"/publish", body, &resp); err != nil {
		return nil, err
	}
	return &PubAck{
		Stream:    resp.Stream,
		Seq:       resp.Seq,
		Subject:   resp.Subject,
		Partition: resp.Partition,
		Timestamp: resp.Timestamp,
	}, nil
}

// PullSubscribeConfig selects a durable pull consumer.
type PullSubscribeConfig struct {
	Stream     string
	Consumer   string
	Filter     string
	MaxDeliver int
	AckWaitSec int
}

// PullSubscribe registers (or reuses) a durable consumer and returns a subscription handle.
func (c *Client) PullSubscribe(ctx context.Context, cfg PullSubscribeConfig) (*Subscription, error) {
	if cfg.Stream == "" || cfg.Consumer == "" {
		return nil, errors.New("iomeshclient: stream and consumer required")
	}

	req := createConsumerRequest{
		Name:          cfg.Consumer,
		FilterSubject: cfg.Filter,
		MaxDeliver:    cfg.MaxDeliver,
		AckWaitSec:    cfg.AckWaitSec,
	}
	path := "/v1/streams/" + cfg.Stream + "/consumers"
	if err := c.doJSON(ctx, http.MethodPost, path, req, new(struct{})); err != nil {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
			return nil, err
		}
	}

	return &Subscription{
		client:   c,
		stream:   cfg.Stream,
		consumer: cfg.Consumer,
	}, nil
}

// Subscription is a durable pull subscription.
type Subscription struct {
	client   *Client
	stream   string
	consumer string
}

// Fetch pulls up to batch messages.
func (s *Subscription) Fetch(batch int, opts ...FetchOpt) ([]*Msg, error) {
	if batch <= 0 {
		return nil, errors.New("iomeshclient: batch must be > 0")
	}

	fo := fetchOptions{maxWait: 5 * time.Second}
	for _, opt := range opts {
		opt(&fo)
	}

	req := fetchRequest{
		Batch:     batch,
		MaxWaitMs: int(fo.maxWait.Milliseconds()),
	}
	var resp fetchResponse
	path := fmt.Sprintf("/v1/streams/%s/consumers/%s/fetch", s.stream, s.consumer)
	if err := s.client.doJSON(context.Background(), http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}

	msgs := make([]*Msg, len(resp.Messages))
	for i, m := range resp.Messages {
		payload, err := base64.StdEncoding.DecodeString(m.Payload)
		if err != nil {
			return nil, fmt.Errorf("iomeshclient: decode payload seq %d: %w", m.Seq, err)
		}
		msgs[i] = &Msg{
			sub:       s,
			stream:    m.Stream,
			seq:       m.Seq,
			subject:   m.Subject,
			partition: m.Partition,
			data:      payload,
			headers:   m.Headers,
		}
	}
	return msgs, nil
}

// Ack acknowledges one or more message sequences.
func (s *Subscription) Ack(seqs ...uint64) error {
	if len(seqs) == 0 {
		return errors.New("iomeshclient: seqs required")
	}
	req := ackRequest{Seqs: seqs}
	path := fmt.Sprintf("/v1/streams/%s/consumers/%s/ack", s.stream, s.consumer)
	return s.client.doJSON(context.Background(), http.MethodPost, path, req, new(struct{}))
}

// Nack negatively acknowledges sequences (optional PoC hook).
func (s *Subscription) Nack(seqs ...uint64) error {
	if len(seqs) == 0 {
		return errors.New("iomeshclient: seqs required")
	}
	req := ackRequest{Seqs: seqs}
	path := fmt.Sprintf("/v1/streams/%s/consumers/%s/nack", s.stream, s.consumer)
	return s.client.doJSON(context.Background(), http.MethodPost, path, req, new(struct{}))
}

// Unsubscribe is a no-op for durable consumers in the PoC HTTP client.
func (s *Subscription) Unsubscribe() error {
	return nil
}

// Msg is a fetched stream message.
type Msg struct {
	sub       *Subscription
	stream    string
	seq       uint64
	subject   string
	partition int
	data      []byte
	headers   map[string]string
}

// Seq returns the stream sequence number.
func (m *Msg) Seq() uint64 { return m.seq }

// Subject returns the message subject.
func (m *Msg) Subject() string { return m.subject }

// Partition returns the stream partition index for this message.
func (m *Msg) Partition() int { return m.partition }

// Data returns the message payload.
func (m *Msg) Data() []byte { return m.data }

// Headers returns message metadata.
func (m *Msg) Headers() map[string]string { return m.headers }

// Ack acknowledges this message.
func (m *Msg) Ack() error { return m.sub.Ack(m.seq) }

// Nack negatively acknowledges this message.
func (m *Msg) Nack() error { return m.sub.Nack(m.seq) }

type fetchOptions struct {
	maxWait time.Duration
}

// FetchOpt configures Fetch behavior.
type FetchOpt func(*fetchOptions)

// MaxWait sets the long-poll wait duration for Fetch.
func MaxWait(d time.Duration) FetchOpt {
	return func(o *fetchOptions) {
		o.maxWait = d
	}
}

type publishOpts struct {
	partition    int
	partitionKey string
	setPartition bool
}

// PublishOpt configures optional publish routing fields.
type PublishOpt func(*publishOpts)

// WithPartitionKey routes the message by hashing key across stream partitions.
func WithPartitionKey(key string) PublishOpt {
	return func(o *publishOpts) {
		o.partitionKey = key
	}
}

// WithPartition publishes to an explicit partition index.
func WithPartition(p int) PublishOpt {
	return func(o *publishOpts) {
		o.partition = p
		o.setPartition = true
	}
}

func applyPublishOpts(opts []PublishOpt) publishOpts {
	var po publishOpts
	for _, opt := range opts {
		opt(&po)
	}
	return po
}

type publishRequest struct {
	Subject      string            `json:"subject"`
	Payload      string            `json:"payload"`
	Headers      map[string]string `json:"headers,omitempty"`
	Partition    int               `json:"partition,omitempty"`
	PartitionKey string            `json:"partition_key,omitempty"`
}

type pubAckResponse struct {
	Stream    string    `json:"stream"`
	Seq       uint64    `json:"seq"`
	Subject   string    `json:"subject"`
	Partition int       `json:"partition,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type createConsumerRequest struct {
	Name          string `json:"name"`
	FilterSubject string `json:"filter_subject,omitempty"`
	AckWaitSec    int    `json:"ack_wait_sec,omitempty"`
	MaxDeliver    int    `json:"max_deliver,omitempty"`
}

type fetchRequest struct {
	Batch     int `json:"batch"`
	MaxWaitMs int `json:"max_wait_ms"`
}

type fetchMessage struct {
	Stream    string            `json:"stream"`
	Seq       uint64            `json:"seq"`
	Subject   string            `json:"subject"`
	Partition int               `json:"partition,omitempty"`
	Payload   string            `json:"payload"`
	Headers   map[string]string `json:"headers"`
	Timestamp time.Time         `json:"timestamp"`
}

type fetchResponse struct {
	Messages []fetchMessage `json:"messages"`
}

type ackRequest struct {
	Seqs []uint64 `json:"seqs"`
}

// APIError is returned when the broker responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("iomeshclient: HTTP %d: %s", e.StatusCode, e.Body)
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, respBody any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.applyAuthHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(raw))}
	}

	if respBody == nil {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, respBody)
}

// applyAuthHeaders sets tenant / org / workspace / bearer on the request.
func (c *Client) applyAuthHeaders(req *http.Request) {
	if c == nil || req == nil {
		return
	}
	if c.tenant != "" {
		req.Header.Set(tenantHeader, c.tenant)
	}
	if c.org != "" {
		req.Header.Set(orgHeader, c.org)
	}
	if c.workspace != "" {
		req.Header.Set(workspaceHeader, c.workspace)
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
}
