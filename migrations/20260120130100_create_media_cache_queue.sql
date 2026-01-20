-- +goose Up
-- +goose StatementBegin
-- Create queue table for media caching with rate limiting
CREATE TABLE media_cache_queue (
    id SERIAL PRIMARY KEY,
    media_id INTEGER NOT NULL,
    media_type TEXT NOT NULL, -- 'ANIME' or 'MANGA'
    status TEXT NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed
    retry_after TIMESTAMP, -- When to retry after rate limit
    attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for fetching pending items
CREATE INDEX idx_media_cache_queue_status ON media_cache_queue(status);
CREATE INDEX idx_media_cache_queue_retry_after ON media_cache_queue(retry_after);
CREATE UNIQUE INDEX idx_media_cache_queue_media_id ON media_cache_queue(media_id) WHERE status IN ('pending', 'processing');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE media_cache_queue;
-- +goose StatementEnd
