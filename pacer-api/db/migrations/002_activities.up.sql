-- 002_activities.up.sql

CREATE TYPE workout_type AS ENUM ('easy','long','tempo','interval','race','recovery','unstructured');

CREATE TABLE activities (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  strava_id              TEXT NOT NULL,
  activity_date          DATE NOT NULL,
  workout_type           workout_type NOT NULL DEFAULT 'unstructured',
  distance_m             INTEGER NOT NULL,
  duration_s             INTEGER NOT NULL,
  elevation_gain_m       FLOAT NOT NULL DEFAULT 0,
  avg_pace_s             FLOAT,            -- seconds per km
  avg_hr                 INTEGER,
  max_hr                 INTEGER,
  tss                    FLOAT,            -- Training Stress Score
  intensity_factor       FLOAT,
  zone_distribution      JSONB,            -- {z1_pct, z2_pct, z3_pct, z4_pct, z5_pct}
  cardiac_decoupling_pct FLOAT,
  garmin_cadence_spm     INTEGER,
  garmin_gct_ms          INTEGER,          -- ground contact time
  garmin_vert_osc_cm     FLOAT,
  garmin_lr_balance_pct  FLOAT,
  garmin_training_load   FLOAT,
  rpe_reported           INTEGER CHECK (rpe_reported BETWEEN 1 AND 10),
  matched_workout_id     UUID,             -- FK to workouts added in migration 003
  adherence_score        FLOAT,
  splits_km              JSONB,            -- [{km, pace_s, hr, elev_delta}]
  streams_s3_key         TEXT,             -- S3 key for raw streams (archived > 6mo)
  gear_id                TEXT,             -- Strava gear ID
  created_at             TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(user_id, strava_id)
);

CREATE INDEX idx_activities_user_date ON activities(user_id, activity_date DESC);
CREATE INDEX idx_activities_user_type ON activities(user_id, workout_type);

CREATE TABLE daily_health (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  health_date         DATE NOT NULL,
  hrv_last_night_avg  FLOAT,
  hrv_baseline_low    FLOAT,
  hrv_baseline_high   FLOAT,
  hrv_status          TEXT,             -- 'BALANCED'|'UNBALANCED'|'POOR'
  sleep_total_s       INTEGER,
  sleep_deep_s        INTEGER,
  sleep_rem_s         INTEGER,
  sleep_score         INTEGER,
  body_battery_high   INTEGER,
  body_battery_low    INTEGER,
  resting_hr          INTEGER,
  stress_avg          INTEGER,
  readiness_score     INTEGER,          -- 1-10 computed by Pacer
  readiness_level     TEXT,             -- 'green'|'amber'|'red'
  readiness_note      TEXT,
  created_at          TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(user_id, health_date)
);

CREATE TABLE sync_logs (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider     oauth_provider NOT NULL,
  synced_at    TIMESTAMP NOT NULL DEFAULT now(),
  status       TEXT NOT NULL,   -- 'success'|'partial'|'failed'
  records_sync INTEGER,
  error_msg    TEXT
);
