package adlists

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// formMode distinguishes an add form (all fields editable) from an edit form
// (address read-only; comment/groups/type/enabled editable).
type formMode int

const (
	formAdd formMode = iota
	formEdit
)

// fieldKind identifies a focusable form field. Text fields wrap a textinput;
// the type and enabled fields are toggles driven by left/right/space.
type fieldKind int

const (
	fieldAddress fieldKind = iota
	fieldComment
	fieldGroups
	fieldType
	fieldEnabled
)

// formModel is the inline add/edit form. It owns its own focus cycle so the
// screen can defer all keystrokes to it while active.
type formModel struct {
	active bool
	mode   formMode

	address textinput.Model
	comment textinput.Model
	groups  textinput.Model

	listType pihole.ListType
	enabled  bool

	fields []fieldKind
	focus  int
}

// newForm constructs an empty, inactive form with its text inputs prepared.
func newForm() formModel {
	addr := textinput.New()
	addr.Placeholder = "https://example.com/list.txt"
	addr.CharLimit = 2048

	com := textinput.New()
	com.Placeholder = "comment"
	com.CharLimit = 256

	grp := textinput.New()
	grp.Placeholder = "0,1"
	grp.CharLimit = 128

	return formModel{
		address: addr,
		comment: com,
		groups:  grp,
	}
}

// openAdd returns a copy of the form activated in add mode for the given type.
func (f formModel) openAdd(listType pihole.ListType) formModel {
	f.active = true
	f.mode = formAdd
	f.listType = listType
	f.enabled = true
	f.address.SetValue("")
	f.comment.SetValue("")
	f.groups.SetValue("")
	f.fields = []fieldKind{fieldAddress, fieldComment, fieldGroups, fieldType}
	f.focus = 0
	f.syncFocus()
	return f
}

// openEdit returns a copy of the form activated in edit mode, pre-filled from
// the given list. Address is displayed read-only (not in the focus cycle).
func (f formModel) openEdit(l pihole.List) formModel {
	f.active = true
	f.mode = formEdit
	f.listType = pihole.ListType(l.Type)
	f.enabled = l.Enabled
	f.address.SetValue(l.Address)
	f.comment.SetValue(l.Comment)
	f.groups.SetValue(groupsCSV(l.Groups))
	f.fields = []fieldKind{fieldComment, fieldGroups, fieldType, fieldEnabled}
	f.focus = 0
	f.syncFocus()
	return f
}

// close returns a copy with the form deactivated and inputs blurred.
func (f formModel) close() formModel {
	f.active = false
	f.address.Blur()
	f.comment.Blur()
	f.groups.Blur()
	return f
}

// current returns the field kind currently focused.
func (f formModel) current() fieldKind {
	if f.focus < 0 || f.focus >= len(f.fields) {
		return fieldType
	}
	return f.fields[f.focus]
}

// syncFocus focuses the active text input (if any) and blurs the rest.
func (f *formModel) syncFocus() {
	f.address.Blur()
	f.comment.Blur()
	f.groups.Blur()
	switch f.current() {
	case fieldComment:
		f.comment.Focus()
	case fieldGroups:
		f.groups.Focus()
	case fieldAddress:
		f.address.Focus()
	}
}

// advance moves focus by delta (wrapping) and re-syncs input focus.
func (f formModel) advance(delta int) formModel {
	n := len(f.fields)
	if n == 0 {
		return f
	}
	f.focus = ((f.focus+delta)%n + n) % n
	f.syncFocus()
	return f
}

// toggle flips the type or enabled field when it is focused.
func (f formModel) toggle() formModel {
	switch f.current() {
	case fieldType:
		if f.listType == pihole.ListBlock {
			f.listType = pihole.ListAllow
		} else {
			f.listType = pihole.ListBlock
		}
	case fieldEnabled:
		f.enabled = !f.enabled
	}
	return f
}

// update routes a keystroke into the focused text input (toggles are handled by
// the screen before this is reached).
func (f formModel) update(msg tea.Msg) (formModel, tea.Cmd) {
	var cmd tea.Cmd
	switch f.current() {
	case fieldComment:
		f.comment, cmd = f.comment.Update(msg)
	case fieldGroups:
		f.groups, cmd = f.groups.Update(msg)
	case fieldAddress:
		f.address, cmd = f.address.Update(msg)
	}
	return f, cmd
}

// setWidth reflows the text inputs to the available width.
func (f *formModel) setWidth(w int) {
	iw := w - 14
	if iw < 8 {
		iw = 8
	}
	f.address.SetWidth(iw)
	f.comment.SetWidth(iw)
	f.groups.SetWidth(iw)
}

// values returns the trimmed field values needed by AddList/UpdateList.
func (f formModel) values() (address, comment, groupsCSVVal string) {
	return strings.TrimSpace(f.address.Value()),
		strings.TrimSpace(f.comment.Value()),
		f.groups.Value()
}

// view renders the form panel using the live theme.
func (f formModel) view(th *theme.Theme, width int) string {
	heading := "Add list"
	if f.mode == formEdit {
		heading = "Edit list"
	}
	lines := []string{th.AccentStyle().Bold(true).Render(heading), ""}

	label := th.SubtleStyle().Width(10)
	sel := func(k fieldKind) string {
		if f.current() == k {
			return th.AccentStyle().Render("›")
		}
		return " "
	}

	addressField := th.SubtleStyle().Render(f.address.Value())
	if f.mode == formAdd {
		addressField = f.address.View()
	}
	lines = append(lines, sel(fieldAddress)+" "+label.Render("Address")+addressField)
	lines = append(lines, sel(fieldComment)+" "+label.Render("Comment")+f.comment.View())
	lines = append(lines, sel(fieldGroups)+" "+label.Render("Groups")+f.groups.View())

	typeStyle := styleForToken(th, typeToken(string(f.listType)))
	lines = append(lines, sel(fieldType)+" "+label.Render("Type")+typeStyle.Render(string(f.listType))+th.SubtleStyle().Render("  (←/→)"))

	if f.mode == formEdit {
		enTok := tokenAllow
		if !f.enabled {
			enTok = tokenNeutral
		}
		lines = append(lines, sel(fieldEnabled)+" "+label.Render("Enabled")+styleForToken(th, enTok).Render(enabledCell(f.enabled)))
	}

	lines = append(lines, "", th.SubtleStyle().Render("tab next · ←/→ toggle · enter save · esc cancel"))

	panel := th.PanelStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(1, 2).
		Width(maxInt(minInt(width-2, 70), 24)).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return panel
}
