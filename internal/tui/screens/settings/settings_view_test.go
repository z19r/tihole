package settings

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// specialKey builds a KeyPressMsg for a non-printable key (arrows, tab) whose
// String() the handlers switch on.
func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// bigModel builds a config-backed model sized like the reference tests (120x30)
// with the tree already loaded, so View() exercises the populated path.
func bigModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	m.SetSize(120, 30)
	m.focused = true
	_, _ = m.Update(configMsg{epoch: m.epoch, tree: sampleTree()})
	return m
}

// --- lifecycle ------------------------------------------------------------

func TestInitReturnsNil(t *testing.T) {
	if cmd := newTestModel(t).Init(); cmd != nil {
		t.Fatalf("Init should return nil until Focus")
	}
}

func TestTitleIsSettings(t *testing.T) {
	if got := newTestModel(t).Title(); got != "Settings" {
		t.Fatalf("unexpected title: %q", got)
	}
}

func TestBlurClearsOverlaysAndCancels(t *testing.T) {
	// Arrange: Focus arms an in-flight cancel; open several overlays.
	m := newTestModel(t)
	m.Focus()
	if m.cancel == nil {
		t.Fatalf("precondition: Focus should arm a cancel func")
	}
	m.filtering = true
	_, _ = m.Update(keyPress("c")) // connections mode
	_, _ = m.Update(keyPress("a")) // add form
	m.edit = newLeafEdit(leaf{path: "x", value: "y"})

	// Act
	m.Blur()

	// Assert
	if m.focused {
		t.Fatalf("Blur should clear focus")
	}
	if m.filtering || m.edit != nil || m.connForm != nil || m.themePicker != nil {
		t.Fatalf("Blur should tear down all overlays")
	}
	if m.pendingDel != -1 {
		t.Fatalf("Blur should reset pendingDel")
	}
	if m.cancel != nil {
		t.Fatalf("Blur should cancel and clear the in-flight request")
	}
}

func TestSetSizeTinyBothModesNoPanic(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(-3, -3)
	_ = m.View()
	_, _ = m.Update(keyPress("c"))
	m.SetSize(2, 2)
	_ = m.View()
}

// --- config-mode View states ---------------------------------------------

func TestViewConfigLoadedRendersRows(t *testing.T) {
	m := bigModel(t)
	out := m.View().Content
	for _, want := range []string{"Settings", "Configuration", "shown", "total", "dns.host"} {
		if !strings.Contains(out, want) {
			t.Fatalf("config view missing %q:\n%s", want, out)
		}
	}
}

func TestViewConfigLoadingShowsSpinner(t *testing.T) {
	m := newTestModel(t)
	m.SetSize(120, 30)
	m.loading = true
	if out := m.View().Content; !strings.Contains(out, "loading config") {
		t.Fatalf("empty+loading should render loading text:\n%s", out)
	}
}

func TestViewConfigEmptyFilterShowsNoMatch(t *testing.T) {
	m := bigModel(t)
	m.filter = "zzz-nothing-matches"
	m.applyFilter()
	if out := m.View().Content; !strings.Contains(out, "no config keys match") {
		t.Fatalf("filtered-to-empty should render the no-match message:\n%s", out)
	}
}

func TestViewConfigErrorBanner(t *testing.T) {
	m := bigModel(t)
	m.err = errPerm{}
	if out := m.View().Content; !strings.Contains(out, "error:") {
		t.Fatalf("a set error should render the banner:\n%s", out)
	}
}

type errPerm struct{}

func (errPerm) Error() string { return "permission denied" }

func TestViewConfigFilterInputActive(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("/"))
	if !m.filtering {
		t.Fatalf("'/' should enter filtering mode")
	}
	m.filterInput.SetValue("myquery")
	if out := m.View().Content; !strings.Contains(out, "myquery") {
		t.Fatalf("filtering should render the filter input value:\n%s", out)
	}
}

