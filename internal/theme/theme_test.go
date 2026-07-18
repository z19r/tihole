package theme

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// tokensNonNil asserts every semantic token on a theme is populated.
func tokensNonNil(t *testing.T, th *Theme) {
	t.Helper()
	tokens := map[string]any{
		"Surface": th.Surface, "Panel": th.Panel, "Text": th.Text,
		"Subtle": th.Subtle, "Accent": th.Accent, "Allow": th.Allow,
		"Block": th.Block, "Warn": th.Warn, "Border": th.Border,
	}
	for name, tok := range tokens {
		if tok == nil {
			t.Errorf("token %s is nil", name)
		}
	}
}

func TestDeepNightHasNonNilTokensAndName(t *testing.T) {
	// Arrange / Act
	th := DeepNight()

	// Assert
	if th.Name != NameDeepNight {
		t.Errorf("expected name %q, got %q", NameDeepNight, th.Name)
	}
	tokensNonNil(t, th)
}

func TestLightLuxuryHasNonNilTokensAndName(t *testing.T) {
	th := LightLuxury()

	if th.Name != NameLightLuxury {
		t.Errorf("expected name %q, got %q", NameLightLuxury, th.Name)
	}
	tokensNonNil(t, th)
}

func TestPiholeClassicHasNonNilTokensAndName(t *testing.T) {
	th := PiholeClassic()

	if th.Name != NamePiholeClassic {
		t.Errorf("expected name %q, got %q", NamePiholeClassic, th.Name)
	}
	tokensNonNil(t, th)
}

func TestBuiltinReturnsKnownThemeByName(t *testing.T) {
	// Arrange / Act
	th, ok := Builtin(NamePiholeClassic)

	// Assert
	if !ok {
		t.Fatalf("expected Builtin(%q) to be found", NamePiholeClassic)
	}
	if th.Name != NamePiholeClassic {
		t.Errorf("expected %q, got %q", NamePiholeClassic, th.Name)
	}
}

func TestBuiltinReturnsFalseForUnknownName(t *testing.T) {
	th, ok := Builtin("no-such-theme")

	if ok || th != nil {
		t.Errorf(
			"expected (nil, false) for unknown theme, got (%v, %v)",
			th,
			ok,
		)
	}
}

func TestBuiltinReturnsFreshInstances(t *testing.T) {
	// Arrange / Act: two lookups of the same name.
	a, _ := Builtin(NameDeepNight)
	b, _ := Builtin(NameDeepNight)

	// Assert: distinct pointers so callers cannot mutate a shared theme.
	if a == b {
		t.Error(
			"expected Builtin to return fresh instances, got shared pointer",
		)
	}
}

func TestGlossHasNonNilTokensAndName(t *testing.T) {
	th := Gloss()

	if th.Name != NameGloss {
		t.Errorf("expected name %q, got %q", NameGloss, th.Name)
	}
	tokensNonNil(t, th)

	// Gloss carries the signature multi-stop ramp; GradientStops must surface
	// it.
	if len(th.Ramp) < 2 {
		t.Errorf(
			"expected Gloss to define a multi-stop ramp, got %d stops",
			len(th.Ramp),
		)
	}
	if got := th.GradientStops(); len(got) != len(th.Ramp) {
		t.Errorf(
			"GradientStops should return the ramp (%d stops), got %d",
			len(th.Ramp),
			len(got),
		)
	}
}

func TestGradientStopsFallsBackToAccentAllow(t *testing.T) {
	// A single-hue theme (no Ramp) falls back to a two-stop Accent→Allow blend.
	th := DeepNight()
	got := th.GradientStops()

	if len(got) != 2 {
		t.Fatalf("expected 2 fallback stops, got %d", len(got))
	}
	if got[0] != th.Accent || got[1] != th.Allow {
		t.Errorf("expected [Accent, Allow] fallback, got %v", got)
	}
}

