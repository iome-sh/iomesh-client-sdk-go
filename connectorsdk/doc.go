// Package connectorsdk provides partner-facing helpers for third-party webhook
// adapters: HMAC verification (GitHub sha256= style), observation envelope
// normalization, and broker publish metadata.
//
// See examples/connector-sdk-template. Use EventsPath for org-wide ingest and
// InstallEventsPath after mesh install. HMAC webhook verify ≠ OAuth; catalog
// listing ≠ Connected; Knowledge stays Beta.
package connectorsdk