func TestViewConfigLeafEditPanel(t *testing.T) {
	m := bigModel(t)
	m.tree.SetCursor(1) // dns.host (string)
	_, _ = m.Update(keyPress("enter"))
	if m.edit == nil {
		t.Fatalf("enter should open the leaf editor")
	}
	out := m.View().Content
	for _, want := range []string{"Edit dns.host", "Value", "save", "cancel"} {
		if !strings.Contains(out, want) {
			t.Fatalf("leaf edit panel missing %q:\n%s", want, out)
		}
	}
}

// --- connections-mode View states ----------------------------------------

func TestViewConnEmptyInstances(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	m.ctx.Config.Instances = nil
	if out := m.View().Content; !strings.Contains(out, "no instances configured") {
		t.Fatalf("zero instances should render the empty message:\n%s", out)
	}
}

func TestViewConnAddFormRendered(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("a"))
	out := m.View().Content
	for _, want := range []string{"Add instance", "Name", "URL", "Password", "Verify TLS"} {
		if !strings.Contains(out, want) {
			t.Fatalf("add form missing %q:\n%s", want, out)
		}
	}
}

func TestViewConnEditFormMasksPassword(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(0) // primary has inline secret
	_, _ = m.Update(keyPress("e"))
	out := m.View().Content
	if !strings.Contains(out, "Edit instance") {
		t.Fatalf("edit form heading missing:\n%s", out)
	}
	if strings.Contains(out, "secret-pw") {
		t.Fatalf("edit form must never surface the stored password")
	}
}

func TestViewConnThemePickerRendered(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("t"))
	out := m.View().Content
	for _, want := range []string{"Select theme", theme.NameDeepNight, "(current)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("theme picker missing %q:\n%s", want, out)
		}
	}
}

func TestViewConnConfirmDialogRendered(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(1)
	_, _ = m.Update(keyPress("x"))
	if !m.confirm.Active {
		t.Fatalf("'x' should open the confirm dialog")
	}
	if out := m.View().Content; !strings.Contains(out, "Delete instance?") {
		t.Fatalf("confirm dialog prompt missing:\n%s", out)
	}
}

// --- leaf edit key handling ----------------------------------------------

func TestLeafEditTextTypeAndSubmitBuildsPatch(t *testing.T) {
	m := bigModel(t)
	m.tree.SetCursor(1) // dns.host (string)
	_, _ = m.Update(keyPress("enter"))
	if m.edit == nil || m.edit.isBool {
		t.Fatalf("string leaf should open a text editor")
	}
	m.edit.input.SetValue("newhost")

	_, cmd := m.Update(keyPress("enter"))
	if m.edit != nil {
		t.Fatalf("submit should close the editor")
	}
	if cmd == nil {
		t.Fatalf("submit should return a patch command")
	}
}

func TestLeafEditEscCancels(t *testing.T) {
	m := bigModel(t)
	m.tree.SetCursor(1)
	_, _ = m.Update(keyPress("enter"))
	_, _ = m.Update(keyPress("esc"))
	if m.edit != nil {
		t.Fatalf("esc should cancel the editor")
	}
}

func TestLeafEditBoolToggleViaSpace(t *testing.T) {
	m := bigModel(t)
	m.tree.SetCursor(0) // dns.blocking.active (bool)
	_, _ = m.Update(keyPress("enter"))
	if m.edit == nil || !m.edit.isBool {
		t.Fatalf("bool leaf should open a bool editor")
	}
	before := m.edit.boolVal
	_, _ = m.Update(keyPress("space"))
	if m.edit.boolVal == before {
		t.Fatalf("space should toggle the boolean value")
	}
}

func TestLeafEditTextTypingEditsInput(t *testing.T) {
	m := bigModel(t)
	m.tree.SetCursor(1)
	_, _ = m.Update(keyPress("enter"))
	_, _ = m.Update(keyPress("z"))
	if !strings.Contains(m.edit.input.Value(), "z") {
		t.Fatalf("typing should edit the text input, got %q", m.edit.input.Value())
	}
}

