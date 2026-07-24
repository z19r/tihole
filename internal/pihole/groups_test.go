package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestGroupsListDecodes(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/groups" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"groups":[{"name":"kids","comment":"c","enabled":true,"id":2}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.Groups(context.Background())
	if err != nil {
		t.Fatalf("Groups error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "kids" || got[0].ID != 2 {
		t.Errorf("got = %+v, want group kids id=2", got)
	}
}

func TestAddGroupPostsBody(t *testing.T) {
	var gotBody groupCreateRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/groups" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		writeJSON(w, http.StatusCreated, `{"groups":[{"name":"kids","enabled":true,"id":5}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.AddGroup(context.Background(), "kids", "family")
	if err != nil {
		t.Fatalf("AddGroup error: %v", err)
	}
	if got.ID != 5 {
		t.Errorf("got id = %d, want 5", got.ID)
	}
	if gotBody.Name != "kids" || gotBody.Comment != "family" || !gotBody.Enabled {
		t.Errorf("body = %+v, want name=kids comment=family enabled", gotBody)
	}
}

func TestUpdateGroupPutsToNamePath(t *testing.T) {
	var gotBody groupUpdateRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/groups/kids" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		writeJSON(w, http.StatusOK, `{"groups":[{"name":"kids","enabled":false,"id":5}],"took":0.0}`)
	})
	client.setSID("S")

	got, err := client.UpdateGroup(context.Background(), "kids", "off", false)
	if err != nil {
		t.Fatalf("UpdateGroup error: %v", err)
	}
	if got.Enabled {
		t.Errorf("got enabled = true, want false")
	}
	if gotBody.Comment != "off" || gotBody.Enabled {
		t.Errorf("body = %+v, want comment=off enabled=false", gotBody)
	}
}

func TestDeleteGroupReturns204(t *testing.T) {
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/groups/kids" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteGroup(context.Background(), "kids"); err != nil {
		t.Fatalf("DeleteGroup error: %v", err)
	}
}

func TestBatchDeleteGroupsSendsItemBody(t *testing.T) {
	var raw []map[string]any
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/groups:batchDelete" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &raw)
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	err := client.BatchDeleteGroups(context.Background(), []GroupRef{{Item: "kids"}})
	if err != nil {
		t.Fatalf("BatchDeleteGroups error: %v", err)
	}
	if len(raw) != 1 || raw[0]["item"] != "kids" {
		t.Errorf("body = %+v, want [{item:kids}]", raw)
	}
}
