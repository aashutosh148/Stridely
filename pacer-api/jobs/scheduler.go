package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/services"
)

type Dependencies struct {
	DB       *db.Postgres
	Redis    *db.Redis
	Planning *services.PlanningService
	Agent    *services.AgentService
	Memory   *services.MemoryService
	Analysis *services.AnalysisService

	Garmin   GarminSyncer
	Notifier Notifier
	Archiver S3Archiver
}

type GarminSyncer interface {
	GetHRVData(ctx context.Context, userID uuid.UUID, date string) (*HRVData, error)
	GetSleepData(ctx context.Context, userID uuid.UUID, date string) (*SleepData, error)
	SyncDailyData(ctx context.Context, userID uuid.UUID, date string) (*GarminDailyData, error)
}

type Notifier interface {
	Push(ctx context.Context, userID uuid.UUID, eventType string, payload map[string]any) error
}

type S3Archiver interface {
	PutJSON(ctx context.Context, key string, payload any) error
}

type noopGarminSyncer struct{}

func (n *noopGarminSyncer) GetHRVData(context.Context, uuid.UUID, string) (*HRVData, error) {
	return nil, nil
}
func (n *noopGarminSyncer) GetSleepData(context.Context, uuid.UUID, string) (*SleepData, error) {
	return nil, nil
}
func (n *noopGarminSyncer) SyncDailyData(context.Context, uuid.UUID, string) (*GarminDailyData, error) {
	return nil, nil
}

type logNotifier struct{}

func (l *logNotifier) Push(ctx context.Context, userID uuid.UUID, eventType string, payload map[string]any) error {
	body, _ := json.Marshal(payload)
	slog.InfoContext(ctx, "job notification", "user_id", userID, "event", eventType, "payload", string(body))
	return nil
}

func SetupScheduler(deps *Dependencies) *cron.Cron {
	if deps.Garmin == nil {
		deps.Garmin = &noopGarminSyncer{}
	}
	if deps.Notifier == nil {
		deps.Notifier = &logNotifier{}
	}

	c := cron.New(cron.WithLocation(time.UTC))

	_, _ = c.AddFunc("0 6 * * *", func() {
		if err := RunMorningReadiness(deps); err != nil {
			slog.Error("morning readiness job failed", "error", err)
		}
	})

	_, _ = c.AddFunc("0 2 * * *", func() {
		if err := RunGarminSync(deps); err != nil {
			slog.Error("garmin sync job failed", "error", err)
		}
	})

	_, _ = c.AddFunc("0 8 * * 1", func() {
		if err := RunWeeklyCheckin(deps); err != nil {
			slog.Error("weekly checkin job failed", "error", err)
		}
	})

	_, _ = c.AddFunc("0 3 * * 0", func() {
		if err := RunMemoryCompression(deps); err != nil {
			slog.Error("memory compression job failed", "error", err)
		}
	})

	_, _ = c.AddFunc("0 4 1 * *", func() {
		if err := RunGenomeRecalibration(deps); err != nil {
			slog.Error("genome recalibration job failed", "error", err)
		}
	})

	c.Start()
	slog.Info("background scheduler started", "jobs", 5)
	return c
}
