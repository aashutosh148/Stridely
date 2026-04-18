package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

const weeklyCheckinPrompt = "Generate the weekly check-in for this athlete. Look at last week's training data and ask 3 relevant questions about how they are feeling and any upcoming life events that might affect training."

func RunWeeklyCheckin(deps *Dependencies) error {
	if deps.Agent == nil {
		slog.Warn("weekly checkin skipped: agent service not configured")
		return nil
	}

	ctx := context.Background()
	userIDs, err := listUsersWithActivePlan(ctx, deps)
	if err != nil {
		return err
	}

	for _, uid := range userIDs {
		result, err := deps.Agent.RunLoop(ctx, uid.String(), weeklyCheckinPrompt, nil)
		if err != nil {
			slog.Warn("weekly checkin generation failed", "user_id", uid, "error", err)
			continue
		}

		payload := map[string]any{
			"title":   "Weekly Check-in",
			"message": result,
		}
		_ = deps.Notifier.Push(ctx, uid, "weekly.checkin", payload)

		_, _ = deps.DB.Pool.Exec(ctx, `
      INSERT INTO episodic_memories (
        user_id, memory_type, event_date, title, summary, content, importance_score, tags, compressed
      ) VALUES (
        $1, 'weekly_summary', CURRENT_DATE, $2, $3, $4::jsonb, 0.7, ARRAY['weekly_checkin','agent'], false
      )
    `, uid, "Weekly Check-in", truncate(result, 380), fmt.Sprintf(`{"full_text":%q}`, result))
	}

	slog.Info("weekly checkin job complete", "users", len(userIDs))
	return nil
}

func listUsersWithActivePlan(ctx context.Context, deps *Dependencies) ([]uuid.UUID, error) {
	rows, err := deps.DB.Pool.Query(ctx, `
    SELECT DISTINCT user_id
    FROM training_blocks
    WHERE is_active = true
  `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		out = append(out, uid)
	}
	return out, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
