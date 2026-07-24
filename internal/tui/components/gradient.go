package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// gradientText renders s with a smooth per-rune color sweep from `from` to `to`.
// It is the small bit of whimsy the wordmark and gauges lean on; a single-rune
// (or empty) string simply takes the start color.
func gradientText(s string, from, to color.Color) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return s
	}
	fr, fg, fb := rgb(from)
	tr, tg, tb := rgb(to)

	var b strings.Builder
	for i, r := range runes {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		col := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x",
			lerp(fr, tr, t), lerp(fg, tg, t), lerp(fb, tb, t)))
		b.WriteString(lipgloss.NewStyle().Foreground(col).Bold(true).Render(string(r)))
	}
	return b.String()
}

// gradientWordmark is the sidebar brand: a gradient "tihole" that runs from the
// accent to the allow token when the rail is focused, and quiets to a subtle
// monochrome when the panel owns input.
func gradientWordmark(th *theme.Theme, focused bool) string {
	frame := lipgloss.NewStyle().Padding(0, 2).MarginBottom(1)
	if !focused {
		return frame.Foreground(th.Subtle).Bold(true).Render("● tihole")
	}
	return frame.Render(gradientText("● tihole", th.Accent, th.Allow))
}

// rgb unpacks a color.Color into 8-bit channels. lipgloss colors expose 16-bit
// premultiplied values via RGBA(); we downshift to the 0–255 range.
func rgb(c color.Color) (int, int, int) {
	if c == nil {
		return 0, 0, 0
	}
	r, g, b, _ := c.RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

// lerp linearly interpolates between two ints by t in [0,1].
func lerp(a, b int, t float64) int {
	return a + int(float64(b-a)*t)
}
