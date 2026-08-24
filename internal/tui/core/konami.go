package core

// konamiSequence is the classic cheat code, matched against literal key
// strings (arrow keys, not their vim h/j/k/l aliases) so vim-style
// navigation never triggers it by accident.
var konamiSequence = []string{
	"up", "up", "down", "down",
	"left", "right", "left", "right",
	"b", "a", "enter",
}

// KonamiDetector recognizes the Konami code across a stream of key presses.
// The zero value is ready to use.
type KonamiDetector struct {
	pos int
}

// Feed advances the detector by one key press (as returned by
// tea.KeyPressMsg.String()) and reports whether the full sequence just
// completed. It resets to the start after either a completed sequence or a
// mismatched key, so it can be fed continuously without the caller
// filtering input first.
func (k *KonamiDetector) Feed(key string) bool {
	if key == konamiSequence[k.pos] {
		k.pos++
		if k.pos == len(konamiSequence) {
			k.pos = 0
			return true
		}
		return false
	}
	if key == konamiSequence[0] {
		k.pos = 1
	} else {
		k.pos = 0
	}
	return false
}
