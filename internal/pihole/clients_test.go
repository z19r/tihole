package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestClientsListDecodes(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/clients" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"clients":[{"client":"192.168.1.5","comment":"laptop","groups":[0,2],"id":3}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.Clients(context.Background())
	if err != nil {
		t.Fatalf("Clients error: %v", err)
	}
	if len(got) != 1 || got[0].Client != "192.168.1.5" || len(got[0].Groups) != 2 {
		t.Errorf("got = %+v, want client 192.168.1.5 with 2 groups", got)
	}
}

func TestAddClientPostsBody(t *testing.T) {
	var gotBody clientCreateRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/clients" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		writeJSON(w, http.StatusCreated, `{"clients":[{"client":"192.168.1.5","id":8}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.AddClient(context.Background(), "192.168.1.5", "laptop", []int{2})
	if err != nil {
		t.Fatalf("AddClient error: %v", err)
	}
	if got.ID != 8 {
		t.Errorf("got id = %d, want 8", got.ID)
	}
	if gotBody.Client != "192.168.1.5" || gotBody.Comment != "laptop" || len(gotBody.Groups) != 1 {
		t.Errorf("body = %+v, want client/comment/groups set", gotBody)
	}
}

func TestUpdateClientPutsToClientPath(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/clients/192.168.1.5" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"clients":[{"client":"192.168.1.5","id":8}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.UpdateClient(context.Background(), "192.168.1.5", "renamed", []int{1})
	if err != nil {
		t.Fatalf("UpdateClient error: %v", err)
	}
	if got.ID != 8 {
		t.Errorf("got id = %d, want 8", got.ID)
	}
}

func TestDeleteClientReturns204(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/clients/192.168.1.5" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteClient(context.Background(), "192.168.1.5"); err != nil {
		t.Fatalf("DeleteClient error: %v", err)
	}
}

func TestClientSuggestionsDecodes(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/clients/_suggestions" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"clients":[{"hwaddr":"aa:bb","macVendor":"Acme","lastQuery":123,"name":"tv","addresses":"10.0.0.9"}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.ClientSuggestions(context.Background())
	if err != nil {
		t.Fatalf("ClientSuggestions error: %v", err)
	}
	if len(got) != 1 || got[0].HWAddr != "aa:bb" || got[0].MACVendor != "Acme" {
		t.Errorf("got = %+v, want one suggestion aa:bb/Acme", got)
	}
}

func TestBatchDeleteClientsSendsItemBody(t *testing.T) {
	var raw []map[string]any
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/clients:batchDelete" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &raw)
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	err := client.BatchDeleteClients(context.Background(), []ClientRef{{Item: "192.168.1.5"}})
	if err != nil {
		t.Fatalf("BatchDeleteClients error: %v", err)
	}
	if len(raw) != 1 || raw[0]["item"] != "192.168.1.5" {
		t.Errorf("body = %+v, want [{item:192.168.1.5}]", raw)
	}
}
