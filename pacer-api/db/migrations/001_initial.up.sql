-- 001_initial.up.sql

CREATE TYPE runner_tier AS ENUM ('beginner', 'recreational', 'competitive', 'serious');
CREATE TYPE subscription_tier AS ENUM ('free', 'core', 'pro');

CREATE TABLE users (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email                  TEXT UNIQUE NOT NULL,
  runner_tier            runner_tier NOT NULL DEFAULT 'recreational',
  subscription_tier      subscription_tier NOT NULL DEFAULT 'free',
  goal_time_s            INTEGER,           -- target marathon finish seconds
  target_race_date       DATE,
  threshold_pace_s       FLOAT,             -- lactate threshold pace s/km
  threshold_hr           INTEGER,           -- lactate threshold HR bpm
  max_hr                 INTEGER,
  weight_kg              FLOAT,
  onboarded_at           TIMESTAMP,
  strava_athlete_id      TEXT UNIQUE,
  garmin_user_id         TEXT UNIQUE,
  preferred_language     TEXT NOT NULL DEFAULT 'en',
  notification_prefs     JSONB NOT NULL DEFAULT '{"push": true, "quiet_hours": ["22:00", "06:00"]}'::jsonb,
  created_at             TIMESTAMP NOT NULL DEFAULT now(),
  updated_at             TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TYPE oauth_provider AS ENUM ('strava', 'garmin');

CREATE TABLE oauth_tokens (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider          oauth_provider NOT NULL,
  access_token_enc  BYTEA NOT NULL,   -- AES-256 encrypted
  refresh_token_enc BYTEA NOT NULL,   -- AES-256 encrypted
  expires_at        TIMESTAMP NOT NULL,
  scope             TEXT,
  created_at        TIMESTAMP NOT NULL DEFAULT now(),
  updated_at        TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(user_id, provider)
);
