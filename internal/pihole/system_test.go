package pihole

import (
	"context"
	"net/http"
	"testing"
)

// systemJSON is a trimmed but structurally faithful GET /api/info/system body
// captured from a live FTL v6 host.
const systemJSON = `{
  "system": {
    "uptime": 392102,
    "memory": {
      "ram": {"total": 3886904, "free": 3003932, "used": 263808, "available": 3508352, "%used": 6.787098420748236},
      "swap": {"total": 2097148, "free": 2097148, "used": 0, "%used": 0}
    },
    "procs": 192,
    "cpu": {
      "nprocs": 4,
      "%cpu": 0.899999976158142,
      "load": {"raw": [0.099, 0.083, 0.049], "percent": [2.478, 2.099, 1.232]}
    },
    "ftl": {"%mem": 1.0797282457351685, "%cpu": 0.15000000596046448}
  },
  "took": 0.0002968311309814453
}`

func TestSystemDecodesNestedMetrics(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/info/system" {
			t.Errorf("path = %q, want /api/info/system", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, systemJSON)
	})
	client.setSID("S")

	// Act
	got, err := client.System(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("System error: %v", err)
	}
	if got.Uptime != 392102 {
		t.Errorf("uptime = %d, want 392102", got.Uptime)
	}
	if got.Procs != 192 || got.NProcs != 4 {
		t.Errorf("procs/nprocs = %d/%d, want 192/4", got.Procs, got.NProcs)
	}
	if got.CPUPercent < 0.89 || got.CPUPercent > 0.91 {
		t.Errorf("cpu%% = %.3f, want ~0.9", got.CPUPercent)
	}
	if got.Load1Percent < 2.47 || got.Load1Percent > 2.48 {
		t.Errorf(
			"load1%% = %.3f, want ~2.478 (first load bucket)",
			got.Load1Percent,
		)
	}
	if got.MemUsedPercent < 6.78 || got.MemUsedPercent > 6.79 {
		t.Errorf("mem%% = %.3f, want ~6.787", got.MemUsedPercent)
	}
	if got.MemUsedKiB != 263808 || got.MemTotalKiB != 3886904 {
		t.Errorf(
			"mem KiB = %d/%d, want 263808/3886904",
			got.MemUsedKiB,
			got.MemTotalKiB,
		)
	}
	if got.SwapUsedPercent != 0 {
		t.Errorf("swap%% = %.3f, want 0", got.SwapUsedPercent)
	}
	if got.FTLCPUPercent < 0.14 || got.FTLMemPercent < 1.07 {
		t.Errorf(
			"ftl cpu/mem = %.3f/%.3f, want ~0.15/~1.08",
			got.FTLCPUPercent,
			got.FTLMemPercent,
		)
	}
}

func TestSystemLoadDefaultsWhenAbsent(t *testing.T) {
	// Arrange: a host with no load buckets must not panic on the empty slice.
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusOK,
			`{"system":{"cpu":{"nprocs":2,"load":{"percent":[]}}},"took":0}`,
		)
	})
	client.setSID("S")

	// Act
	got, err := client.System(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("System error: %v", err)
	}
	if got.Load1Percent != 0 {
		t.Errorf(
			"load1%% = %.3f, want 0 when load buckets are empty",
			got.Load1Percent,
		)
	}
}

func TestSensorsInfoDecodesTemperature(t *testing.T) {
	// Arrange
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/info/sensors" {
			t.Errorf("path = %q, want /api/info/sensors", r.URL.Path)
		}
		writeJSON(
			w,
			http.StatusOK,
			`{"sensors":{"list":[],"cpu_temp":48.199,"hot_limit":60,"unit":"C"},"took":0.0}`,
		)
	})
	client.setSID("S")

	// Act
	got, err := client.SensorsInfo(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("SensorsInfo error: %v", err)
	}
	if !got.HasTemp {
		t.Fatal("expected HasTemp true when cpu_temp present")
	}
	if got.CPUTemp < 48.1 || got.CPUTemp > 48.3 {
		t.Errorf("cpu temp = %.3f, want ~48.2", got.CPUTemp)
	}
	if got.HotLimit != 60 || got.Unit != "C" {
		t.Errorf("hot/unit = %.0f/%q, want 60/C", got.HotLimit, got.Unit)
	}
}

func TestSensorsInfoNoTemperature(t *testing.T) {
	// Arrange: a host with no CPU temperature sensor (cpu_temp null).
	client, _ := mockFTL(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusOK,
			`{"sensors":{"list":[],"cpu_temp":null,"hot_limit":60,"unit":"C"},"took":0}`,
		)
	})
	client.setSID("S")

	// Act
	got, err := client.SensorsInfo(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("SensorsInfo error: %v", err)
	}
	if got.HasTemp {
		t.Fatal("expected HasTemp false when cpu_temp is null")
	}
}
