package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/llm"
	"github.com/aashutosh148/Stridely/pacer-api/models"
	"github.com/aashutosh148/Stridely/pacer-api/tools"
)

const maxToolIterations = 12

// AgentContext holds all context needed for an agent conversation
type AgentContext struct {
	User             *models.User
	WorkingMemory    *WorkingMemory
	SemanticFacts    []SemanticFact
	ActivePlan       *models.TrainingBlock
	CurrentWeek      []models.Workout
	RecentActivities []models.Activity
}

// WorkingMemory represents the current state cached in Redis
type WorkingMemory struct {
	AsOfDate       string           `json:"as_of_date"`
	ReadinessScore int              `json:"readiness_score"`
	ReadinessLevel string           `json:"readiness_level"`
	CTL            float64          `json:"ctl"`
	ATL            float64          `json:"atl"`
	TSB            float64          `json:"tsb"`
	DaysToRace     int              `json:"days_to_race"`
	ActiveFlags    []string         `json:"active_flags"`
	StreakDays     int              `json:"streak_days"`
	LastActivity   *ActivitySummary `json:"last_activity"`
	LastLongRun    *ActivitySummary `json:"last_long_run"`
	TodaysWorkout  *WorkoutSummary  `json:"todays_workout"`
	WeekProgress   *WeekProgress    `json:"week_progress"`
}

type ActivitySummary struct {
	Date       string  `json:"date"`
	Type       string  `json:"type"`
	DistanceKM float64 `json:"distance_km"`
	DurationS  int     `json:"duration_s"`
	AvgPaceS   int     `json:"avg_pace_s"`
}

type WorkoutSummary struct {
	Type        string  `json:"type"`
	DistanceKM  float64 `json:"distance_km"`
	Description string  `json:"description"`
}

type WeekProgress struct {
	CompletedKM float64 `json:"completed_km"`
	PlannedKM   float64 `json:"planned_km"`
	Compliance  float64 `json:"compliance"`
}

// SemanticFact represents a learned pattern about the athlete
type SemanticFact struct {
	ID         uuid.UUID `json:"id"`
	FactType   string    `json:"fact_type"`
	Notes      string    `json:"notes"`
	Confidence float64   `json:"confidence"`
}

// AgentService handles the agentic loop
type AgentService struct {
	db          *db.Postgres
	redis       *db.Redis
	llmClient   llm.Client
	tools       *tools.Registry
	analysisSvc *AnalysisService
	memorySvc   *MemoryService
}

// NewAgentService creates a new agent service
func NewAgentService(
	database *db.Postgres,
	redisDB *db.Redis,
	llmClient llm.Client,
	toolRegistry *tools.Registry,
	analysisSvc *AnalysisService,
	memorySvc *MemoryService,
) *AgentService {
	return &AgentService{
		db:          database,
		redis:       redisDB,
		llmClient:   llmClient,
		tools:       toolRegistry,
		analysisSvc: analysisSvc,
		memorySvc:   memorySvc,
	}
}

// AssembleContext fetches all context needed for the agent in parallel
func (a *AgentService) AssembleContext(ctx context.Context, userID string) (*AgentContext, error) {
	var ac AgentContext
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make(chan error, 5)

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Parallel fetch from Redis + Postgres
	wg.Add(5)

	// Fetch working memory from Redis
	go func() {
		defer wg.Done()
		wm, err := a.getWorkingMemory(ctx, userID)
		mu.Lock()
		ac.WorkingMemory = wm
		mu.Unlock()
		if err != nil {
			errs <- fmt.Errorf("working memory: %w", err)
		}
	}()

	// Fetch user profile
	go func() {
		defer wg.Done()
		user, err := a.getUser(ctx, uid)
		mu.Lock()
		ac.User = user
		mu.Unlock()
		if err != nil {
			errs <- fmt.Errorf("user: %w", err)
		}
	}()

	// Fetch semantic facts
	go func() {
		defer wg.Done()
		facts, err := a.getSemanticFacts(ctx, uid, 0.5)
		mu.Lock()
		ac.SemanticFacts = facts
		mu.Unlock()
		if err != nil {
			errs <- fmt.Errorf("semantic facts: %w", err)
		}
	}()

	// Fetch active training plan
	go func() {
		defer wg.Done()
		plan, err := a.getActivePlan(ctx, uid)
		mu.Lock()
		ac.ActivePlan = plan
		mu.Unlock()
		if err != nil && err != sql.ErrNoRows {
			errs <- fmt.Errorf("active plan: %w", err)
		}
	}()

	// Fetch recent activities
	go func() {
		defer wg.Done()
		acts, err := a.getRecentActivities(ctx, uid, 5)
		mu.Lock()
		ac.RecentActivities = acts
		mu.Unlock()
		if err != nil {
			errs <- fmt.Errorf("recent activities: %w", err)
		}
	}()

	wg.Wait()
	close(errs)

	// Check for errors
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return &ac, nil
}

