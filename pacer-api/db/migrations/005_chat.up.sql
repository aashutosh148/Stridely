-- 005_chat.up.sql

CREATE TABLE chat_messages (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_id   UUID NOT NULL,  -- groups messages into a conversation
  role         TEXT NOT NULL CHECK (role IN ('user','assistant')),
  content      TEXT NOT NULL,
  tool_calls   JSONB,          -- raw tool call blocks for debugging
  created_at   TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_chat_user_session ON chat_messages(user_id, session_id, created_at);
