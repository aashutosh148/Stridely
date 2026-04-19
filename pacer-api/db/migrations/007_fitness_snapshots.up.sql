-- 007_fitness_snapshots.up.sql

CREATE TABLE fitness_snapshots (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  snapshot_date DATE NOT NULL,
  ctl           FLOAT NOT NULL DEFAULT 0,
  atl           FLOAT NOT NULL DEFAULT 0,
  tsb           FLOAT NOT NULL DEFAULT 0,
  created_at    TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(user_id, snapshot_date)
);

CREATE INDEX idx_fitness_snapshots_user_date ON fitness_snapshots(user_id, snapshot_date DESC);
