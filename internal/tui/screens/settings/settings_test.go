package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/config"
	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/core"
)

// boolPtr is a small helper for the *bool VerifyTLS field.
func boolPtr(b bool) *bool { return &b }

// newTestConfig builds a config with two instances and a theme, plus a temp
// config path that Save can write to.
func newTestConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	c := &config.Config{
		Active: "primary",
		Theme:  theme.NameDeepNight,
		Instances: []config.Instance{
			{
				Name:      "primary",
				URL:       "http://pi.hole",
				Password:  "secret-pw",
				VerifyTLS: boolPtr(true),
			},
			{
				Name:        "secondary",
				URL:         "http://pi2.hole",
				PasswordEnv: "PIHOLE_PW",
			},
		},
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	return c, path
}

// newTestModel builds a Settings model with a config-backed app context.
func newTestModel(t *testing.T) *Model {
	t.Helper()
	c, path := newTestConfig(t)
	th := theme.DeepNight()
	api := pihole.New("http://x", "pw")
	m := New(&core.AppContext{
		API:          api,
		Theme:        th,
		Keys:         core.DefaultKeyMap(),
		Config:       c,
		ConfigPath:   path,
		InstanceName: "primary",
	})
	m.SetSize(100, 24)
	return m
}

// keyPress builds a KeyPressMsg whose String() matches the given keystroke.
func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace}
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg{Code: r, Text: s}
	}
}

func sampleTree() map[string]any {
	return map[string]any{
		"dns": map[string]any{
			"blocking": map[string]any{
				"active": true,
			},
			"port": 53,
			"host": "pi.hole",
		},
		"webserver": map[string]any{
			"port": 80,
		},
		"misc": map[string]any{
			"nice": false,
		},
	}
}

func TestFlattenConfigProducesSortedDottedLeaves(t *testing.T) {
	// Arrange
	tree := sampleTree()

	// Act
	leaves := flattenConfig(tree)

	// Assert
	want := []string{
		"dns.blocking.active",
		"dns.host",
		"dns.port",
		"misc.nice",
		"webserver.port",
	}
	if len(leaves) != len(want) {
		t.Fatalf(
			"expected %d leaves, got %d: %+v",
			len(want),
			len(leaves),
			leaves,
		)
	}
	for i, p := range want {
		if leaves[i].path != p {
			t.Fatalf("leaf %d: expected %q, got %q", i, p, leaves[i].path)
		}
	}
	// Types preserved.
	if leaves[0].value != true {
		t.Fatalf(
			"expected bool leaf, got %T %v",
			leaves[0].value,
			leaves[0].value,
		)
	}
	if leaves[2].value != 53 {
		t.Fatalf(
			"expected int leaf 53, got %T %v",
			leaves[2].value,
			leaves[2].value,
		)
	}
	if leaves[1].value != "pi.hole" {
		t.Fatalf(
			"expected string leaf, got %T %v",
			leaves[1].value,
			leaves[1].value,
		)
	}
}

func TestFlattenConfigCollapsesDetailedLeaves(t *testing.T) {
	// Arrange: FTL detailed=true wraps each leaf with metadata.
	tree := map[string]any{
		"dns": map[string]any{
			"port": map[string]any{
				"value":       float64(53),
				"type":        "integer",
				"description": "DNS port",
			},
		},
	}

	// Act
	leaves := flattenConfig(tree)

	// Assert
	if len(leaves) != 1 {
		t.Fatalf("expected 1 collapsed leaf, got %d: %+v", len(leaves), leaves)
	}
	if leaves[0].path != "dns.port" {
		t.Fatalf("expected dns.port, got %q", leaves[0].path)
	}
	if leaves[0].value != float64(53) {
		t.Fatalf("expected collapsed value 53, got %v", leaves[0].value)
	}
}

func TestFlattenConfigCapturesLeafMetadata(t *testing.T) {
	// Arrange: a detailed leaf carrying the full metadata set FTL returns.
	tree := map[string]any{
		"dns": map[string]any{
			"blocking": map[string]any{
				"mode": map[string]any{
					"value":       "NULL",
					"type":        "string",
					"description": "How blocked queries are answered",
					"default":     "NULL",
					"modified":    true,
					"allowed": []any{
						map[string]any{
							"item":        "NULL",
							"description": "0.0.0.0",
						},
						map[string]any{"item": "NXDOMAIN"},
						"IP",
					},
				},
			},
		},
	}

	// Act
	leaves := flattenConfig(tree)

	// Assert
	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d: %+v", len(leaves), leaves)
	}
	l := leaves[0]
	if l.description != "How blocked queries are answered" {
		t.Fatalf("description not captured: %q", l.description)
	}
	if l.dataType != "string" {
		t.Fatalf("type not captured: %q", l.dataType)
	}
	if l.defaultVal != "NULL" {
		t.Fatalf("default not captured: %v", l.defaultVal)
	}
	if !l.modified {
		t.Fatalf("modified flag not captured")
	}
	if got := allowedInputs(l); len(got) != 3 ||
		got[0] != "NULL" || got[1] != "NXDOMAIN" || got[2] != "IP" {
		t.Fatalf("allowed inputs not captured: %v", got)
	}
}

