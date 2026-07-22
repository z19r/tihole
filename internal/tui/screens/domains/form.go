package domains

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
)

// formField enumerates the focusable fields in the add/edit form, in tab order.
type formField int

const (
	fieldType formField = iota
	fieldKind
	fieldDomain
	fieldComment
	fieldGroups
	fieldCount // sentinel
)

// formState holds the inline add/edit form. When editing, original identifies
// the row being updated so its enabled flag is preserved.
type formState struct {
	editing  bool
	original pihole.Domain

	dtype pihole.DomainType
	dkind pihole.DomainKind

	domain  textinput.Model
	comment textinput.Model
	groups  textinput.Model

	focus formField
}

// newAddForm builds a blank form seeded from the currently active filter tab so
// the new entry defaults to a sensible type/kind.
func newAddForm(tab filterTab) *formState {
	dt, dk := tab.typeKind()
	if tab == filterAll {
		dt, dk = pihole.DomainDeny, pihole.KindExact
	}

	f := &formState{editing: false, dtype: dt, dkind: dk, focus: fieldDomain}
	f.domain = newInput("domain or regex", 253)
	f.comment = newInput("comment (optional)", 255)
	f.groups = newInput("group ids, e.g. 0, 2", 64)
	f.syncFocus()
	return f
}

// newEditForm builds a form pre-filled from an existing domain.
func newEditForm(d pihole.Domain) *formState {
	f := &formState{
		editing:  true,
		original: d,
		dtype:    pihole.DomainType(d.Type),
		dkind:    pihole.DomainKind(d.Kind),
		focus:    fieldDomain,
	}
	f.domain = newInput("domain or regex", 253)
	f.domain.SetValue(d.Domain)
	f.comment = newInput("comment (optional)", 255)
	f.comment.SetValue(d.Comment)
	f.groups = newInput("group ids, e.g. 0, 2", 64)
	f.groups.SetValue(groupsToString(d.Groups))
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
	if f.editing {
		return "Edit domain"
	}
	return "Add domain"
}

// focusNext advances the focused field, wrapping around.
func (f *formState) focusNext() { f.focus = (f.focus + 1) % fieldCount }

// focusPrev moves the focused field back, wrapping around.
func (f *formState) focusPrev() { f.focus = (f.focus + fieldCount - 1) % fieldCount }

// toggleType flips allow/deny.
func (f *formState) toggleType() {
	if f.dtype == pihole.DomainAllow {
		f.dtype = pihole.DomainDeny
	} else {
		f.dtype = pihole.DomainAllow
	}
}

// toggleKind flips exact/regex.
func (f *formState) toggleKind() {
	if f.dkind == pihole.KindExact {
		f.dkind = pihole.KindRegex
	} else {
		f.dkind = pihole.KindExact
	}
}

// syncFocus focuses the active textinput and blurs the rest, returning the
// blink command for the focused one (nil for the non-text toggle fields).
func (f *formState) syncFocus() tea.Cmd {
	f.domain.Blur()
	f.comment.Blur()
	f.groups.Blur()

	switch f.focus {
	case fieldDomain:
		return f.domain.Focus()
	case fieldComment:
		return f.comment.Focus()
	case fieldGroups:
		return f.groups.Focus()
	default:
		return nil
	}
}

// updateFocused routes a key message to the currently focused textinput.
func (f *formState) updateFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch f.focus {
	case fieldDomain:
		f.domain, cmd = f.domain.Update(msg)
	case fieldComment:
		f.comment, cmd = f.comment.Update(msg)
	case fieldGroups:
		f.groups, cmd = f.groups.Update(msg)
	}
	return cmd
}

// values extracts the submitted field values.
func (f *formState) values() (domain, comment string, groups []int) {
	return strings.TrimSpace(f.domain.Value()),
		strings.TrimSpace(f.comment.Value()),
		parseGroups(f.groups.Value())
}

// setWidth reflows the text inputs to the available width.
func (f *formState) setWidth(w int) {
	iw := w - 12
	if iw < 8 {
		iw = 8
	}
	f.domain.SetWidth(iw)
	f.comment.SetWidth(iw)
	f.groups.SetWidth(iw)
}

// render draws the form as a bordered panel, centered in the body.
func (f *formState) render(th *theme.Theme, w, h int) string {
	heading := th.AccentStyle().Bold(true).Render(f.title())

	lines := []string{
		heading,
		"",
		f.toggleLine(th, fieldType, "Type", string(f.dtype)),
		f.toggleLine(th, fieldKind, "Kind", string(f.dkind)),
		f.inputLine(th, fieldDomain, "Domain", f.domain.View()),
		f.inputLine(th, fieldComment, "Comment", f.comment.View()),
		f.inputLine(th, fieldGroups, "Groups", f.groups.View()),
		"",
		th.SubtleStyle().
			Render("tab next · ←/→ toggle · enter save · esc cancel"),
	}

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

func (f *formState) toggleLine(
	th *theme.Theme,
	field formField,
	label, value string,
) string {
	return f.fieldLabel(
		th,
		field,
		label,
	) + "  " + th.TextStyle().
		Render("‹ "+value+" ›")
}

func (f *formState) inputLine(
	th *theme.Theme,
	field formField,
	label, view string,
) string {
	return f.fieldLabel(th, field, label) + "  " + view
}

func (f *formState) fieldLabel(
	th *theme.Theme,
	field formField,
	label string,
) string {
	marker := "  "
	style := th.SubtleStyle()
	if f.focus == field {
		marker = "▸ "
		style = th.AccentStyle()
	}
	return marker + style.Width(9).Render(label)
}
