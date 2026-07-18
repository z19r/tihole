package dashboard

import "strconv"

// barFilled and barEmpty are the glyphs used by miniBar for the drawn and
// undrawn portions of a horizontal meter.
const (
	barFilled = '█'
	barEmpty  = '░'
)

// miniBar renders a horizontal meter width runes wide, filled proportionally to
// fraction (clamped to 0..1). A width <= 0 yields "". The fraction is rounded
// to
// the nearest cell so a small non-zero value still shows at least one filled
// cell once it rounds up.
func miniBar(fraction float64, width int) string {
	if width <= 0 {
		return ""
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	filled := int(fraction*float64(width) + 0.5)
	if filled > width {
		filled = width
	}

	out := make([]rune, width)
	for i := 0; i < width; i++ {
		if i < filled {
			out[i] = barFilled
		} else {
			out[i] = barEmpty
		}
	}
	return string(out)
}

// formatCount renders an integer count with thousands separators (e.g. 12345 ->
// "12,345"). Negative values keep their sign.
func formatCount(n int) string {
	neg := n < 0
	if neg {
		n = -n
	}

	s := strconv.Itoa(n)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}

	// Insert commas every three digits from the right.
	var out []byte
	lead := len(s) % 3
	if lead > 0 {
		out = append(out, s[:lead]...)
	}
	for i := lead; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, s[i:i+3]...)
	}

	res := string(out)
	if neg {
		return "-" + res
	}
	return res
}

// formatPercent renders a 0..100 percentage with one decimal place and a
// trailing '%' (e.g. 42.15 -> "42.2%"). Values are not clamped; callers pass
// the
// API-provided percent directly.
func formatPercent(p float64) string {
	return strconv.FormatFloat(p, 'f', 1, 64) + "%"
}
