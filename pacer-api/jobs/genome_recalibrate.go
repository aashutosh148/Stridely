package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
)

func RunGenomeRecalibration(deps *Dependencies) error {
	ctx := context.Background()

	rows, err := deps.DB.Pool.Query(ctx, `
    SELECT user_id, COUNT(*) AS points
    FROM episodic_memories
    GROUP BY user_id
    HAVING COUNT(*) >= 60
  `)
	if err != nil {
		return err
	}
	defer rows.Close()

	processed := 0
	for rows.Next() {
		var uid uuid.UUID
		var points int
		if err := rows.Scan(&uid, &points); err != nil {
			return err
		}

		if err := recalibrateUserGenome(ctx, deps, uid, points); err != nil {
			slog.Warn("genome recalibration failed", "user_id", uid, "error", err)
			continue
		}
		processed++
	}

	slog.Info("genome recalibration job complete", "users", processed)
	return nil
}

func recalibrateUserGenome(ctx context.Context, deps *Dependencies, userID uuid.UUID, points int) error {
	ctlRate, atlRate, taperResp, err := computeGenomeSignals(ctx, deps, userID)
	if err != nil {
		return err
	}

	confidence := "low"
	switch {
	case points >= 180:
		confidence = "high"
	case points >= 100:
		confidence = "medium"
	}

	payload := map[string]any{
		"ctl_accumulation_rate": round3(ctlRate),
		"atl_recovery_rate":     round3(atlRate),
		"taper_response":        round3(taperResp),
		"updated_by_job":        "genome_recalibrate",
	}
	payloadJSON, _ := json.Marshal(payload)

	_, err = deps.DB.Pool.Exec(ctx, `
    INSERT INTO fatigue_genome (user_id, model_version, data_points, confidence, genome_data, last_calibrated)
    VALUES ($1, 1, $2, $3, $4::jsonb, $5)
    ON CONFLICT (user_id)
    DO UPDATE SET
      data_points = EXCLUDED.data_points,
      confidence = EXCLUDED.confidence,
      genome_data = EXCLUDED.genome_data,
      last_calibrated = EXCLUDED.last_calibrated
  `, userID, points, confidence, payloadJSON, time.Now().UTC())

	return err
}

func computeGenomeSignals(ctx context.Context, deps *Dependencies, userID uuid.UUID) (float64, float64, float64, error) {
	rows, err := deps.DB.Pool.Query(ctx, `
    SELECT COALESCE(ctl, 0), COALESCE(atl, 0), COALESCE(tsb, 0)
    FROM fitness_snapshots
    WHERE user_id = $1
    ORDER BY snapshot_date ASC
  `, userID)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	ctlVals := make([]float64, 0)
	atlVals := make([]float64, 0)
	tsbVals := make([]float64, 0)

	for rows.Next() {
		var ctl, atl, tsb float64
		if err := rows.Scan(&ctl, &atl, &tsb); err != nil {
			return 0, 0, 0, err
		}
		ctlVals = append(ctlVals, ctl)
		atlVals = append(atlVals, atl)
		tsbVals = append(tsbVals, tsb)
	}

	if len(ctlVals) < 2 {
		return 0.07, 0.2, 1.0, nil
	}

	ctlAccumulationRate := averagePositiveDelta(ctlVals)
	atlRecoveryRate := averageNegativeDelta(atlVals)
	taperResponse := average(tsbVals)
	if taperResponse == 0 {
		taperResponse = 1.0
	} else {
		taperResponse = math.Max(0.5, math.Min(1.5, 1.0+(taperResponse/40.0)))
	}

	return ctlAccumulationRate, atlRecoveryRate, taperResponse, nil
}

func averagePositiveDelta(values []float64) float64 {
	sum := 0.0
	count := 0.0
	for i := 1; i < len(values); i++ {
		d := values[i] - values[i-1]
		if d > 0 {
			sum += d
			count++
		}
	}
	if count == 0 {
		return 0.07
	}
	return math.Max(0.01, math.Min(0.25, sum/count/10.0))
}

func averageNegativeDelta(values []float64) float64 {
	sum := 0.0
	count := 0.0
	for i := 1; i < len(values); i++ {
		d := values[i] - values[i-1]
		if d < 0 {
			sum += math.Abs(d)
			count++
		}
	}
	if count == 0 {
		return 0.2
	}
	return math.Max(0.05, math.Min(0.5, sum/count/8.0))
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
