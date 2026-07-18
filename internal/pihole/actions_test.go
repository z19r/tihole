package pihole

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestRestartDNSPostsPath(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	})
	client.setSID("S")

	if err := client.RestartDNS(context.Background()); err != nil {
		t.Fatalf("RestartDNS error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/action/restartdns" {
		t.Errorf("path = %q, want /api/action/restartdns", gotPath)
	}
}

func TestFlushLogsPostsPath(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	})
	client.setSID("S")

	if err := client.FlushLogs(context.Background()); err != nil {
		t.Fatalf("FlushLogs error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/action/flush/logs" {
		t.Errorf(
			"got %s %s, want POST /api/action/flush/logs",
			gotMethod,
			gotPath,
		)
	}
}

func TestFlushNetworkPostsPath(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	})
	client.setSID("S")

	if err := client.FlushNetwork(context.Background()); err != nil {
		t.Fatalf("FlushNetwork error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/action/flush/network" {
		t.Errorf(
			"got %s %s, want POST /api/action/flush/network",
			gotMethod,
			gotPath,
		)
	}
}

func TestDestructiveActionForbiddenSurfacesAPIError(t *testing.T) {
	// Arrange: FTL returns 403 when allow_destructive is disabled.
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusForbidden,
			`{"error":{"key":"forbidden","message":"destructive actions disabled","hint":"set allow_destructive"},"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	err := client.RestartDNS(context.Background())

	// Assert
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.Status != http.StatusForbidden || apiErr.Key != "forbidden" {
		t.Errorf("apiErr = %#v", apiErr)
	}
}
