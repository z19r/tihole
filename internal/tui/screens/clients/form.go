package clients

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// Form validation and parse errors surfaced inline in the form panel.
var (
	errEmptyClient = errors.New("client id is required")
	errBadGroups   = errors.New("groups must be comma-separated integers")
)

// formMode distinguishes an add (client editable) from an edit (client
// read-only, keyed on its id) form.
type formMode int

const (
	formAdd formMode = iota
	formEdit
)

// clientForm holds the inline add/edit form state: three text inputs plus the
// index of the focused editable field. In edit mode the client id is fixed, so
// only comment and groups are focusable.
type clientForm struct {
	active   bool
	mode     formMode
	client   textinput.Model
	comment  textinput.Model
	groups   textinput.Model
	focusIdx int
	err      error
}

// newForm builds the (initially inactive) form inputs.
func newForm() clientForm {
	c := textinput.New()
	c.Placeholder = "IP / subnet / MAC / host"
	c.CharLimit = 253

	cm := textinput.New()
	cm.Placeholder = "comment"
	cm.CharLimit = 253

	g := textinput.New()
	g.Placeholder = "e.g. 0, 1"
	g.CharLimit = 128

	return clientForm{client: c, comment: cm, groups: g}
}

// fields returns the focusable inputs in tab order for the current mode.
func (f *clientForm) fields() []*textinput.Model {
	if f.mode == formEdit {
		return []*textinput.Model{&f.comment, &f.groups}
	}
	return []*textinput.Model{&f.client, &f.comment, &f.groups}
}

// focusNext / focusPrev cycle the focused field index within the mode's fields.
func (f *clientForm) focusNext() {
	n := len(f.fields())
	if n > 0 {
		f.focusIdx = (f.focusIdx + 1) % n
	}
}

func (f *clientForm) focusPrev() {
	n := len(f.fields())
	if n > 0 {
		f.focusIdx = (f.focusIdx - 1 + n) % n
	}
}

// focusCmd blurs every field, focuses the current one, and returns its Focus
// command (the cursor-blink cmd).
func (f *clientForm) focusCmd() tea.Cmd {
	fields := f.fields()
	var cmd tea.Cmd
	for i, fi := range fields {
		if i == f.focusIdx {
			cmd = fi.Focus()
		} else {
			fi.Blur()
		}
	}
	return cmd
}

// updateFocused forwards a message to the currently focused input.
func (f *clientForm) updateFocused(msg tea.Msg) tea.Cmd {
	fields := f.fields()
	if f.focusIdx < 0 || f.focusIdx >= len(fields) {
		return nil
	}
	var cmd tea.Cmd
	*fields[f.focusIdx], cmd = fields[f.focusIdx].Update(msg)
	return cmd
}

// close deactivates the form and blurs all inputs.
func (f *clientForm) close() {
	f.active = false
	f.err = nil
	f.client.Blur()
	f.comment.Blur()
	f.groups.Blur()
}

// setWidth reflows the input widths to fit the panel.
func (f *clientForm) setWidth(w int) {
	iw := clampMin(minInt(w-6, 48), 8)
	f.client.SetWidth(iw)
	f.comment.SetWidth(iw)
	f.groups.SetWidth(iw)
}

// render draws the form as a centered panel within the given area.
func (f *clientForm) render(th *theme.Theme, w, h int) string {
	heading := "Add client"
	if f.mode == formEdit {
		heading = "Edit client"
	}

	labelStyle := th.SubtleStyle().Width(9)
	var lines []string
	lines = append(lines, th.AccentStyle().Bold(true).Render(heading), "")

	clientField := f.client.View()
	if f.mode == formEdit {
		// Client id is read-only in edit mode: show it as static text.
		clientField = th.TextStyle().Render(f.client.Value())
	}
	lines = append(lines,
		labelStyle.Render("Client")+"  "+clientField,
		labelStyle.Render("Comment")+"  "+f.comment.View(),
		labelStyle.Render("Groups")+"  "+f.groups.View(),
	)

	if f.err != nil {
		lines = append(
			lines,
			"",
			th.BlockStyle().Bold(true).Render("error: "+f.err.Error()),
		)
	}
	lines = append(
		lines,
		"",
		th.SubtleStyle().Render("tab next · enter save · esc cancel"),
	)

	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(1, 2).
		Width(maxInt(minInt(w-2, 64), 24)).
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

// formValues extracts the trimmed, parsed field values for submission.
func (f *clientForm) formValues() (client, comment string, groups []int, err error) {
	groups, gerr := parseGroups(f.groups.Value())
	if gerr != nil {
		return "", "", nil, gerr
	}
	client = strings.TrimSpace(f.client.Value())
	comment = strings.TrimSpace(f.comment.Value())
	if client == "" {
		return "", "", nil, errEmptyClient
	}
	return client, comment, groups, nil
}
