package connectorsdk

import (
	"fmt"
	"net/url"
	"strings"
)

// EventsPath is the org-wide connector ingress path:
// /v10/connectors/{id}/events
//
// Honesty: this is the catalog/org-wide URL. Mesh-install ingest uses
// InstallEventsPath. Webhook verify ≠ OAuth; catalog listing ≠ Connected;
// Knowledge stays Beta. An org-wide POST without a captured/env secret may
// 503 — that is serving honesty, not Connected.
func EventsPath(connectorID string) (string, error) {
	id, err := requireConnectorID(connectorID)
	if err != nil {
		return "", err
	}
	return "/v10/connectors/" + url.PathEscape(id) + "/events", nil
}

// InstallEventsPath is the mesh-install connector ingress path:
// /v10/connectors/{id}/i/{install_id}/events
//
// Use this after a mesh install (Notion and other installed connectors).
// Do not present EventsPath as the install ingest URL.
func InstallEventsPath(connectorID, installID string) (string, error) {
	id, err := requireConnectorID(connectorID)
	if err != nil {
		return "", err
	}
	install := strings.TrimSpace(installID)
	if install == "" {
		return "", fmt.Errorf("connectorsdk: install id required")
	}
	return "/v10/connectors/" + url.PathEscape(id) + "/i/" + url.PathEscape(install) + "/events", nil
}

// EventsURL joins an absolute http(s) broker base with EventsPath.
func EventsURL(baseURL, connectorID string) (string, error) {
	path, err := EventsPath(connectorID)
	if err != nil {
		return "", err
	}
	return joinBrokerURL(baseURL, path)
}

// InstallEventsURL joins an absolute http(s) broker base with InstallEventsPath.
func InstallEventsURL(baseURL, connectorID, installID string) (string, error) {
	path, err := InstallEventsPath(connectorID, installID)
	if err != nil {
		return "", err
	}
	return joinBrokerURL(baseURL, path)
}

func requireConnectorID(connectorID string) (string, error) {
	id := strings.TrimSpace(connectorID)
	if id == "" {
		return "", fmt.Errorf("connectorsdk: connector id required")
	}
	return id, nil
}

func joinBrokerURL(baseURL, path string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("connectorsdk: base URL required")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("connectorsdk: invalid base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("connectorsdk: unsupported URL scheme %q (want http or https)", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("connectorsdk: URL host required")
	}
	if u.User != nil {
		return "", fmt.Errorf("connectorsdk: URL must not contain userinfo")
	}
	return base + path, nil
}
