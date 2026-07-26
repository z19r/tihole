package system

const ellipsis = "…"

// truncate shortens s to at most width display cells, appending an ellipsis
// when
// it must cut. Rune-aware so multibyte strings are not split mid-rune.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return ellipsis
	}
	return string(r[:width-1]) + ellipsis
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampMin(v, min int) int {
	if v < min {
		return min
	}
	return v
}
