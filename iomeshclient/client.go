// Package iomeshclient is the Go HTTP client for the I/O Mesh broker (/v1 API).
//
// Wire headers: X-IOMesh-Tenant, X-IOMesh-Org, X-IOMesh-Workspace (and related X-IOMesh-* ingress headers).
// Default User-Agent: iomesh-client-sdk-go/<Version> for operator supportability.
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

// Version is the public module version string for User-Agent and diagnostics.
// Bump when cutting a release (keep aligned with CHANGELOG / git tags).
const Version = "0.53.0"

// DefaultFetchMaxWait is the default long-poll wait for Fetch / ConsumerFetch
// when MaxWait is not set. Used as the fetchOptions.maxWait baseline.
const DefaultFetchMaxWait = 5 * time.Second

// defaultUserAgent identifies this SDK on outbound HTTP (override with WithUserAgent).
const defaultUserAgent = "iomesh-client-sdk-go/" + Version

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
	userAgent   string
}

type connectOpts struct {
	tenant      string
	org         string
	workspace   string
	bearerToken string
	userAgent   string
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
// Parity with iomesh-tui mesh client and I/O Mesh workspace header.
func WithWorkspace(workspaceID string) ConnectOpt {
	return func(o *connectOpts) {
		o.workspace = strings.TrimSpace(workspaceID)
	}
}

// WithUserAgent overrides the default User-Agent (iomesh-client-sdk-go/<Version>).
// Empty string keeps the default.
func WithUserAgent(ua string) ConnectOpt {
	return func(o *connectOpts) {
		o.userAgent = strings.TrimSpace(ua)
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
	ua := co.userAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	return &Client{
		baseURL:     raw,
		http:        httpClient,
		timeout:     timeout,
		tenant:      co.tenant,
		org:         co.org,
		workspace:   co.workspace,
		bearerToken: co.bearerToken,
		userAgent:   ua,
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

// CreateConsumerConfig for durable pull consumer create.
type CreateConsumerConfig struct {
	Stream        string
	Name          string
	FilterSubject string
	MaxDeliver    int
	AckWaitSec    int
}

// ConsumerInfo is durable consumer metadata returned on create (201).
// On 409 conflict, CreateConsumer succeeds with Stream and Name only.
type ConsumerInfo struct {
	Stream        string `json:"stream"`
	Name          string `json:"name"`
	AckFloor      uint64 `json:"ack_floor"`
	PendingCount  int    `json:"pending_count"`
	FilterSubject string `json:"filter_subject,omitempty"`
}

// CreateConsumer registers a durable pull consumer via POST /v1/streams/{stream}/consumers.
// On 201, decodes ConsumerInfo from the response body.
// On 409 conflict, treats as success and returns &ConsumerInfo{Stream, Name} (name-only, like EnsureBucket).
// Empty stream/name / nil client → error. Stream path segment is url.PathEscape'd.
func (c *Client) CreateConsumer(ctx context.Context, cfg CreateConsumerConfig) (*ConsumerInfo, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	if cfg.Stream == "" || cfg.Name == "" {
		return nil, errors.New("iomeshclient: stream and name required")
	}

	req := createConsumerRequest{
		Name:          cfg.Name,
		FilterSubject: cfg.FilterSubject,
		MaxDeliver:    cfg.MaxDeliver,
		AckWaitSec:    cfg.AckWaitSec,
	}
	path := "/v1/streams/" + url.PathEscape(cfg.Stream) + "/consumers"
	var info ConsumerInfo
	err := c.doJSON(ctx, http.MethodPost, path, req, &info)
	if err == nil {
		if info.Stream == "" {
			info.Stream = cfg.Stream // defensive when broker omits stream
		}
		if info.Name == "" {
			info.Name = cfg.Name // defensive when broker omits name
		}
		return &info, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
		return &ConsumerInfo{Stream: cfg.Stream, Name: cfg.Name}, nil
	}
	return nil, err
}

// EnsureConsumer creates the durable consumer if it does not already exist
// (CreateConsumer semantics: 409 conflict is success). Same return as CreateConsumer.
func (c *Client) EnsureConsumer(ctx context.Context, cfg CreateConsumerConfig) (*ConsumerInfo, error) {
	return c.CreateConsumer(ctx, cfg)
}

// DeleteConsumer removes a durable pull consumer via
// DELETE /v1/streams/{stream}/consumers/{name}.
// Empty stream/name / nil client → error. Non-2xx → *APIError (404 if missing).
// 204 No Content is success (doJSON handles empty body on 2xx).
// Stream and name path segments are url.PathEscape'd.
func (c *Client) DeleteConsumer(ctx context.Context, stream, name string) error {
	if c == nil {
		return errors.New("iomeshclient: nil client")
	}
	stream = strings.TrimSpace(stream)
	name = strings.TrimSpace(name)
	if stream == "" || name == "" {
		return errors.New("iomeshclient: stream and name required")
	}

	path := "/v1/streams/" + url.PathEscape(stream) + "/consumers/" + url.PathEscape(name)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// PullSubscribe registers (or reuses) a durable consumer and returns a subscription handle.
// Uses CreateConsumer: on 201, full ConsumerInfo from the body; on 409, Stream/Name only.
// Stream path segment is url.PathEscape'd.
func (c *Client) PullSubscribe(ctx context.Context, cfg PullSubscribeConfig) (*Subscription, error) {
	if cfg.Stream == "" || cfg.Consumer == "" {
		return nil, errors.New("iomeshclient: stream and consumer required")
	}

	info, err := c.CreateConsumer(ctx, CreateConsumerConfig{
		Stream:        cfg.Stream,
		Name:          cfg.Consumer,
		FilterSubject: cfg.Filter,
		MaxDeliver:    cfg.MaxDeliver,
		AckWaitSec:    cfg.AckWaitSec,
	})
	if err != nil {
		return nil, err
	}
	var subInfo ConsumerInfo
	if info != nil {
		subInfo = *info
	}

	return &Subscription{
		client:   c,
		stream:   cfg.Stream,
		consumer: cfg.Consumer,
		info:     subInfo,
	}, nil
}

// Subscription is a durable pull subscription.
type Subscription struct {
	client   *Client
	stream   string
	consumer string
	info     ConsumerInfo
}

// ConsumerInfo returns consumer metadata from create (201 full body; 409 Stream/Name only).
func (s *Subscription) ConsumerInfo() ConsumerInfo {
	if s == nil {
		return ConsumerInfo{}
	}
	return s.info
}

// ConsumerFetch pulls up to batch messages from a durable consumer without holding a
// long-lived Subscription. Returned Msg values are wired to an ephemeral Subscription
// so Msg.Ack / Msg.Nack work. Stream and consumer path segments are url.PathEscape'd.
func (c *Client) ConsumerFetch(ctx context.Context, stream, consumer string, batch int, opts ...FetchOpt) ([]*Msg, error) {
	if c == nil {
		return nil, errors.New("iomeshclient: nil client")
	}
	if stream == "" || consumer == "" {
		return nil, errors.New("iomeshclient: stream and consumer required")
	}
	if batch <= 0 {
		return nil, errors.New("iomeshclient: batch must be > 0")
	}

	fo := fetchOptions{maxWait: DefaultFetchMaxWait}
	for _, opt := range opts {
		opt(&fo)
	}

	req := fetchRequest{
		Batch:     batch,
		MaxWaitMs: int(fo.maxWait.Milliseconds()),
	}
	var resp fetchResponse
	path := fmt.Sprintf("/v1/streams/%s/consumers/%s/fetch",
		url.PathEscape(stream), url.PathEscape(consumer))
	if err := c.doJSON(ctx, http.MethodPost, path, req, &resp); err != nil {
		return nil, err
	}

	// Ephemeral sub so Msg.Ack / Msg.Nack can call Client.ConsumerAck/Nack.
	sub := &Subscription{client: c, stream: stream, consumer: consumer}
	msgs := make([]*Msg, len(resp.Messages))
	for i, m := range resp.Messages {
		payload, err := base64.StdEncoding.DecodeString(m.Payload)
		if err != nil {
			return nil, fmt.Errorf("iomeshclient: decode payload seq %d: %w", m.Seq, err)
		}
		msgs[i] = &Msg{
			sub:       sub,
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

// ConsumerAck acknowledges one or more message sequences on a durable consumer.
// Stream and consumer path segments are url.PathEscape'd.
func (c *Client) ConsumerAck(ctx context.Context, stream, consumer string, seqs ...uint64) error {
	if c == nil {
		return errors.New("iomeshclient: nil client")
	}
	if stream == "" || consumer == "" {
		return errors.New("iomeshclient: stream and consumer required")
	}
	if len(seqs) == 0 {
		return errors.New("iomeshclient: seqs required")
	}
	req := ackRequest{Seqs: seqs}
	path := fmt.Sprintf("/v1/streams/%s/consumers/%s/ack",
		url.PathEscape(stream), url.PathEscape(consumer))
	return c.doJSON(ctx, http.MethodPost, path, req, new(struct{}))
}

// ConsumerNack negatively acknowledges one or more message sequences on a durable consumer.
// Stream and consumer path segments are url.PathEscape'd.
func (c *Client) ConsumerNack(ctx context.Context, stream, consumer string, seqs ...uint64) error {
	if c == nil {
		return errors.New("iomeshclient: nil client")
	}
	if stream == "" || consumer == "" {
		return errors.New("iomeshclient: stream and consumer required")
	}
	if len(seqs) == 0 {
		return errors.New("iomeshclient: seqs required")
	}
	req := ackRequest{Seqs: seqs}
	path := fmt.Sprintf("/v1/streams/%s/consumers/%s/nack",
		url.PathEscape(stream), url.PathEscape(consumer))
	return c.doJSON(ctx, http.MethodPost, path, req, new(struct{}))
}

// FetchContext pulls up to batch messages using ctx for cancellation and deadlines.
// Prefer this over Fetch when the caller already has a request-scoped context.
// MaxWait defaults to DefaultFetchMaxWait when not set via opts.
func (s *Subscription) FetchContext(ctx context.Context, batch int, opts ...FetchOpt) ([]*Msg, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("iomeshclient: nil subscription")
	}
	msgs, err := s.client.ConsumerFetch(ctx, s.stream, s.consumer, batch, opts...)
	if err != nil {
		return nil, err
	}
	// Rebind Msg.sub to this Subscription so Ack/Nack use the caller's handle.
	for _, m := range msgs {
		if m != nil {
			m.sub = s
		}
	}
	return msgs, nil
}

// Fetch pulls up to batch messages. Equivalent to FetchContext(context.Background(), …).
// MaxWait defaults to DefaultFetchMaxWait when not set via opts.
func (s *Subscription) Fetch(batch int, opts ...FetchOpt) ([]*Msg, error) {
	return s.FetchContext(context.Background(), batch, opts...)
}

// AckContext acknowledges one or more message sequences using ctx.
func (s *Subscription) AckContext(ctx context.Context, seqs ...uint64) error {
	if s == nil || s.client == nil {
		return errors.New("iomeshclient: nil subscription")
	}
	return s.client.ConsumerAck(ctx, s.stream, s.consumer, seqs...)
}

// Ack acknowledges one or more message sequences.
// Equivalent to AckContext(context.Background(), seqs...).
func (s *Subscription) Ack(seqs ...uint64) error {
	return s.AckContext(context.Background(), seqs...)
}

// NackContext negatively acknowledges sequences using ctx.
func (s *Subscription) NackContext(ctx context.Context, seqs ...uint64) error {
	if s == nil || s.client == nil {
		return errors.New("iomeshclient: nil subscription")
	}
	return s.client.ConsumerNack(ctx, s.stream, s.consumer, seqs...)
}

// Nack negatively acknowledges sequences (optional PoC hook).
// Equivalent to NackContext(context.Background(), seqs...).
func (s *Subscription) Nack(seqs ...uint64) error {
	return s.NackContext(context.Background(), seqs...)
}

// Delete removes the durable consumer via DeleteConsumer (stream/name from subscription).
// Nil subscription / nil client → error.
func (s *Subscription) Delete(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("iomeshclient: nil subscription")
	}
	return s.client.DeleteConsumer(ctx, s.stream, s.consumer)
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

// FetchOpt configures Fetch / FetchContext / ConsumerFetch behavior.
type FetchOpt func(*fetchOptions)

// MaxWait sets the long-poll wait duration for Fetch / FetchContext / ConsumerFetch.
// When omitted, DefaultFetchMaxWait (5s) is used.
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

// applyAuthHeaders sets User-Agent, tenant / org / workspace / bearer on the request.
func (c *Client) applyAuthHeaders(req *http.Request) {
	if c == nil || req == nil {
		return
	}
	ua := c.userAgent
	if ua == "" {
		ua = defaultUserAgent
	}
	req.Header.Set("User-Agent", ua)
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
