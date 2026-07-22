package dashboard

import (
	"fmt"

	"github.com/z19r/tihole/internal/pihole"
)

// memLabel formats the memory cell value: percent, plus used/total in human
// units when the totals are known.
func memLabel(s pihole.SystemInfo) string {
	pct := fmt.Sprintf("%.1f%%", s.MemUsedPercent)
	if s.MemTotalKiB > 0 {
		return fmt.Sprintf(
			"%s · %s/%s",
			pct,
			humanKiB(s.MemUsedKiB),
			humanKiB(s.MemTotalKiB),
		)
	}
	return pct
}

// humanUptime renders a duration in seconds as a compact "Nd Nh" / "Nh Nm" /
// "Nm" string. Sub-minute and negative values read as "0m".
func humanUptime(seconds int64) string {
	if seconds < 60 {
		return "0m"
	}
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// humanKiB renders a kibibyte figure (as FTL reports memory) in the largest
// unit
// that keeps it readable: KiB, MiB or GiB.
func humanKiB(kib int64) string {
	const (
		mib = 1024
		gib = 1024 * 1024
	)
	switch {
	case kib >= gib:
		return fmt.Sprintf("%.1fG", float64(kib)/float64(gib))
	case kib >= mib:
		return fmt.Sprintf("%.0fM", float64(kib)/float64(mib))
	default:
		return fmt.Sprintf("%dK", kib)
	}
}

// pluralIssues renders an FTL diagnosis count as a warn badge, e.g. "⚠ 1 issue"
// or "⚠ 3 issues".
func pluralIssues(n int) string {
	if n == 1 {
		return "⚠ 1 issue"
	}
	return fmt.Sprintf("⚠ %d issues", n)
}
