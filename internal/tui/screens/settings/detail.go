package settings

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

const (
	// detailBreakpoint is the minimum inner width at which the config view
	// splits into a table plus a side detail panel; below it the panel stacks
	// beneath the table instead.
	detailBreakpoint = 88
	detailMinW       = 34
	detailMaxW       = 60
	detailLabelW     = 8
	detailMinBodyH   = 8
	// detailMinPanelW is the narrowest width worth drawing a bordered panel in
	// (2 border + 4 padding + a little content). Below it we drop the panel
	// rather than render wider than the container and wrap the whole layout.
	detailMinPanelW = 12
)

// configSplit divides the inner width between the config table and the detail
// panel. Wide terminals get a side-by-side split; narrow ones keep a full-width
// table and stack the panel beneath it (so detailW is the full width).
func configSplit(w int) (tableW, detailW int, horizontal bool) {
	if w < detailBreakpoint {
		return w, w, false
	}
	detailW = clampInt(w*2/5, detailMinW, detailMaxW)
	tableW = w - detailW - 1 // one-column surface gutter between the panes
	if tableW < 40 {
		tableW = 40
		detailW = w - tableW - 1
	}
	return tableW, detailW, true
}

// showDetail reports whether there is enough vertical room to render the detail
// panel at all.
func showDetail(bodyH int) bool { return bodyH >= detailMinBodyH }

// detailStackH is the height of the detail panel when it stacks beneath the
// table on narrow terminals, always leaving a few rows for the table itself.
func detailStackH(bodyH int) int {
	// Reserve enough rows for the description plus the value block, while
	// always
	// leaving the table a few rows of its own.
	h := clampInt(bodyH-6, 9, 16)
	if h > bodyH-3 {
		h = bodyH - 3
	}
	return clampMin(h, 1)
}

// treeHeight returns the height to give the config table, reserving room for a
// stacked detail panel on narrow terminals.
func treeHeight(w, bodyH int) int {
	if !showDetail(bodyH) {
		return bodyH
	}
	if _, _, horizontal := configSplit(w); horizontal {
		return bodyH
	}
	return bodyH - detailStackH(bodyH)
}

// renderLeafDetail draws the metadata panel for a single config leaf, filling a
// w×h surface block so it aligns cleanly beside or beneath the table. It
// mirrors what Pi-hole's web UI shows per setting: description, accepted input,
// default,
// current value, enumerated options and whether the value has been changed.
func renderLeafDetail(th *theme.Theme, l leaf, w, h int) string {
	// Too narrow (or short) to draw a bordered panel without overflowing the
	// container and wrapping the layout: yield a blank surface block instead.
	if w < detailMinPanelW || h < 3 {
		return lipgloss.Place(
			w, h, lipgloss.Left, lipgloss.Top, "", surfaceWhitespace(th),
		)
	}
	// The panel (border + padding) must never render wider than the width it is
	// placed into: on a narrow, stacked layout the outer surface render would
	// wrap the overflow and shift every row below it. Cap against w.
	panelW := clampInt(w-4, detailMinPanelW, detailMaxW)
	if panelW > w {
		panelW = w
	}
	// Border adds 2 columns, the horizontal padding (1,2) adds 4; the text sits
	// in what's left, and the panel is 4 rows taller than its content (2 border
	// + 2 vertical padding).
	innerW := clampMin(panelW-6, 1)
	maxContent := clampMin(h-4, 1)

	header := []string{
		th.AccentStyle().Bold(true).Render(truncate(l.path, innerW)),
		th.SubtleStyle().Render(strings.Repeat("─", innerW)),
	}

	meta := []string{
		detailRow(th, "Type", typeLabel(l), th.TextStyle(), innerW),
		detailRow(th, "Default", stringifyValue(l.defaultVal),
			th.SubtleStyle(), innerW),
		detailRow(th, "Current", stringifyValue(l.value),
			valueStyle(th, l.value), innerW),
	}

	var allowed []string
	if items := allowedInputs(l); len(items) > 0 {
		options := lipgloss.NewStyle().Width(innerW).
			Foreground(th.Text).Render(strings.Join(items, " · "))
		allowed = []string{"", th.SubtleStyle().Render("Allowed"), options}
	}

	statusVal := th.SubtleStyle().Render("unchanged from default")
	if l.modified {
		statusVal = th.WarnStyle().Bold(true).Render("modified")
	}
	status := []string{"", detailRowRaw(th, "Status", statusVal)}

	// Assemble within the height budget. The description is the headline the
	// user came for, so it takes priority: it claims the rows it needs (capped
	// so the value block still fits when there's room for both), and the
	// Allowed/Status blocks are what drop first when space is tight.
	lines := append([]string{}, header...)
	used := len(header)
	if desc := strings.TrimSpace(l.description); desc != "" {
		budget := maxContent - used - 1 // 1 for the trailing blank
		if budget > len(meta)+2 {
			budget -= len(meta) // leave room for the value block too
		}
		if budget >= 1 {
			descLines := wrapLines(th, desc, innerW, budget)
			lines = append(lines, descLines...)
			lines = append(lines, "")
			used += len(descLines) + 1
		}
	}
	fits := func(block []string) bool { return used+len(block) <= maxContent }
	if fits(meta) {
		lines = append(lines, meta...)
		used += len(meta)
	}
	if len(allowed) > 0 && fits(allowed) {
		lines = append(lines, allowed...)
		used += len(allowed)
	}
	if fits(status) {
		lines = append(lines, status...)
	}

	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(1, 2).
		Width(panelW).
		MaxHeight(h).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(
		w, h, lipgloss.Left, lipgloss.Top, panel, surfaceWhitespace(th),
	)
}

