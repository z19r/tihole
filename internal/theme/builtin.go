package theme

import "charm.land/lipgloss/v2"

// Built-in theme names.
const (
	NameDeepNight     = "deep-night"
	NameLightLuxury   = "light-luxury"
	NamePiholeClassic = "pihole-classic"
)

// DeepNight is the signature dark theme: a deep midnight-navy surface with a
// soft periwinkle accent and warm signal colors.
func DeepNight() *Theme {
	return &Theme{
		Name:    NameDeepNight,
		Surface: lipgloss.Color("#0B0F1A"),
		Panel:   lipgloss.Color("#141B2D"),
		Text:    lipgloss.Color("#E6EAF2"),
		Subtle:  lipgloss.Color("#8A93A6"),
		Accent:  lipgloss.Color("#7AA2F7"),
		Allow:   lipgloss.Color("#4ADE80"),
		Block:   lipgloss.Color("#F7768E"),
		Warn:    lipgloss.Color("#F5C97B"),
		Border:  lipgloss.Color("#2A3550"),
	}
}

// LightLuxury is a warm, high-end light theme: ivory paper, brass accent, and
// muted, editorial signal colors.
func LightLuxury() *Theme {
	return &Theme{
		Name:    NameLightLuxury,
		Surface: lipgloss.Color("#FBF9F4"),
		Panel:   lipgloss.Color("#FFFFFF"),
		Text:    lipgloss.Color("#1F1B16"),
		Subtle:  lipgloss.Color("#6E665B"),
		Accent:  lipgloss.Color("#B08D57"),
		Allow:   lipgloss.Color("#3F9D5A"),
		Block:   lipgloss.Color("#C0392B"),
		Warn:    lipgloss.Color("#C9871F"),
		Border:  lipgloss.Color("#E4DCCF"),
	}
}

// PiholeClassic is a red-forward homage to Pi-hole: warm near-black surface
// with the signature Pi-hole red as both accent and block signal.
func PiholeClassic() *Theme {
	return &Theme{
		Name:    NamePiholeClassic,
		Surface: lipgloss.Color("#1A1213"),
		Panel:   lipgloss.Color("#241819"),
		Text:    lipgloss.Color("#F5E9E9"),
		Subtle:  lipgloss.Color("#A88B8B"),
		Accent:  lipgloss.Color("#E5322B"),
		Allow:   lipgloss.Color("#35C46A"),
		Block:   lipgloss.Color("#E5322B"),
		Warn:    lipgloss.Color("#F2B134"),
		Border:  lipgloss.Color("#4A2E30"),
	}
}

// builtinFactories maps each built-in name to its constructor. Constructors
// return fresh pointers so callers can never mutate a shared instance.
var builtinFactories = map[string]func() *Theme{
	NameDeepNight:     DeepNight,
	NameLightLuxury:   LightLuxury,
	NamePiholeClassic: PiholeClassic,
}

// Builtin returns the built-in theme for name and true, or (nil, false) if no
// such built-in exists.
func Builtin(name string) (*Theme, bool) {
	factory, ok := builtinFactories[name]
	if !ok {
		return nil, false
	}
	return factory(), true
}

// Names returns the built-in theme names in a stable, display order.
func Names() []string {
	return []string{NameDeepNight, NameLightLuxury, NamePiholeClassic}
}