func TestNamesListsAllBuiltinsInOrder(t *testing.T) {
	want := []string{
		NameAuto,
		NameGloss,
		NameDeepNight,
		NameLightLuxury,
		NamePiholeClassic,
	}

	got := Names()

	if len(got) != len(want) {
		t.Fatalf("expected %d names, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("names[%d]: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestStyleBuildersUseTokens(t *testing.T) {
	// Arrange
	th := DeepNight()

	// Act: build every style helper. This is a smoke test that they compile
	// and derive from the theme without panicking.
	styles := []any{
		th.SurfaceStyle(), th.PanelStyle(), th.TextStyle(), th.SubtleStyle(),
		th.AccentStyle(), th.AllowStyle(), th.BlockStyle(), th.WarnStyle(),
		th.BorderStyle(),
	}

	// Assert
	if len(styles) != 9 {
		t.Fatalf("expected 9 style builders")
	}
}

func TestAdaptReturnsReceiverForCommittedTheme(t *testing.T) {
	// Arrange: built-in themes are single-mode, adapt is nil.
	th := DeepNight()

	// Act
	got := th.Adapt(false)

	// Assert
	if got != th {
		t.Error("expected Adapt to return the same committed theme")
	}
}

func TestAdaptConsultsCustomAdapter(t *testing.T) {
	// Arrange: a theme carrying a mode-specific adapter.
	dark := DeepNight()
	light := LightLuxury()
	th := &Theme{Name: "pair", adapt: func(isDark bool) *Theme {
		if isDark {
			return dark
		}
		return light
	}}

	// Act / Assert
	if th.Adapt(true) != dark {
		t.Error("expected dark variant when isDark=true")
	}
	if th.Adapt(false) != light {
		t.Error("expected light variant when isDark=false")
	}
}

func TestPickChoosesByBackground(t *testing.T) {
	// Arrange
	th := DeepNight()

	// Act
	darkPick := Pick(true, th.Surface, th.Panel)
	lightPick := Pick(false, th.Surface, th.Panel)

	// Assert: dark background -> the dark (second) argument, and vice versa.
	if darkPick != th.Panel {
		t.Error("expected dark pick to select the dark color")
	}
	if lightPick != th.Surface {
		t.Error("expected light pick to select the light color")
	}
}

// writeOmarchyFixture writes a sample alacritty.toml (Tokyo Night palette) into
// dir and returns the known-good hex values for assertions.
func writeOmarchyFixture(
	t *testing.T,
	dir string,
) (bg, fg, red, green, yellow, brightBlue string) {
	t.Helper()
	bg, fg = "#1a1b26", "#c0caf5"
	red, green, yellow = "#f7768e", "#9ece6a", "#e0af68"
	brightBlue = "#7aa2f7"
	content := `# Tokyo Night alacritty colors
[colors.primary]
background = "#1a1b26"
foreground = "#c0caf5"  # main fg

[colors.normal]
black   = '#15161e'
red     = "#f7768e"
green   = "#9ece6a"
yellow  = "#e0af68"
blue    = "#7aa2f7"
magenta = "#bb9af7"
cyan    = "#7dcfff"
white   = "#a9b1d6"

[colors.bright]
black   = "#414868"
blue    = "#7aa2f7"
magenta = "#bb9af7"
`
	path := filepath.Join(dir, "alacritty.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return
}

func TestParseTOMLReadsSectionsAndKeys(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeOmarchyFixture(t, dir)
	raw, _ := os.ReadFile(filepath.Join(dir, "alacritty.toml"))

	// Act
	sections := parseTOML(string(raw))

	// Assert
	if got := sections["colors.primary"]["background"]; got != "#1a1b26" {
		t.Errorf("background: expected #1a1b26, got %q", got)
	}
	if got := sections["colors.primary"]["foreground"]; got != "#c0caf5" {
		t.Errorf("foreground with trailing comment mis-parsed: got %q", got)
	}
	if got := sections["colors.normal"]["red"]; got != "#f7768e" {
		t.Errorf("normal.red: expected #f7768e, got %q", got)
	}
	if got := sections["colors.normal"]["black"]; got != "#15161e" {
		t.Errorf("single-quoted value mis-parsed: got %q", got)
	}
}

func TestLoadOmarchyMapsAnsiToSemanticTokens(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	bg, fg, red, green, yellow, brightBlue := writeOmarchyFixture(t, dir)

	// Act
	th, err := loadOmarchyFrom(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert: ANSI -> semantic mapping.
	assertColor(t, "Block<-red", th.Block, red)
	assertColor(t, "Allow<-green", th.Allow, green)
	assertColor(t, "Warn<-yellow", th.Warn, yellow)
	assertColor(t, "Surface<-bg", th.Surface, bg)
	assertColor(t, "Text<-fg", th.Text, fg)
	assertColor(t, "Accent<-brightBlue", th.Accent, brightBlue)
	if th.Name != "omarchy" {
		t.Errorf("expected name omarchy, got %q", th.Name)
	}
	// Panel must be derived (a shift of Surface), not equal to Surface.
	if colorHex(th.Panel) == bg {
		t.Error(
			"expected Panel to be a derived shift of Surface, not identical",
		)
	}
}

func TestLoadOmarchyDetectsLightModeMarker(t *testing.T) {
	// Arrange: two dirs, one with a light.mode marker.
	darkDir := t.TempDir()
	writeOmarchyFixture(t, darkDir)
	lightDir := t.TempDir()
	writeOmarchyFixture(t, lightDir)
	if err := os.WriteFile(filepath.Join(lightDir, "light.mode"), nil, 0o644); err != nil {
		t.Fatalf("writing marker: %v", err)
	}

	// Act
	darkTheme, err1 := loadOmarchyFrom(darkDir)
	lightTheme, err2 := loadOmarchyFrom(lightDir)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v %v", err1, err2)
	}

	// Assert: light.mode flips the sign of the Panel derivation, so the two
	// Panels differ even though Surface is identical.
	if colorHex(darkTheme.Panel) == colorHex(lightTheme.Panel) {
		t.Error("expected light.mode marker to change Panel derivation")
	}
}

func TestLoadOmarchyErrorsWhenFileAbsent(t *testing.T) {
	// Arrange: empty dir, no alacritty.toml.
	dir := t.TempDir()

	// Act
	_, err := loadOmarchyFrom(dir)

	// Assert
	if err == nil {
		t.Error("expected error when alacritty.toml is absent")
	}
}

func TestResolveOmarchyFallsBackToDeepNightWhenAbsent(t *testing.T) {
	// Arrange: empty dir simulates Omarchy not installed.
	dir := t.TempDir()

	// Act
	th, err := resolveFrom("omarchy", dir)

	// Assert: fallback theme is DeepNight, and the error is surfaced for logs.
	if th == nil || th.Name != NameDeepNight {
		t.Errorf("expected DeepNight fallback, got %v", th)
	}
	if err == nil {
		t.Error("expected the underlying load error to be returned for logging")
	}
}

func TestResolveOmarchySucceedsWhenPresent(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeOmarchyFixture(t, dir)

	// Act
	th, err := resolveFrom("omarchy", dir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if th.Name != "omarchy" {
		t.Errorf("expected omarchy theme, got %q", th.Name)
	}
}

func TestResolveReturnsBuiltinByName(t *testing.T) {
	th, err := Resolve(NameLightLuxury)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if th.Name != NameLightLuxury {
		t.Errorf("expected %q, got %q", NameLightLuxury, th.Name)
	}
}

func TestResolveUnknownNameDefaultsToAuto(t *testing.T) {
	// Arrange / Act
	th, err := Resolve("totally-unknown")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if th.Name != NameAuto {
		t.Errorf("expected Auto default, got %q", th.Name)
	}
}

func TestAutoAdaptsToBackground(t *testing.T) {
	// Arrange
	th := Auto()

	// Assert: base renders as the safe dark default before detection.
	if th.Name != NameAuto {
		t.Errorf("expected name %q, got %q", NameAuto, th.Name)
	}
	tokensNonNil(t, th)

	// Act / Assert: Adapt resolves to a concrete single-mode theme per
	// background.
	// Dark terminals land on Gloss, tihole's signature default look.
	if got := th.Adapt(true); got.Name != NameGloss {
		t.Errorf("dark background: expected %q, got %q", NameGloss, got.Name)
	}
	if got := th.Adapt(false); got.Name != NameLightLuxury {
		t.Errorf(
			"light background: expected %q, got %q",
			NameLightLuxury,
			got.Name,
		)
	}
}

func TestLoadOmarchyUsesHomeConfigDir(t *testing.T) {
	// Arrange: fake HOME with a full omarchy theme dir.
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "omarchy", "current", "theme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeOmarchyFixture(t, dir)
	t.Setenv("HOME", home)

	// Act
	th, err := LoadOmarchy()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if th.Name != "omarchy" {
		t.Errorf("expected omarchy theme, got %q", th.Name)
	}
}

func TestResolveOmarchyEndToEndFallsBackWhenAbsent(t *testing.T) {
	// Arrange: HOME with no omarchy config at all.
	t.Setenv("HOME", t.TempDir())

	// Act
	th, err := Resolve("omarchy")

	// Assert: usable DeepNight fallback plus a loggable error.
	if th.Name != NameDeepNight {
		t.Errorf("expected DeepNight fallback, got %q", th.Name)
	}
	if err == nil {
		t.Error("expected an error to surface for logging")
	}
}

func TestParseTOMLValueHandlesUnquotedAndComments(t *testing.T) {
	cases := map[string]string{
		`"#abcdef"`:      "#abcdef",
		`'#abcdef'`:      "#abcdef",
		`bare`:           "bare",
		`token # note`:   "token",
		`"open`:          "open", // unterminated quote -> trimmed
		`123 # trailing`: "123",
	}
	for in, want := range cases {
		if got := parseTOMLValue(in); got != want {
			t.Errorf("parseTOMLValue(%q): expected %q, got %q", in, want, got)
		}
	}
}

func TestShiftAndMixIgnoreNonHex(t *testing.T) {
	if got := shift("not-a-hex", 0.1); got != "not-a-hex" {
		t.Errorf("shift non-hex: expected passthrough, got %q", got)
	}
	if got := mix("nope", "#000000", 0.5); got != "nope" {
		t.Errorf("mix non-hex a: expected passthrough, got %q", got)
	}
	if got := mix("#ffffff", "bad", 0.5); got != "#ffffff" {
		t.Errorf("mix non-hex b: expected passthrough of a, got %q", got)
	}
}

func TestShiftClampsAtBounds(t *testing.T) {
	// Lightening white stays white; darkening black stays black.
	if got := shift("#ffffff", 0.5); got != "#ffffff" {
		t.Errorf("shift clamp high: got %q", got)
	}
	if got := shift("#000000", -0.5); got != "#000000" {
		t.Errorf("shift clamp low: got %q", got)
	}
}

func TestParseHexRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"#fff", "#gggggg", "", "12345"} {
		if _, _, _, ok := parseHex(bad); ok {
			t.Errorf("parseHex(%q): expected ok=false", bad)
		}
	}
}

func TestFallbackAndFirstNonEmpty(t *testing.T) {
	if got := fallback("", "def"); got != "def" {
		t.Errorf("fallback empty: got %q", got)
	}
	if got := fallback("val", "def"); got != "val" {
		t.Errorf("fallback present: got %q", got)
	}
	if got := firstNonEmpty("", "  ", "x"); got != "x" {
		t.Errorf("firstNonEmpty: got %q", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("firstNonEmpty all empty: got %q", got)
	}
}

// --- color assertion helpers ---------------------------------------------

// colorHex renders a color.Color as "#rrggbb". lipgloss.Color returns a
// color.RGBA, so we read the 8-bit channels via RGBA() (which returns 16-bit
// premultiplied values) and downscale.
func colorHex(c any) string {
	col, ok := c.(color.Color)
	if !ok || col == nil {
		return ""
	}
	r, g, b, _ := col.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func assertColor(t *testing.T, label string, got any, wantHex string) {
	t.Helper()
	if h := colorHex(got); h != wantHex {
		t.Errorf("%s: expected %q, got %q", label, wantHex, h)
	}
}
