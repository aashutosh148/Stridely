-- 004_memory.up.sql

CREATE TYPE memory_type AS ENUM
  ('activity','race','injury','milestone','note','coaching_moment','weekly_summary');

CREATE TABLE episodic_memories (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  memory_type      memory_type NOT NULL,
  event_date       DATE NOT NULL,
  title            TEXT NOT NULL,
  summary          TEXT NOT NULL,
  content          JSONB NOT NULL,
  importance_score FLOAT NOT NULL DEFAULT 0.5 CHECK (importance_score BETWEEN 0 AND 1),
  tags             TEXT[] NOT NULL DEFAULT '{}',
  compressed       BOOL NOT NULL DEFAULT false,
  created_at       TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_episodic_user_date   ON episodic_memories(user_id, event_date DESC);
CREATE INDEX idx_episodic_user_type   ON episodic_memories(user_id, memory_type);
CREATE INDEX idx_episodic_importance  ON episodic_memories(user_id, importance_score DESC);
CREATE INDEX idx_episodic_tags        ON episodic_memories USING gin(tags);

CREATE TABLE semantic_facts (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  fact_key        TEXT NOT NULL,
  fact_value      JSONB NOT NULL,
  confidence      FLOAT NOT NULL DEFAULT 0.3 CHECK (confidence BETWEEN 0 AND 1),
  evidence_count  INTEGER NOT NULL DEFAULT 1,
  last_updated    TIMESTAMP NOT NULL DEFAULT now(),
  first_observed  TIMESTAMP NOT NULL DEFAULT now(),
  notes           TEXT,
  UNIQUE(user_id, fact_key)
);

CREATE TABLE fatigue_genome (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
  model_version     INTEGER NOT NULL DEFAULT 1,
  data_points       INTEGER NOT NULL DEFAULT 0,
  confidence        TEXT NOT NULL DEFAULT 'insufficient', -- 'insufficient'|'low'|'medium'|'high'
  genome_data       JSONB NOT NULL DEFAULT '{}'::jsonb,
  last_calibrated   TIMESTAMP NOT NULL DEFAULT now()
);
