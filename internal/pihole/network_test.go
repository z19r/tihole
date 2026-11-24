package pihole

import (
	"context"
	"net/http"
	"testing"
)

func TestNetworkDevicesDecode(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/network/devices" {
			t.Errorf("path = %q", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"devices":[{
			"id":1,"hwaddr":"aa:bb:cc:dd:ee:ff","interface":"eth0",
			"firstSeen":1700000000,"lastQuery":1700000500,"numQueries":42,
			"macVendor":"Acme",
			"ips":[{"ip":"192.168.1.5","name":"host.local","lastSeen":1700000400}]
		}],"took":0.0}`)
	})
	client.setSID("S")

	// Act
	devices, err := client.NetworkDevices(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("NetworkDevices error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("len = %d, want 1", len(devices))
	}
	d := devices[0]
	if d.ID != 1 || d.HWAddr != "aa:bb:cc:dd:ee:ff" || d.Interface != "eth0" {
		t.Errorf("device = %#v", d)
	}
	if d.NumQueries != 42 || d.MACVendor != "Acme" ||
		d.FirstSeen != 1700000000 ||
		d.LastQuery != 1700000500 {
		t.Errorf("device = %#v", d)
	}
	if len(d.IPs) != 1 || d.IPs[0].IP != "192.168.1.5" ||
		d.IPs[0].Name != "host.local" ||
		d.IPs[0].LastSeen != 1700000400 {
		t.Errorf("ips = %#v", d.IPs)
	}
}

func TestDeleteNetworkDevicePath(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	client.setSID("S")

	if err := client.DeleteNetworkDevice(context.Background(), 7); err != nil {
		t.Fatalf("DeleteNetworkDevice error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/api/network/devices/7" {
		t.Errorf("path = %q, want /api/network/devices/7", gotPath)
	}
}
