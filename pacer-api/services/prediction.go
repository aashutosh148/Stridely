package services

import (
	"math"
)

type PredictionService struct{}

func NewPredictionService() *PredictionService {
	return &PredictionService{}
}

type FinishTimePrediction struct {
	PredictedTimeSeconds int     `json:"predicted_time_seconds"`
	ConfidenceBandLow    int     `json:"confidence_band_low"`
	ConfidenceBandHigh   int     `json:"confidence_band_high"`
	WeeklyDeltaSeconds   int     `json:"weekly_delta_seconds"`
}

func (s *PredictionService) PredictFinishTime(d1 float64, t1 float64, d2 float64, vo2max float64, trainingLoad float64, lastWeekTime float64) *FinishTimePrediction {
	// Riegel: t2 = t1 * (d2/d1)^1.06
	riegelTime := t1 * math.Pow(d2/d1, 1.06)
	
	// Composite weighted by vo2max and training load
	predictedTime := riegelTime * 0.95 // Dummy adjustment to simulate weighting
	
	weeklyDelta := 0
	if lastWeekTime > 0 {
		weeklyDelta = int(predictedTime - lastWeekTime)
	}
	
	return &FinishTimePrediction{
		PredictedTimeSeconds: int(predictedTime),
		ConfidenceBandLow:    int(predictedTime * 0.95),
		ConfidenceBandHigh:   int(predictedTime * 1.05),
		WeeklyDeltaSeconds:   weeklyDelta,
	}
}

type InjuryRiskPrediction struct {
	Score float64 `json:"score"`
}

func (s *PredictionService) PredictInjuryRisk(acwr float64, hrvTrend float64, shoeMileage float64, lrBalance float64, historyScore float64) *InjuryRiskPrediction {
	// ACWR(35%) + HRV trend(20%) + shoe mileage(15%) + LR balance(15%) + history(15%)
	score := (acwr * 0.35) + (hrvTrend * 0.20) + (shoeMileage * 0.15) + (lrBalance * 0.15) + (historyScore * 0.15)
	return &InjuryRiskPrediction{Score: score}
}

type WallPointPrediction struct {
	DistanceKm float64 `json:"distance_km"`
}

func (s *PredictionService) PredictWallPoint(glycogenLevel float64, pace float64) *WallPointPrediction {
	// Simple distance until glycogen depletion
	distance := glycogenLevel / (pace * 0.1)
	return &WallPointPrediction{DistanceKm: distance}
}
