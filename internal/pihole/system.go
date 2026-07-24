package pihole

import (
	"context"
	"net/http"
)

// SystemInfo is the decoded, dashboard-relevant slice of GET /api/info/system.
// The FTL response nests everything under a "system" key; only the fields the
// UI actually renders are pulled out here.
type SystemInfo struct {
	// Uptime is the host uptime in seconds.
	Uptime int64
	// Procs is the number of running processes.
	Procs int
	// NProcs is the number of CPU cores/threads.
	NProcs int
	// CPUPercent is the host-wide CPU utilization (0..100).
	CPUPercent float64
	// Load1Percent is the 1-minute load average expressed as a percent of total
	// CPU capacity (0..100).
	Load1Percent float64
	// MemUsedPercent is RAM utilization (0..100).
	MemUsedPercent float64
	// MemUsedBytes / MemTotalBytes are RAM figures in kibibytes as FTL reports
	// them (the values are raw /proc/meminfo units).
	MemUsedKiB  int64
	MemTotalKiB int64
	// SwapUsedPercent is swap utilization (0..100).
	SwapUsedPercent float64
	// FTLCPUPercent / FTLMemPercent are pihole-FTL's own resource usage.
	FTLCPUPercent float64
	FTLMemPercent float64
}

// systemEnvelope mirrors the wire shape of GET /api/info/system. JSON keys with
// leading '%' are valid struct tags and decode fine.
type systemEnvelope struct {
	System struct {
		Uptime int64 `json:"uptime"`
		Procs  int   `json:"procs"`
		Memory struct {
			RAM struct {
				Total    int64   `json:"total"`
				Used     int64   `json:"used"`
				UsedPerc float64 `json:"%used"`
			} `json:"ram"`
			Swap struct {
				UsedPerc float64 `json:"%used"`
			} `json:"swap"`
		} `json:"memory"`
		CPU struct {
			NProcs  int     `json:"nprocs"`
			CPUPerc float64 `json:"%cpu"`
			Load    struct {
				Percent []float64 `json:"percent"`
			} `json:"load"`
		} `json:"cpu"`
		FTL struct {
			MemPerc float64 `json:"%mem"`
			CPUPerc float64 `json:"%cpu"`
		} `json:"ftl"`
	} `json:"system"`
}

// System fetches GET /api/info/system and decodes the fields the dashboard's
// health strip renders.
func (c *Client) System(ctx context.Context) (SystemInfo, error) {
	var env systemEnvelope
	if err := c.do(ctx, http.MethodGet, "/info/system", nil, &env); err != nil {
		return SystemInfo{}, err
	}

	s := env.System
	var load1 float64
	if len(s.CPU.Load.Percent) > 0 {
		load1 = s.CPU.Load.Percent[0]
	}

	return SystemInfo{
		Uptime:          s.Uptime,
		Procs:           s.Procs,
		NProcs:          s.CPU.NProcs,
		CPUPercent:      s.CPU.CPUPerc,
		Load1Percent:    load1,
		MemUsedPercent:  s.Memory.RAM.UsedPerc,
		MemUsedKiB:      s.Memory.RAM.Used,
		MemTotalKiB:     s.Memory.RAM.Total,
		SwapUsedPercent: s.Memory.Swap.UsedPerc,
		FTLCPUPercent:   s.FTL.CPUPerc,
		FTLMemPercent:   s.FTL.MemPerc,
	}, nil
}

// Sensors is the decoded, dashboard-relevant slice of GET /api/info/sensors.
type Sensors struct {
	// HasTemp is false when the host exposes no readable temperature sensor.
	HasTemp bool
	// CPUTemp is the primary CPU temperature in Unit.
	CPUTemp float64
	// HotLimit is the temperature FTL considers hot (drives the gauge scale).
	HotLimit float64
	// Unit is the temperature unit letter ("C"/"F"/"K").
	Unit string
}

// sensorsEnvelope mirrors GET /api/info/sensors.
type sensorsEnvelope struct {
	Sensors struct {
		CPUTemp  *float64 `json:"cpu_temp"`
		HotLimit float64  `json:"hot_limit"`
		Unit     string   `json:"unit"`
	} `json:"sensors"`
}

// SensorsInfo fetches GET /api/info/sensors. When the host reports no CPU
// temperature (cpu_temp null/absent), HasTemp is false and callers should hide
// the temperature gauge rather than render a misleading zero.
func (c *Client) SensorsInfo(ctx context.Context) (Sensors, error) {
	var env sensorsEnvelope
	if err := c.do(ctx, http.MethodGet, "/info/sensors", nil, &env); err != nil {
		return Sensors{}, err
	}

	s := env.Sensors
	out := Sensors{HotLimit: s.HotLimit, Unit: s.Unit}
	if s.CPUTemp != nil {
		out.HasTemp = true
		out.CPUTemp = *s.CPUTemp
	}
	return out, nil
}
