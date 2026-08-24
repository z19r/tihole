package theme

import (
	"image/color"
	"testing"
)

func TestPartyHasNonNilTokensAndName(t *testing.T) {
	th := Party()

	if th.Name != NameParty {
		t.Errorf("expected name %q, got %q", NameParty, th.Name)
	}
	if !th.Animated {
		t.Error("expected Party to be Animated so callers re-render every tick")
	}
	tokensNonNil(t, th)

	if len(th.Ramp) < 2 {
		t.Errorf(
			"expected Party to define a multi-stop ramp, got %d stops",
			len(th.Ramp),
		)
	}
}

func TestPartyIsNotABuiltin(t *testing.T) {
	// Party is a hidden Konami-code easter egg, not a selectable theme: it
	// must stay out of Names()/Builtin() so cycleTheme and the palette never
	// surface it.
	if _, ok := Builtin(NameParty); ok {
		t.Error("expected Builtin(\"party\") to report false")
	}
	for _, name := range Names() {
		if name == NameParty {
			t.Error("expected Names() to omit \"party\"")
		}
	}
}

func TestHSVToRGBKnownHues(t *testing.T) {
	tests := []struct {
		name string
		hue  float64
		want color.RGBA
	}{
		{"red", 0, color.RGBA{255, 0, 0, 255}},
		{"green", 120, color.RGBA{0, 255, 0, 255}},
		{"blue", 240, color.RGBA{0, 0, 255, 255}},
		{"wraps past 360", 480, color.RGBA{0, 255, 0, 255}},
		{"wraps negative", -240, color.RGBA{0, 255, 0, 255}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hsvToRGB(tc.hue, 1, 1).(color.RGBA)
			if got != tc.want {
				t.Errorf(
					"hsvToRGB(%v, 1, 1) = %+v, want %+v",
					tc.hue,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestWildHueOffsetsSpreadTokens(t *testing.T) {
	// Different offsets sampled at the same instant must land on different
	// points of the wheel, or every "wild" token would flash the same color
	// together instead of reading as a wash of color.
	a := wildHue(0)
	b := wildHue(90)
	if a == b {
		t.Error("expected distinct offsets to produce distinct hues")
	}
}
