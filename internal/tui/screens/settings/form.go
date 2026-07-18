package settings

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/config"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// instField enumerates the focusable fields in the instance form, in tab order.
type instField int

const (
	fieldName instField = iota
	fieldURL
	fieldPassword
	fieldPasswordEnv
	fieldVerifyTLS
	instFieldCount // sentinel
)

// instanceForm is the inline add/edit form for a single configured instance.
// When editing, index locates the row in the instance slice and original holds
// its prior definition so a blank password field preserves the stored secret.
type instanceForm struct {
	editing  bool
	index    int
	original config.Instance

	name        textinput.Model
	url         textinput.Model
	password    textinput.Model
	passwordEnv textinput.Model
	verifyTLS   bool

	focus instField
}

// newAddInstanceForm builds a blank instance form (verify_tls defaults true).
func newAddInstanceForm() *instanceForm {
	f := &instanceForm{verifyTLS: true, focus: fieldName}
	f.name = newInput("name", 128)
	f.url = newInput("https://pihole.local", 512)
	f.password = newPasswordInput()
	f.passwordEnv = newInput("PIHOLE_PASSWORD", 128)
	f.syncFocus()
	return f
}

// newEditInstanceForm builds a form pre-filled from an existing instance. The
// password is intentionally NOT pre-filled: a blank password on save preserves
// the stored secret so it never has to be surfaced.
func newEditInstanceForm(inst config.Instance, index int) *instanceForm {
	f := &instanceForm{
		editing:   true,
		index:     index,
		original:  inst,
		verifyTLS: inst.VerifyTLSValue(),
		focus:     fieldName,
	}
	f.name = newInput("name", 128)
	f.name.SetValue(inst.Name)
	f.url = newInput("https://pihole.local", 512)
	f.url.SetValue(inst.URL)
	f.password = newPasswordInput()
	f.passwordEnv = newInput("PIHOLE_PASSWORD", 128)
	f.passwordEnv.SetValue(inst.PasswordEnv)
	f.syncFocus()
	return f
}

func newInput(placeholder string, limit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = limit
	return ti
}

func newPasswordInput() textinput.Model {
	ti := newInput("password (blank keeps current)", 512)
	ti.EchoMode = textinput.EchoPassword
	return ti
}

func (f *instanceForm) title() string {
	if f.editing {
		return "Edit instance"
	}
	return "Add instance"
}

func (f *instanceForm) focusNext() { f.focus = (f.focus + 1) % instFieldCount }

func (f *instanceForm) focusPrev() { f.focus = (f.focus + instFieldCount - 1) % instFieldCount }

// toggleVerify flips the verify_tls field.
func (f *instanceForm) toggleVerify() { f.verifyTLS = !f.verifyTLS }

// syncFocus focuses the active text input and blurs the rest, returning the
// blink command (nil for the non-text verify_tls field).
func (f *instanceForm) syncFocus() tea.Cmd {
	f.name.Blur()
	f.url.Blur()
	f.password.Blur()
	f.passwordEnv.Blur()

	switch f.focus {
	case fieldName:
		return f.name.Focus()
	case fieldURL:
		return f.url.Focus()
	case fieldPassword:
		return f.password.Focus()
	case fieldPasswordEnv:
		return f.passwordEnv.Focus()
	default:
		return nil
	}
}

// updateFocused routes a key message to the currently focused text input.
func (f *instanceForm) updateFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch f.focus {
	case fieldName:
		f.name, cmd = f.name.Update(msg)
	case fieldURL:
		f.url, cmd = f.url.Update(msg)
	case fieldPassword:
		f.password, cmd = f.password.Update(msg)
	case fieldPasswordEnv:
		f.passwordEnv, cmd = f.passwordEnv.Update(msg)
	}
	return cmd
}

// instance builds the resulting instance definition from the form's fields.
// A blank password when editing preserves the original stored secret.
func (f *instanceForm) instance() config.Instance {
	inst := config.Instance{
		Name: strings.TrimSpace(f.name.Value()),
		URL:  strings.TrimSpace(f.url.Value()),
	}

	pw := f.password.Value()
	switch {
	case pw != "":
		inst.Password = pw
	case f.editing:
		inst.Password = f.original.Password
	}

	inst.PasswordEnv = strings.TrimSpace(f.passwordEnv.Value())

	tls := f.verifyTLS
	inst.VerifyTLS = &tls
	return inst
}

// setWidth reflows the text inputs to the available width.
func (f *instanceForm) setWidth(w int) {
	iw := clampMin(w-16, 8)
	f.name.SetWidth(iw)
	f.url.SetWidth(iw)
	f.password.SetWidth(iw)
	f.passwordEnv.SetWidth(iw)
}

// render draws the form as a centered, bordered panel.
func (f *instanceForm) render(th *theme.Theme, w, h int) string {
	heading := th.AccentStyle().Bold(true).Render(f.title())

	tls := "off"
	if f.verifyTLS {
		tls = "on"
	}

	lines := []string{
		heading,
		"",
		f.inputLine(th, fieldName, "Name", f.name.View()),
		f.inputLine(th, fieldURL, "URL", f.url.View()),
		f.inputLine(th, fieldPassword, "Password", f.password.View()),
		f.inputLine(th, fieldPasswordEnv, "Pass env", f.passwordEnv.View()),
		f.toggleLine(th, fieldVerifyTLS, "Verify TLS", tls),
		"",
		th.SubtleStyle().
			Render("tab next · ←/→ toggle · enter save · esc cancel"),
	}

	panelW := clampInt(w-4, 24, 72)
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
		surfaceWhitespace(th),
	)
}

func (f *instanceForm) toggleLine(
	th *theme.Theme,
	field instField,
	label, value string,
) string {
	return f.fieldLabel(
		th,
		field,
		label,
	) + "  " + th.TextStyle().
		Render("‹ "+value+" ›")
}

func (f *instanceForm) inputLine(
	th *theme.Theme,
	field instField,
	label, view string,
) string {
	return f.fieldLabel(th, field, label) + "  " + view
}

func (f *instanceForm) fieldLabel(
	th *theme.Theme,
	field instField,
	label string,
) string {
	marker := "  "
	style := th.SubtleStyle()
	if f.focus == field {
		marker = "▸ "
		style = th.AccentStyle()
	}
	return marker + style.Width(10).Render(label)
}
