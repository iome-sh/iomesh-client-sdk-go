package iomeshclient

import (
	"net/url"
	"strings"
)

const (
	// SessionPalaceStateBound is a workspace MemoryURL chosen as the session palace.
	// Selection does not dial the host.
	SessionPalaceStateBound = "bound"
	// SessionPalaceStateNotBound means no session palace was chosen.
	// Not bound is not Connected.
	SessionPalaceStateNotBound = "not_bound"

	sessionPalacePlaceholderPrefix = "aion-mem-"
)

// SessionPalace is the workspace MemoryURL chosen for a session, or not bound.
// Not bound is not Connected. URL is empty when not bound.
// dual_write stays OFF. Not Memory GA. leftover_is_bind stays open.
type SessionPalace struct {
	// URL is the accepted workspace MemoryURL. Empty when not bound.
	URL string
	// Bound is true only when a workspace MemoryURL was accepted.
	Bound bool
	// State is SessionPalaceStateBound or SessionPalaceStateNotBound.
	// Not bound is not Connected.
	State string
}

// String returns the palace URL when bound, otherwise "not_bound".
// Not bound is not Connected.
func (p SessionPalace) String() string {
	if !p.Bound || strings.TrimSpace(p.URL) == "" {
		return SessionPalaceStateNotBound
	}
	return p.URL
}

// SessionPalaceOptions chooses a session palace from a workspace MemoryURL.
// No network I/O. Empty input is not bound.
type SessionPalaceOptions struct {
	// WorkspaceMemoryURL is the workspace MemoryURL. Empty means none was provided.
	WorkspaceMemoryURL string
	// CfgMemoryURL is the shared cfg.MemoryURL.
	// It is never copied into the session when WorkspaceMemoryURL is empty.
	// A shared config URL is not a bind.
	CfgMemoryURL string
	// HostedPalaceEnabled true returns no session palace (not bound).
	// The flag is not a bind.
	HostedPalaceEnabled bool
}

// SelectSessionPalace chooses the session palace URL from a workspace MemoryURL.
// No network I/O is performed.
//
// The workspace URL is accepted only when it is an absolute http or https URL
// with a host. The synthetic placeholder is refused: one DNS label, suffix
// .internal, prefix aion-mem- (https://aion-mem-abc.internal). A longer internal
// name such as aion-mem-abc.us-east4-a.c.iomesh-stage-001.internal is not that
// placeholder. Hosts that are only the broker or rqlite are refused. URL ports
// 4001, 4002, and 4003 are refused.
//
// HostedPalaceEnabled true is not a bind. CfgMemoryURL is not copied when no
// workspace URL was provided. Empty input is not bound. Not bound is not Connected.
// dual_write stays OFF. Not Memory GA. leftover_is_bind stays open.
func SelectSessionPalace(opts SessionPalaceOptions) SessionPalace {
	// Shared cfg.MemoryURL is not a session palace, including when it is the only URL.
	_ = opts.CfgMemoryURL
	if opts.HostedPalaceEnabled {
		return sessionPalaceNotBound()
	}
	raw := strings.TrimSpace(opts.WorkspaceMemoryURL)
	if raw == "" {
		return sessionPalaceNotBound()
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || !u.IsAbs() || !sessionPalaceHTTPScheme(u.Scheme) || strings.TrimSpace(u.Host) == "" {
		return sessionPalaceNotBound()
	}
	host := strings.ToLower(strings.Trim(u.Hostname(), "."))
	if host == "" {
		return sessionPalaceNotBound()
	}
	if sessionPalaceRefusedPort(u.Port()) || sessionPalacePlaceholder(host) || sessionPalaceBrokerOrRqlite(host) {
		return sessionPalaceNotBound()
	}
	return SessionPalace{
		URL:   raw,
		Bound: true,
		State: SessionPalaceStateBound,
	}
}

func sessionPalaceNotBound() SessionPalace {
	return SessionPalace{State: SessionPalaceStateNotBound}
}

func sessionPalaceHTTPScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

// sessionPalaceRefusedPort reports control-plane HTTP/Raft ports and the
// broker stream port. :14001 is not :4001. An empty port is not refused.
func sessionPalaceRefusedPort(port string) bool {
	switch port {
	case "4001", "4002", "4003":
		return true
	default:
		return false
	}
}

// sessionPalacePlaceholder reports the synthetic one-label host
// aion-mem-<slug>.internal. A longer internal name is not this placeholder.
func sessionPalacePlaceholder(host string) bool {
	if !strings.HasSuffix(host, ".internal") {
		return false
	}
	name := strings.TrimSuffix(host, ".internal")
	if name == "" || strings.Contains(name, ".") {
		return false
	}
	return strings.HasPrefix(name, sessionPalacePlaceholderPrefix)
}

// sessionPalaceBrokerOrRqlite reports a host that is only the broker or rqlite.
// A DNS label of broker, a hostname containing aion-broker, or a hostname
// containing rqlite is refused. A palace path or query is not a host.
func sessionPalaceBrokerOrRqlite(host string) bool {
	if strings.Contains(host, "aion-broker") || strings.Contains(host, "rqlite") {
		return true
	}
	for _, label := range strings.Split(host, ".") {
		if label == "broker" {
			return true
		}
	}
	return false
}
