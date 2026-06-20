package settings

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// leaf is a single scalar entry in the flattened config tree, addressed by its
// dotted path (e.g. "dns.blocking.active").
type leaf struct {
	path  string
	value any
}

// flattenConfig walks a nested config tree into a sorted, flat list of scalar
// leaves keyed by dotted paths. Nested objects are descended into; a detailed
// leaf descriptor (as returned by GET /config?detailed=true) is collapsed to its
// underlying value so both detailed and raw trees flatten the same way.
func flattenConfig(m map[string]any) []leaf {
	var out []leaf
	walkTree("", m, &out)
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

// walkTree recursively appends leaves under prefix.
func walkTree(prefix string, m map[string]any, out *[]leaf) {
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if child, ok := v.(map[string]any); ok {
			if isDetailLeaf(child) {
				*out = append(*out, leaf{path: path, value: child["value"]})
			} else {
				walkTree(path, child, out)
			}
			continue
		}
		*out = append(*out, leaf{path: path, value: v})
	}
}

// isDetailLeaf reports whether a map is a detailed-leaf descriptor (carries a
// "value" plus at least one metadata key) rather than a nested config object.
func isDetailLeaf(m map[string]any) bool {
	if _, ok := m["value"]; !ok {
		return false
	}
	for _, k := range []string{"type", "description", "default", "flags", "modified"} {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// filterLeaves returns the subset whose path contains the (lower-cased)
// substring. An empty query returns the input unchanged.
func filterLeaves(leaves []leaf, query string) []leaf {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return leaves
	}
	var out []leaf
	for _, l := range leaves {
		if strings.Contains(strings.ToLower(l.path), q) {
			out = append(out, l)
		}
	}
	return out
}

// stringifyValue renders a leaf value for display and for pre-filling an editor.
func stringifyValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case string:
		return x
	case float64:
		if x == math.Trunc(x) && !math.IsInf(x, 0) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = stringifyValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		return "{…}"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// parseLeafValue parses an edited string back into a value whose type matches
// the original leaf's type (bool/number stay typed; everything else is a
// string). Booleans are handled by the editor directly, so this covers the
// text-input path.
func parseLeafValue(orig any, s string) (any, error) {
	s = strings.TrimSpace(s)
	switch orig.(type) {
	case bool:
		return strconv.ParseBool(s)
	case int:
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		return n, nil
	case int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return n, nil
	case float64:
		if !strings.ContainsAny(s, ".eE") {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				return n, nil
			}
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return s, nil
	}
}

// buildPatch turns a dotted path plus a value into the nested map the config
// PATCH endpoint expects (the caller wraps it under "config").
func buildPatch(path string, value any) map[string]any {
	parts := strings.Split(path, ".")
	root := map[string]any{}
	cur := root
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			break
		}
		next := map[string]any{}
		cur[p] = next
		cur = next
	}
	return root
}

// leafEdit holds the inline editor for a single leaf. Boolean leaves toggle via
// boolVal; all other leaves are edited through the text input.
type leafEdit struct {
	leaf    leaf
	isBool  bool
	boolVal bool
	input   textinput.Model
}

// newLeafEdit builds an editor pre-filled from the leaf's current value.
func newLeafEdit(l leaf) *leafEdit {
	e := &leafEdit{leaf: l}
	if b, ok := l.value.(bool); ok {
		e.isBool = true
		e.boolVal = b
		return e
	}
	ti := textinput.New()
	ti.CharLimit = 512
	ti.SetValue(stringifyValue(l.value))
	ti.CursorEnd()
	ti.Focus()
	e.input = ti
	return e
}

// value resolves the edited value: the toggled boolean, or the parsed input.
func (e *leafEdit) value() (any, error) {
	if e.isBool {
		return e.boolVal, nil
	}
	return parseLeafValue(e.leaf.value, e.input.Value())
}

// setWidth reflows the text input.
func (e *leafEdit) setWidth(w int) {
	iw := w - 12
	if iw < 8 {
		iw = 8
	}
	if !e.isBool {
		e.input.SetWidth(iw)
	}
}

// render draws the leaf editor as a centered panel.
func (e *leafEdit) render(th *theme.Theme, w, h int) string {
	heading := th.AccentStyle().Bold(true).Render("Edit " + e.leaf.path)

	var valueLine string
	if e.isBool {
		valueLine = th.SubtleStyle().Render("Value") + "  " + th.TextStyle().Render("‹ "+strconv.FormatBool(e.boolVal)+" ›")
	} else {
		valueLine = th.SubtleStyle().Render("Value") + "  " + e.input.View()
	}

	hint := "enter save · esc cancel"
	if e.isBool {
		hint = "←/→/space toggle · enter save · esc cancel"
	}

	lines := []string{heading, "", valueLine, "", th.SubtleStyle().Render(hint)}

	panelW := clampInt(w-4, 24, 72)
	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(1, 2).
		Width(panelW).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Top, panel)
}

// --- tree table rendering ---

// treeColumnTitles is the header row for the config-tree table.
var treeColumnTitles = []string{"Path", "Value"}

// computeTreeWidths splits the available width between the path and value
// columns, clamping each to a sane minimum.
func computeTreeWidths(total int) []int {
	avail := total - 2*treeCellPad
	if avail < 2*minTreeCol {
		avail = 2 * minTreeCol
	}
	pathW := avail * 3 / 5
	valW := avail - pathW
	return []int{clampMin(pathW, minTreeCol), clampMin(valW, minTreeCol)}
}

// treeColumns builds table columns from computed widths.
func treeColumns(widths []int) []table.Column {
	cols := make([]table.Column, len(treeColumnTitles))
	for i, title := range treeColumnTitles {
		w := minTreeCol
		if i < len(widths) {
			w = widths[i]
		}
		cols[i] = table.Column{Title: title, Width: w}
	}
	return cols
}

// valueStyle resolves the display colour for a value by its type: booleans in
// allow/block, numbers in accent, everything else in primary text.
func valueStyle(th *theme.Theme, v any) lipgloss.Style {
	switch x := v.(type) {
	case bool:
		if x {
			return lipgloss.NewStyle().Foreground(th.Allow)
		}
		return lipgloss.NewStyle().Foreground(th.Block)
	case float64, int, int64:
		return lipgloss.NewStyle().Foreground(th.Accent)
	default:
		return lipgloss.NewStyle().Foreground(th.Text)
	}
}

// treeRows builds styled table rows from leaves, colouring the value cell.
func treeRows(th *theme.Theme, leaves []leaf, widths []int) []table.Row {
	pathW, valW := minTreeCol, minTreeCol
	if len(widths) == 2 {
		pathW, valW = widths[0], widths[1]
	}
	rows := make([]table.Row, len(leaves))
	for i, l := range leaves {
		path := truncate(l.path, pathW)
		val := truncate(stringifyValue(l.value), valW)
		rows[i] = table.Row{path, valueStyle(th, l.value).Render(val)}
	}
	return rows
}
