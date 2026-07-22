package iomeshclient

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// FormatPutResult is a multi-line view for one Put outcome (operator / CLI style).
// Pure helper with no network I/O.
func FormatPutResult(r PutResult) string {
	var b strings.Builder
	b.WriteString("iomesh kv put\n")
	fmt.Fprintf(&b, "bucket:     %s\n", r.Bucket)
	fmt.Fprintf(&b, "key:        %s\n", r.Key)
	fmt.Fprintf(&b, "revision:   %d\n", r.Revision)
	return b.String()
}

// FormatBucketInfo is a multi-line view for one KV bucket (operator / CLI style).
// Pure helper with no network I/O.
// Always emits optional knobs for scrapers: history, max_bytes, ttl_seconds
// (0 / blank when unset; *int64 nil → blank value, not omitted).
func FormatBucketInfo(info BucketInfo) string {
	var b strings.Builder
	b.WriteString("iomesh kv bucket\n")
	fmt.Fprintf(&b, "name:         %s\n", info.Name)
	fmt.Fprintf(&b, "history:      %d\n", info.History)
	if info.MaxBytes != nil {
		fmt.Fprintf(&b, "max_bytes:    %d\n", *info.MaxBytes)
	} else {
		fmt.Fprintf(&b, "max_bytes:    \n")
	}
	if info.TTLSeconds != nil {
		fmt.Fprintf(&b, "ttl_seconds:  %d\n", *info.TTLSeconds)
	} else {
		fmt.Fprintf(&b, "ttl_seconds:  \n")
	}
	return b.String()
}

// FormatKVEntry is a multi-line view for one KV entry (operator / CLI style).
// Pure helper with no network I/O. Value is shown as UTF-8 text when printable;
// otherwise as a base64-friendly byte length note with a short hex preview.
func FormatKVEntry(e KVEntry) string {
	var b strings.Builder
	b.WriteString("iomesh kv entry\n")
	fmt.Fprintf(&b, "bucket:     %s\n", e.Bucket)
	fmt.Fprintf(&b, "key:        %s\n", e.Key)
	fmt.Fprintf(&b, "revision:   %d\n", e.Revision)
	if !e.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "created_at: %s\n", e.CreatedAt.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(&b, "value:      %s\n", formatKVValue(e.Value, 256))
	return b.String()
}

// FormatKVKeys renders a compact key listing for operator discovery.
// Mirrors iomesh-tui CLI style; pure helper with no network I/O.
func FormatKVKeys(bucket string, keys []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh kv keys bucket=%s count=%d\n", bucket, len(keys))
	if len(keys) == 0 {
		b.WriteString("(no keys)\n")
		return b.String()
	}
	for i, k := range keys {
		if i >= 50 {
			fmt.Fprintf(&b, "… (%d more)\n", len(keys)-50)
			break
		}
		b.WriteString(truncateRunes(k, 96))
		b.WriteByte('\n')
	}
	return b.String()
}

func formatKVValue(v []byte, maxRunes int) string {
	if len(v) == 0 {
		return `""`
	}
	if isMostlyPrintableUTF8(v) {
		s := string(v)
		if utf8.RuneCountInString(s) > maxRunes {
			return truncateRunes(s, maxRunes)
		}
		return s
	}
	preview := v
	if len(preview) > 16 {
		preview = preview[:16]
	}
	return fmt.Sprintf("<%d bytes hex=%x…>", len(v), preview)
}

func isMostlyPrintableUTF8(v []byte) bool {
	if !utf8.Valid(v) {
		return false
	}
	for _, r := range string(v) {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
