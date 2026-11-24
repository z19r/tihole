package dashboard

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/z19r/tihole/internal/pihole"
)

// summaryClient is a real client against a fake Pi-hole whose summary
// reports total queries.
func summaryClient(t *testing.T, total string) *pihole.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/auth" {
				_, _ = w.Write(
					[]byte(
						`{"session":{"valid":true,"sid":"s","csrf":"c","validity":1800}}`,
					),
				)
				return
			}
			_, _ = w.Write([]byte(`{"queries":{"total":` + total + `}}`))
		}))
	t.Cleanup(srv.Close)
	return pihole.New(srv.URL, "pw", pihole.WithHTTPClient(srv.Client()))
}

func TestFetchAllUsesSource(t *testing.T) {
	m := newTestModel()
	m.ctx.Fleet = pihole.NewFleet([]pihole.Member{
		{Name: "a", Client: summaryClient(t, "4000")},
		{Name: "b", Client: summaryClient(t, "242")},
	}, nil)
	m.ctx.Combined = true
	m.epoch = 1
	m.baseCtx = context.Background()

	msg := fetchSummary(m.baseCtx, m.ctx.Source(), m.epoch)()
	_, _ = m.Update(msg)

	if got := m.summary.Queries.Total; got != 4242 {
		t.Errorf("Total = %d, want 4242", got)
	}
}

func combinedModel(failed ...string) *Model {
	m := newTestModel()
	f := map[string]error{}
	for _, n := range failed {
		f[n] = errors.New("x")
	}
	m.ctx.Fleet = pihole.NewFleet([]pihole.Member{
		{Name: "primary", Client: m.ctx.API},
		{Name: "other", Client: m.ctx.API},
	}, f)
	m.ctx.Combined = true
	m.ctx.InstanceName = "primary"
	m.loaded = true
	return m
}

func TestUnreachableWarningRenders(t *testing.T) {
	for _, tc := range []struct {
		failed []string
		want   string
	}{
		{[]string{"secondary"}, "secondary unreachable"},
		{[]string{"tertiary", "secondary"}, "secondary, tertiary unreachable"},
	} {
		out := combinedModel(tc.failed...).View().Content
		if !strings.Contains(out, tc.want) {
			t.Errorf("View missing %q\n%s", tc.want, out)
		}
	}
	m := combinedModel("secondary")
	m.ctx.Combined = false
	if strings.Contains(m.View().Content, "unreachable") {
		t.Error("warning shown when not combined")
	}
}

func TestGravityMismatchMarker(t *testing.T) {
	m := combinedModel()
	m.summary = sampleSummary()
	if strings.Contains(m.renderTiles(), "≠") {
		t.Error("marker without mismatch")
	}
	m.summary.GravityMismatch = true
	if !strings.Contains(m.renderTiles(), "≠") {
		t.Error("gravity tile missing ≠")
	}
}

func TestSystemPanelNamesActiveWhenCombined(t *testing.T) {
	m := combinedModel()
	m.ctx.InstanceName = "pi-two"
	m.system = sampleSystem()
	if !strings.Contains(m.renderHealth(), "pi-two") {
		t.Error("combined health panel missing instance name")
	}
	m.ctx.Combined = false
	if strings.Contains(m.renderHealth(), "pi-two") {
		t.Error("single-mode health panel names instance")
	}
}
