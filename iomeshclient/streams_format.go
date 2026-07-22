package iomeshclient

import (
	"fmt"
	"strings"
	"time"
)

// FormatStreams renders a compact table for operator discovery (name, msgs, subjects).
// Mirrors iomesh-tui CLI style; pure helper with no network I/O.
func FormatStreams(streams []StreamInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh streams count=%d\n", len(streams))
	if len(streams) == 0 {
		b.WriteString("(no streams)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%-24s %8s %8s %8s %5s %-10s %s\n",
		"NAME", "MSGS", "FIRST", "LAST", "PART", "RETENTION", "SUBJECTS")
	for i, s := range streams {
		if i >= 50 {
			fmt.Fprintf(&b, "… (%d more)\n", len(streams)-50)
			break
		}
		subj := strings.Join(s.Subjects, ",")
		fmt.Fprintf(&b, "%-24s %8d %8d %8d %5d %-10s %s\n",
			truncateRunes(s.Name, 24),
			s.Messages, s.FirstSeq, s.LastSeq, s.Partitions,
			truncateRunes(s.Retention, 10),
			truncateRunes(subj, 48),
		)
	}
	return b.String()
}

// FormatMsg is a compact one-line view for a fetched message (seq, subject, byte length).
// Pure helper with no network I/O. Nil msg renders as "(nil)".
func FormatMsg(m *Msg) string {
	if m == nil {
		return "iomesh msg (nil)\n"
	}
	return fmt.Sprintf("iomesh msg seq=%d subject=%s bytes=%d\n", m.Seq(), m.Subject(), len(m.Data()))
}

// FormatMsgs renders multiple fetched messages for operator logs.
// Pure helper with no network I/O. nil/empty → "iomesh msgs count=0\n";
// otherwise a count header plus one FormatMsg line per element.
func FormatMsgs(msgs []*Msg) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh msgs count=%d\n", len(msgs))
	for _, m := range msgs {
		b.WriteString(FormatMsg(m))
	}
	return b.String()
}

// FormatConsumerInfo is a multi-line view for one durable consumer (operator / CLI style).
// Pure helper with no network I/O. Always emits filter_subject (empty string when unset).
func FormatConsumerInfo(info ConsumerInfo) string {
	var b strings.Builder
	b.WriteString("iomesh consumer\n")
	fmt.Fprintf(&b, "stream:          %s\n", info.Stream)
	fmt.Fprintf(&b, "name:            %s\n", info.Name)
	fmt.Fprintf(&b, "ack_floor:       %d\n", info.AckFloor)
	fmt.Fprintf(&b, "pending_count:   %d\n", info.PendingCount)
	fmt.Fprintf(&b, "filter_subject:  %s\n", info.FilterSubject)
	return b.String()
}

// FormatSubscription renders a pull subscription handle for operator logs.
// Pure helper with no network I/O.
// Nil subscription → "iomesh subscription: nil\n".
// Otherwise multi-line with stream and consumer from the Subscription handle
// (so 409 name-only create still shows identity) plus FormatConsumerInfo body
// fields (ack_floor, pending_count, filter_subject — always emitted, empty when unset).
func FormatSubscription(s *Subscription) string {
	if s == nil {
		return "iomesh subscription: nil\n"
	}
	var b strings.Builder
	b.WriteString("iomesh subscription\n")
	fmt.Fprintf(&b, "stream:          %s\n", s.stream)
	fmt.Fprintf(&b, "consumer:        %s\n", s.consumer)
	fmt.Fprintf(&b, "ack_floor:       %d\n", s.info.AckFloor)
	fmt.Fprintf(&b, "pending_count:   %d\n", s.info.PendingCount)
	fmt.Fprintf(&b, "filter_subject:  %s\n", s.info.FilterSubject)
	return b.String()
}

// FormatStreamDetail is a multi-line view for one stream (operator / CLI style).
// Pure helper with no network I/O.
func FormatStreamDetail(s StreamInfo) string {
	var b strings.Builder
	b.WriteString("iomesh stream\n")
	fmt.Fprintf(&b, "name:        %s\n", s.Name)
	if s.Description != "" {
		fmt.Fprintf(&b, "description: %s\n", s.Description)
	}
	if s.Retention != "" {
		fmt.Fprintf(&b, "retention:   %s\n", s.Retention)
	}
	if s.Partitions > 0 {
		fmt.Fprintf(&b, "partitions:  %d\n", s.Partitions)
	}
	if s.MaxMsgs != nil {
		fmt.Fprintf(&b, "max_msgs:    %d\n", *s.MaxMsgs)
	}
	if s.MaxAgeSec != nil {
		fmt.Fprintf(&b, "max_age_sec: %d\n", *s.MaxAgeSec)
	}
	fmt.Fprintf(&b, "messages:    %d\n", s.Messages)
	fmt.Fprintf(&b, "first_seq:   %d\n", s.FirstSeq)
	fmt.Fprintf(&b, "last_seq:    %d\n", s.LastSeq)
	if !s.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "created_at:  %s\n", s.CreatedAt.UTC().Format(time.RFC3339))
	}
	if len(s.Subjects) > 0 {
		b.WriteString("subjects:\n")
		for i, sub := range s.Subjects {
			if i >= 24 {
				fmt.Fprintf(&b, "  … +%d more\n", len(s.Subjects)-24)
				break
			}
			fmt.Fprintf(&b, "  - %s\n", sub)
		}
	}
	return b.String()
}
