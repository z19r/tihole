package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// omarchyThemeDir returns the Omarchy current-theme directory under the user's
// home config, e.g. ~/.config/omarchy/current/theme.
func omarchyThemeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "omarchy", "current", "theme"), nil
}

// LoadOmarchy loads a Theme derived from the active Omarchy desktop theme by
// parsing ~/.config/omarchy/current/theme/alacritty.toml and mapping its ANSI
// palette onto tihole's semantic tokens. It returns an error if the Omarchy
// theme cannot be located or parsed; callers that want a graceful fallback
// should use Resolve("omarchy") instead.
func LoadOmarchy() (*Theme, error) {
	dir, err := omarchyThemeDir()
	if err != nil {
		return nil, err
	}
	return loadOmarchyFrom(dir)
}

// loadOmarchyFrom is the injectable core of LoadOmarchy: it reads
// alacritty.toml from dir and detects light mode via a light.mode marker file
// in the same directory. Tests point it at a temp dir.
func loadOmarchyFrom(dir string) (*Theme, error) {
	path := filepath.Join(dir, "alacritty.toml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("omarchy: reading %s: %w", path, err)
	}

	sections := parseTOML(string(raw))

	primary := sections["colors.primary"]
	normal := sections["colors.normal"]
	bright := sections["colors.bright"]

	bg := firstNonEmpty(primary["background"])
	fg := firstNonEmpty(primary["foreground"])
	if bg == "" || fg == "" {
		return nil, fmt.Errorf("omarchy: %s missing [colors.primary] background/foreground", path)
	}

	// light.mode marker beside the toml nudges derivation of ambiguous tokens.
	isLight := fileExists(filepath.Join(dir, "light.mode"))

	// Panel is a slight shift of Surface: lighter in dark mode, darker in light.
	panelDelta := 0.06
	borderDelta := 0.14
	if isLight {
		panelDelta = -0.05
		borderDelta = -0.16
	}

	red := firstNonEmpty(normal["red"], bright["red"])
	green := firstNonEmpty(normal["green"], bright["green"])
	yellow := firstNonEmpty(normal["yellow"], bright["yellow"])
	// A bright accent: prefer bright blue, then bright magenta, then normals.
	accent := firstNonEmpty(bright["blue"], bright["magenta"], normal["blue"], normal["magenta"])
	// Border from a neutral ANSI: normal white or bright black, else derived.
	border := firstNonEmpty(normal["white"], bright["black"])

	t := &Theme{
		Name:    "omarchy",
		Surface: lipgloss.Color(bg),
		Panel:   lipgloss.Color(shift(bg, panelDelta)),
		Text:    lipgloss.Color(fg),
		Subtle:  lipgloss.Color(mix(fg, bg, 0.45)),
		Accent:  lipgloss.Color(fallback(accent, fg)),
		Allow:   lipgloss.Color(fallback(green, fg)),
		Block:   lipgloss.Color(fallback(red, fg)),
		Warn:    lipgloss.Color(fallback(yellow, fg)),
		Border:  lipgloss.Color(fallback(border, shift(bg, borderDelta))),
	}
	return t, nil
}

// Resolve maps a theme name to a Theme, with graceful fallback semantics.
//
//   - "omarchy": attempt LoadOmarchy. On ANY failure it returns DeepNight()
//     together with the underlying error so the caller may log it; the returned
//     *Theme is always non-nil and usable, so callers may safely ignore err.
//   - a known built-in name: that built-in, nil error.
//   - anything else (including ""): DeepNight() as the default, nil error.
func Resolve(name string) (*Theme, error) {
	if name == "omarchy" {
		dir, err := omarchyThemeDir()
		if err != nil {
			return DeepNight(), err
		}
		return resolveFrom(name, dir)
	}
	if t, ok := Builtin(name); ok {
		return t, nil
	}
	return DeepNight(), nil
}

// resolveFrom is the injectable core of Resolve for the Omarchy path.
func resolveFrom(name, dir string) (*Theme, error) {
	if name == "omarchy" {
		t, err := loadOmarchyFrom(dir)
		if err != nil {
			return DeepNight(), err
		}
		return t, nil
	}
	if t, ok := Builtin(name); ok {
		return t, nil
	}
	return DeepNight(), nil
}

// --- minimal TOML parsing -------------------------------------------------

// parseTOML is a deliberately tiny TOML reader good enough for alacritty color
// files. It understands `[section]` headers and `key = "value"` /
// `key = 'value'` lines, ignores blank lines and `#` comments, and returns a
// map of section name -> key -> value. Hex values keep their leading '#'
// because the value is taken from inside the quotes.
func parseTOML(content string) map[string]map[string]string {
	sections := map[string]map[string]string{}
	current := ""
	sections[current] = map[string]string{}

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			if _, ok := sections[current]; !ok {
				sections[current] = map[string]string{}
			}
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := parseTOMLValue(strings.TrimSpace(line[eq+1:]))
		if key == "" {
			continue
		}
		sections[current][key] = val
	}
	return sections
}

// parseTOMLValue extracts a scalar string value: the contents of a quoted
// string, or a bare token with any trailing `#` comment stripped.
func parseTOMLValue(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') {
		quote := v[0]
		if end := strings.IndexByte(v[1:], quote); end >= 0 {
			return v[1 : 1+end]
		}
	}
	if hash := strings.IndexByte(v, '#'); hash > 0 {
		v = strings.TrimSpace(v[:hash])
	}
	return strings.Trim(v, "\"'")
}

// --- helpers --------------------------------------------------------------

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// parseHex parses "#rrggbb" (or "rrggbb") into r,g,b bytes. ok is false on any
// malformed input, letting callers fall back to the original string.
func parseHex(hex string) (r, g, b int, ok bool) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return 0, 0, 0, false
	}
	rv, err1 := strconv.ParseInt(h[0:2], 16, 0)
	gv, err2 := strconv.ParseInt(h[2:4], 16, 0)
	bv, err3 := strconv.ParseInt(h[4:6], 16, 0)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}

func clamp8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// shift lightens (delta>0) or darkens (delta<0) a hex color by a fraction of
// full range. Non-hex input is returned unchanged.
func shift(hex string, delta float64) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return hex
	}
	d := int(delta * 255)
	return fmt.Sprintf("#%02x%02x%02x", clamp8(r+d), clamp8(g+d), clamp8(b+d))
}

// mix blends a toward b by t in [0,1]. Non-hex input returns a unchanged.
func mix(a, b string, t float64) string {
	ar, ag, ab, ok1 := parseHex(a)
	br, bg, bb, ok2 := parseHex(b)
	if !ok1 || !ok2 {
		return a
	}
	lerp := func(x, y int) int { return clamp8(x + int(float64(y-x)*t)) }
	return fmt.Sprintf("#%02x%02x%02x", lerp(ar, br), lerp(ag, bg), lerp(ab, bb))
}
