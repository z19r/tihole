package core

import "testing"

func feedAll(t *testing.T, keys ...string) bool {
	t.Helper()
	var d KonamiDetector
	var done bool
	for _, k := range keys {
		done = d.Feed(k)
	}
	return done
}

func TestKonamiDetectorFullSequenceCompletes(t *testing.T) {
	if !feedAll(t, konamiSequence...) {
		t.Error(
			"expected the full sequence to report completion on the last key",
		)
	}
}

func TestKonamiDetectorOnlyFinalKeyReportsTrue(t *testing.T) {
	var d KonamiDetector
	for i, key := range konamiSequence {
		got := d.Feed(key)
		want := i == len(konamiSequence)-1
		if got != want {
			t.Errorf("Feed(%q) at position %d = %v, want %v", key, i, got, want)
		}
	}
}

func TestKonamiDetectorResetsOnMismatch(t *testing.T) {
	var d KonamiDetector
	d.Feed("up")
	d.Feed("up")
	if d.Feed("x") {
		t.Fatal("mismatched key must never report completion")
	}
	// Having reset, the real sequence must still work from scratch.
	if !feedAll(t, konamiSequence...) {
		t.Error("expected detector to recover after a mismatch")
	}
}

func TestKonamiDetectorMismatchThatEqualsFirstKeyReloads(t *testing.T) {
	var d KonamiDetector
	d.Feed("up")
	d.Feed("up")
	d.Feed("down")
	// "up" isn't the expected fourth key ("down"), but it does equal the
	// sequence's first key, so the detector should treat it as a fresh
	// start (position 1) rather than resetting all the way to untyped.
	d.Feed("up")
	rest := konamiSequence[1:]
	var done bool
	for _, k := range rest {
		done = d.Feed(k)
	}
	if !done {
		t.Error(
			"expected re-feeding from the second key onward to still complete",
		)
	}
}

func TestKonamiDetectorIsReusableAfterCompletion(t *testing.T) {
	var d KonamiDetector
	for _, k := range konamiSequence {
		d.Feed(k)
	}
	if !feedAll(t, konamiSequence...) {
		t.Error(
			"expected the detector to be triggerable again after completing once",
		)
	}
}

func TestKonamiDetectorWrongKeyAtStartNeverCompletes(t *testing.T) {
	var d KonamiDetector
	if d.Feed("enter") {
		t.Fatal("an unrelated first key must never complete the sequence")
	}
}
