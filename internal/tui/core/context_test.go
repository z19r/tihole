package core

import (
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbletea/v2"

	"github.com/z19r/tihole/internal/pihole"
)

func TestSourcePicksFleet(t *testing.T) {
	api := pihole.New("http://api", "")
	fleet := pihole.NewFleet([]pihole.Member{
		{Name: "a", Client: pihole.New("http://a", "")},
		{Name: "b", Client: pihole.New("http://b", "")},
	}, nil)

	c := &AppContext{API: api, Fleet: fleet}
	if c.Source() != Source(api) {
		t.Fatal("Combined=false should use the API")
	}
	c.Combined = true
	if c.Source() != Source(fleet) {
		t.Fatal("Combined with a 2-member fleet should use the fleet")
	}
	c.Fleet = nil
	if c.Source() != Source(api) {
		t.Fatal("nil Fleet should fall back to the API")
	}
}

func TestToggleCombinedBinding(t *testing.T) {
	msg := tea.KeyPressMsg{Code: 'm', Text: "m"}
	if !key.Matches(msg, DefaultKeyMap().ToggleCombined) {
		t.Fatal("m should match ToggleCombined")
	}
}