func TestRenderLeafDetailShowsMetadata(t *testing.T) {
	// Arrange
	th := theme.Gloss()
	l := leaf{
		path:        "dns.blocking.mode",
		value:       "NULL",
		description: "How blocked queries are answered",
		dataType:    "string",
		defaultVal:  "NULL",
		allowed:     []any{map[string]any{"item": "NXDOMAIN"}},
		modified:    true,
	}

	// Act
	out := renderLeafDetail(th, l, 48, 20)

	// Assert
	for _, want := range []string{
		"dns.blocking.mode", "How blocked queries",
		"Type", "Default", "Current", "Allowed", "NXDOMAIN", "modified",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("detail panel missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderLeafDetailNeverExceedsNarrowWidth(t *testing.T) {
	// Arrange: a narrow (sub-24-col) but tall stacked layout, where the old
	// 20-col panelW floor would render wider than the container and make the
	// outer surface wrap — shifting every row below the panel.
	th := theme.Gloss()
	l := leaf{
		path:        "dns.blocking.mode",
		value:       "NULL",
		description: "How blocked queries are answered by the resolver",
		dataType:    "string",
		defaultVal:  "NULL",
		modified:    true,
	}

	// Act
	for _, w := range []int{4, 8, 14, 20, 23} {
		out := renderLeafDetail(th, l, w, 16)

		// Assert: no rendered line is wider than the allotted width.
		for i, line := range strings.Split(out, "\n") {
			if lw := lipgloss.Width(line); lw > w {
				t.Fatalf(
					"w=%d line %d width %d exceeds container:\n%s",
					w, i, lw, out,
				)
			}
		}
	}
}

func TestParseLeafValueRoundTripsByType(t *testing.T) {
	// Arrange / Act / Assert — bool
	b, err := parseLeafValue(true, "false")
	if err != nil || b != false {
		t.Fatalf("bool parse: got %v err %v", b, err)
	}
	// int
	n, err := parseLeafValue(42, "99")
	if err != nil || n != 99 {
		t.Fatalf("int parse: got %v (%T) err %v", n, n, err)
	}
	// string
	s, err := parseLeafValue("old", "new value")
	if err != nil || s != "new value" {
		t.Fatalf("string parse: got %v err %v", s, err)
	}
	// invalid int surfaces an error
	if _, err := parseLeafValue(1, "not-a-number"); err == nil {
		t.Fatalf("expected error parsing non-numeric int")
	}
}

func TestBuildPatchNestsDottedPath(t *testing.T) {
	// Arrange / Act
	patch := buildPatch("dns.blocking.active", false)

	// Assert
	dns, ok := patch["dns"].(map[string]any)
	if !ok {
		t.Fatalf("expected dns object, got %T", patch["dns"])
	}
	blocking, ok := dns["blocking"].(map[string]any)
	if !ok {
		t.Fatalf("expected blocking object, got %T", dns["blocking"])
	}
	if blocking["active"] != false {
		t.Fatalf("expected active=false leaf, got %v", blocking["active"])
	}
}

func TestFocusSetsLoadingAndBumpsEpoch(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	startEpoch := m.epoch

	// Act
	cmd := m.Focus()

	// Assert
	if !m.loading {
		t.Fatalf("focus should set loading")
	}
	if m.epoch != startEpoch+1 {
		t.Fatalf(
			"focus should bump epoch: got %d want %d",
			m.epoch,
			startEpoch+1,
		)
	}
	if cmd == nil {
		t.Fatalf("focus should return a fetch command")
	}
}

func TestConfigMsgPopulatesTreeAndClearsLoading(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	m.loading = true

	// Act
	_, _ = m.Update(configMsg{epoch: m.epoch, tree: sampleTree()})

	// Assert
	if m.loading {
		t.Fatalf("result should clear loading")
	}
	if len(m.leaves) != 5 {
		t.Fatalf("expected 5 leaves stored, got %d", len(m.leaves))
	}
	if len(m.visible) != 5 {
		t.Fatalf("expected 5 visible with no filter, got %d", len(m.visible))
	}
}

func TestConfigMsgErrorSetsBanner(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	m.loading = true

	// Act
	_, _ = m.Update(configMsg{epoch: m.epoch, err: os.ErrPermission})

	// Assert
	if m.err == nil {
		t.Fatalf("error result should set the banner")
	}
	if m.loading {
		t.Fatalf("error result should clear loading")
	}
}

func TestUpdateIgnoresStaleConfigResult(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	m.epoch = 9
	m.loading = true

	// Act
	_, _ = m.Update(configMsg{epoch: 8, tree: sampleTree()})

	// Assert
	if len(m.leaves) != 0 {
		t.Fatalf("stale result should not populate the tree")
	}
	if !m.loading {
		t.Fatalf("stale result should not clear loading")
	}
}

func TestPatchSuccessTriggersRefetch(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	m.epoch = 4

	// Act
	_, cmd := m.Update(patchMsg{epoch: 4, err: nil})

	// Assert
	if m.epoch != 5 {
		t.Fatalf(
			"successful patch should bump epoch for refetch, got %d",
			m.epoch,
		)
	}
	if !m.loading {
		t.Fatalf("refetch should set loading")
	}
	if cmd == nil {
		t.Fatalf("successful patch should return a refetch command")
	}
}

func TestEnterOpensLeafEditorAndSubmitPatches(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(configMsg{epoch: m.epoch, tree: sampleTree()})
	m.tree.SetCursor(0) // dns.blocking.active (bool)

	// Act: open editor
	_, _ = m.Update(keyPress("enter"))

	// Assert
	if m.edit == nil {
		t.Fatalf("enter should open the leaf editor")
	}
	if !m.edit.isBool {
		t.Fatalf("bool leaf editor should be in bool mode")
	}

	// Act: submit
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.edit != nil {
		t.Fatalf("submitting should close the editor")
	}
	if cmd == nil {
		t.Fatalf("submitting should return a patch command")
	}
}

func TestFilterNarrowsVisibleLeaves(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(configMsg{epoch: m.epoch, tree: sampleTree()})

	// Act: open filter, type "webserver", submit
	_, _ = m.Update(keyPress("/"))
	m.filterInput.SetValue("webserver")
	_, _ = m.Update(keyPress("enter"))

	// Assert
	if m.filtering {
		t.Fatalf("enter should close the filter input")
	}
	if len(m.visible) != 1 {
		t.Fatalf("filter should narrow to 1 leaf, got %d", len(m.visible))
	}
	if m.visible[0].path != "webserver.port" {
		t.Fatalf("wrong leaf after filter: %q", m.visible[0].path)
	}
}

func TestCToggleSwitchesModes(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	if m.mode != modeConfig {
		t.Fatalf("default mode should be config")
	}

	// Act: config -> connections
	_, _ = m.Update(keyPress("c"))

	// Assert
	if m.mode != modeConnections {
		t.Fatalf("'c' should switch to connections mode")
	}

	// Act: connections -> config
	_, _ = m.Update(keyPress("c"))

	// Assert
	if m.mode != modeConfig {
		t.Fatalf("'c' should switch back to config mode")
	}
}

func TestAddInstanceSubmitWritesFileAndEmits(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(keyPress("c")) // connections mode
	_, _ = m.Update(keyPress("a")) // add form
	if m.connForm == nil {
		t.Fatalf("'a' should open the instance form")
	}
	m.connForm.name.SetValue("tertiary")
	m.connForm.url.SetValue("http://pi3.hole")
	m.connForm.password.SetValue("pw3")

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert: form closed, no error
	if m.connForm != nil {
		t.Fatalf("submitting should close the form")
	}
	if m.err != nil {
		t.Fatalf("valid submit should not set an error: %v", m.err)
	}
	// File persisted.
	if _, err := os.Stat(m.ctx.ConfigPath); err != nil {
		t.Fatalf("config file should exist after save: %v", err)
	}
	// Local config updated.
	if len(m.ctx.Config.Instances) != 3 {
		t.Fatalf(
			"expected 3 instances after add, got %d",
			len(m.ctx.Config.Instances),
		)
	}
	// Command emits InstancesChangedMsg.
	if cmd == nil {
		t.Fatalf("valid submit should return a command")
	}
	msg := cmd()
	changed, ok := msg.(core.InstancesChangedMsg)
	if !ok {
		t.Fatalf("expected InstancesChangedMsg, got %T", msg)
	}
	if len(changed.Config.Instances) != 3 {
		t.Fatalf(
			"emitted config should carry 3 instances, got %d",
			len(changed.Config.Instances),
		)
	}
}

func TestInvalidEditSetsBannerAndDoesNotEmit(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(1) // edit "secondary"
	_, _ = m.Update(keyPress("e"))
	// Rename to a duplicate of the first instance.
	m.connForm.name.SetValue("primary")

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.err == nil {
		t.Fatalf("duplicate name should set an error banner")
	}
	if m.connForm == nil {
		t.Fatalf("invalid submit should keep the form open")
	}
	if cmd != nil {
		t.Fatalf("invalid submit should not emit a command")
	}
}

func TestDeleteInstanceConfirmEmits(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(1) // delete "secondary"
	_, _ = m.Update(keyPress("x"))
	if !m.confirm.Active {
		t.Fatalf("'x' should open the confirm dialog")
	}

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert
	if m.confirm.Active {
		t.Fatalf("confirming should hide the dialog")
	}
	if len(m.ctx.Config.Instances) != 1 {
		t.Fatalf(
			"expected 1 instance after delete, got %d",
			len(m.ctx.Config.Instances),
		)
	}
	if cmd == nil {
		t.Fatalf("confirming delete should return a command")
	}
	if _, ok := cmd().(core.InstancesChangedMsg); !ok {
		t.Fatalf("delete should emit InstancesChangedMsg")
	}
}

func TestDeleteLastInstanceIsRejected(t *testing.T) {
	// Arrange: config with a single instance.
	c := &config.Config{
		Active: "only",
		Theme:  theme.NameDeepNight,
		Instances: []config.Instance{
			{Name: "only", URL: "http://pi.hole", Password: "x"},
		},
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	m := New(
		&core.AppContext{
			API:        pihole.New("http://x", "pw"),
			Theme:      theme.DeepNight(),
			Config:     c,
			ConfigPath: path,
		},
	)
	m.SetSize(100, 24)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("x"))

	// Act
	_, cmd := m.Update(keyPress("y"))

	// Assert: rejected — still one instance, banner set, no emit.
	if len(m.ctx.Config.Instances) != 1 {
		t.Fatalf("deleting the last instance should be rejected")
	}
	if m.err == nil {
		t.Fatalf("rejected delete should set a banner")
	}
	if cmd != nil {
		t.Fatalf("rejected delete should not emit a command")
	}
}

func TestThemeSelectEmitsSetThemeMsg(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("t"))
	if m.themePicker == nil {
		t.Fatalf("'t' should open the theme picker")
	}
	// Move to a different theme than the current one.
	_, _ = m.Update(keyPress("down"))

	// Act
	_, cmd := m.Update(keyPress("enter"))

	// Assert
	if m.themePicker != nil {
		t.Fatalf("selecting should close the picker")
	}
	if cmd == nil {
		t.Fatalf("theme select should return a command")
	}
	msg := cmd()
	setTheme, ok := msg.(core.SetThemeMsg)
	if !ok {
		t.Fatalf("expected SetThemeMsg, got %T", msg)
	}
	if setTheme.Theme == nil {
		t.Fatalf("SetThemeMsg should carry a resolved theme")
	}
	if _, err := os.Stat(m.ctx.ConfigPath); err != nil {
		t.Fatalf("theme change should persist the config: %v", err)
	}
}

func TestPasswordNeverAppearsInView(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(keyPress("c")) // connections mode shows instances

	// Act
	view := m.View()

	// Assert
	if strings.Contains(view.Content, "secret-pw") {
		t.Fatalf("rendered view must never contain the stored password")
	}
	// Auth source is described instead.
	if !strings.Contains(view.Content, "inline") {
		t.Fatalf("expected the inline auth label to appear")
	}
}

func TestPasswordNeverAppearsInEditFormView(t *testing.T) {
	// Arrange
	m := newTestModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(0) // primary has an inline password
	_, _ = m.Update(keyPress("e"))

	// Act
	view := m.View()

	// Assert: the edit form must not surface the stored secret.
	if strings.Contains(view.Content, "secret-pw") {
		t.Fatalf("edit form must not pre-fill or render the stored password")
	}
}

func TestSetSizeDoesNotPanicAtTinySizes(t *testing.T) {
	// Arrange
	m := newTestModel(t)

	// Act / Assert (no panic)
	m.SetSize(1, 1)
	m.SetSize(0, 0)
	_ = m.View()
	_, _ = m.Update(keyPress("c"))
	_ = m.View()
}

func TestHelpDiffersByMode(t *testing.T) {
	// Arrange
	m := newTestModel(t)

	// Act
	configHelp := m.Help()
	_, _ = m.Update(keyPress("c"))
	connHelp := m.Help()

	// Assert
	if len(configHelp) == 0 || len(connHelp) == 0 {
		t.Fatalf("both modes should provide help bindings")
	}
	if configHelp[0].Help().Key == connHelp[0].Help().Key {
		t.Fatalf("help bindings should differ by mode")
	}
}

var _ core.Screen = (*Model)(nil)
var _ tea.Model = (*Model)(nil)
