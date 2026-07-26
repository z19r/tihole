package settings

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

const (
	fetchTimeout = 8 * time.Second

	headerHeight = 3 // section-tabs line + explanation line + banner/spacer
	footerHeight = 1 // hint line

	treeCellPad = 2
	minTreeCol  = 8

	ellipsis = "…"
)

// truncate shortens s to at most width display cells, appending an ellipsis
// when
// it must cut. Rune-aware so multibyte content is not split mid-rune.
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

// surfaceWhitespace paints lipgloss.Place's fill with the surface background so
// centered panels and empty states don't bleed the terminal background around
// the placed content.
func surfaceWhitespace(th *theme.Theme) lipgloss.WhitespaceOption {
	return lipgloss.WithWhitespaceStyle(
		lipgloss.NewStyle().Background(th.Surface),
	)
}

func maxInt(a, b int) int {
	if a > b {
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
