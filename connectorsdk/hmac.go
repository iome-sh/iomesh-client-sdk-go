package connectorsdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	// DefaultHMACPrefix is the GitHub-style HMAC-SHA256 prefix (sha256=<hex>).
	DefaultHMACPrefix = "sha256="

	// HeaderSignature256 is the default webhook HMAC header (GitHub-compatible).
	HeaderSignature256 = "X-Hub-Signature-256"
	// HeaderEvent carries the inbound event type for broker ingress.
	HeaderEvent = "X-GitHub-Event"
	// HeaderDelivery is a unique delivery id for dedupe and audit.
	HeaderDelivery = "X-GitHub-Delivery"
)

// VerifyOptions configures HMAC verification (prefix override for non-GitHub webhooks).
type VerifyOptions struct {
	Prefix string
}

var (
	ErrMissingSecret    = errors.New("connectorsdk: webhook secret required")
	ErrMissingSignature = errors.New("connectorsdk: missing signature header")
	ErrInvalidSignature = errors.New("connectorsdk: invalid signature")
)

// ComputeHMACSHA256 returns an HMAC-SHA256 digest for body. When prefix is non-empty
// the result is prefix+<hex> (GitHub style with DefaultHMACPrefix); otherwise raw hex.
func ComputeHMACSHA256(secret string, body []byte, prefix string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	digest := hex.EncodeToString(mac.Sum(nil))
	if prefix == "" {
		return digest
	}
	return prefix + digest
}

// VerifyHMAC checks signature against body using the configured prefix (GitHub sha256=
// by default).
func VerifyHMAC(secret string, body string, signature string, opts VerifyOptions) error {
	if strings.TrimSpace(secret) == "" {
		return ErrMissingSecret
	}
	sig := strings.TrimSpace(signature)
	if sig == "" {
		return ErrMissingSignature
	}
	prefix := strings.TrimSpace(opts.Prefix)
	if prefix == "" {
		prefix = DefaultHMACPrefix
	}
	expected := ComputeHMACSHA256(secret, []byte(body), prefix)
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return ErrInvalidSignature
	}
	return nil
}
