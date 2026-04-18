-- 006_pgvector.up.sql

CREATE EXTENSION IF NOT EXISTS vector;

-- Add embedding column to episodic_memories
ALTER TABLE episodic_memories ADD COLUMN embedding vector(1536);

-- HNSW index for fast cosine similarity search
CREATE INDEX idx_episodic_embedding ON episodic_memories
  USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);
