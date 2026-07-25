package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestDomainsListDecodes(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/domains" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		writeJSON(
			w,
			http.StatusOK,
			`{"domains":[{"domain":"ads.example.com","type":"deny","kind":"exact","comment":"junk","groups":[0],"enabled":true,"id":7,"date_added":100,"date_modified":200}],"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	got, err := client.Domains(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Domains error: %v", err)
	}
	if len(got) != 1 || got[0].Domain != "ads.example.com" || got[0].ID != 7 ||
		!got[0].Enabled {
		t.Errorf("got = %+v, want one enabled domain id=7", got)
	}
}

func TestAddDomainPostsBodyAndEchoes(t *testing.T) {
	// Arrange
	var gotBody domainRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/api/domains/deny/exact/ads.example.com" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		writeJSON(
			w,
			http.StatusCreated,
			`{"domains":[{"domain":"ads.example.com","type":"deny","kind":"exact","comment":"junk","groups":[1],"enabled":true,"id":9}],"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	got, err := client.AddDomain(
		context.Background(),
		DomainDeny,
		KindExact,
		"ads.example.com",
		"junk",
		[]int{1},
	)

	// Assert
	if err != nil {
		t.Fatalf("AddDomain error: %v", err)
	}
	if got.ID != 9 || got.Domain != "ads.example.com" {
		t.Errorf("got = %+v, want id=9", got)
	}
	if gotBody.Comment != "junk" || !gotBody.Enabled ||
		len(gotBody.Groups) != 1 ||
		gotBody.Groups[0] != 1 {
		t.Errorf("body = %+v, want comment=junk enabled groups=[1]", gotBody)
	}
}

func TestAddDomainRegexEscapesPath(t *testing.T) {
	// Arrange
	var gotRawPath string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawPath = r.RequestURI
		writeJSON(
			w,
			http.StatusCreated,
			`{"domains":[{"domain":"(^|\\.)ads\\.com$","type":"allow","kind":"regex","id":3}],"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	_, err := client.AddDomain(
		context.Background(),
		DomainAllow,
		KindRegex,
		`(^|\.)ads\.com$`,
		"",
		nil,
	)

	// Assert
	if err != nil {
		t.Fatalf("AddDomain error: %v", err)
	}
	want := "/api/domains/allow/regex/%28%5E%7C%5C.%29ads%5C.com$"
	if gotRawPath != want {
		t.Errorf("escaped path = %q, want %q", gotRawPath, want)
	}
}

func TestUpdateDomainPutsBody(t *testing.T) {
	// Arrange
	var gotBody domainRequest
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut ||
			r.URL.Path != "/api/domains/allow/exact/ok.example.com" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		writeJSON(
			w,
			http.StatusOK,
			`{"domains":[{"domain":"ok.example.com","enabled":false,"id":4}],"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	got, err := client.UpdateDomain(
		context.Background(),
		DomainAllow,
		KindExact,
		"ok.example.com",
		"note",
		[]int{2},
		false,
	)

	// Assert
	if err != nil {
		t.Fatalf("UpdateDomain error: %v", err)
	}
	if got.ID != 4 || got.Enabled {
		t.Errorf("got = %+v, want id=4 disabled", got)
	}
	if gotBody.Comment != "note" || gotBody.Enabled {
		t.Errorf("body = %+v, want comment=note enabled=false", gotBody)
	}
}

func TestDeleteDomainReturns204(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete ||
			r.URL.Path != "/api/domains/deny/exact/x.example.com" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	// Act
	err := client.DeleteDomain(
		context.Background(),
		DomainDeny,
		KindExact,
		"x.example.com",
	)

	// Assert
	if err != nil {
		t.Fatalf("DeleteDomain error: %v", err)
	}
}

func TestBatchDeleteDomainsSendsItemBody(t *testing.T) {
	// Arrange
	var raw []map[string]any
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/api/domains:batchDelete" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &raw)
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	// Act
	err := client.BatchDeleteDomains(context.Background(), []DomainRef{
		{Item: "a.example.com", Type: DomainDeny, Kind: KindExact},
	})

	// Assert
	if err != nil {
		t.Fatalf("BatchDeleteDomains error: %v", err)
	}
	if len(raw) != 1 || raw[0]["item"] != "a.example.com" {
		t.Errorf("body = %+v, want one entry with item key", raw)
	}
}
