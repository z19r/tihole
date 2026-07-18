package querylog

import (
	"testing"

	"github.com/z19r/tihole/internal/pihole"
)

// modelWithQuery returns a focused, panel-ready model with one query row so the
// cursor resolves to a real domain.
func modelWithQuery(domain string) *Model {
	m := newTestModel()
	m.SetSize(100, 24)
	m.focused = true
	m.queries = []pihole.Query{{Domain: domain, Status: "GRAVITY"}}
	m.syncRows()
	return m
}

func TestPressingAllowOpensConfirm(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")

	// Act
	_, _ = m.Update(keyPress("a"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("pressing a should open the confirm dialog")
	}
	if m.pendingType != pihole.DomainAllow {
		t.Fatalf("pending type should be allow, got %q", m.pendingType)
	}
	if m.confirm.Message != "ads.example.com" {
		t.Fatalf("confirm should name the domain, got %q", m.confirm.Message)
	}
	if m.confirm.Danger {
		t.Fatalf("allow confirm should not be styled as danger")
	}
}

func TestPressingBlockOpensDangerConfirm(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")

	// Act
	_, _ = m.Update(keyPress("b"))

	// Assert
	if !m.confirm.Active {
		t.Fatalf("pressing b should open the confirm dialog")
	}
	if m.pendingType != pihole.DomainDeny {
		t.Fatalf("pending type should be deny, got %q", m.pendingType)
	}
	if !m.confirm.Danger {
		t.Fatalf("block confirm should be styled as danger")
	}
}

func TestAllowIgnoredWithNoSelection(t *testing.T) {
	// Arrange: focused model with no rows
	m := newTestModel()
	m.SetSize(100, 24)
	m.focused = true

	// Act
	_, _ = m.Update(keyPress("a"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("allow with no selected row should not open a confirm")
	}
}

func TestQuickActionSuppressedWhileSearching(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")
	m.searching = true
	m.search.Focus()

	// Act: 'a' should be typed into the search box, not open a confirm
	_, _ = m.Update(keyPress("a"))

	// Assert
	if m.confirm.Active {
		t.Fatalf(
			"quick-classify must not trigger while the search field is focused",
		)
	}
}

func TestConfirmCancelHidesDialog(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")
	_, _ = m.Update(keyPress("a"))

	// Act
	_, _ = m.Update(keyPress("n"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("n should dismiss the confirm dialog")
	}
	if m.pendingDomain != "" {
		t.Fatalf(
			"cancel should clear the pending domain, got %q",
			m.pendingDomain,
		)
	}
}

func TestConfirmAcceptClearsDialogAndDispatches(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")
	_, _ = m.Update(keyPress("b"))

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("y should close the confirm dialog")
	}
	if cmd == nil {
		t.Fatalf("y should dispatch the classify command")
	}
}

func TestCapturesInputDuringConfirm(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")
	_, _ = m.Update(keyPress("a"))

	// Assert: while the confirm is up the screen must own y/n/esc so global
	// shortcuts (d=toggle-block) and the esc-to-rail climb don't steal them.
	if !m.CapturesInput() {
		t.Fatalf(
			"screen should capture input while the confirm dialog is active",
		)
	}
}

func TestClassifySuccessSetsNoteAndExpires(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")

	// Act: a successful allow result
	_, cmd := m.Update(classifyMsg{domain: "ads.example.com", verb: "allowed"})

	// Assert
	if m.note != "allowed ads.example.com" {
		t.Fatalf("success should set the note, got %q", m.note)
	}
	if m.err != nil {
		t.Fatalf("success should clear any error")
	}
	if cmd == nil {
		t.Fatalf("success should schedule the note expiry")
	}

	// Act: the expiry fires
	_, _ = m.Update(noteExpireMsg{})

	// Assert
	if m.note != "" {
		t.Fatalf("note should clear on expiry, got %q", m.note)
	}
}

func TestClassifyErrorSurfacesBanner(t *testing.T) {
	// Arrange
	m := modelWithQuery("ads.example.com")

	// Act
	_, _ = m.Update(
		classifyMsg{domain: "ads.example.com", verb: "blocked", err: errTest},
	)

	// Assert
	if m.err == nil {
		t.Fatalf("classify error should surface an error banner")
	}
	if m.note != "" {
		t.Fatalf("classify error should not set a success note, got %q", m.note)
	}
}

func TestClassifyWorksFromDetailPane(t *testing.T) {
	// Arrange: detail pane open on a specific query
	m := modelWithQuery("row.example.com")
	d := pihole.Query{Domain: "detail.example.com"}
	m.detail = &d

	// Act
	_, _ = m.Update(keyPress("a"))

	// Assert: the confirm targets the detail domain, not the table cursor
	if m.confirm.Message != "detail.example.com" {
		t.Fatalf(
			"classify from detail should target the detail domain, got %q",
			m.confirm.Message,
		)
	}
}

var errTest = testErr("boom")

type testErr string

func (e testErr) Error() string { return string(e) }