func TestSubmitEditParseErrorKeepsEditorOpen(t *testing.T) {
	m := bigModel(t)
	m.tree.SetCursor(2) // dns.port (int)
	_, _ = m.Update(keyPress("enter"))
	m.edit.input.SetValue("not-a-number")
	_, cmd := m.Update(keyPress("enter"))
	if m.edit == nil {
		t.Fatalf("parse error should keep the editor open")
	}
	if m.err == nil {
		t.Fatalf("parse error should set a banner")
	}
	if cmd != nil {
		t.Fatalf("parse error should not dispatch a patch")
	}
}

// --- filter key handling --------------------------------------------------

func TestFilterTypingThenEscRestores(t *testing.T) {
	m := bigModel(t)
	m.filter = "dns"
	_, _ = m.Update(keyPress("/"))
	_, _ = m.Update(keyPress("x")) // types into the box
	_, _ = m.Update(keyPress("esc"))
	if m.filtering {
		t.Fatalf("esc should leave filtering mode")
	}
	if m.filter != "dns" {
		t.Fatalf("esc should not change the applied filter, got %q", m.filter)
	}
	if m.filterInput.Value() != "dns" {
		t.Fatalf("esc should reset the box to the applied filter, got %q", m.filterInput.Value())
	}
}

// --- theme picker key handling -------------------------------------------

func TestThemePickerNavigationAndEsc(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("t"))
	start := m.themePicker.cursor
	_, _ = m.Update(keyPress("j")) // down
	if m.themePicker.cursor != start+1 {
		t.Fatalf("j should move the theme cursor down")
	}
	_, _ = m.Update(keyPress("k")) // up
	if m.themePicker.cursor != start {
		t.Fatalf("k should move the theme cursor up")
	}
	_, _ = m.Update(specialKey(tea.KeyDown))
	if m.themePicker.cursor != start+1 {
		t.Fatalf("down arrow should move the cursor down")
	}
	_, _ = m.Update(specialKey(tea.KeyUp))
	_, _ = m.Update(keyPress("esc"))
	if m.themePicker != nil {
		t.Fatalf("esc should close the theme picker")
	}
}

// --- confirm dialog cancel paths -----------------------------------------

func TestConfirmCancelWithN(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(1)
	_, _ = m.Update(keyPress("x"))
	_, _ = m.Update(keyPress("n"))
	if m.confirm.Active {
		t.Fatalf("'n' should hide the confirm dialog")
	}
	if m.pendingDel != -1 {
		t.Fatalf("'n' should reset pendingDel")
	}
	if len(m.ctx.Config.Instances) != 2 {
		t.Fatalf("'n' should not delete anything")
	}
}

func TestConfirmCancelWithEsc(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(1)
	_, _ = m.Update(keyPress("x"))
	_, _ = m.Update(keyPress("esc"))
	if m.confirm.Active {
		t.Fatalf("esc should hide the confirm dialog")
	}
}

// --- deleting the active instance reassigns Active ------------------------

func TestDeleteActiveInstanceReassignsActive(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(0) // "primary" is the active instance
	_, _ = m.Update(keyPress("x"))
	_, cmd := m.Update(keyPress("y"))
	if len(m.ctx.Config.Instances) != 1 {
		t.Fatalf("expected 1 instance after delete, got %d", len(m.ctx.Config.Instances))
	}
	if m.ctx.Config.Active != "secondary" {
		t.Fatalf("deleting the active instance should reassign Active, got %q", m.ctx.Config.Active)
	}
	if cmd == nil {
		t.Fatalf("delete should emit a command")
	}
}

// --- editing the active instance preserves the Active pointer -------------

