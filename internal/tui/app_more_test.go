package tui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
	"github.com/zackkitzmiller/tihole/internal/tui/core"
)

// failClient points at a closed port so API calls fail fast with a connection
// error, letting us exercise the error branches of the command closures without
// any real network.
func failClient() *pihole.Client {
	return pihole.New("http://127.0.0.1:1", "pw")
}

func TestInitReturnsCommand(t *testing.T) {
	m := newTestModel(t)
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init should return a batch command")
	}
}

func TestNavigateMsgSwitchesPageAndFocuses(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)

	updated, cmd := m.Update(core.NavigateMsg{To: core.PageSystem})
	m = updated.(*AppModel)

	if m.active != core.PageSystem {
		t.Fatalf("expected System active after NavigateMsg, got %v", m.active)
	}
	if cmd == nil {
		t.Fatal("switching page should return the new screen's Focus command")
	}
}

func TestNavigateMsgToSamePageIsNoop(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)

	_, cmd := m.Update(core.NavigateMsg{To: core.PageDashboard})
	if cmd != nil {
		t.Fatal("navigating to the active page should be a no-op (nil cmd)")
	}
}

func TestDigitKeyJumpsToEachPage(t *testing.T) {
	order := core.PageOrder()
	for i, p := range order {
		if i >= 9 {
			break
		}
		m := sized(t, newTestModel(t), 120, 36)
		d := rune('1' + i)
		updated, _ := m.Update(keyPress(string(d), d))
		m = updated.(*AppModel)
		if m.active != p.ID {
			t.Fatalf("digit %c expected page %v, got %v", d, p.ID, m.active)
		}
	}
}

func TestNavArrowsCycleForwardWithWrap(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	order := core.PageOrder()

	// On the rail, ↓ walks the sidebar and wraps back to the top.
	for i := 1; i <= len(order); i++ {
		updated, _ := m.Update(keyPress("down", tea.KeyDown))
		m = updated.(*AppModel)
		want := order[i%len(order)].ID
		if m.active != want {
			t.Fatalf("after %d downs expected %v, got %v", i, want, m.active)
		}
	}
	if m.active != core.PageDashboard {
		t.Fatalf("nav cycle should wrap to Dashboard, got %v", m.active)
	}
}

func TestNavArrowsCycleBackwardWithWrap(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	order := core.PageOrder()

	for i := 1; i <= len(order); i++ {
		updated, _ := m.Update(keyPress("up", tea.KeyUp))
		m = updated.(*AppModel)
		want := order[(len(order)-i)%len(order)].ID
		if m.active != want {
			t.Fatalf("after %d ups expected %v, got %v", i, want, m.active)
		}
	}
	if m.active != core.PageDashboard {
		t.Fatalf("nav cycle should wrap to Dashboard, got %v", m.active)
	}
}

func TestPanelFocusDelegatesNonEscToScreen(t *testing.T) {
	// Arrange: move onto an interactive screen, then descend into the panel.
	m := sized(t, newTestModel(t), 120, 36)
	updated, _ := m.Update(keyPress("down", tea.KeyDown)) // -> Query Log
	m = updated.(*AppModel)
	updated, _ = m.Update(keyPress("tab", tea.KeyTab))
	m = updated.(*AppModel)

	// Act: a page-jump digit is NOT a global in panel focus — it falls through
	// to the screen, so the active page is unchanged.
	before := m.active
	updated, _ = m.Update(keyPress("3", '3'))
	m = updated.(*AppModel)

	// Assert
	if m.active != before {
		t.Fatalf("panel focus must not honor digit page-jumps, moved to %v", m.active)
	}
	if m.focus != focusPanel {
		t.Fatalf("expected to remain in panel focus, got %v", m.focus)
	}
}

func TestSetThemeMsgSwapsTheme(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	other, err := theme.Resolve(theme.NameLightLuxury)
	if err != nil {
		t.Fatalf("resolve theme: %v", err)
	}

	updated, _ := m.Update(core.SetThemeMsg{Theme: other})
	m = updated.(*AppModel)

	if m.ctx.Theme.Name != theme.NameLightLuxury {
		t.Fatalf("expected theme %q, got %q", theme.NameLightLuxury, m.ctx.Theme.Name)
	}
}

