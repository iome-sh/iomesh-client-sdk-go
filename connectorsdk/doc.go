// Package connectorsdk provides partner-facing helpers for third-party webhook
// adapters: HMAC verification (GitHub sha256= style), observation envelope
// normalization, and broker publish metadata.
//
// See examples/connector-sdk-template. HMAC verify + envelope normalize only —
// webhook verify ≠ OAuth; catalog listing ≠ Connected; Knowledge stays Beta.
package connectorsdk
