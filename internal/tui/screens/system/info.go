package system

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

// infoSectionNames are the read-only sections fetched for the Info tab, in
// display order.
var infoSectionNames = []string{"ftl", "host", "system", "version"}

// kv is a single flattened label/value row.
type kv struct {
	key string
	val string
}

// infoSection is a labeled panel of scalar rows for one Info section.
type infoSection struct {
	name string
	rows []kv
}

// infoMsg carries the multi-section Info fetch result, tagged with its epoch.
type infoMsg struct {
	epoch    int
	sections []infoSection
	err      error
}

// fetchInfo requests every Info section in one closure and flattens each to
// scalar rows. Partial failures still surface the sections that succeeded.
func (m *Model) fetchInfo() tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	m.loading = true

	return tea.Batch(func() tea.Msg {
		var sections []infoSection
		var firstErr error
		for _, name := range infoSectionNames {
			data, err := api.Info(ctx, name)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			sections = append(
				sections,
				infoSection{name: name, rows: flattenInfo(data)},
			)
		}
		return infoMsg{epoch: e, sections: sections, err: firstErr}
	}, m.spinner.Tick)
}

// flattenInfo reduces an open-ended Info object to its scalar (string/number/
// bool) fields, sorted by key. Nested maps and slices are skipped.
func flattenInfo(data map[string]any) []kv {
	keys := make([]string, 0, len(data))
	for k, v := range data {
		if isScalar(v) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	rows := make([]kv, len(keys))
	for i, k := range keys {
		rows[i] = kv{key: k, val: scalarString(data[k])}
	}
	return rows
}

// isScalar reports whether v is a flat JSON scalar (string, bool, or number).
func isScalar(v any) bool {
	switch v.(type) {
	case string, bool, float64:
		return true
	default:
		return false
	}
}

// scalarString renders a scalar value; integral floats lose their decimals.
func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == math.Trunc(t) && !math.IsInf(t, 0) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// newCtx cancels any in-flight request and starts a fresh timeout context,
// storing the cancel on the model (safe: Update runs on the main loop).
func (m *Model) newCtx() context.Context {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	m.cancel = cancel
	return ctx
}

// renderInfo draws the fetched sections as labeled key/value panels.
func (m *Model) renderInfo(th *theme.Theme, bodyH int) string {
	if m.loading && len(m.info) == 0 {
		return m.spinnerLine(th, "loading info…", bodyH)
	}
	if len(m.info) == 0 {
		empty := th.SubtleStyle().Render("no info available")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			empty,
			th.SurfaceWhitespace(),
		)
	}

	keyW := 16
	var panels []string
	for _, sec := range m.info {
		heading := th.AccentStyle().Bold(true).Render(sec.name)
		lines := []string{heading}
		for _, row := range sec.rows {
			label := th.SubtleStyle().
				Background(th.Surface).
				Width(keyW).
				Render(truncate(row.key, keyW))
			val := th.TextStyle().
				Render(truncate(row.val, maxInt(m.w-keyW-4, 8)))
			lines = append(lines, label+"  "+val)
		}
		panels = append(panels, strings.Join(lines, "\n"))
	}

	content := strings.Join(panels, "\n")
	return lipgloss.NewStyle().MaxHeight(bodyH).Render(content)
}