// BuildSystemPrompt constructs the system prompt with all context
func (a *AgentService) BuildSystemPrompt(ac *AgentContext) string {
	var sb strings.Builder

	sb.WriteString(`You are Pacer — an expert AI marathon coaching agent.
You have full access to the athlete's training data, health metrics,
and training history. You are proactive, evidence-based, and honest.
Use tools to look up current data before giving advice.
Always respond in a tone appropriate to the athlete's tier.`)

	// Inject runner profile
	if ac.User != nil {
		goalTime := "not set"
		if ac.User.GoalTimeS.Valid {
			goalTime = formatGoalTime(int(ac.User.GoalTimeS.Int32))
		}
		raceDate := "not set"
		daysTo := 0
		if ac.User.TargetRaceDate.Valid {
			raceDate = ac.User.TargetRaceDate.Time.Format("2006-01-02")
			daysTo = int(time.Until(ac.User.TargetRaceDate.Time).Hours() / 24)
		}

		sb.WriteString(fmt.Sprintf(`

## Athlete Profile
Name: %s | Tier: %s | Goal: %s | Race: %s | Days to race: %d`,
			ac.User.Email,
			ac.User.RunnerTier,
			goalTime,
			raceDate,
			daysTo,
		))
	}

	// Inject working memory state
	if ac.WorkingMemory != nil {
		wm := ac.WorkingMemory
		flags := "none"
		if len(wm.ActiveFlags) > 0 {
			flags = strings.Join(wm.ActiveFlags, ", ")
		}
		sb.WriteString(fmt.Sprintf(`

## Current State
CTL: %.1f | ATL: %.1f | TSB: %.1f | Readiness: %s (%d/10)
Active flags: %s`,
			wm.CTL, wm.ATL, wm.TSB,
			wm.ReadinessLevel, wm.ReadinessScore,
			flags,
		))
	}

	// Inject top semantic facts (confidence > 0.7 only for prompt)
	if len(ac.SemanticFacts) > 0 {
		highConfFacts := []SemanticFact{}
		for _, f := range ac.SemanticFacts {
			if f.Confidence > 0.7 {
				highConfFacts = append(highConfFacts, f)
			}
		}
		if len(highConfFacts) > 0 {
			sb.WriteString("\n\n## Known Patterns About This Athlete")
			for _, f := range highConfFacts {
				sb.WriteString(fmt.Sprintf("\n- %s", f.Notes))
			}
		}
	}

	sb.WriteString(`

Always call relevant tools before answering.
Never guess at current data — look it up.`)

	return sb.String()
}

// RunLoop executes the agentic reasoning loop with tool calls
func (a *AgentService) RunLoop(
	ctx context.Context,
	userID string,
	userMessage string,
	streamCh chan<- string,
) (string, error) {
	// Assemble context
	agentCtx, err := a.AssembleContext(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("context assembly: %w", err)
	}

	systemPrompt := a.BuildSystemPrompt(agentCtx)

	// Message history for this invocation
	messages := []llm.Message{
		llm.NewTextMessage("user", userMessage),
	}

	// Main loop
	lastPartialText := ""
	for i := 0; i < maxToolIterations; i++ {
		// Build completion request
		req := llm.CompletionRequest{
			Model:        a.llmClient.GetDefaultModel(),
			Messages:     messages,
			SystemPrompt: systemPrompt,
			Tools:        a.tools.GetAllDefinitions(),
			MaxTokens:    2048,
		}

		resp, err := a.llmClient.Complete(ctx, req)
		if err != nil {
			slog.Warn("llm call failed, retrying once", "error", err)
			time.Sleep(500 * time.Millisecond)
			resp, err = a.llmClient.Complete(ctx, req)
			if err != nil {
				slog.Error("llm call failed after retry", "error", err)
				if lastPartialText != "" {
					return lastPartialText, nil
				}
				return "", fmt.Errorf("llm call: %w", err)
			}
		}

		// Collect final text for return
		finalText := llm.ExtractText(resp.Content)
		if finalText != "" {
			lastPartialText = finalText
		}

		// Handle end_turn stop reason
		if resp.StopReason == "end_turn" {
			// Stream final tokens if channel provided
			if streamCh != nil && finalText != "" {
				for _, r := range finalText {
					streamCh <- fmt.Sprintf(`{"type":"token","text":%q}`, string(r))
				}
			}

			// LLD Section 8.4 - Post-loop update: store coaching moment
			if a.memorySvc != nil {
				go func() {
					bgCtx := context.Background()
					uid, _ := uuid.Parse(userID)
					summary := fmt.Sprintf("User asked: %s. Agent responded with coaching advice.", userMessage)
					insights := []string{finalText}

					if err := a.memorySvc.PostLoopUpdate(bgCtx, uid, summary, insights); err != nil {
						// Log error but don't fail the request
						fmt.Printf("PostLoopUpdate error: %v\n", err)
					}
				}()
			}

			return finalText, nil
		}

		// Handle tool_use stop reason
		if resp.StopReason == "tool_use" {
			// Extract tool uses
			toolUses := llm.ExtractToolUses(resp.Content)
			if len(toolUses) == 0 {
				fmt.Printf("Tool error: tool_use stop reason but no tools found\n")
				return finalText, fmt.Errorf("tool_use stop reason but no tools found")
			}

			// Notify stream that tools are running
			if streamCh != nil {
				for _, tu := range toolUses {
					streamCh <- fmt.Sprintf(`{"type":"tool_call","tool":"%s","status":"running"}`, tu.Name)
				}
			}

			// Execute tools in parallel
			toolResults := a.tools.ExecuteAll(ctx, userID, toolUses)

			if streamCh != nil {
				for _, tu := range toolUses {
					streamCh <- fmt.Sprintf(`{"type":"tool_result","tool":"%s","status":"done"}`, tu.Name)
				}
			}

			// Append assistant turn + tool results to history
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: resp.Content,
			})
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: toolResults,
			})
			continue
		}

		// Unexpected stop reason, return what we have
		return finalText, nil
	}

	loopErr := fmt.Errorf("max tool iterations (%d) exceeded", maxToolIterations)
	fmt.Printf("Agent loop error: %v\n", loopErr)
	return "", loopErr
}

