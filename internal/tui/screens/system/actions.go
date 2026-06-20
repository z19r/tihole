package system

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
	"github.com/zackkitzmiller/tihole/internal/theme"
)

// actionItem is one destructive action in the Actions menu.
type actionItem struct {
	label string
	desc  string
	run   func(context.Context, *pihole.Client) error
}

// actionItems is the fixed Actions menu. Each is destructive and may require
// the FTL webserver's allow_destructive setting.
func actionItems() []actionItem {
	return []actionItem{
		{
			label: "Restart DNS",
			desc:  "Restart the FTL DNS resolver. Brief resolution outage.",
			run:   func(ctx context.Context, api *pihole.Client) error { return api.RestartDNS(ctx) },
		},
		{
			label: "Flush logs",
			desc:  "Permanently clear the FTL query log.",
			run:   func(ctx context.Context, api *pihole.Client) error { return api.FlushLogs(ctx) },
		},
		{
			label: "Flush network",
			desc:  "Permanently clear the FTL network device table.",
			run:   func(ctx context.Context, api *pihole.Client) error { return api.FlushNetwork(ctx) },
		},
	}
}

// handleActionsKey moves the menu cursor and arms the confirm dialog on enter.
func (m *Model) handleActionsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := actionItems()
	switch msg.String() {
	case "up", "k":
		if m.actionCursor > 0 {
			m.actionCursor--
		}
		return m, nil
	case "down", "j":
		if m.actionCursor < len(items)-1 {
			m.actionCursor++
		}
		return m, nil
	case "enter":
		if m.actionCursor < 0 || m.actionCursor >= len(items) {
			return m, nil
		}
		item := items[m.actionCursor]
		m.pendingOp = item.run
		m.pendingReload = false
		m.confirm = m.confirm.Show(
			item.label+"?",
			item.desc+"\nMay require allow_destructive in the FTL config.",
			true,
		)
		return m, nil
	}
	return m, nil
}

// destructiveHint augments a failed destructive action with an allow_destructive
// remediation note when the error looks like a forbidden/destructive rejection.
func destructiveHint(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *pihole.APIError
	if errors.As(err, &apiErr) {
		low := strings.ToLower(apiErr.Message + " " + apiErr.Key + " " + apiErr.Hint)
		if apiErr.Status == 403 || strings.Contains(low, "destructive") || strings.Contains(low, "forbidden") {
			return fmt.Errorf("%w — enable allow_destructive in the FTL config to permit this", err)
		}
	}
	return err
}

// renderActions draws the destructive-action menu.
func (m *Model) renderActions(th *theme.Theme, bodyH int) string {
	items := actionItems()
	var lines []string
	lines = append(lines, th.AccentStyle().Bold(true).Render("Destructive actions"), "")
	for i, item := range items {
		marker := "  "
		style := th.TextStyle()
		if i == m.actionCursor {
			marker = th.AccentStyle().Render("› ")
			style = th.AccentStyle()
		}
		lines = append(lines, marker+style.Render(item.label))
		lines = append(lines, "    "+th.SubtleStyle().Render(item.desc))
	}
	lines = append(lines, "", th.SubtleStyle().Render("enter run · may require allow_destructive"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.NewStyle().MaxHeight(bodyH).Render(content)
}
