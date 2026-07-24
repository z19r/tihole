package pihole

import (
	"context"
	"net/http"
	"testing"
)

func TestInfoSectionDecode(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/info/ftl" {
			t.Errorf("path = %q, want /api/info/ftl", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"ftl":{"privacy_level":0,"pid":1234},"took":0.0}`)
	})
	client.setSID("S")

	// Act
	got, err := client.Info(context.Background(), "ftl")

	// Assert
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if got["pid"].(float64) != 1234 {
		t.Errorf("pid = %v, want 1234", got["pid"])
	}
}

func TestMessageCount(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/info/messages/count" {
			t.Errorf("path = %q", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"count":3,"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.MessageCount(context.Background())
	if err != nil {
		t.Fatalf("MessageCount error: %v", err)
	}
	if got != 3 {
		t.Errorf("count = %d, want 3", got)
	}
}

func TestMessagesDecode(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/info/messages" {
			t.Errorf("path = %q", r.URL.Path)
		}
		writeJSON(w, http.StatusOK,
			`{"messages":[{"id":7,"timestamp":1700000000.5,"type":"REGEX","plain":"bad regex","html":"<b>bad</b>","url":"http://x"}],"took":0.0}`)
	})
	client.setSID("S")

	// Act
	msgs, err := client.Messages(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Messages error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len = %d, want 1", len(msgs))
	}
	want := DiagnosisMessage{ID: 7, Timestamp: 1700000000.5, Type: "REGEX", Plain: "bad regex", HTML: "<b>bad</b>", URL: "http://x"}
	if msgs[0] != want {
		t.Errorf("msg = %#v, want %#v", msgs[0], want)
	}
}

func TestDeleteMessagesCommaJoinedPath(t *testing.T) {
	// Arrange
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	// Act
	err := client.DeleteMessages(context.Background(), []int{7, 8, 9})

	// Assert
	if err != nil {
		t.Fatalf("DeleteMessages error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/api/info/messages/7,8,9" {
		t.Errorf("path = %q, want /api/info/messages/7,8,9", gotPath)
	}
}

func TestDeleteMessagesEmptyIsNoop(t *testing.T) {
	called := false
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteMessages(context.Background(), nil); err != nil {
		t.Fatalf("DeleteMessages error: %v", err)
	}
	if called {
		t.Errorf("expected no request for empty id slice")
	}
}

func TestDNSLogWithNextID(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/info/logs/dns" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("nextID"); got != "42" {
			t.Errorf("nextID = %q, want 42", got)
		}
		writeJSON(w, http.StatusOK,
			`{"log":[{"timestamp":1700000000.0,"message":"query A","prio":"info"}],"nextID":43}`)
	})
	client.setSID("S")

	// Act
	page, err := client.DNSLog(context.Background(), 42)

	// Assert
	if err != nil {
		t.Fatalf("DNSLog error: %v", err)
	}
	if page.NextID != 43 {
		t.Errorf("nextID = %d, want 43", page.NextID)
	}
	if len(page.Log) != 1 || page.Log[0].Message != "query A" || page.Log[0].PRIO != "info" {
		t.Errorf("log = %#v", page.Log)
	}
}

func TestDNSLogOmitsNextIDWhenNonPositive(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("query = %q, want empty", r.URL.RawQuery)
		}
		writeJSON(w, http.StatusOK, `{"log":[],"nextID":1}`)
	})
	client.setSID("S")

	if _, err := client.DNSLog(context.Background(), 0); err != nil {
		t.Fatalf("DNSLog error: %v", err)
	}
}
