package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/models"
)

// PlanningService handles training plan generation and adjustment
type PlanningService struct {
	db *db.Postgres
}

func NewPlanningService(database *db.Postgres) *PlanningService {
	return &PlanningService{db: database}
}

// GenerateBlock generates a complete training block
func (s *PlanningService) GenerateBlock(ctx context.Context, userID uuid.UUID, params models.BlockParams) (*models.TrainingBlock, []models.WeekPlan, error) {
	// Calculate weeks to race
	weeksToRace := int(math.Ceil(time.Until(params.RaceDate).Hours() / 168.0))

	// Clamp to 12–20 weeks
	if weeksToRace < 12 {
		weeksToRace = 12
	}
	if weeksToRace > 20 {
		weeksToRace = 20
	}

	// Phase division: Base 35%, Build 35%, Peak 15%, Taper 15%
	baseWeeks := int(float64(weeksToRace) * 0.35)
	buildWeeks := int(float64(weeksToRace) * 0.35)
	peakWeeks := int(float64(weeksToRace) * 0.15)
	taperWeeks := weeksToRace - baseWeeks - buildWeeks - peakWeeks

	// Generate weekly plans
	var allWeeks []models.WeekPlan
	weekNum := 1
	startDate := time.Now()

	// Base phase
	baseWeeksPlans := s.generateBasePhase(params, baseWeeks, weekNum, startDate)
	allWeeks = append(allWeeks, baseWeeksPlans...)
	weekNum += baseWeeks
	startDate = startDate.AddDate(0, 0, baseWeeks*7)

	// Build phase
	buildWeeksPlans := s.generateBuildPhase(params, buildWeeks, weekNum, startDate)
	allWeeks = append(allWeeks, buildWeeksPlans...)
	weekNum += buildWeeks
	startDate = startDate.AddDate(0, 0, buildWeeks*7)

	// Peak phase
	peakWeeksPlans := s.generatePeakPhase(params, peakWeeks, weekNum, startDate)
	allWeeks = append(allWeeks, peakWeeksPlans...)
	weekNum += peakWeeks
	startDate = startDate.AddDate(0, 0, peakWeeks*7)

	// Taper phase
	taperWeeksPlans := s.generateTaperPhase(params, taperWeeks, weekNum, startDate)
	allWeeks = append(allWeeks, taperWeeksPlans...)

	// Hard cap weekly mileage increase to 10% across phase boundaries
	for i := 1; i < len(allWeeks); i++ {
		prev := allWeeks[i-1].TotalKM
		curr := allWeeks[i].TotalKM
		if prev <= 0 {
			continue
		}
		maxAllowed := prev * 1.10
		if curr > maxAllowed {
			allWeeks[i].TotalKM = math.Floor(maxAllowed*10) / 10
		}
	}

	// Create training block record
	block := &models.TrainingBlock{
		ID:         uuid.New(),
		UserID:     userID,
		Phase:      models.PhaseBase,
		BlockStart: time.Now(),
		BlockEnd:   params.RaceDate,
		TargetRace: sql.NullTime{Time: params.RaceDate, Valid: true},
		GoalTimeS:  sql.NullInt32{Int32: int32(params.GoalTimeS), Valid: true},
		PeakCTL:    sql.NullFloat64{Float64: s.calculatePeakCTL(params), Valid: true},
		IsActive:   true,
		CreatedAt:  time.Now(),
	}

	return block, allWeeks, nil
}

// generateBasePhase creates base building weeks
func (s *PlanningService) generateBasePhase(params models.BlockParams, weeks int, startWeek int, startDate time.Time) []models.WeekPlan {
	weekPlans := make([]models.WeekPlan, weeks)
	currentKM := params.CurrentWeeklyKM
	if currentKM == 0 {
		currentKM = 30 // Default starting mileage
	}

	for i := 0; i < weeks; i++ {
		isRecoveryWeek := (i+1)%4 == 0

		if isRecoveryWeek {
			currentKM *= 0.7 // Recovery week: 70% of previous
		} else if i > 0 {
			currentKM *= 1.08 // Gradual 8% increase (staying under 10%)
		}

		weekPlans[i] = models.WeekPlan{
			WeekNumber: startWeek + i,
			StartDate:  startDate.AddDate(0, 0, i*7),
			TotalKM:    math.Round(currentKM*10) / 10,
			Workouts:   s.generateBaseWeekWorkouts(params, currentKM, isRecoveryWeek),
		}
	}

	return weekPlans
}

