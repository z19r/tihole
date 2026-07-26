package localdns

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

// formState holds the inline add form. Its fields depend on the record kind:
// host records take ip + domain; CNAME records take domain + target + optional
// ttl. Only adds are supported (the config API has no single-record update), so
// there is no editing mode.
type formState struct {
	kind   recordKind
	labels []string
	inputs []textinput.Model
	focus  int
}

// newHostForm builds a blank host-record add form (ip, domain).
func newHostForm() *formState {
	f := &formState{
		kind:   kindHosts,
		labels: []string{"IP", "Domain"},
		inputs: []textinput.Model{
			newInput("192.0.2.10 or ::1", 64),
			newInput("host.lan", 253),
		},
	}
	f.syncFocus()
	return f
}

// newCNAMEForm builds a blank CNAME-record add form (domain, target, ttl).
func newCNAMEForm() *formState {
	f := &formState{
		kind:   kindCNAMEs,
		labels: []string{"Domain", "Target", "TTL"},
		inputs: []textinput.Model{
			newInput("alias.lan", 253),
			newInput("host.lan", 253),
			newInput("ttl (optional)", 10),
		},
	}
	f.syncFocus()
	return f
}

func newInput(placeholder string, limit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = limit
	return ti
}

// title is the form heading.
func (f *formState) title() string {
	if f.kind == kindCNAMEs {
		return "Add CNAME record"
	}
	return "Add host record"
}

// focusNext advances the focused field, wrapping around.
func (f *formState) focusNext() { f.focus = (f.focus + 1) % len(f.inputs) }

// focusPrev moves the focused field back, wrapping around.
func (f *formState) focusPrev() { f.focus = (f.focus + len(f.inputs) - 1) % len(f.inputs) }

// syncFocus focuses the active textinput and blurs the rest, returning the
// blink command for the focused one.
func (f *formState) syncFocus() tea.Cmd {
	var cmd tea.Cmd
	for i := range f.inputs {
		if i == f.focus {
			cmd = f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
	return cmd
}

// updateFocused routes a key message to the currently focused textinput.
func (f *formState) updateFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return cmd
}

// value returns the trimmed value of the i-th input.
func (f *formState) value(i int) string {
	if i < 0 || i >= len(f.inputs) {
		return ""
	}
	return strings.TrimSpace(f.inputs[i].Value())
}

// parseTTL parses the optional ttl field. A blank or non-numeric value yields
// 0, which the client omits from the stored record.
func parseTTL(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// setWidth reflows the text inputs to the available width.
func (f *formState) setWidth(w int) {
	iw := w - 14
	if iw < 8 {
		iw = 8
	}
	for i := range f.inputs {
		f.inputs[i].SetWidth(iw)
	}
}

// render draws the form as a bordered panel, centered in the body.
func (f *formState) render(th *theme.Theme, w, h int) string {
	heading := th.AccentStyle().Bold(true).Render(f.title())

	lines := []string{heading, ""}
	for i, label := range f.labels {
		lines = append(lines, f.inputLine(th, i, label, f.inputs[i].View()))
	}
	lines = append(
		lines,
		"",
		th.SubtleStyle().Render("tab next · enter save · esc cancel"),
	)

	panelW := w - 4
	if panelW < 24 {
		panelW = 24
	}
	if panelW > 72 {
		panelW = 72
	}

	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(1, 2).
		Width(panelW).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(
		w,
		h,
		lipgloss.Center,
		lipgloss.Top,
		panel,
		th.SurfaceWhitespace(),
	)
}

func (f *formState) inputLine(
	th *theme.Theme,
	field int,
	label, view string,
) string {
	marker := "  "
	style := th.SubtleStyle()
	if f.focus == field {
		marker = "▸ "
		style = th.AccentStyle()
	}
	return marker + style.Width(8).Render(label) + "  " + view
}
