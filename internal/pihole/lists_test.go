package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListsSendsTypeParam(t *testing.T) {
	var gotType string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/lists" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		gotType = r.URL.Query().Get("type")
		writeJSON(w, http.StatusOK, `{"lists":[{"address":"https://x/list.txt","type":"block","enabled":true,"id":4}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.Lists(context.Background(), ListBlock)
	if err != nil {
		t.Fatalf("Lists error: %v", err)
	}
	if gotType != "block" {
		t.Errorf("type param = %q, want block", gotType)
	}
	if len(got) != 1 || got[0].Address != "https://x/list.txt" || got[0].ID != 4 {
		t.Errorf("got = %+v, want one list id=4", got)
	}
}

func TestAddListPostsBody(t *testing.T) {
	var gotBody listCreateRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/lists" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		writeJSON(w, http.StatusCreated, `{"lists":[{"address":"https://x/list.txt","type":"allow","id":6}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.AddList(context.Background(), "https://x/list.txt", ListAllow, "mine", []int{0})
	if err != nil {
		t.Fatalf("AddList error: %v", err)
	}
	if got.ID != 6 {
		t.Errorf("got id = %d, want 6", got.ID)
	}
	if gotBody.Address != "https://x/list.txt" || gotBody.Type != ListAllow || !gotBody.Enabled {
		t.Errorf("body = %+v, want address/type=allow/enabled", gotBody)
	}
}

func TestUpdateListPutsWithTypeParam(t *testing.T) {
	var gotType string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/lists/https://x/list.txt" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		gotType = r.URL.Query().Get("type")
		writeJSON(w, http.StatusOK, `{"lists":[{"address":"https://x/list.txt","type":"block","enabled":false,"id":6}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.UpdateList(context.Background(), "https://x/list.txt", ListBlock, "c", nil, false)
	if err != nil {
		t.Fatalf("UpdateList error: %v", err)
	}
	if gotType != "block" {
		t.Errorf("type param = %q, want block", gotType)
	}
	if got.Enabled {
		t.Errorf("got enabled = true, want false")
	}
}

func TestDeleteListReturns204(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/lists/https://x/list.txt" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("type") != "block" {
			t.Errorf("type param = %q, want block", r.URL.Query().Get("type"))
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteList(context.Background(), "https://x/list.txt", ListBlock); err != nil {
		t.Fatalf("DeleteList error: %v", err)
	}
}

func TestBatchDeleteListsSendsItemBody(t *testing.T) {
	var raw []map[string]any
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/lists:batchDelete" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &raw)
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	err := client.BatchDeleteLists(context.Background(), []ListRef{{Item: "https://x/list.txt", Type: ListBlock}})
	if err != nil {
		t.Fatalf("BatchDeleteLists error: %v", err)
	}
	if len(raw) != 1 || raw[0]["item"] != "https://x/list.txt" || raw[0]["type"] != "block" {
		t.Errorf("body = %+v, want [{item,type:block}]", raw)
	}
}