func TestSwitchInstanceMsgSameNameNoop(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)

	_, cmd := m.Update(core.SwitchInstanceMsg{Name: "home"})
	if cmd != nil {
		t.Fatal("switching to the active instance should be a no-op")
	}
	if m.ctx.InstanceName != "home" {
		t.Fatalf("instance name should stay home, got %q", m.ctx.InstanceName)
	}
}

func TestErrorMsgSetsBanner(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)

	updated, _ := m.Update(core.ErrorMsg{Screen: "x", Err: errBoom{}})
	m = updated.(*AppModel)

	if m.err != "boom" {
		t.Fatalf("expected err 'boom', got %q", m.err)
	}
}

func TestErrorMsgNilErrIgnored(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)

	updated, _ := m.Update(core.ErrorMsg{Screen: "x", Err: nil})
	m = updated.(*AppModel)

	if m.err != "" {
		t.Fatalf("nil ErrorMsg should not set banner, got %q", m.err)
	}
}

func TestBlockingResultWithTimerSetsCountdown(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	secs := 42.0

	updated, _ := m.Update(blockingResultMsg{status: pihole.BlockingStatus{Blocking: false, Timer: &secs}})
	m = updated.(*AppModel)

	if !m.block.Known {
		t.Fatal("blocking state should be known")
	}
	if m.block.CountdownSecs != 42 {
		t.Fatalf("countdown = %d, want 42", m.block.CountdownSecs)
	}
}

func TestSwitchInstanceMsgSurvivesConfigChange(t *testing.T) {
	// Active instance still present: config swapped, nothing else changes.
	m := sized(t, newTestModel(t), 120, 36)
	verify := true
	cfg2 := &config.Config{
		Active: "home",
		Instances: []config.Instance{
			{Name: "home", URL: "http://192.168.1.2", Password: "pw", VerifyTLS: &verify},
			{Name: "office", URL: "http://10.0.0.2", Password: "pw2", VerifyTLS: &verify},
			{Name: "cabin", URL: "http://10.0.0.3", Password: "pw3", VerifyTLS: &verify},
		},
	}

	updated, cmd := m.Update(core.InstancesChangedMsg{Config: cfg2})
	m = updated.(*AppModel)

	if m.cfg != cfg2 || m.ctx.Config != cfg2 {
		t.Fatal("config should be swapped in")
	}
	if m.ctx.InstanceName != "home" {
		t.Fatalf("active instance should remain home, got %q", m.ctx.InstanceName)
	}
	if cmd != nil {
		t.Fatal("surviving active instance should not trigger a switch cmd")
	}
}

func TestApplyInstancesChangeNilAndEmpty(t *testing.T) {
	m := newTestModel(t)
	before := m.cfg

	if cmd := m.applyInstancesChange(nil); cmd != nil {
		t.Fatal("nil config should be a no-op")
	}
	if m.cfg != before {
		t.Fatal("nil config must not swap the config")
	}

	empty := &config.Config{Active: "home"}
	if cmd := m.applyInstancesChange(empty); cmd != nil {
		t.Fatal("empty instance list should not switch")
	}
	if m.cfg != empty {
		t.Fatal("empty config should still be swapped in")
	}
}

func TestApplyInstancesChangeDropsActiveInstance(t *testing.T) {
	// The active instance ("home") is dropped; the surviving instance is
	// re-activated. Auth is lazy, so re-activation proceeds without blocking or
	// synchronously erroring even if the URL is unreachable.
	m := sized(t, newTestModel(t), 120, 36)
	verify := true
	cfg2 := &config.Config{
		Active: "office",
		Instances: []config.Instance{
			{Name: "office", URL: "http://127.0.0.1:1", Password: "pw2", VerifyTLS: &verify},
		},
	}

	m.applyInstancesChange(cfg2)

	if m.cfg != cfg2 {
		t.Fatal("config should be swapped in even when active is dropped")
	}
	if m.ctx.InstanceName != "office" {
		t.Fatalf("surviving instance should be re-activated, got %q", m.ctx.InstanceName)
	}
	if m.err != "" {
		t.Fatalf("re-activation should not synchronously error, got %q", m.err)
	}
}

