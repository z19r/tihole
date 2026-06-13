// Package theme provides semantic theming tokens, built-in themes, an
// adaptive-color helper, and an Omarchy desktop-theme adapter for tihole.
//
// A Theme is a set of SEMANTIC color tokens (not raw palette entries) that
// screens read at View() time, so swapping the active theme re-renders the
// whole UI instantly. Reusable lipgloss.Style values are built on demand from
// those tokens via the *Style() methods.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is a set of semantic color tokens. Each token is an image/color.Color
// (what lipgloss.Color returns) so themes can be built from hex strings, ANSI
// indices, or adaptive pickers uniformly.
type Theme struct {
	// Name is the stable identifier used by Builtin/Resolve (e.g. "deep-night").
	Name string

	Surface color.Color // app background
	Panel   color.Color // raised surface (cards, panels); slightly off Surface
	Text    color.Color // primary foreground
	Subtle  color.Color // muted foreground (secondary text, hints)
	Accent  color.Color // brand / focus / selection highlight
	Allow   color.Color // positive / allowed (green family)
	Block   color.Color // negative / blocked (red family)
	Warn    color.Color // caution (amber family)
	Border  color.Color // panel & divider borders

	// adapt, when non-nil, lets a theme return a mode-specific variant from
	// Adapt(isDark). Built-in themes are committed to a single mode and leave
	// this nil (Adapt returns the receiver unchanged).
	adapt func(isDark bool) *Theme
}

// SurfaceStyle is the base app style: Text on Surface.
func (t *Theme) SurfaceStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Surface)
}

// PanelStyle is a raised, bordered container built from Panel/Border/Text.
func (t *Theme) PanelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Panel).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)
}

// TextStyle renders primary foreground text.
func (t *Theme) TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Text)
}

// SubtleStyle renders muted secondary text.
func (t *Theme) SubtleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Subtle)
}

// AccentStyle renders emphasized/branded text.
func (t *Theme) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
}

// AllowStyle renders positive/allowed state.
func (t *Theme) AllowStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Allow)
}

// BlockStyle renders negative/blocked state.
func (t *Theme) BlockStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Block).Bold(true)
}

// WarnStyle renders caution state.
func (t *Theme) WarnStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Warn)
}

// BorderStyle renders a divider/border using the Border token.
func (t *Theme) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Border)
}

// Adapt returns the theme variant appropriate for the given background.
//
// Built-in themes are each committed to a single light/dark mode, so for them
// Adapt returns the receiver unchanged. Themes that genuinely carry two
// variants (or are derived, like Omarchy) can set an internal adapter; when
// present it is consulted here. This keeps adaptive color a light touch: the
// app calls Adapt(msg.IsDark()) once on a tea.BackgroundColorMsg and re-reads
// tokens at View() time.
func (t *Theme) Adapt(isDark bool) *Theme {
	if t.adapt != nil {
		return t.adapt(isDark)
	}
	return t
}

// Pick chooses between a light and dark color for an ambiguous token, using
// Lip Gloss v2's explicit LightDark resolver. Themes and callers that hold both
// variants of a token can resolve it once the terminal background is known.
func Pick(isDark bool, light, dark color.Color) color.Color {
	return lipgloss.LightDark(isDark)(light, dark)
}
