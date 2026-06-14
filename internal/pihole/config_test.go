package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestConfigDecodesTree(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/config" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("detailed"); got != "true" {
			t.Errorf("detailed = %q, want true", got)
		}
		writeJSON(w, http.StatusOK, `{"config":{"dns":{"port":53},"dhcp":{"active":false}},"took":0.1}`)
	})
	client.setSID("S")

	// Act
	tree, err := client.Config(context.Background(), true)

	// Assert
	if err != nil {
		t.Fatalf("Config error: %v", err)
	}
	dns, ok := tree["dns"].(map[string]any)
	if !ok {
		t.Fatalf("dns subtree missing: %#v", tree)
	}
	if dns["port"].(float64) != 53 {
		t.Errorf("dns.port = %v, want 53", dns["port"])
	}
}

func TestConfigDetailedFalse(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("detailed"); got != "false" {
			t.Errorf("detailed = %q, want false", got)
		}
		writeJSON(w, http.StatusOK, `{"config":{},"took":0.0}`)
	})
	client.setSID("S")

	if _, err := client.Config(context.Background(), false); err != nil {
		t.Fatalf("Config error: %v", err)
	}
}

func TestConfigElementEscapesDottedPath(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/dns/upstreams" {
			t.Errorf("path = %q, want /api/config/dns/upstreams", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"config":{"dns":{"upstreams":["1.1.1.1"]}},"took":0.0}`)
	})
	client.setSID("S")

	// Act
	got, err := client.ConfigElement(context.Background(), "dns.upstreams")

	// Assert
	if err != nil {
		t.Fatalf("ConfigElement error: %v", err)
	}
	if got["dns"] == nil {
		t.Errorf("expected dns subtree, got %#v", got)
	}
}

func TestPatchConfigBodyAndRestartParam(t *testing.T) {
	// Arrange
	var body configPatchRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/config" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("restart"); got != "true" {
			t.Errorf("restart = %q, want true", got)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &body)
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	// Act
	err := client.PatchConfig(context.Background(), map[string]any{"dns": map[string]any{"port": 5353}}, true)

	// Assert
	if err != nil {
		t.Fatalf("PatchConfig error: %v", err)
	}
	dns, ok := body.Config["dns"].(map[string]any)
	if !ok || dns["port"].(float64) != 5353 {
		t.Errorf("patch body = %#v, want config.dns.port=5353", body.Config)
	}
}

func TestAddConfigItemPath(t *testing.T) {
	// Arrange
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	// Act
	err := client.AddConfigItem(context.Background(), "dns.upstreams", "1.1.1.1")

	// Assert
	if err != nil {
		t.Fatalf("AddConfigItem error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/api/config/dns/upstreams/1.1.1.1" {
		t.Errorf("path = %q, want /api/config/dns/upstreams/1.1.1.1", gotPath)
	}
}

func TestDeleteConfigItemPath(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteConfigItem(context.Background(), "dns.upstreams", "1.1.1.1"); err != nil {
		t.Fatalf("DeleteConfigItem error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/api/config/dns/upstreams/1.1.1.1" {
		t.Errorf("path = %q, want /api/config/dns/upstreams/1.1.1.1", gotPath)
	}
}
