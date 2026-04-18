-- 003_plans.up.sql

CREATE TYPE plan_phase AS ENUM ('base','build','peak','taper');

CREATE TABLE training_blocks (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  phase         plan_phase NOT NULL DEFAULT 'base',
  block_start   DATE NOT NULL,
  block_end     DATE NOT NULL,
  target_race   DATE,
  goal_time_s   INTEGER,
  peak_ctl      FLOAT,
  is_active     BOOL NOT NULL DEFAULT true,
  created_at    TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE workouts (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  block_id        UUID NOT NULL REFERENCES training_blocks(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  scheduled_date  DATE NOT NULL,
  workout_type    workout_type NOT NULL,
  distance_km     FLOAT,
  duration_min    INTEGER,
  pace_target_min FLOAT,  -- min target pace s/km
  pace_target_max FLOAT,  -- max target pace s/km
  hr_zone         INTEGER CHECK (hr_zone BETWEEN 1 AND 5),
  rpe_target      INTEGER CHECK (rpe_target BETWEEN 1 AND 10),
  description     TEXT,
  purpose         TEXT,
  status          TEXT NOT NULL DEFAULT 'planned', -- 'planned'|'completed'|'skipped'|'modified'
  completed_activity_id UUID REFERENCES activities(id),
  created_at      TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_workouts_user_date ON workouts(user_id, scheduled_date);
CREATE INDEX idx_workouts_block ON workouts(block_id);

-- Back-fill the FK we left in migration 002
ALTER TABLE activities
  ADD CONSTRAINT fk_matched_workout
  FOREIGN KEY (matched_workout_id) REFERENCES workouts(id);