// generateBuildPhase creates build phase weeks with quality sessions
func (s *PlanningService) generateBuildPhase(params models.BlockParams, weeks int, startWeek int, startDate time.Time) []models.WeekPlan {
	weekPlans := make([]models.WeekPlan, weeks)
	currentKM := params.CurrentWeeklyKM * 1.3 // Build on base phase
	if currentKM == 0 {
		currentKM = 45
	}

	for i := 0; i < weeks; i++ {
		isRecoveryWeek := (i+1)%4 == 0

		if isRecoveryWeek {
			currentKM *= 0.75
		} else if i > 0 {
			currentKM *= 1.06
		}

		weekPlans[i] = models.WeekPlan{
			WeekNumber: startWeek + i,
			StartDate:  startDate.AddDate(0, 0, i*7),
			TotalKM:    math.Round(currentKM*10) / 10,
			Workouts:   s.generateBuildWeekWorkouts(params, currentKM, isRecoveryWeek),
		}
	}

	return weekPlans
}

// generatePeakPhase creates peak weeks with race-specific workouts
func (s *PlanningService) generatePeakPhase(params models.BlockParams, weeks int, startWeek int, startDate time.Time) []models.WeekPlan {
	weekPlans := make([]models.WeekPlan, weeks)
	peakKM := s.calculatePeakWeeklyKM(params)

	for i := 0; i < weeks; i++ {
		weekPlans[i] = models.WeekPlan{
			WeekNumber: startWeek + i,
			StartDate:  startDate.AddDate(0, 0, i*7),
			TotalKM:    peakKM,
			Workouts:   s.generatePeakWeekWorkouts(params, peakKM),
		}
	}

	return weekPlans
}

// generateTaperPhase creates 3-week taper targeting TSB=+12 on race day
func (s *PlanningService) generateTaperPhase(params models.BlockParams, weeks int, startWeek int, startDate time.Time) []models.WeekPlan {
	if weeks < 3 {
		weeks = 3
	}

	weekPlans := make([]models.WeekPlan, weeks)
	peakKM := s.calculatePeakWeeklyKM(params)

	// Taper progression: 75% -> 60% -> 40% of peak week
	taperPercentages := []float64{0.75, 0.60, 0.40}

	for i := 0; i < weeks && i < len(taperPercentages); i++ {
		weekKM := peakKM * taperPercentages[i]
		weekPlans[i] = models.WeekPlan{
			WeekNumber: startWeek + i,
			StartDate:  startDate.AddDate(0, 0, i*7),
			TotalKM:    math.Round(weekKM*10) / 10,
			Workouts:   s.generateTaperWeekWorkouts(params, weekKM, i),
		}
	}

	return weekPlans
}

// generateBaseWeekWorkouts creates workouts for a base week
func (s *PlanningService) generateBaseWeekWorkouts(params models.BlockParams, weekKM float64, isRecovery bool) []models.WorkoutPlan {
	workouts := []models.WorkoutPlan{}
	daysPerWeek := params.AvailableDays
	if daysPerWeek == 0 {
		daysPerWeek = 5 // Default
	}

	// Long run (Sunday): 25-30% of weekly mileage, capped at 35km
	longRunKM := math.Min(weekKM*0.28, 35.0)
	easyPace := params.ThresholdPace + 60 // Threshold + 60s/km = easy pace

	workouts = append(workouts, models.WorkoutPlan{
		DayOfWeek:   0, // Sunday
		Type:        models.WorkoutTypeLong,
		DistanceKM:  math.Round(longRunKM*10) / 10,
		PaceMin:     easyPace,
		PaceMax:     easyPace + 30,
		HRZone:      2,
		RPETarget:   4,
		Description: fmt.Sprintf("%.1f km easy long run", longRunKM),
		Purpose:     "Build aerobic endurance",
	})

	// Distribute remaining mileage across other days
	remainingKM := weekKM - longRunKM
	easyRunKM := remainingKM / float64(daysPerWeek-1)

	// Easy runs (Tuesday, Thursday, Saturday)
	easyDays := []int{2, 4, 6}
	for i := 0; i < daysPerWeek-1 && i < len(easyDays); i++ {
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   easyDays[i],
			Type:        models.WorkoutTypeEasy,
			DistanceKM:  math.Round(easyRunKM*10) / 10,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 30,
			HRZone:      2,
			RPETarget:   3,
			Description: fmt.Sprintf("%.1f km easy run", easyRunKM),
			Purpose:     "Aerobic base building",
		})
	}

	return workouts
}

