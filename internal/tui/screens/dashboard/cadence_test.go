package dashboard

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Round sizes for a single-instance dashboard: summary + system are fast;
// history, breakdown and top lists are slow.
const (
	fastCmds = 2
	allCmds  = 5
)

func TestFocusStartsFullRound(t *testing.T) {
	m := newTestModel()

	m.Focus()

	if m.inFlight != allCmds {
		t.Fatalf("inFlight = %d, want %d", m.inFlight, allCmds)
	}
}

func TestTickWhileRoundInFlightIssuesNoFetch(t *testing.T) {
	m := newTestModel()
	m.Focus()
	m.Update(summaryMsg{epoch: m.epoch}) // one reply in, the rest pending
	pending := m.inFlight

	_, cmd := m.Update(tickMsg{epoch: m.epoch})

	if m.inFlight != pending {
		t.Fatalf("inFlight = %d, want %d (no new round)", m.inFlight, pending)
	}
	if cmd == nil {
		t.Fatal("a busy tick must still reschedule")
	}
}

func TestRepliesSettleRound(t *testing.T) {
	m := newTestModel()
	m.Focus()
	e := m.epoch

	for _, msg := range []tea.Msg{
		summaryMsg{epoch: e}, systemMsg{epoch: e}, historyMsg{epoch: e},
		breakdownMsg{epoch: e}, topMsg{epoch: e},
		summaryMsg{epoch: e - 1}, // stale: must not count
	} {
		m.Update(msg)
	}

	if m.inFlight != 0 {
		t.Fatalf("inFlight = %d, want 0 after every reply", m.inFlight)
	}
}

func TestSlowPanelsRefreshEveryFiveSeconds(t *testing.T) {
	m := newTestModel()
	m.Focus()

	m.inFlight = 0 // the full Focus round has finished
	m.Update(tickMsg{epoch: m.epoch})
	if m.inFlight != fastCmds {
		t.Fatalf("fast tick inFlight = %d, want %d", m.inFlight, fastCmds)
	}

	m.inFlight = 0
	m.lastSlow = time.Now().Add(-slowEvery)
	m.Update(tickMsg{epoch: m.epoch})
	if m.inFlight != allCmds {
		t.Fatalf("slow tick inFlight = %d, want %d", m.inFlight, allCmds)
	}
}

func TestRefreshKeyForcesSlowPanels(t *testing.T) {
	m := newTestModel()
	m.Focus()
	m.inFlight = 0

	m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	if m.inFlight != allCmds {
		t.Fatalf("inFlight = %d, want %d", m.inFlight, allCmds)
	}
}

func TestRefreshKeyWhileBusyRefocuses(t *testing.T) {
	// The dashboard's own 'r' (reached when focus stayed on the panel after a
	// palette jump) behaves like the global one: refocus, fresh full round.
	m := newTestModel()
	m.Focus()
	m.inFlight = 1
	epoch := m.epoch

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	if cmd == nil || m.epoch != epoch+1 || m.inFlight != allCmds {
		t.Fatalf("busy 'r': epoch %d→%d, inFlight %d; want refocus",
			epoch, m.epoch, m.inFlight)
	}
}

func TestCombinedProbesMembersOnSlowCadenceOnly(t *testing.T) {
	m := combinedModel()
	m.Focus()
	full := m.inFlight

	m.inFlight = 0
	m.Update(tickMsg{epoch: m.epoch})

	if full != allCmds+1 || m.inFlight != fastCmds {
		t.Fatalf("combined rounds = %d full / %d fast, want %d / %d",
			full, m.inFlight, allCmds+1, fastCmds)
	}
}

func TestRefocusMidRoundStartsFreshRound(t *testing.T) {
	// A refocus bumps the epoch, so the old round's replies never settle; the
	// count must reset or every later tick would stay "busy".
	m := newTestModel()
	m.Focus()
	m.Update(summaryMsg{epoch: m.epoch})

	m.Focus()

	if m.inFlight != allCmds {
		t.Fatalf("inFlight = %d, want a fresh round of %d", m.inFlight, allCmds)
	}
}
