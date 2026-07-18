package system

import (
	"context"
	"strconv"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/z19r/tihole/internal/pihole"
	"github.com/z19r/tihole/internal/theme"
	"github.com/z19r/tihole/internal/tui/components"
)

const (
	netColHW    = 17 // aa:bb:cc:dd:ee:ff
	netColQuery = 6
	netColLast  = 8
	netMinFlex  = 8
)

var netColumnTitles = []string{"IP", "Name", "HWAddr", "Vendor", "#Q", "Last"}

// networkMsg carries a network-devices fetch result, tagged with its epoch.
type networkMsg struct {
	epoch   int
	devices []pihole.NetworkDevice
	err     error
}

// computeNetWidths returns per-column content widths that never overflow total.
// IP, Name, and Vendor share the flexible space; HWAddr/#Q/Last are fixed.
func computeNetWidths(total int) []int {
	avail := total - len(netColumnTitles)*cellPad
	fixed := netColHW + netColQuery + netColLast
	flex := avail - fixed
	minFlex := 3 * netMinFlex
	if flex < minFlex {
		flex = minFlex
	}
	ip := flex / 3
	name := flex / 3
	vendor := flex - ip - name
	ip = clampMin(ip, netMinFlex)
	name = clampMin(name, netMinFlex)
	vendor = clampMin(vendor, netMinFlex)
	return []int{ip, name, netColHW, vendor, netColQuery, netColLast}
}

func netColumns(widths []int) []table.Column {
	cols := make([]table.Column, len(netColumnTitles))
	for i, title := range netColumnTitles {
		w := netColQuery
		if i < len(widths) {
			w = widths[i]
		}
		cols[i] = table.Column{Title: title, Width: w}
	}
	return cols
}

// firstAddress returns the device's first observed IP and its name, if any.
func firstAddress(d pihole.NetworkDevice) (string, string) {
	if len(d.IPs) == 0 {
		return "-", "-"
	}
	a := d.IPs[0]
	name := a.Name
	if name == "" {
		name = "-"
	}
	ip := a.IP
	if ip == "" {
		ip = "-"
	}
	return ip, name
}

// deviceToRow maps a network device to its ordered, truncated table cells.
func deviceToRow(d pihole.NetworkDevice, widths []int) []string {
	ipW, nameW, vendorW := netMinFlex, netMinFlex, netMinFlex
	if len(widths) == len(netColumnTitles) {
		ipW, nameW, vendorW = widths[0], widths[1], widths[3]
	}
	ip, name := firstAddress(d)
	vendor := d.MACVendor
	if vendor == "" {
		vendor = "-"
	}
	return []string{
		truncate(ip, ipW),
		truncate(name, nameW),
		truncate(d.HWAddr, netColHW),
		truncate(vendor, vendorW),
		strconv.Itoa(d.NumQueries),
		formatUnixTime(d.LastQuery),
	}
}

// formatUnixTime renders a unix-seconds timestamp as HH:MM:SS, or a dash when
// 0.
func formatUnixTime(sec int64) string {
	if sec <= 0 {
		return "-"
	}
	return time.Unix(sec, 0).Format("15:04:05")
}

// fetchNetwork requests the network device table.
func (m *Model) fetchNetwork() tea.Cmd {
	ctx := m.newCtx()
	e := m.epoch
	api := m.ctx.API
	m.loading = true

	return tea.Batch(func() tea.Msg {
		devices, err := api.NetworkDevices(ctx)
		return networkMsg{epoch: e, devices: devices, err: err}
	}, m.spinner.Tick)
}

// selectedDevice returns the network device under the table cursor, if any.
func (m *Model) selectedDevice() (pihole.NetworkDevice, bool) {
	idx := m.netTable.Cursor()
	if idx >= 0 && idx < len(m.devices) {
		return m.devices[idx], true
	}
	return pihole.NetworkDevice{}, false
}

// syncNetRows rebuilds the network table rows from the current data.
func (m *Model) syncNetRows() {
	widths := columnWidthsFrom(m.netTable.Columns())
	idx := m.netTable.Cursor()
	rows := make([]table.Row, len(m.devices))
	for i, d := range m.devices {
		rows[i] = table.Row(deviceToRow(d, widths))
	}
	m.netTable.SetStyles(components.TableStyles(m.ctx.Theme))
	m.netTable.SetRows(rows)
	if idx >= len(rows) {
		idx = len(rows) - 1
	}
	if idx < 0 {
		idx = 0
	}
	m.netTable.SetCursor(idx)
}

// handleNetworkKey routes Network-tab shortcuts, then defers to the table.
func (m *Model) handleNetworkKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "x", "delete":
		if d, ok := m.selectedDevice(); ok {
			id := d.ID
			ip, _ := firstAddress(d)
			m.pendingOp = func(ctx context.Context, api *pihole.Client) error {
				return api.DeleteNetworkDevice(ctx, id)
			}
			m.pendingReload = true
			m.confirm = m.confirm.Show(
				"Delete device?",
				ip+" · "+d.HWAddr,
				true,
			)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.netTable, cmd = m.netTable.Update(msg)
	return m, cmd
}

// renderNetwork draws the network-devices table.
func (m *Model) renderNetwork(th *theme.Theme, bodyH int) string {
	if m.loading && len(m.devices) == 0 {
		return m.spinnerLine(th, "loading devices…", bodyH)
	}
	if len(m.devices) == 0 {
		empty := th.SubtleStyle().Render("no network devices")
		return lipgloss.Place(
			m.w,
			bodyH,
			lipgloss.Center,
			lipgloss.Center,
			empty,
			th.SurfaceWhitespace(),
		)
	}
	m.syncNetRows()
	return m.netTable.View()
}
