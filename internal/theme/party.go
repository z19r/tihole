package theme

import (
	"image/color"
	"math"
	"time"
)

// NameParty is the hidden theme applied while the Konami-code easter egg is
// active. It is deliberately absent from Names()/Builtin() — the only way in
// is the cheat code.
const NameParty = "party"

// Party returns a fresh, wall-clock-driven rainbow theme: every token is a
// full-saturation hue that sweeps over time, offset per token so the UI
// reads as a wash of color rather than one flat tint. Call it again on every
// animation tick (see the root model's party ticker) to advance the sweep;
// Name stays "party" throughout, so Animated is set for callers that cache a
// rebuild behind "has Theme.Name changed?".
func Party() *Theme {
	return &Theme{
		Name:     NameParty,
		Animated: true,
		Surface:  wildDark(0, 0.4, 0.08),
		Panel:    wildDark(210, 0.5, 0.14),
		Text:     wildColor(0),
		Subtle:   wildDark(60, 0.6, 0.75),
		Accent:   wildColor(90),
		Allow:    wildColor(150),
		Block:    wildColor(210),
		Warn:     wildColor(270),
		Border:   wildColor(330),
		Ramp:     wildRamp(),
	}
}

// wildHue returns the current point on the color wheel for a token offset by
// `offset` degrees from the base sweep, so simultaneously-rendered tokens
// land at different hues instead of all flashing the same color together.
func wildHue(offset int) float64 {
	base := float64(time.Now().UnixMilli()/50) * 3
	return base + float64(offset)
}

// wildColor is a full-saturation, full-value swatch: for foreground tokens
// (text, borders, accents) that need to read as vivid against a dark
// background.
func wildColor(offset int) color.Color {
	return hsvToRGB(wildHue(offset), 1, 1)
}

// wildDark is a desaturated, low-value swatch: for background tokens
// (surface, panel) that need to stay dark enough for foreground text to read
// while still drifting through the same sweep.
func wildDark(offset int, saturation, value float64) color.Color {
	return hsvToRGB(wildHue(offset), saturation, value)
}

// wildRamp builds the signature-gradient stops (boot banner, wordmark) as
// evenly spaced points around the current sweep.
func wildRamp() []color.Color {
	const stops = 7
	ramp := make([]color.Color, stops)
	for i := range ramp {
		ramp[i] = wildColor(i * (360 / stops))
	}
	return ramp
}

// hsvToRGB converts hue (degrees, wraps), saturation, and value (both
// [0,1]) to an RGB color.
func hsvToRGB(hue, saturation, value float64) color.Color {
	h := math.Mod(math.Mod(hue, 360)+360, 360)
	c := value * saturation
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := value - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}
