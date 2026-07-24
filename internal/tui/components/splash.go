package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// splashArt is the startup wordmark: "tihole" in an ANSI-shadow block font. It
// is rendered with a top-to-bottom accent→allow gradient so the boot screen
// carries the same whimsy as the sidebar brand.
const splashArt = ` ████████╗██╗██╗  ██╗ ██████╗ ██╗     ███████╗
 ╚══██╔══╝██║██║  ██║██╔═══██╗██║     ██╔════╝
    ██║   ██║███████║██║   ██║██║     █████╗
    ██║   ██║██╔══██║██║   ██║██║     ██╔══╝
    ██║   ██║██║  ██║╚██████╔╝███████╗███████╗
    ╚═╝   ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝`

// Splash is the boot screen shown before the dashboard reveals. It draws the
// gradient wordmark, a tagline, and a status line, centered in the viewport.
type Splash struct {
	// Instance is the active instance name, shown under the wordmark.
	Instance string
	// Status is the current boot status line (e.g. a spinner + "connecting…").
	Status string
	Width  int
	Height int
}

// Render composes the centered boot screen on the theme surface.
func (s Splash) Render(th *theme.Theme) string {
	banner := verticalGradient(splashArt, th.Accent, th.Allow)

	tagline := lipgloss.NewStyle().Foreground(th.Subtle).
		Render("PiHole v6 · terminal control")

	parts := []string{banner, "", tagline}
	if s.Instance != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(th.Accent).Bold(true).
			Render("▸ "+s.Instance))
	}
	if s.Status != "" {
		parts = append(parts, "", lipgloss.NewStyle().Foreground(th.Subtle).Render(s.Status))
	}

	// A compact key legend so the essential shortcuts are learned on the way in,
	// not hunted for later.
	legend := lipgloss.NewStyle().Foreground(th.Subtle).Render(
		"ctrl+k palette · ? help · ⌃T theme · i splash")
	parts = append(parts, "", legend)

	block := lipgloss.JoinVertical(lipgloss.Center, parts...)
	centered := lipgloss.Place(s.Width, s.Height, lipgloss.Center, lipgloss.Center, block)
	return th.SurfaceStyle().Width(s.Width).Height(s.Height).Render(centered)
}

// verticalGradient colors each line of s by its row position, blending from
// `from` at the top to `to` at the bottom. A single line takes `from`.
func verticalGradient(s string, from, to color.Color) string {
	lines := strings.Split(s, "\n")
	n := len(lines)
	fr, fg, fb := rgb(from)
	tr, tg, tb := rgb(to)

	// Right-pad every line to a common width so ragged glyph rows (e.g. the "E"
	// in an ANSI-shadow font) keep their left edges aligned when centered.
	maxW := 0
	for _, line := range lines {
		if w := len([]rune(line)); w > maxW {
			maxW = w
		}
	}

	out := make([]string, n)
	for i, line := range lines {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		if pad := maxW - len([]rune(line)); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		col := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x",
			lerp(fr, tr, t), lerp(fg, tg, t), lerp(fb, tb, t)))
		out[i] = lipgloss.NewStyle().Foreground(col).Bold(true).Render(line)
	}
	return strings.Join(out, "\n")
}
