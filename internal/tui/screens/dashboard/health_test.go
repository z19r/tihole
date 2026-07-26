package dashboard

import (
	"errors"
	"strings"
	"testing"

	"github.com/z19r/tihole/internal/pihole"
)

// sampleSystem returns populated host metrics for health-strip rendering.
func sampleSystem() pihole.SystemInfo {
	return pihole.SystemInfo{
		Uptime:         4*86400 + 9*3600,
		Procs:          192,
		NProcs:         4,
		CPUPercent:     12.5,
		Load1Percent:   2.478,
		MemUsedPercent: 6.787,
		MemUsedKiB:     263808,
		MemTotalKiB:    3886904,
	}
}

func TestRenderHealth_ShowsCPUMemoryAndUptime(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.system = sampleSystem()

	// Act
	out := m.renderHealth()

	// Assert
	for _, want := range []string{"CPU", "12.5%", "Memory", "6.8%", "192p", "4d 9h"} {
		if !strings.Contains(out, want) {
			t.Errorf("health strip missing %q\n%s", want, out)
		}
	}
}

func TestRenderHealth_TemperatureHiddenWithoutSensor(t *testing.T) {
	// Arrange: no readable sensor.
	m := newTestModel()
	m.system = sampleSystem()
	m.sensors = pihole.Sensors{HasTemp: false}

	// Act
	out := m.renderHealth()

	// Assert
	if !strings.Contains(out, "n/a") {
		t.Errorf("expected a temperature n/a cell when no sensor\n%s", out)
	}
}

func TestRenderHealth_TemperatureShownWithSensor(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.system = sampleSystem()
	m.sensors = pihole.Sensors{
		HasTemp:  true,
		CPUTemp:  48.2,
		HotLimit: 60,
		Unit:     "C",
	}

	// Act
	out := m.renderHealth()

	// Assert
	if !strings.Contains(out, "48°C") {
		t.Errorf("expected temperature reading in strip\n%s", out)
	}
}

func TestRenderHealth_AllClearWhenNoMessages(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.system = sampleSystem()
	m.messages = nil

	// Act
	out := m.renderHealth()

	// Assert
	if !strings.Contains(out, "All clear") {
		t.Errorf("expected an all-clear badge with zero messages\n%s", out)
	}
}

func TestRenderHealth_WarnsOnDiagnosisMessages(t *testing.T) {
	// Arrange
	m := newTestModel()
	m.system = sampleSystem()
	m.messages = []pihole.DiagnosisMessage{
		{Plain: "Gravity database is corrupt"},
		{Plain: "second message"},
	}

	// Act
	out := m.renderHealth()

	// Assert
	if !strings.Contains(out, "2 issues") {
		t.Errorf("expected issue count badge\n%s", out)
	}
	if !strings.Contains(out, "Gravity") {
		t.Errorf("expected newest message text (possibly truncated)\n%s", out)
	}
}

func TestRenderHealth_InlineNoteWhenSystemUnavailable(t *testing.T) {
	// Arrange: fetch failed and nothing decoded (NProcs still zero).
	m := newTestModel()
	m.errSystem = "connection refused"

	// Act
	out := m.renderHealth()

	// Assert
	if !strings.Contains(out, "system unavailable") {
		t.Errorf("expected inline unavailable note\n%s", out)
	}
}

func TestApplySystem_SpringsGaugesAndRecordsErrorsIndependently(t *testing.T) {
	// Arrange
	m := newTestModel()
	msg := systemMsg{
		system: pihole.SystemInfo{
			CPUPercent:     50,
			MemUsedPercent: 25,
			NProcs:         4,
		},
		sensors: pihole.Sensors{HasTemp: true, CPUTemp: 30, HotLimit: 60},
		msgErr:  errors.New("boom"),
	}

	// Act
	cmd := m.applySystem(msg)

	// Assert: gauge targets sprung, message error captured, system kept.
	if cmd == nil {
		t.Fatal("expected animation command from applySystem")
	}
	if m.cpuBar.Percent() < 0.49 || m.cpuBar.Percent() > 0.51 {
		t.Errorf("cpu gauge target = %.3f, want ~0.5", m.cpuBar.Percent())
	}
	if m.errMessages != "boom" {
		t.Errorf("errMessages = %q, want boom", m.errMessages)
	}
	if m.system.NProcs != 4 {
		t.Errorf("system not stored despite message error")
	}
}

func TestClampFrac(t *testing.T) {
	// Arrange / Act / Assert
	if clampFrac(-0.5) != 0 {
		t.Error("negative should clamp to 0")
	}
	if clampFrac(1.5) != 1 {
		t.Error("over-one should clamp to 1")
	}
	if clampFrac(0.42) != 0.42 {
		t.Error("in-range should pass through")
	}
}