// wrapLines word-wraps s to width w and returns at most max lines, appending an
// ellipsis to the last line when the text is clipped. It wraps and truncates on
// the plain text, then applies the foreground, so styling never straddles a cut
// escape sequence.
func wrapLines(th *theme.Theme, s string, w, max int) []string {
	all := strings.Split(lipgloss.NewStyle().Width(w).Render(s), "\n")
	if len(all) > max {
		all = all[:max]
		last := strings.TrimRight(all[max-1], " ")
		all[max-1] = truncate(last, clampMin(w-2, 1)) + " " + ellipsis
	}
	st := lipgloss.NewStyle().Foreground(th.Text)
	for i := range all {
		all[i] = st.Render(all[i])
	}
	return all
}

// detailRow renders a "label   value" line, styling each part and truncating
// the value to fit the panel width.
func detailRow(
	th *theme.Theme, label, value string, vs lipgloss.Style, panelW int,
) string {
	lbl := th.SubtleStyle().Render(fmt.Sprintf("%-*s", detailLabelW, label))
	avail := clampMin(panelW-detailLabelW-1, 4)
	return lbl + " " + vs.Render(truncate(value, avail))
}

// detailRowRaw renders a labeled line whose value is already styled.
func detailRowRaw(th *theme.Theme, label, rendered string) string {
	lbl := th.SubtleStyle().Render(fmt.Sprintf("%-*s", detailLabelW, label))
	return lbl + " " + rendered
}

// typeLabel returns a human description of the leaf's accepted input,
// preferring
// FTL's own type string and falling back to the Go value's kind.
func typeLabel(l leaf) string {
	if s := strings.TrimSpace(l.dataType); s != "" {
		return s
	}
	switch l.value.(type) {
	case bool:
		return "boolean (true · false)"
	case float64, int, int64:
		return "number"
	case string:
		return "text"
	case []any:
		return "list"
	default:
		return "value"
	}
}

// allowedInputs extracts the enumerated allowed values for a leaf, if FTL
// provided them. Each entry may be a bare scalar or a {item,description} map.
func allowedInputs(l leaf) []string {
	if len(l.allowed) == 0 {
		return nil
	}
	out := make([]string, 0, len(l.allowed))
	for _, a := range l.allowed {
		if m, ok := a.(map[string]any); ok {
			if item, ok := m["item"]; ok {
				out = append(out, stringifyValue(item))
			}
			continue
		}
		out = append(out, stringifyValue(a))
	}
	return out
}
