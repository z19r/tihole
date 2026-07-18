package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

// gradientText renders s with a smooth per-rune color sweep from `from` to
// `to`.
// It is the small bit of whimsy the wordmark and gauges lean on; a single-rune
// (or empty) string simply takes the start color.
func gradientText(s string, from, to color.Color) string {
	return gradientRamp(s, []color.Color{from, to})
}

// gradientRamp renders s with a per-rune sweep across an arbitrary number of
// color stops. Two stops reduce to a plain linear blend; more stops chain
// piecewise so a brand ramp (pink→magenta→purple→acid→cyan) reads smoothly
// across a wordmark. An empty string, or an empty stop set, is returned as-is.
func gradientRamp(s string, stops []color.Color) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 || len(stops) == 0 {
		return s
	}
	if len(stops) == 1 {
		return lipgloss.NewStyle().Foreground(stops[0]).Bold(true).Render(s)
	}

	var b strings.Builder
	for i, r := range runes {
		col := rampColorAt(stops, positionOf(i, n))
		b.WriteString(
			lipgloss.NewStyle().Foreground(col).Bold(true).Render(string(r)),
		)
	}
	return b.String()
}

// rampColorAt samples a piecewise-linear gradient defined by stops at position
// t in [0,1]: t maps across len(stops)-1 equal segments, blending the two stops
// that bracket it. Out-of-range t clamps to the end stops.
func rampColorAt(stops []color.Color, t float64) color.Color {
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	segs := len(stops) - 1
	x := t * float64(segs)
	i := int(x)
	if i >= segs {
		i = segs - 1
	}
	local := x - float64(i)
	ar, ag, ab := rgb(stops[i])
	br, bg, bb := rgb(stops[i+1])
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x",
		lerp(ar, br, local), lerp(ag, bg, local), lerp(ab, bb, local)))
}

// positionOf is the [0,1] position of index i within n items; a single item
// sits at the start.
func positionOf(i, n int) float64 {
	if n <= 1 {
		return 0
	}
	return float64(i) / float64(n-1)
}

// gradientWordmark is the sidebar brand: a gradient "tihole" that sweeps the
// theme's signature ramp when the rail is focused, and quiets to a subtle
// monochrome when the panel owns input.
func gradientWordmark(th *theme.Theme, focused bool) string {
	frame := lipgloss.NewStyle().
		Padding(0, 2).
		MarginBottom(1).
		Background(th.Panel)
	if !focused {
		return frame.Foreground(th.Subtle).Bold(true).Render("● tihole")
	}
	return frame.Render(gradientRamp("● tihole", th.GradientStops()))
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
