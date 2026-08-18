package connectorsdk

import "testing"

func TestEventsPath(t *testing.T) {
	got, err := EventsPath("acme-crm")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/v10/connectors/acme-crm/events" {
		t.Fatalf("EventsPath = %q", got)
	}
	if _, err := EventsPath("  "); err == nil {
		t.Fatal("expected connector id required")
	}
	escaped, err := EventsPath("acme/crm")
	if err != nil {
		t.Fatal(err)
	}
	if escaped != "/v10/connectors/acme%2Fcrm/events" {
		t.Fatalf("EventsPath escape = %q", escaped)
	}
}

func TestInstallEventsPath(t *testing.T) {
	got, err := InstallEventsPath("notion", "inst-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/v10/connectors/notion/i/inst-1/events" {
		t.Fatalf("InstallEventsPath = %q", got)
	}
	if _, err := InstallEventsPath("notion", ""); err == nil {
		t.Fatal("expected install id required")
	}
	if _, err := InstallEventsPath("", "inst-1"); err == nil {
		t.Fatal("expected connector id required")
	}
}

func TestEventsURLAndInstallEventsURL(t *testing.T) {
	u, err := EventsURL("http://127.0.0.1:8422/", "acme-crm")
	if err != nil {
		t.Fatal(err)
	}
	if u != "http://127.0.0.1:8422/v10/connectors/acme-crm/events" {
		t.Fatalf("EventsURL = %q", u)
	}
	u, err = InstallEventsURL("https://broker.example", "notion", "inst-1")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://broker.example/v10/connectors/notion/i/inst-1/events" {
		t.Fatalf("InstallEventsURL = %q", u)
	}
	if _, err := EventsURL("file:///tmp", "acme-crm"); err == nil {
		t.Fatal("expected scheme reject")
	}
	if _, err := EventsURL("http://user:pass@127.0.0.1:8422", "acme-crm"); err == nil {
		t.Fatal("expected userinfo reject")
	}
}
