package dashboard

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/core"
)

// newTestModel builds a dashboard model with a non-nil client. No network is
// performed; render tests populate state directly (white-box, same package).
func newTestModel() *Model {
	th := theme.DeepNight()
	api := pihole.New("http://x", "pw")
	m := New(&core.AppContext{API: api, Theme: th, Keys: core.DefaultKeyMap()})
	m.SetSize(100, 30)
	return m
}

// sampleSummary returns a populated summary for tile rendering.
func sampleSummary() pihole.Summary {
	var s pihole.Summary
	s.Queries.Total = 12345
	s.Queries.Blocked = 2000
	s.Queries.PercentBlocked = 16.2
	s.Clients.Active = 5
	s.Gravity.DomainsBeingBlocked = 1000
	return s
}

func sampleHistory() pihole.History {
	return pihole.History{History: []pihole.HistoryPoint{
		{Total: 10, Blocked: 2},
		{Total: 40, Blocked: 5},
		{Total: 25, Blocked: 3},
	}}
}

func sampleTypes() pihole.QueryTypes {
	return pihole.QueryTypes{
		Types: map[string]int{"A": 10, "AAAA": 5, "PTR": 1},
	}
}

func sampleUpstreams() pihole.Upstreams {
	return pihole.Upstreams{Upstreams: []pihole.UpstreamServer{
		{Name: "cloudflared", Count: 100},
		{IP: "1.1.1.1", Count: 50},
	}}
}

func sampleTopDomains() pihole.TopDomains {
	var td pihole.TopDomains
	_ = json.Unmarshal(
		[]byte(
			`{"domains":[{"domain":"example.com","count":42},{"domain":"news.example.org","count":17}]}`,
		),
		&td,
	)
	return td
}

func sampleTopClients() pihole.TopClients {
	var tc pihole.TopClients
	_ = json.Unmarshal(
		[]byte(
			`{"clients":[{"ip":"10.0.0.1","name":"laptop","count":80},{"ip":"10.0.0.2","name":"","count":30}]}`,
		),
		&tc,
	)
	return tc
}

// loadedModel returns a fully-populated, loaded model ready for View().
func loadedModel() *Model {
	m := newTestModel()
	m.loaded = true
	m.summary = sampleSummary()
	m.history = sampleHistory()
	m.types = sampleTypes()
	m.upstreams = sampleUpstreams()
	m.domains = sampleTopDomains()
	m.blocked = sampleTopDomains()
	m.clients = sampleTopClients()
	return m
}

func TestView_LoadedRendersAllPanels(t *testing.T) {
	// Arrange
	m := loadedModel()

	// Act
	v := m.View()

	// Assert
	if v.Content == "" {
		t.Fatal("expected non-empty view content")
	}
	for _, want := range []string{"Total Queries", "Blocked", "Active Clients", "Gravity Domains", "Queries over time", "Query Types", "Upstreams", "Top Domains", "Top Blocked", "Top Clients"} {
		if !strings.Contains(v.Content, want) {
			t.Errorf("view content missing %q", want)
		}
	}
}

func TestView_LoadingShowsSpinnerMessage(t *testing.T) {
	// Arrange
	m := newTestModel() // not loaded

	// Act
	v := m.View()

	// Assert
	if !strings.Contains(v.Content, "Loading dashboard") {
		t.Fatalf("expected loading message, got %q", v.Content)
	}
}

func TestView_TooSmallRendersEmptySurface(t *testing.T) {
	// Arrange
	m := loadedModel()
	m.SetSize(10, 4)

	// Act / Assert (no panic, degrades gracefully)
	_ = m.View()
}

func TestView_ErrorBannersRenderWithoutPanic(t *testing.T) {
	// Arrange
	m := loadedModel()
	m.errSummary = "summary boom"
	m.errHistory = "history boom"
	m.errTypes = "types boom"
	m.errUpstreams = "upstreams boom"
	m.errDomains = "domains boom"
	m.errBlocked = "blocked boom"
	m.errClients = "clients boom"

	// Act
	v := m.View()

	// Assert
	if v.Content == "" {
		t.Fatal("expected content even with all panels erroring")
	}
	if !strings.Contains(v.Content, "unavailable") {
		t.Errorf("expected an 'unavailable' banner, got %q", v.Content)
	}
}

func TestRenderSparkline_EmptyHistoryShowsPlaceholder(t *testing.T) {
	// Arrange
	m := loadedModel()
	m.history = pihole.History{}

	// Act
	out := m.renderSparkline()

	// Assert
	if !strings.Contains(out, "no history yet") {
		t.Fatalf("expected placeholder for empty history, got %q", out)
	}
}

func TestBreakdownAndListPanels_EmptyShowNoData(t *testing.T) {
	// Arrange
	m := loadedModel()

	// Act
	bp := m.breakdownPanel("Query Types", nil, "", 30)
	lp := m.listPanel("Top Domains", nil, "", 30)

	// Assert
	if !strings.Contains(bp, "no data") {
		t.Errorf("empty breakdown should say no data, got %q", bp)
	}
	if !strings.Contains(lp, "no data") {
		t.Errorf("empty list should say no data, got %q", lp)
	}
}

func TestTypeItems_SortedDescending(t *testing.T) {
	// Arrange / Act
	items := typeItems(map[string]int{"A": 1, "AAAA": 9})

	// Assert
	if len(items) != 2 || items[0].label != "AAAA" {
		t.Fatalf("expected AAAA first, got %+v", items)
	}
}

func TestUpstreamItems_FallsBackToIPWhenNameEmpty(t *testing.T) {
	// Arrange / Act
	items := upstreamItems(sampleUpstreams().Upstreams)

	// Assert
	if items[0].label != "cloudflared" {
		t.Errorf("expected name label, got %q", items[0].label)
	}
	if items[1].label != "1.1.1.1" {
		t.Errorf("expected IP fallback label, got %q", items[1].label)
	}
}

func TestDomainItems_MapsDomainAndCount(t *testing.T) {
	// Arrange / Act
	items := domainItems(sampleTopDomains())

	// Assert
	if len(items) != 2 || items[0].label != "example.com" ||
		items[0].count != 42 {
		t.Fatalf("unexpected domain items: %+v", items)
	}
}

func TestClientItems_FallsBackToIPWhenNameEmpty(t *testing.T) {
	// Arrange / Act
	items := clientItems(sampleTopClients())

	// Assert
	if items[0].label != "laptop" {
		t.Errorf("expected client name, got %q", items[0].label)
	}
	if items[1].label != "10.0.0.2" {
		t.Errorf("expected IP fallback for empty name, got %q", items[1].label)
	}
}
