package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

func RunMemoryCompression(deps *Dependencies) error {
	ctx := context.Background()
	userIDs, err := listAllUserIDs(ctx, deps)
	if err != nil {
		return err
	}

	for _, uid := range userIDs {
		if err := compressUserMemories(ctx, deps, uid); err != nil {
			slog.Warn("memory compression failed", "user_id", uid, "error", err)
		}
	}

	slog.Info("memory compression job complete", "users", len(userIDs))
	return nil
}

func compressUserMemories(ctx context.Context, deps *Dependencies, userID uuid.UUID) error {
	rows, err := deps.DB.Pool.Query(ctx, `
    SELECT id, event_date, title, summary, content
    FROM episodic_memories
    WHERE user_id = $1
      AND event_date < CURRENT_DATE - INTERVAL '90 days'
      AND compressed = false
    ORDER BY event_date ASC
  `, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type memRow struct {
		id      uuid.UUID
		date    time.Time
		title   string
		summary string
		content map[string]any
	}

	byWeek := map[string][]memRow{}

	for rows.Next() {
		var r memRow
		var contentJSON []byte
		if err := rows.Scan(&r.id, &r.date, &r.title, &r.summary, &contentJSON); err != nil {
			return err
		}
		_ = json.Unmarshal(contentJSON, &r.content)
		weekStart := startOfWeekUTC(r.date).Format("2006-01-02")
		byWeek[weekStart] = append(byWeek[weekStart], r)
	}

	for weekStart, items := range byWeek {
		if len(items) <= 7 {
			continue
		}

		summary := fmt.Sprintf("Compressed %d memories for week starting %s.", len(items), weekStart)
		title := fmt.Sprintf("Weekly memory summary (%s)", weekStart)

		titles := make([]string, 0, len(items))
		for _, item := range items {
			titles = append(titles, item.title)
		}

		content := map[string]any{
			"week_start": weekStart,
			"count":      len(items),
			"titles":     titles,
		}
		contentJSON, _ := json.Marshal(content)

		if deps.Archiver != nil {
			key := fmt.Sprintf("memory-archive/%s/%s.json", userID, weekStart)
			archivePayload := map[string]any{"records": items}
			if err := deps.Archiver.PutJSON(ctx, key, archivePayload); err != nil {
				slog.Warn("archive upload failed", "user_id", userID, "week_start", weekStart, "error", err)
			}
		}

		_, err := deps.DB.Pool.Exec(ctx, `
      INSERT INTO episodic_memories (
        user_id, memory_type, event_date, title, summary, content, importance_score, tags, compressed
      ) VALUES ($1, 'weekly_summary', $2, $3, $4, $5, 0.65, ARRAY['compression','weekly_summary'], false)
    `, userID, weekStart, title, summary, contentJSON)
		if err != nil {
			return err
		}

		ids := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.id)
		}
		_, err = deps.DB.Pool.Exec(ctx, `UPDATE episodic_memories SET compressed = true WHERE id = ANY($1)`, ids)
		if err != nil {
			return err
		}
	}

	return nil
}

func startOfWeekUTC(t time.Time) time.Time {
	t = t.UTC()
	day := int(t.Weekday())
	if day == 0 {
		day = 7
	}
	d := t.AddDate(0, 0, -(day - 1))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func listAllUserIDs(ctx context.Context, deps *Dependencies) ([]uuid.UUID, error) {
	rows, err := deps.DB.Pool.Query(ctx, `SELECT id FROM users`)
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