func TestEditActiveInstanceRenamesActivePointer(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	m.connTable.SetCursor(0) // primary (active)
	_, _ = m.Update(keyPress("e"))
	m.connForm.name.SetValue("primary-renamed")
	_, cmd := m.Update(keyPress("enter"))
	if m.err != nil {
		t.Fatalf("valid rename should not error: %v", m.err)
	}
	if m.ctx.Config.Active != "primary-renamed" {
		t.Fatalf("renaming the active instance should follow the Active pointer, got %q", m.ctx.Config.Active)
	}
	if cmd == nil {
		t.Fatalf("valid edit should emit a command")
	}
}

// --- instance form key handling ------------------------------------------

func TestConnFormTabNavigationAndToggle(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("a"))
	if m.connForm.focus != fieldName {
		t.Fatalf("new form should focus the name field")
	}
	_, _ = m.Update(keyPress("tab"))
	if m.connForm.focus != fieldURL {
		t.Fatalf("tab should advance focus to URL, got %v", m.connForm.focus)
	}
	_, _ = m.Update(specialKey(tea.KeyUp))
	if m.connForm.focus != fieldName {
		t.Fatalf("up should move focus back to name, got %v", m.connForm.focus)
	}

	// Move to the verify-tls toggle and flip it.
	m.connForm.focus = fieldVerifyTLS
	before := m.connForm.verifyTLS
	_, _ = m.Update(keyPress("space"))
	if m.connForm.verifyTLS == before {
		t.Fatalf("space on the verify field should toggle it")
	}
}

func TestConnFormTypingRoutesToFocusedInput(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("a"))
	_, _ = m.Update(keyPress("q")) // focus is name
	if !strings.Contains(m.connForm.name.Value(), "q") {
		t.Fatalf("typing should route to the focused input, got %q", m.connForm.name.Value())
	}
}

func TestConnFormEscCloses(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	_, _ = m.Update(keyPress("a"))
	_, _ = m.Update(keyPress("esc"))
	if m.connForm != nil {
		t.Fatalf("esc should close the instance form")
	}
}

// --- pure form helpers ----------------------------------------------------

func TestFormFocusNextPrevWraps(t *testing.T) {
	f := newAddInstanceForm()
	f.focus = fieldName
	f.focusPrev()
	if f.focus != fieldVerifyTLS {
		t.Fatalf("focusPrev from first should wrap to last, got %v", f.focus)
	}
	f.focusNext()
	if f.focus != fieldName {
		t.Fatalf("focusNext from last should wrap to first, got %v", f.focus)
	}
}

func TestFormToggleVerify(t *testing.T) {
	f := newAddInstanceForm()
	before := f.verifyTLS
	f.toggleVerify()
	if f.verifyTLS == before {
		t.Fatalf("toggleVerify should flip verifyTLS")
	}
}

func TestFormUpdateFocusedRoutesEachField(t *testing.T) {
	f := newAddInstanceForm()
	for _, field := range []instField{fieldName, fieldURL, fieldPassword, fieldPasswordEnv} {
		f.focus = field
		f.syncFocus()
		f.updateFocused(keyPress("a"))
	}
	if f.name.Value() == "" || f.url.Value() == "" ||
		f.password.Value() == "" || f.passwordEnv.Value() == "" {
		t.Fatalf("updateFocused should route input to each focused field")
	}
}

func TestFormTitleByMode(t *testing.T) {
	if newAddInstanceForm().title() != "Add instance" {
		t.Fatalf("add form title wrong")
	}
	edit := newEditInstanceForm(config.Instance{Name: "x"}, 0)
	if edit.title() != "Edit instance" {
		t.Fatalf("edit form title wrong")
	}
}

// --- pure value helpers ---------------------------------------------------

func TestStringifyValueAllTypes(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, "null"},
		{true, "true"},
		{false, "false"},
		{"hello", "hello"},
		{float64(53), "53"},
		{float64(1.5), "1.5"},
		{int(7), "7"},
		{int64(9), "9"},
		{[]any{1, "a"}, "[1, a]"},
		{map[string]any{"k": 1}, "{…}"},
		{struct{ X int }{1}, "{1}"},
	}
	for _, c := range cases {
		if got := stringifyValue(c.in); got != c.want {
			t.Fatalf("stringifyValue(%v): got %q want %q", c.in, got, c.want)
		}
	}
}

