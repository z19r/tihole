package groups

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/theme"
)

// form field indices. In add mode all three are focusable; in edit mode the
// name is read-only (UpdateGroup keys on the original name), so focus cycles
// over comment and enabled only.
const (
	fieldName = iota
	fieldComment
	fieldEnabled
)

// groupForm is the inline add/edit form. It carries only display + input state;
// the owning screen interprets enter (submit) and esc (cancel).
type groupForm struct {
	active  bool
	editing bool // true = edit (name read-only), false = add

	origName string // the group's key when editing

	name    textinput.Model
	comment textinput.Model
	enabled bool

	order []int // focusable field ids in navigation order
	cur   int   // index into order
}

// newGroupForm builds the form with its two text inputs configured.
func newGroupForm() groupForm {
	name := textinput.New()
	name.Placeholder = "name"
	name.CharLimit = 63

	comment := textinput.New()
	comment.Placeholder = "comment (optional)"
	comment.CharLimit = 255

	return groupForm{name: name, comment: comment}
}

// openAdd resets the form for creating a new group and focuses the name field.
func (f *groupForm) openAdd(width int) tea.Cmd {
	f.active = true
	f.editing = false
	f.origName = ""
	f.enabled = true
	f.name.SetValue("")
	f.comment.SetValue("")
	f.order = []int{fieldName, fieldComment, fieldEnabled}
	f.cur = 0
	f.setWidth(width)
	f.comment.Blur()
	return f.name.Focus()
}

// openEdit pre-fills the form for editing an existing group. The name is kept
// read-only; focus starts on the comment field.
func (f *groupForm) openEdit(
	name, comment string,
	enabled bool,
	width int,
) tea.Cmd {
	f.active = true
	f.editing = true
	f.origName = name
	f.enabled = enabled
	f.name.SetValue(name)
	f.comment.SetValue(comment)
	f.order = []int{fieldComment, fieldEnabled}
	f.cur = 0
	f.setWidth(width)
	f.name.Blur()
	f.comment.CursorEnd()
	return f.comment.Focus()
}

// close deactivates and blurs the form.
func (f *groupForm) close() {
	f.active = false
	f.name.Blur()
	f.comment.Blur()
}

// setWidth reflows the input fields to the available width.
func (f *groupForm) setWidth(width int) {
	w := width - 4
	if w < 8 {
		w = 8
	}
	f.name.SetWidth(w)
	f.comment.SetWidth(w)
}

// focused returns the currently focused field id.
func (f *groupForm) focused() int {
	if f.cur < 0 || f.cur >= len(f.order) {
		return fieldEnabled
	}
	return f.order[f.cur]
}

// canSubmit reports whether the form has enough data to submit. Add requires a
// non-blank name; edit always can (name is fixed).
func (f *groupForm) canSubmit() bool {
	if f.editing {
		return true
	}
	return strings.TrimSpace(f.name.Value()) != ""
}

// values returns the submitted name/comment/enabled and whether this is an
// edit.
// For edits the original name is authoritative.
func (f *groupForm) values() (name, comment string, enabled, editing bool) {
	name = strings.TrimSpace(f.name.Value())
	if f.editing {
		name = f.origName
	}
	return name, strings.TrimSpace(f.comment.Value()), f.enabled, f.editing
}

// update handles navigation (tab/shift+tab), the enabled toggle (space), and
// forwards printable input to the focused text field. Enter/esc are handled by
// the owning screen before this is reached.
func (f groupForm) update(msg tea.KeyPressMsg) (groupForm, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		return f.moveFocus(1)
	case "shift+tab", "up":
		return f.moveFocus(-1)
	case "space":
		if f.focused() == fieldEnabled {
			f.enabled = !f.enabled
			return f, nil
		}
	}

	var cmd tea.Cmd
	switch f.focused() {
	case fieldName:
		f.name, cmd = f.name.Update(msg)
	case fieldComment:
		f.comment, cmd = f.comment.Update(msg)
	}
	return f, cmd
}

// moveFocus advances the focused field by delta, wrapping, and refocuses the
// matching text input (blurring the others).
func (f groupForm) moveFocus(delta int) (groupForm, tea.Cmd) {
	if len(f.order) == 0 {
		return f, nil
	}
	f.cur = (f.cur + delta + len(f.order)) % len(f.order)

	f.name.Blur()
	f.comment.Blur()

	var cmd tea.Cmd
	switch f.focused() {
	case fieldName:
		cmd = f.name.Focus()
	case fieldComment:
		cmd = f.comment.Focus()
	}
	return f, cmd
}

// view renders the form panel using the live theme.
func (f *groupForm) view(th *theme.Theme, width int) string {
	heading := "Add group"
	if f.editing {
		heading = "Edit group"
	}

	var lines []string
	lines = append(lines, th.AccentStyle().Bold(true).Render(heading), "")

	lines = append(
		lines,
		f.fieldRow(th, "Name", f.nameView(th), f.focused() == fieldName),
	)
	lines = append(
		lines,
		f.fieldRow(
			th,
			"Comment",
			f.comment.View(),
			f.focused() == fieldComment,
		),
	)

	toggle := disabledGlyph + " disabled"
	if f.enabled {
		toggle = th.AllowStyle().Render(enabledGlyph + " enabled")
	}
	lines = append(
		lines,
		f.fieldRow(th, "Enabled", toggle, f.focused() == fieldEnabled),
	)

	lines = append(
		lines,
		"",
		th.SubtleStyle().
			Render("tab move · space toggle · enter save · esc cancel"),
	)

	panelW := maxInt(minInt(width-2, 60), 20)
	return th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(1, 2).
		Width(panelW).
		Render(strings.Join(lines, "\n"))
}

// nameView renders the name field: the live input in add mode, or read-only
// subtle text in edit mode.
func (f *groupForm) nameView(th *theme.Theme) string {
	if f.editing {
		return th.SubtleStyle().Render(f.origName + " (read-only)")
	}
	return f.name.View()
}

// fieldRow renders a labelled field with a focus marker.
func (f *groupForm) fieldRow(
	th *theme.Theme,
	label, value string,
	focused bool,
) string {
	marker := "  "
	if focused {
		marker = th.AccentStyle().Render("▸ ")
	}
	return marker + th.SubtleStyle().Width(9).Render(label) + value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
