// Connector SDK template — minimal third-party webhook adapter that
// verifies HMAC, normalizes an observation envelope, and POSTs to an
// I/O Mesh broker connector ingress using connectorsdk.
//
// Prerequisites:
//   - CONNECTOR_SDK_SECRET set (same value used to sign the sample payload)
//   - Optional: I/O Mesh broker (local foundation). Unknown connector
//     ids return 404 from the broker — normalization still runs locally.
//
// Run:
//
//	CONNECTOR_SDK_SECRET=dev-connector-sdk-secret \
//	IOMESH_URL=http://127.0.0.1:8422 \
//	IOMESH_ORG=acme-org \
//	IOMESH_TENANT=dept.engineering \
//	IOMESH_DEPARTMENT=engineering \
//	CONNECTOR_ID=acme-crm \
//	go run ./examples/connector-sdk-template
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-client-sdk-go/connectorsdk"
)

const (
	tenantHeader = "X-IOMesh-Tenant"

	eventType  = "contact.created"
	deliveryID = "acme-crm-delivery-001"
	externalID = "crm-contact-9001"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	secret := strings.TrimSpace(os.Getenv("CONNECTOR_SDK_SECRET"))
	if secret == "" {
		return fmt.Errorf("CONNECTOR_SDK_SECRET required")
	}

	baseURL := strings.TrimRight(envOr("IOMESH_URL", "http://127.0.0.1:8422"), "/")
	department := envOr("IOMESH_DEPARTMENT", "engineering")
	connectorID := envOr("CONNECTOR_ID", "acme-crm")
	org := envOr("IOMESH_ORG", "acme-org")
	tenant := envOr("IOMESH_TENANT", "dept."+department)

	eventBody := []byte(`{
"event":"contact.created",
"id":"crm-contact-9001",
"email":"partner@acme.example",
"name":"Connector SDK Template",
"account_id":"acct-42"
}`)

	// 1) Verify inbound partner webhook (GitHub sha256= style).
	sig := connectorsdk.ComputeHMACSHA256(secret, eventBody, connectorsdk.DefaultHMACPrefix)
	if err := connectorsdk.VerifyHMAC(secret, string(eventBody), sig, connectorsdk.VerifyOptions{}); err != nil {
		return fmt.Errorf("verify inbound webhook: %w", err)
	}
	log.Printf("verify OK (signature redacted; len=%d)", len(sig))

	// 2) Normalize to I/O Mesh observation envelope.
	normalizedPayload, err := connectorsdk.NormalizeEnvelope(
		connectorID,
		department,
		connectorID,
		externalID,
		eventType,
		eventBody,
	)
	if err != nil {
		return fmt.Errorf("normalize: %w", err)
	}
	subject, err := connectorsdk.SubjectForDepartment(department, connectorID)
	if err != nil {
		return fmt.Errorf("subject: %w", err)
	}
	log.Printf("normalize OK subject=%s data_product_id=%s payload_bytes=%d", subject, subject, len(normalizedPayload))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 15 * time.Second}

	eventsURL := fmt.Sprintf("%s/v10/connectors/%s/events?department=%s", baseURL, connectorID, department)
	ingressHeaders := map[string]string{
		"X-IOMesh-Org": org,
		tenantHeader:   tenant,
	}
	for k, v := range connectorsdk.PublishHeaders(connectorID, department, externalID, connectorID) {
		ingressHeaders[k] = v
	}

	log.Printf("POST %s → %s", eventType, eventsURL)
	resp, err := signedPOST(ctx, client, eventsURL, secret, eventType, deliveryID, eventBody, ingressHeaders)
	if err != nil {
		return fmt.Errorf("post event: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		var published struct {
			Status  string `json:"status"`
			Subject string `json:"subject"`
		}
		if err := json.Unmarshal(raw, &published); err != nil {
			return fmt.Errorf("decode publish response: %w", err)
		}
		log.Printf("broker OK status=%s subject=%s", published.Status, published.Subject)
		fmt.Printf("PASS connector-sdk-template: %s → %s (broker published)\n", eventType, subject)
	case http.StatusNotFound:
		log.Printf("broker note: status=404 (connector %q not registered — expected for template)", connectorID)
		fmt.Printf("PASS connector-sdk-template: verify + normalize → %s (broker 404 OK for unregistered connector)\n", subject)
	default:
		return fmt.Errorf("broker status=%d body=%s", resp.StatusCode, raw)
	}
	return nil
}

func signedPOST(ctx context.Context, client *http.Client, url, secret, eventType, deliveryID string, body []byte, extra map[string]string) (*http.Response, error) {
	sig := connectorsdk.ComputeHMACSHA256(secret, body, connectorsdk.DefaultHMACPrefix)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(connectorsdk.HeaderSignature256, sig)
	req.Header.Set(connectorsdk.HeaderEvent, eventType)
	req.Header.Set(connectorsdk.HeaderDelivery, deliveryID)
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	return client.Do(req)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