func TestSwitchInstanceProceedsWithoutBlocking(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	// Even pointed at a closed port, switching must not block on auth: it
	// activates the instance and returns a fetch command; failures surface later.
	m.cfg.Instances[1].URL = "http://127.0.0.1:1"

	cmd := m.switchInstance("office")

	if cmd == nil {
		t.Fatal("switch should return a (fetch) command, not nil")
	}
	if m.err != "" {
		t.Fatalf("switch should not synchronously error, got %q", m.err)
	}
	if m.ctx.InstanceName != "office" {
		t.Fatalf("instance name should advance to office, got %q", m.ctx.InstanceName)
	}
}

func TestSwitchInstanceStructuralErrorSetsBanner(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	// Give office an unset password_env: clientFor can't resolve it, which is a
	// structural config error that must still surface as a banner (not a crash),
	// leaving the active instance unchanged.
	m.cfg.Instances[1].Password = ""
	m.cfg.Instances[1].PasswordEnv = "TIHOLE_TEST_UNSET_PW"

	cmd := m.switchInstance("office")

	if cmd != nil {
		t.Fatal("structural failure should return nil cmd")
	}
	if m.err == "" {
		t.Fatal("structural config error should surface a banner")
	}
	if m.ctx.InstanceName != "home" {
		t.Fatalf("active instance should remain home, got %q", m.ctx.InstanceName)
	}
}

func TestThemeCmdValidResolvesTheme(t *testing.T) {
	msg := themeCmd(theme.NameLightLuxury)()

	setMsg, ok := msg.(core.SetThemeMsg)
	if !ok {
		t.Fatalf("expected core.SetThemeMsg, got %T", msg)
	}
	if setMsg.Theme == nil || setMsg.Theme.Name != theme.NameLightLuxury {
		t.Fatalf("unexpected theme in SetThemeMsg: %+v", setMsg.Theme)
	}
}

func TestThemeCmdUnknownFallsBackToDefault(t *testing.T) {
	// theme.Resolve returns the Auto default (no error) for unknown names, so
	// themeCmd emits a SetThemeMsg carrying that fallback rather than an error.
	msg := themeCmd("does-not-exist")()

	setMsg, ok := msg.(core.SetThemeMsg)
	if !ok {
		t.Fatalf("expected core.SetThemeMsg fallback, got %T", msg)
	}
	if setMsg.Theme == nil || setMsg.Theme.Name != theme.NameAuto {
		t.Fatalf("expected Auto fallback, got %+v", setMsg.Theme)
	}
}

func TestPaletteCommandRunsEmitExpectedMessages(t *testing.T) {
	m := newTestModel(t)
	cmds := m.paletteCommands()

	var navRun, themeRun, instRun tea.Cmd
	for _, c := range cmds {
		switch {
		case navRun == nil && strings.HasPrefix(c.Title, "Go to "):
			navRun = c.Run
		case themeRun == nil && strings.HasPrefix(c.Title, "Theme: "):
			themeRun = c.Run
		case instRun == nil && strings.HasPrefix(c.Title, "Switch to "):
			instRun = c.Run
		}
	}
	if navRun == nil || themeRun == nil || instRun == nil {
		t.Fatalf("missing palette commands: nav=%v theme=%v inst=%v", navRun != nil, themeRun != nil, instRun != nil)
	}

	if _, ok := navRun().(core.NavigateMsg); !ok {
		t.Fatal("nav command should emit core.NavigateMsg")
	}
	if _, ok := themeRun().(core.SetThemeMsg); !ok {
		t.Fatal("theme command should emit core.SetThemeMsg")
	}
	if _, ok := instRun().(core.SwitchInstanceMsg); !ok {
		t.Fatal("instance command should emit core.SwitchInstanceMsg")
	}
}