func TestParseLeafValueNumericTypes(t *testing.T) {
	// int64 stays int64
	v, err := parseLeafValue(int64(1), "123")
	if err != nil || v != int64(123) {
		t.Fatalf("int64 parse: got %v (%T) err %v", v, v, err)
	}
	// float64 with fractional part
	v, err = parseLeafValue(float64(1), "2.5")
	if err != nil || v != 2.5 {
		t.Fatalf("float parse: got %v err %v", v, err)
	}
	// float64 fed an integer string keeps int64 form
	v, err = parseLeafValue(float64(1), "42")
	if err != nil || v != int64(42) {
		t.Fatalf("float-as-int parse: got %v (%T) err %v", v, v, err)
	}
	// errors
	if _, err := parseLeafValue(int64(1), "nope"); err == nil {
		t.Fatalf("int64 parse should error on garbage")
	}
	if _, err := parseLeafValue(float64(1), "nope"); err == nil {
		t.Fatalf("float parse should error on garbage")
	}
}

func TestNewLeafEditTextPath(t *testing.T) {
	e := newLeafEdit(leaf{path: "dns.host", value: "pi.hole"})
	if e.isBool {
		t.Fatalf("string leaf should not be a bool editor")
	}
	if e.input.Value() != "pi.hole" {
		t.Fatalf("editor should pre-fill the current value, got %q", e.input.Value())
	}
}

func TestLeafEditValueBoolAndText(t *testing.T) {
	b := newLeafEdit(leaf{path: "x", value: true})
	if v, err := b.value(); err != nil || v != true {
		t.Fatalf("bool editor value: got %v err %v", v, err)
	}
	txt := newLeafEdit(leaf{path: "x", value: "orig"})
	txt.input.SetValue("changed")
	if v, err := txt.value(); err != nil || v != "changed" {
		t.Fatalf("text editor value: got %v err %v", v, err)
	}
}

func TestLeafEditRenderContainsPath(t *testing.T) {
	th := theme.DeepNight()
	e := newLeafEdit(leaf{path: "dns.host", value: "pi.hole"})
	e.setWidth(80)
	if out := e.render(th, 80, 20); !strings.Contains(out, "dns.host") {
		t.Fatalf("leaf render should contain the path:\n%s", out)
	}
	// Bool variant renders the toggle hint.
	be := newLeafEdit(leaf{path: "dns.blocking.active", value: true})
	if out := be.render(th, 80, 20); !strings.Contains(out, "toggle") {
		t.Fatalf("bool leaf render should show the toggle hint:\n%s", out)
	}
}

func TestIsDetailLeaf(t *testing.T) {
	if !isDetailLeaf(map[string]any{"value": 1, "type": "integer"}) {
		t.Fatalf("value+type should be a detail leaf")
	}
	if isDetailLeaf(map[string]any{"value": 1}) {
		t.Fatalf("value without metadata is not a detail leaf")
	}
	if isDetailLeaf(map[string]any{"child": 1}) {
		t.Fatalf("no value key is not a detail leaf")
	}
}

// --- connection/theme helpers --------------------------------------------

func TestAuthLabelAllBranches(t *testing.T) {
	if got := authLabel(config.Instance{Password: "x"}); got != "inline" {
		t.Fatalf("inline password should label inline, got %q", got)
	}
	if got := authLabel(config.Instance{PasswordEnv: "PW"}); got != "env:PW" {
		t.Fatalf("env password should label env:NAME, got %q", got)
	}
	if got := authLabel(config.Instance{}); got != "—" {
		t.Fatalf("no auth should label a dash, got %q", got)
	}
}

