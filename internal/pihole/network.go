package pihole

import (
	"context"
	"fmt"
	"net/http"
)

// NetworkAddress is a single IP/name pair observed for a network device.
type NetworkAddress struct {
	IP       string `json:"ip"`
	Name     string `json:"name"`
	LastSeen int64  `json:"lastSeen"`
}

// NetworkDevice is a device row from GET /api/network/devices.
type NetworkDevice struct {
	ID         int              `json:"id"`
	HWAddr     string           `json:"hwaddr"`
	Interface  string           `json:"interface"`
	FirstSeen  int64            `json:"firstSeen"`
	LastQuery  int64            `json:"lastQuery"`
	NumQueries int              `json:"numQueries"`
	MACVendor  string           `json:"macVendor"`
	IPs        []NetworkAddress `json:"ips"`
}

// networkDevicesEnvelope is the response of GET /api/network/devices.
type networkDevicesEnvelope struct {
	Devices []NetworkDevice `json:"devices"`
	Took    float64         `json:"took"`
}

// NetworkDevices fetches GET /api/network/devices — every device FTL has seen
// on the local network, with its observed addresses.
func (c *Client) NetworkDevices(ctx context.Context) ([]NetworkDevice, error) {
	var out networkDevicesEnvelope
	if err := c.do(ctx, http.MethodGet, "/network/devices", nil, &out); err != nil {
		return nil, err
	}
	return out.Devices, nil
}

// Gateway fetches GET /api/network/gateway — the default gateway details, as an
// open-ended object.
func (c *Client) Gateway(ctx context.Context) (map[string]any, error) {
	return c.networkObject(ctx, "/network/gateway")
}

// Interfaces fetches GET /api/network/interfaces — the host network interfaces.
func (c *Client) Interfaces(ctx context.Context) (map[string]any, error) {
	return c.networkObject(ctx, "/network/interfaces")
}

// Routes fetches GET /api/network/routes — the host routing table.
func (c *Client) Routes(ctx context.Context) (map[string]any, error) {
	return c.networkObject(ctx, "/network/routes")
}

// networkObject fetches a network endpoint that returns an open-ended object.
func (c *Client) networkObject(
	ctx context.Context,
	path string,
) (map[string]any, error) {
	var out map[string]any
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteNetworkDevice removes a device from FTL's network table via
// DELETE /api/network/devices/{id}.
func (c *Client) DeleteNetworkDevice(ctx context.Context, id int) error {
	path := fmt.Sprintf("/network/devices/%d", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}
