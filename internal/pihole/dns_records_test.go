package pihole

import (
	"context"
	"net/http"
	"testing"
)

func TestHostRecordsParse(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/dns/hosts" {
			t.Errorf("path = %q, want /api/config/dns/hosts", r.URL.Path)
		}
		writeJSON(w, http.StatusOK,
			`{"config":{"dns":{"hosts":["192.168.1.10 nas.local","10.0.0.5 printer.local"]}},"took":0.0}`)
	})
	client.setSID("S")

	// Act
	records, err := client.HostRecords(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("HostRecords error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want 2", len(records))
	}
	if records[0] != (HostRecord{IP: "192.168.1.10", Domain: "nas.local"}) {
		t.Errorf("record[0] = %#v", records[0])
	}
	if records[1] != (HostRecord{IP: "10.0.0.5", Domain: "printer.local"}) {
		t.Errorf("record[1] = %#v", records[1])
	}
}

func TestAddHostRecordEscapesValueAsOneSegment(t *testing.T) {
	// Arrange
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	// Act
	err := client.AddHostRecord(context.Background(), "192.168.1.10", "nas.local")

	// Assert
	if err != nil {
		t.Fatalf("AddHostRecord error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	// The space in "IP domain" must be escaped as %20 within a single segment.
	if gotPath != "/api/config/dns/hosts/192.168.1.10%20nas.local" {
		t.Errorf("path = %q, want .../192.168.1.10%%20nas.local", gotPath)
	}
}

func TestDeleteHostRecordPath(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteHostRecord(context.Background(), "192.168.1.10", "nas.local"); err != nil {
		t.Fatalf("DeleteHostRecord error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/api/config/dns/hosts/192.168.1.10%20nas.local" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestCNAMERecordsParse(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/dns/cnameRecords" {
			t.Errorf("path = %q, want /api/config/dns/cnameRecords", r.URL.Path)
		}
		writeJSON(w, http.StatusOK,
			`{"config":{"dns":{"cnameRecords":["www.example.com,example.com,3600","alias.local,host.local"]}},"took":0.0}`)
	})
	client.setSID("S")

	// Act
	records, err := client.CNAMERecords(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("CNAMERecords error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len = %d, want 2", len(records))
	}
	if records[0] != (CNAMERecord{Domain: "www.example.com", Target: "example.com", TTL: 3600}) {
		t.Errorf("record[0] = %#v", records[0])
	}
	if records[1] != (CNAMERecord{Domain: "alias.local", Target: "host.local", TTL: 0}) {
		t.Errorf("record[1] = %#v", records[1])
	}
}

func TestAddCNAMERecordWithTTL(t *testing.T) {
	var gotPath string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.AddCNAMERecord(context.Background(), "www.example.com", "example.com", 3600); err != nil {
		t.Fatalf("AddCNAMERecord error: %v", err)
	}
	// Commas within the value are escaped as %2C in a single segment.
	if gotPath != "/api/config/dns/cnameRecords/www.example.com%2Cexample.com%2C3600" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestAddCNAMERecordOmitsTTLWhenNonPositive(t *testing.T) {
	var gotPath string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.AddCNAMERecord(context.Background(), "alias.local", "host.local", 0); err != nil {
		t.Fatalf("AddCNAMERecord error: %v", err)
	}
	if gotPath != "/api/config/dns/cnameRecords/alias.local%2Chost.local" {
		t.Errorf("path = %q, want ttl omitted", gotPath)
	}
}

func TestDeleteCNAMERecordPath(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteCNAMERecord(context.Background(), "www.example.com", "example.com", 3600); err != nil {
		t.Fatalf("DeleteCNAMERecord error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/api/config/dns/cnameRecords/www.example.com%2Cexample.com%2C3600" {
		t.Errorf("path = %q", gotPath)
	}
}
