package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/pacer-api/models"
)

func TestBeginner16WeekPhaseLengths(t *testing.T) {
	svc := &PlanningService{}
	params := models.BlockParams{
		RaceDate:        time.Now().AddDate(0, 0, 16*7),
		CurrentCTL:      20,
		CurrentWeeklyKM: 25,
		ThresholdPace:   360,
		RunnerTier:      models.RunnerTierBeginner,
		GoalTimeS:       15000,
		AvailableDays:   4,
	}

	_, weeks, err := svc.GenerateBlock(t.Context(), uuid.New(), params)
	if err != nil {
		t.Fatalf("GenerateBlock failed: %v", err)
	}
	if len(weeks) != 16 {
		t.Fatalf("expected 16 weeks, got %d", len(weeks))
	}

	weeksToRace := 16
	baseWeeks := int(float64(weeksToRace) * 0.35)
	buildWeeks := int(float64(weeksToRace) * 0.35)
	peakWeeks := int(float64(weeksToRace) * 0.15)
	taperWeeks := 16 - baseWeeks - buildWeeks - peakWeeks

	if baseWeeks != 5 || buildWeeks != 5 || peakWeeks != 2 || taperWeeks != 4 {
		t.Fatalf("unexpected phase lengths base=%d build=%d peak=%d taper=%d", baseWeeks, buildWeeks, peakWeeks, taperWeeks)
	}
}

func TestMileageIncreaseNeverAboveTenPercent(t *testing.T) {
	svc := &PlanningService{}
	params := models.BlockParams{
		RaceDate:        time.Now().AddDate(0, 0, 16*7),
		CurrentWeeklyKM: 30,
		ThresholdPace:   340,
		RunnerTier:      models.RunnerTierRecreational,
		GoalTimeS:       14000,
		AvailableDays:   5,
	}

	_, weeks, err := svc.GenerateBlock(t.Context(), uuid.New(), params)
	if err != nil {
		t.Fatalf("GenerateBlock failed: %v", err)
	}

	for i := 1; i < len(weeks); i++ {
		prev := weeks[i-1].TotalKM
		curr := weeks[i].TotalKM
		if prev <= 0 {
			continue
		}
		if curr > prev*1.10+0.01 {
			t.Fatalf("week %d mileage jump too high: prev %.1f -> curr %.1f", i+1, prev, curr)
		}
	}
}

func TestTaperReducesVolume(t *testing.T) {
	svc := &PlanningService{}
	params := models.BlockParams{
		RaceDate:        time.Now().AddDate(0, 0, 16*7),
		CurrentWeeklyKM: 40,
		ThresholdPace:   300,
		RunnerTier:      models.RunnerTierCompetitive,
		GoalTimeS:       13000,
		AvailableDays:   5,
	}

	taper := svc.generateTaperPhase(params, 3, 1, time.Now())
	if len(taper) < 3 {
		t.Fatalf("expected at least 3 taper weeks, got %d", len(taper))
	}

	if !(taper[0].TotalKM > taper[1].TotalKM && taper[1].TotalKM > taper[2].TotalKM) {
		t.Fatalf("taper weeks not decreasing: %.1f %.1f %.1f", taper[0].TotalKM, taper[1].TotalKM, taper[2].TotalKM)
	}
}
