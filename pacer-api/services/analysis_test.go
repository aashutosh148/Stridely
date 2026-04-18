package services

import (
	"math"
	"testing"
)

func TestTSSTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		durationS      int
		avgPaceS       float64
		thresholdPaceS float64
		expected       float64
		tolerance      float64
	}{
		{"1h threshold", 3600, 240.0, 240.0, 100.0, 0.1},
		{"30m threshold", 1800, 240.0, 240.0, 50.0, 0.1},
		{"1h 10pct slower", 3600, 264.0, 240.0, 82.6, 0.2},
		{"1h 10pct faster", 3600, 218.0, 240.0, 121.2, 0.5},
		{"2h easy", 7200, 300.0, 240.0, 128.0, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateTSS(tt.durationS, tt.avgPaceS, tt.thresholdPaceS)
			if math.Abs(got-tt.expected) > tt.tolerance {
				t.Fatalf("TSS got %.2f want %.2f", got, tt.expected)
			}
		})
	}
}

func TestCTLFormulaKnownVector(t *testing.T) {
	// Constant TSS vector should converge toward steady state
	vector := make([]float64, 60)
	for i := range vector {
		vector[i] = 80
	}

	ctl := CalculateCTLFromTSS(vector)
	if ctl < 60 || ctl > 80 {
		t.Fatalf("unexpected CTL %.2f for constant vector", ctl)
	}
}

func TestATLFormulaKnownVector(t *testing.T) {
	vector := []float64{20, 40, 60, 80, 100, 90, 70}
	atl := CalculateATLFromTSS(vector)
	if atl <= 0 {
		t.Fatalf("ATL must be positive, got %.2f", atl)
	}
	if atl < 40 || atl > 90 {
		t.Fatalf("unexpected ATL %.2f", atl)
	}
}

func TestTSBFormulaAndStates(t *testing.T) {
	tests := []struct {
		ctl   float64
		atl   float64
		state string
	}{
		{100, 80, "fresh"},
		{100, 90, "optimal_race_form"},
		{100, 100, "productive_training"},
		{100, 115, "tired"},
		{100, 130, "very_tired"},
	}

	for _, tt := range tests {
		tsb := CalculateTSB(tt.ctl, tt.atl)
		state := "productive_training"
		switch {
		case tsb > 15:
			state = "fresh"
		case tsb > 5:
			state = "optimal_race_form"
		case tsb > -10:
			state = "productive_training"
		case tsb > -20:
			state = "tired"
		default:
			state = "very_tired"
		}
		if state != tt.state {
			t.Fatalf("TSB state mismatch for ctl %.1f atl %.1f: got %s want %s", tt.ctl, tt.atl, state, tt.state)
		}
	}
}