// Helper functions

func formatGoalTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
}

// Database helper methods

func (a *AgentService) getUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `SELECT id, email, runner_tier, subscription_tier, goal_time_s, 
	          target_race_date, threshold_pace_s, threshold_hr, max_hr, weight_kg,
	          onboarded_at, strava_athlete_id, garmin_user_id, preferred_language,
	          notification_prefs, created_at, updated_at
	          FROM users WHERE id = $1`
	var u models.User
	err := a.db.Pool.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.Email, &u.RunnerTier, &u.SubscriptionTier, &u.GoalTimeS,
		&u.TargetRaceDate, &u.ThresholdPaceS, &u.ThresholdHR, &u.MaxHR, &u.WeightKg,
		&u.OnboardedAt, &u.StravaAthleteID, &u.GarminUserID, &u.PreferredLanguage,
		&u.NotificationPrefs, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (a *AgentService) getActivePlan(ctx context.Context, userID uuid.UUID) (*models.TrainingBlock, error) {
	query := `SELECT id, user_id, phase, block_start, block_end, target_race, 
	          goal_time_s, peak_ctl, is_active, created_at
	          FROM training_blocks WHERE user_id = $1 AND is_active = true
	          ORDER BY created_at DESC LIMIT 1`
	var tb models.TrainingBlock
	err := a.db.Pool.QueryRow(ctx, query, userID).Scan(
		&tb.ID, &tb.UserID, &tb.Phase, &tb.BlockStart, &tb.BlockEnd,
		&tb.TargetRace, &tb.GoalTimeS, &tb.PeakCTL, &tb.IsActive, &tb.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tb, nil
}

func (a *AgentService) getRecentActivities(ctx context.Context, userID uuid.UUID, limit int) ([]models.Activity, error) {
	query := `SELECT id, user_id, strava_id, activity_date, workout_type, 
			distance_m, duration_s, elevation_gain_m, avg_pace_s, 
			avg_hr, max_hr, tss, intensity_factor
		FROM activities
		WHERE user_id = $1
		ORDER BY activity_date DESC
		LIMIT $2`
	rows, err := a.db.Pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []models.Activity
	for rows.Next() {
		var a models.Activity
		err := rows.Scan(
			&a.ID, &a.UserID, &a.StravaID, &a.ActivityDate, &a.WorkoutType,
			&a.DistanceM, &a.DurationS, &a.ElevationGainM, &a.AvgPaceS,
			&a.AvgHR, &a.MaxHR, &a.TSS, &a.IntensityFactor,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, nil
}

func (a *AgentService) getSemanticFacts(ctx context.Context, userID uuid.UUID, minConfidence float64) ([]SemanticFact, error) {
	// Session 6 - Get semantic facts from memory service
	if a.memorySvc == nil {
		return []SemanticFact{}, nil
	}

	facts, err := a.memorySvc.GetSemanticFacts(ctx, userID, minConfidence)
	if err != nil {
		return nil, fmt.Errorf("get semantic facts: %w", err)
	}

	// Convert models.SemanticFact to agent.SemanticFact format
	result := make([]SemanticFact, 0, len(facts))
	for _, f := range facts {
		result = append(result, SemanticFact{
			ID:         f.ID,
			FactType:   f.FactKey,
			Notes:      f.Notes,
			Confidence: f.Confidence,
		})
	}

	return result, nil
}

func (a *AgentService) getWorkingMemory(ctx context.Context, userID string) (*WorkingMemory, error) {
	uid, _ := uuid.Parse(userID)

	// Session 6 - Try to get from Redis first
	if a.redis != nil {
		wm, err := a.redis.GetWorkingMemory(ctx, uid)
		if err == nil && wm != nil {
			// Convert db.WorkingMemory to agent WorkingMemory
			return convertWorkingMemory(wm), nil
		}
		// Cache miss or error - rebuild from Postgres
		if a.redis.Pool != nil {
			wm, err := a.redis.RebuildWorkingMemory(ctx, uid)
			if err == nil && wm != nil {
				return convertWorkingMemory(wm), nil
			}
		}
	}

	// Fallback: compute working memory on-the-fly from CTL/ATL/TSB
	ctl, atl, tsb, err := a.analysisSvc.GetLatestFitnessMetrics(ctx, uid)
	if err != nil {
		// No metrics yet, return minimal working memory
		return &WorkingMemory{
			AsOfDate:       time.Now().Format("2006-01-02"),
			ReadinessScore: 5,
			ReadinessLevel: "unknown",
			CTL:            0,
			ATL:            0,
			TSB:            0,
			ActiveFlags:    []string{},
			StreakDays:     0,
		}, nil
	}

	// Determine readiness level based on TSB
	readinessLevel := "productive_training"
	readinessScore := 7
	if tsb > 15 {
		readinessLevel = "fresh"
		readinessScore = 9
	} else if tsb > 5 {
		readinessLevel = "optimal_race_form"
		readinessScore = 8
	} else if tsb < -20 {
		readinessLevel = "very_tired"
		readinessScore = 3
	} else if tsb < -10 {
		readinessLevel = "tired"
		readinessScore = 5
	}

	return &WorkingMemory{
		AsOfDate:       time.Now().Format("2006-01-02"),
		ReadinessScore: readinessScore,
		ReadinessLevel: readinessLevel,
		CTL:            ctl,
		ATL:            atl,
		TSB:            tsb,
		ActiveFlags:    []string{},
		StreakDays:     0,
	}, nil
}

// convertWorkingMemory converts db.WorkingMemory to agent.WorkingMemory
func convertWorkingMemory(dbWM *db.WorkingMemory) *WorkingMemory {
	wm := &WorkingMemory{
		AsOfDate:       dbWM.LastRebuilt.Format("2006-01-02"),
		ReadinessScore: int(dbWM.ReadinessScore),
		CTL:            dbWM.CTL,
		ATL:            dbWM.ATL,
		TSB:            dbWM.TSB,
		ActiveFlags:    dbWM.ActiveFlags,
	}

	// Determine readiness level based on TSB
	if dbWM.TSB > 15 {
		wm.ReadinessLevel = "fresh"
	} else if dbWM.TSB > 5 {
		wm.ReadinessLevel = "optimal_race_form"
	} else if dbWM.TSB < -20 {
		wm.ReadinessLevel = "very_tired"
	} else if dbWM.TSB < -10 {
		wm.ReadinessLevel = "tired"
	} else {
		wm.ReadinessLevel = "productive_training"
	}

	// Convert today's workout
	if dbWM.TodayWorkout != nil {
		wm.TodaysWorkout = &WorkoutSummary{
			Type:        dbWM.TodayWorkout.WorkoutType,
			DistanceKM:  dbWM.TodayWorkout.DistanceKM,
			Description: dbWM.TodayWorkout.Description,
		}
	}

	// Convert last activity
	if dbWM.LastActivity != nil {
		wm.LastActivity = &ActivitySummary{
			Date:       dbWM.LastActivity.Date.Format("2006-01-02"),
			Type:       dbWM.LastActivity.Type,
			DistanceKM: dbWM.LastActivity.DistanceKM,
			DurationS:  dbWM.LastActivity.DurationS,
			AvgPaceS:   dbWM.LastActivity.AvgPaceS,
		}
	}

	// Convert week progress
	if dbWM.WeekProgress != nil {
		wm.WeekProgress = &WeekProgress{
			CompletedKM: dbWM.WeekProgress.CompletedKM,
			PlannedKM:   dbWM.WeekProgress.PlannedKM,
			Compliance:  0,
		}
		if dbWM.WeekProgress.PlannedKM > 0 {
			wm.WeekProgress.Compliance = dbWM.WeekProgress.CompletedKM / dbWM.WeekProgress.PlannedKM
		}
	}

	return wm
}