// generateBuildWeekWorkouts creates workouts with quality sessions
func (s *PlanningService) generateBuildWeekWorkouts(params models.BlockParams, weekKM float64, isRecovery bool) []models.WorkoutPlan {
	workouts := []models.WorkoutPlan{}
	easyPace := params.ThresholdPace + 60
	tempoPace := params.ThresholdPace + 15

	// Long run
	longRunKM := math.Min(weekKM*0.28, 35.0)
	workouts = append(workouts, models.WorkoutPlan{
		DayOfWeek:   0,
		Type:        models.WorkoutTypeLong,
		DistanceKM:  math.Round(longRunKM*10) / 10,
		PaceMin:     easyPace,
		PaceMax:     easyPace + 20,
		HRZone:      2,
		RPETarget:   5,
		Description: fmt.Sprintf("%.1f km long run with last 3km at tempo", longRunKM),
		Purpose:     "Endurance with quality finish",
	})

	// Tempo run (Wednesday)
	tempoKM := 10.0
	workouts = append(workouts, models.WorkoutPlan{
		DayOfWeek:   3,
		Type:        models.WorkoutTypeTempo,
		DistanceKM:  tempoKM,
		PaceMin:     tempoPace,
		PaceMax:     tempoPace + 10,
		HRZone:      4,
		RPETarget:   7,
		Description: "2km warmup + 6km tempo + 2km cooldown",
		Purpose:     "Lactate threshold development",
	})

	// Recovery and easy runs
	remainingKM := weekKM - longRunKM - tempoKM
	easyRunKM := remainingKM / 3

	easyDays := []int{2, 4, 6}
	for _, day := range easyDays {
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   day,
			Type:        models.WorkoutTypeEasy,
			DistanceKM:  math.Round(easyRunKM*10) / 10,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 30,
			HRZone:      2,
			RPETarget:   3,
			Description: fmt.Sprintf("%.1f km easy run", easyRunKM),
			Purpose:     "Active recovery",
		})
	}

	return workouts
}

// generatePeakWeekWorkouts creates race-specific peak workouts
func (s *PlanningService) generatePeakWeekWorkouts(params models.BlockParams, weekKM float64) []models.WorkoutPlan {
	workouts := []models.WorkoutPlan{}
	easyPace := params.ThresholdPace + 60
	tempoPace := params.ThresholdPace + 15
	racePace := float64(params.GoalTimeS) / 42.195

	// Long run with marathon pace segments
	longRunKM := math.Min(weekKM*0.30, 35.0)
	workouts = append(workouts, models.WorkoutPlan{
		DayOfWeek:   0,
		Type:        models.WorkoutTypeLong,
		DistanceKM:  math.Round(longRunKM*10) / 10,
		PaceMin:     easyPace,
		PaceMax:     racePace + 10,
		HRZone:      3,
		RPETarget:   6,
		Description: fmt.Sprintf("%.1f km with 3x3km at race pace", longRunKM),
		Purpose:     "Race pace simulation",
	})

	// Marathon pace workout
	workouts = append(workouts, models.WorkoutPlan{
		DayOfWeek:   3,
		Type:        models.WorkoutTypeTempo,
		DistanceKM:  16.0,
		PaceMin:     racePace - 5,
		PaceMax:     racePace + 5,
		HRZone:      3,
		RPETarget:   6,
		Description: "2km warmup + 12km at marathon pace + 2km cooldown",
		Purpose:     "Race pace confidence",
	})

	// Tempo intervals
	workouts = append(workouts, models.WorkoutPlan{
		DayOfWeek:   5,
		Type:        models.WorkoutTypeInterval,
		DistanceKM:  12.0,
		PaceMin:     tempoPace - 10,
		PaceMax:     tempoPace,
		HRZone:      4,
		RPETarget:   7,
		Description: "2km warmup + 4x2km @ tempo (90s rest) + 2km cooldown",
		Purpose:     "VO2max maintenance",
	})

	// Easy runs
	remainingKM := weekKM - longRunKM - 16.0 - 12.0
	easyRunKM := remainingKM / 2

	for _, day := range []int{2, 4} {
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   day,
			Type:        models.WorkoutTypeEasy,
			DistanceKM:  math.Round(easyRunKM*10) / 10,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 30,
			HRZone:      2,
			RPETarget:   3,
			Description: fmt.Sprintf("%.1f km easy run", easyRunKM),
			Purpose:     "Recovery",
		})
	}

	return workouts
}

