package main

import (
	"strings"
	"testing"
)

func TestDispatchHelp(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		if err := dispatch([]string{arg}); err != nil {
			t.Errorf("dispatch(%q) = %v, want nil", arg, err)
		}
	}
}

func TestDispatchUnknownCommand(t *testing.T) {
	err := dispatch([]string{"bogus"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown-command error, got %v", err)
	}
}
