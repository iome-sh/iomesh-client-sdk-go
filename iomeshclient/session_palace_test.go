package iomeshclient_test

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-client-sdk-go/iomeshclient"
)

func TestSelectSessionPalace(t *testing.T) {
	const (
		palace       = "https://palace.example/mcp"
		longInternal = "https://aion-mem-abc.us-east4-a.c.iomesh-stage-001.internal"
		shared       = "https://shared-cfg.example/memory"
	)

	tests := []struct {
		name  string
		opts  iomeshclient.SessionPalaceOptions
		want  string
		bound bool
	}{
		{name: "empty", opts: iomeshclient.SessionPalaceOptions{}},
		{name: "whitespace", opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: " \t "}},
		{
			name: "shared cfg.MemoryURL is not a bind",
			opts: iomeshclient.SessionPalaceOptions{CfgMemoryURL: shared},
		},
		{
			name: "HostedPalaceEnabled is not a bind",
			opts: iomeshclient.SessionPalaceOptions{
				WorkspaceMemoryURL:  palace,
				CfgMemoryURL:        shared,
				HostedPalaceEnabled: true,
			},
		},
		{
			name: "HostedPalaceEnabled with empty workspace",
			opts: iomeshclient.SessionPalaceOptions{
				CfgMemoryURL:        shared,
				HostedPalaceEnabled: true,
			},
		},
		{
			name:  "https workspace",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: palace, CfgMemoryURL: shared},
			want:  palace,
			bound: true,
		},
		{
			name:  "http workspace trimmed",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "  http://127.0.0.1:8080/mcp  "},
			want:  "http://127.0.0.1:8080/mcp",
			bound: true,
		},
		{
			name: "synthetic one-label placeholder",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://aion-mem-abc.internal"},
		},
		{
			name: "placeholder with path",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://aion-mem-abc.internal/ready"},
		},
		{
			name: "placeholder case and trailing dot",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "HTTPS://Aion-Mem-Abc.INTERNAL."},
		},
		{
			name:  "longer internal name is not the placeholder",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: longInternal},
			want:  longInternal,
			bound: true,
		},
		{
			name:  "longer internal with palace port",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://aion-mem-abc.us-east4-a.c.iomesh-stage-001.internal:8080/ready"},
			want:  "http://aion-mem-abc.us-east4-a.c.iomesh-stage-001.internal:8080/ready",
			bound: true,
		},
		{
			name: "port 4001",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://palace.example:4001/mcp"},
		},
		{
			name: "port 4002",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://palace.example:4002"},
		},
		{
			name: "port 4003 with query",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://palace.example:4003/mcp?pretty"},
		},
		{
			name:  "port 14001 is not 4001",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://palace.example:14001/mcp"},
			want:  "https://palace.example:14001/mcp",
			bound: true,
		},
		{
			name:  "userinfo is not a URL port",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://user:4001@palace.example/mcp"},
			want:  "http://user:4001@palace.example/mcp",
			bound: true,
		},
		{
			name: "host is only broker",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://broker:8080/mcp"},
		},
		{
			name: "host is only rqlite",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://rqlite/db"},
		},
		{
			name: "broker label",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://BROKER.example/mcp"},
		},
		{
			name: "aion-broker host",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://aion-broker-example.a.run.app"},
		},
		{
			name: "rqlite in hostname",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://aion-rqlite-prod.internal:8080"},
		},
		{
			name:  "rqlite in path is not the host",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://palace.example/rqlite"},
			want:  "https://palace.example/rqlite",
			bound: true,
		},
		{
			name: "relative",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "palace.example/mcp"},
		},
		{
			name: "file scheme",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "file:///tmp/palace"},
		},
		{
			name: "empty host",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http:///mcp"},
		},
		{
			name:  "aion-mem prefix without one-label internal suffix",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "https://aion-mem-abc.example/mcp"},
			want:  "https://aion-mem-abc.example/mcp",
			bound: true,
		},
		{
			name:  "ipv6 palace port",
			opts:  iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://[::1]:8080/mcp"},
			want:  "http://[::1]:8080/mcp",
			bound: true,
		},
		{
			name: "ipv6 refused port",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://[::1]:4002/mcp"},
		},
		{
			name: "longer internal name on a refused port",
			opts: iomeshclient.SessionPalaceOptions{WorkspaceMemoryURL: "http://aion-mem-abc.us-east4-a.c.iomesh-stage-001.internal:4001/ready"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := iomeshclient.SelectSessionPalace(tc.opts)
			if got.Bound != tc.bound {
				t.Fatalf("Bound=%v want %v (%+v)", got.Bound, tc.bound, got)
			}
			if got.URL != tc.want {
				t.Fatalf("URL=%q want %q", got.URL, tc.want)
			}
			if tc.bound {
				if got.State != iomeshclient.SessionPalaceStateBound {
					t.Fatalf("State=%q want bound", got.State)
				}
				if got.String() != tc.want {
					t.Fatalf("String=%q want %q", got.String(), tc.want)
				}
			} else {
				if got.State != iomeshclient.SessionPalaceStateNotBound || got.URL != "" {
					t.Fatalf("not bound = %+v", got)
				}
				if got.String() != iomeshclient.SessionPalaceStateNotBound {
					t.Fatalf("String=%q want not_bound", got.String())
				}
			}
			blob := strings.ToLower(got.URL + " " + got.State + " " + got.String())
			if strings.Contains(blob, "connected") {
				t.Fatalf("described as Connected: %+v", got)
			}
			if strings.Contains(got.URL, "shared-cfg.example") {
				t.Fatalf("copied cfg.MemoryURL: %+v", got)
			}
		})
	}
}