// generateTaperWeekWorkouts creates taper week workouts
func (s *PlanningService) generateTaperWeekWorkouts(params models.BlockParams, weekKM float64, weekIndex int) []models.WorkoutPlan {
	workouts := []models.WorkoutPlan{}
	easyPace := params.ThresholdPace + 60
	racePace := float64(params.GoalTimeS) / 42.195

	// Week-specific taper strategy
	if weekIndex == 0 {
		// Taper week 1: Reduced volume, maintain intensity
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   0,
			Type:        models.WorkoutTypeLong,
			DistanceKM:  20.0,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 20,
			HRZone:      2,
			RPETarget:   4,
			Description: "20km easy long run",
			Purpose:     "Maintain endurance",
		})
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   3,
			Type:        models.WorkoutTypeInterval,
			DistanceKM:  8.0,
			PaceMin:     racePace - 10,
			PaceMax:     racePace,
			HRZone:      3,
			RPETarget:   6,
			Description: "2km warmup + 4x1km @ race pace (60s rest) + 2km cooldown",
			Purpose:     "Race sharpness",
		})
	} else if weekIndex == 1 {
		// Taper week 2: Further reduction
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   0,
			Type:        models.WorkoutTypeLong,
			DistanceKM:  16.0,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 15,
			HRZone:      2,
			RPETarget:   3,
			Description: "16km easy run",
			Purpose:     "Active recovery",
		})
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   3,
			Type:        models.WorkoutTypeEasy,
			DistanceKM:  8.0,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 20,
			HRZone:      2,
			RPETarget:   3,
			Description: "8km easy with 4x100m strides",
			Purpose:     "Neuromuscular activation",
		})
	} else {
		// Race week
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   2,
			Type:        models.WorkoutTypeEasy,
			DistanceKM:  6.0,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 20,
			HRZone:      2,
			RPETarget:   2,
			Description: "6km easy shakeout",
			Purpose:     "Loosen up",
		})
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   5,
			Type:        models.WorkoutTypeEasy,
			DistanceKM:  4.0,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 15,
			HRZone:      2,
			RPETarget:   2,
			Description: "4km easy with 3x100m strides",
			Purpose:     "Final tune-up",
		})
	}

	// Fill remaining days with very easy runs
	currentTotal := 0.0
	for _, w := range workouts {
		currentTotal += w.DistanceKM
	}
	if currentTotal < weekKM {
		remaining := weekKM - currentTotal
		workouts = append(workouts, models.WorkoutPlan{
			DayOfWeek:   4,
			Type:        models.WorkoutTypeEasy,
			DistanceKM:  math.Round(remaining*10) / 10,
			PaceMin:     easyPace,
			PaceMax:     easyPace + 30,
			HRZone:      1,
			RPETarget:   2,
			Description: fmt.Sprintf("%.1f km recovery run", remaining),
			Purpose:     "Recovery",
		})
	}

	return workouts
}

// Helper calculations

func (s *PlanningService) calculatePeakCTL(params models.BlockParams) float64 {
	// Estimate peak CTL based on runner tier and goal
	baseCTL := params.CurrentCTL
	if baseCTL == 0 {
		baseCTL = 50 // Default
	}

	// Target CTL based on tier
	var targetCTL float64
	switch params.RunnerTier {
	case models.RunnerTierSerious:
		targetCTL = 110
	case models.RunnerTierCompetitive:
		targetCTL = 95
	case models.RunnerTierRecreational:
		targetCTL = 75
	default:
		targetCTL = 60
	}

	// Gradually build to target
	return math.Min(baseCTL*1.6, targetCTL)
}

func (s *PlanningService) calculatePeakWeeklyKM(params models.BlockParams) float64 {
	// Peak week mileage based on tier
	switch params.RunnerTier {
	case models.RunnerTierSerious:
		return 85.0
	case models.RunnerTierCompetitive:
		return 70.0
	case models.RunnerTierRecreational:
		return 55.0
	default:
		return 45.0
	}
}

// AdjustWeeklyPlan adjusts the current week based on readiness
func (s *PlanningService) AdjustWeeklyPlan(ctx context.Context, userID uuid.UUID, readinessScore int, missedSessions int) error {
	// Implementation for dynamic plan adjustment
	// This would modify workouts based on TSB and readiness
	return nil
}

// SubstituteWorkout replaces a workout with an equivalent alternative
func (s *PlanningService) SubstituteWorkout(ctx context.Context, workoutID uuid.UUID, reason string) (*models.Workout, error) {
	// Implementation for workout substitution
	// e.g., replace intervals with tempo if fatigued
	return nil, nil
}
