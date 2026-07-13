package connectorsdk

import (
	"errors"
	"testing"
)

func TestVerifyHMAC(t *testing.T) {
	secret := "test-webhook-secret"
	body := `{"action":"opened","number":42}`

	validSig := ComputeHMACSHA256(secret, []byte(body), DefaultHMACPrefix)

	tests := []struct {
		name      string
		secret    string
		body      string
		signature string
		opts      VerifyOptions
		wantErr   error
	}{
		{
			name:      "valid github style",
			secret:    secret,
			body:      body,
			signature: validSig,
			wantErr:   nil,
		},
		{
			name:      "valid custom prefix",
			secret:    secret,
			body:      body,
			signature: ComputeHMACSHA256(secret, []byte(body), "v1="),
			opts:      VerifyOptions{Prefix: "v1="},
			wantErr:   nil,
		},
		{
			name:      "missing secret",
			secret:    "",
			body:      body,
			signature: validSig,
			wantErr:   ErrMissingSecret,
		},
		{
			name:      "missing signature",
			secret:    secret,
			body:      body,
			signature: "",
			wantErr:   ErrMissingSignature,
		},
		{
			name:      "invalid signature",
			secret:    secret,
			body:      body,
			signature: DefaultHMACPrefix + "deadbeef",
			wantErr:   ErrInvalidSignature,
		},
		{
			name:      "wrong body",
			secret:    secret,
			body:      `{"action":"closed"}`,
			signature: validSig,
			wantErr:   ErrInvalidSignature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyHMAC(tt.secret, tt.body, tt.signature, tt.opts)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("VerifyHMAC() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestComputeHMACSHA256(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"zen":"test"}`)

	tests := []struct {
		name   string
		prefix string
	}{
		{name: "github prefix", prefix: DefaultHMACPrefix},
		{name: "custom prefix", prefix: "v1="},
		{name: "no prefix", prefix: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeHMACSHA256(secret, body, tt.prefix)
			if tt.prefix != "" {
				if !hasPrefix(got, tt.prefix) {
					t.Fatalf("ComputeHMACSHA256() = %q, want prefix %q", got, tt.prefix)
				}
				if len(got) != len(tt.prefix)+64 {
					t.Fatalf("digest length = %d, want %d", len(got), len(tt.prefix)+64)
				}
				return
			}
			if len(got) != 64 {
				t.Fatalf("raw hex length = %d, want 64", len(got))
			}
		})
	}

	withPrefix := ComputeHMACSHA256(secret, body, DefaultHMACPrefix)
	if ComputeHMACSHA256(secret, body, DefaultHMACPrefix) != withPrefix {
		t.Fatal("ComputeHMACSHA256() not deterministic")
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