func TestNormalRenderShowsChrome(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	// The gradient wordmark colors each rune separately, so strip ANSI before
	// matching plain text.
	out := ansi.Strip(m.View().Content)

	for _, want := range []string{"tihole", "Dashboard", "home", "Query Log", "Settings"} {
		if !strings.Contains(out, want) {
			t.Errorf("normal render missing %q", want)
		}
	}
}

func TestViewZeroSizeReturnsEmptyAltScreen(t *testing.T) {
	m := newTestModel(t) // no WindowSizeMsg yet -> width/height 0
	v := m.View()
	if v.Content != "" {
		t.Fatalf("expected empty content before sizing, got %q", v.Content)
	}
	if !v.AltScreen {
		t.Fatal("view should request the alt screen")
	}
}

func TestToggleBlockKeyReturnsCommand(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)

	cmd, handled := m.handleKey(keyPress("d", 'd'))
	if !handled {
		t.Fatal("toggle-block key should be handled")
	}
	if cmd == nil {
		t.Fatal("toggle-block key should return a command")
	}
}

func TestSwitchInstanceKeyIsHandled(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	// Point the next instance at a closed port. Switching must NOT block on
	// authentication — the client authenticates lazily, so the switch proceeds
	// and a failure surfaces later as a fetch banner, not a synchronous crash.
	m.cfg.Instances[1].URL = "http://127.0.0.1:1"

	_, handled := m.handleKey(keyPress("s", 's'))
	if !handled {
		t.Fatal("switch-instance key should be handled")
	}
	if m.ctx.InstanceName != "office" {
		t.Fatalf("switch should activate the next instance, got %q", m.ctx.InstanceName)
	}
	if m.err != "" {
		t.Fatalf("switch should not synchronously error, got %q", m.err)
	}
}

func TestContentSizeClampsNegative(t *testing.T) {
	m := sized(t, newTestModel(t), 5, 1)

	cw, ch := m.contentSize()
	if cw != 0 {
		t.Fatalf("width should clamp to 0, got %d", cw)
	}
	if ch != 0 {
		t.Fatalf("height should clamp to 0, got %d", ch)
	}
}

func TestNextInstanceSingleInstance(t *testing.T) {
	m := newTestModel(t)
	m.cfg.Instances = m.cfg.Instances[:1] // only "home" remains

	if got := m.nextInstance(); got != "home" {
		t.Fatalf("single instance should return itself, got %q", got)
	}
}

func TestActiveIndexUnknownPageFallsBackToZero(t *testing.T) {
	if got := activeIndex(core.PageID(999)); got != 0 {
		t.Fatalf("unknown page should map to index 0, got %d", got)
	}
}

func TestLoadOrBootstrapLoadsExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/tihole.yaml"
	src := newTestModel(t).cfg
	if err := config.Save(path, src); err != nil {
		t.Fatalf("save config: %v", err)
	}

	got, err := loadOrBootstrap(path)
	if err != nil {
		t.Fatalf("loadOrBootstrap: %v", err)
	}
	if got.Active != "home" || len(got.Instances) != 2 {
		t.Fatalf("unexpected loaded config: %+v", got)
	}
}

func TestLoadOrBootstrapReportsLoadError(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/broken.yaml"
	if err := os.WriteFile(path, []byte("this: : not: valid: yaml"), 0o600); err != nil {
		t.Fatalf("write broken config: %v", err)
	}

	if _, err := loadOrBootstrap(path); err == nil {
		t.Fatal("expected an error loading a malformed config")
	}
}

func TestBlockingCommandClosuresReportErrors(t *testing.T) {
	m := sized(t, newTestModel(t), 120, 36)
	m.ctx.API = failClient()

	cases := map[string]tea.Cmd{
		"toggle":        m.toggleBlocking(),
		"disable-30s":   m.setBlockingFor(30),
		"disable-indef": m.setBlockingFor(0),
		"fetch":         fetchBlocking(m.ctx.API),
	}
	for name, cmd := range cases {
		if cmd == nil {
			t.Fatalf("%s: command should not be nil", name)
		}
		if _, ok := cmd().(blockingErrMsg); !ok {
			t.Fatalf("%s: expected blockingErrMsg from a failing client", name)
		}
	}
}