func TestCloneConfigIsIndependent(t *testing.T) {
	orig := &config.Config{
		Active:    "primary",
		Theme:     theme.NameDeepNight,
		Refresh:   map[string]int{"queries": 5},
		Instances: []config.Instance{{Name: "primary", URL: "http://p"}},
	}
	clone := cloneConfig(orig)
	clone.Instances[0].Name = "changed"
	clone.Refresh["queries"] = 99
	if orig.Instances[0].Name != "primary" {
		t.Fatalf("clone should not mutate original instances")
	}
	if orig.Refresh["queries"] != 5 {
		t.Fatalf("clone should not mutate original refresh map")
	}
}

func TestThemePickerRenderMarksCurrent(t *testing.T) {
	th := theme.DeepNight()
	p := newThemePicker(theme.NameDeepNight)
	out := p.render(th, 60, 20, theme.NameDeepNight)
	for _, want := range []string{"Select theme", theme.NameDeepNight, "(current)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("theme picker render missing %q:\n%s", want, out)
		}
	}
}

func TestThemePickerBoundsAndSelected(t *testing.T) {
	p := newThemePicker(theme.NameDeepNight)
	p.up() // already at top; no move
	if p.cursor != 0 {
		t.Fatalf("up at top should stay at 0")
	}
	for i := 0; i < len(p.names)+3; i++ {
		p.down()
	}
	if p.cursor != len(p.names)-1 {
		t.Fatalf("down should clamp at the last entry")
	}
	if p.selected() != p.names[len(p.names)-1] {
		t.Fatalf("selected should return the cursor entry")
	}
}

// --- deferred patch/fetch closures ---------------------------------------

// runBatch drains a tea.Batch command, executing each sub-command, and returns
// the collected messages.
func runBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var out []tea.Msg
	for _, c := range batch {
		if c != nil {
			out = append(out, c())
		}
	}
	return out
}

func TestPatchClosureExecutesAgainstCancelledContext(t *testing.T) {
	m := newTestModel(t)
	cmd := m.patch(buildPatch("dns.blocking.active", false))
	// Pre-cancel the context the closure captured so the API call returns at
	// once without touching the network.
	if m.cancel != nil {
		m.cancel()
	}
	var sawPatch bool
	for _, msg := range runBatch(cmd) {
		if pm, ok := msg.(patchMsg); ok {
			sawPatch = true
			if pm.err == nil {
				t.Fatalf("cancelled patch should surface an error")
			}
		}
	}
	if !sawPatch {
		t.Fatalf("patch batch should yield a patchMsg")
	}
}

func TestFetchClosureExecutesAgainstCancelledContext(t *testing.T) {
	m := newTestModel(t)
	cmd := m.fetch()
	if m.cancel != nil {
		m.cancel()
	}
	var sawConfig bool
	for _, msg := range runBatch(cmd) {
		if cm, ok := msg.(configMsg); ok {
			sawConfig = true
			if cm.err == nil {
				t.Fatalf("cancelled fetch should surface an error")
			}
		}
	}
	if !sawConfig {
		t.Fatalf("fetch should yield a configMsg")
	}
}

// --- selectTheme save failure surfaces a banner without emitting ----------

func TestSelectThemeSaveFailureShowsBanner(t *testing.T) {
	m := bigModel(t)
	_, _ = m.Update(keyPress("c"))
	// Point the config path at an unwritable location so Save fails.
	m.ctx.ConfigPath = "/proc/nonexistent-dir/config.yaml"
	_, _ = m.Update(keyPress("t"))
	_, _ = m.Update(keyPress("j")) // pick a different theme
	_, cmd := m.Update(keyPress("enter"))
	if m.err == nil {
		t.Fatalf("a save failure should set the banner")
	}
	if cmd != nil {
		t.Fatalf("a failed theme save should not emit SetThemeMsg")
	}
}

var _ core.Screen = (*Model)(nil)
