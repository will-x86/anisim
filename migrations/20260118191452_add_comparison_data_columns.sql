-- +goose Up
-- +goose StatementBegin
-- Add user data columns to comparisons table
ALTER TABLE comparisons
ADD COLUMN creator_id INTEGER,
ADD COLUMN creator_name TEXT,
ADD COLUMN creator_avatar_large TEXT,
ADD COLUMN creator_avatar_medium TEXT,
ADD COLUMN creator_episodes_watched INTEGER DEFAULT 0,
ADD COLUMN creator_minutes_watched INTEGER DEFAULT 0,
ADD COLUMN creator_chapters_read INTEGER DEFAULT 0,
ADD COLUMN creator_mean_score FLOAT DEFAULT 0,
ADD COLUMN comparator_id INTEGER,
ADD COLUMN comparator_name TEXT,
ADD COLUMN comparator_avatar_large TEXT,
ADD COLUMN comparator_avatar_medium TEXT,
ADD COLUMN comparator_episodes_watched INTEGER DEFAULT 0,
ADD COLUMN comparator_minutes_watched INTEGER DEFAULT 0,
ADD COLUMN comparator_chapters_read INTEGER DEFAULT 0,
ADD COLUMN comparator_mean_score FLOAT DEFAULT 0;

-- Create shared_entries table for comparison results
CREATE TABLE shared_entries (
    id SERIAL PRIMARY KEY,
    comparison_id INTEGER NOT NULL REFERENCES comparisons(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL, -- 'anime' or 'manga'
    media_id INTEGER NOT NULL,
    media_title_romaji TEXT NOT NULL,
    media_title_english TEXT,
    media_cover_large TEXT,
    media_cover_medium TEXT,
    status TEXT NOT NULL, -- 'CURRENT', 'PLANNING', 'COMPLETED', 'DROPPED', 'PAUSED', 'REPEATING'
    creator_score FLOAT DEFAULT 0,
    comparator_score FLOAT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_shared_entries_comparison_id ON shared_entries(comparison_id);
CREATE INDEX idx_shared_entries_media_type ON shared_entries(media_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE shared_entries;

ALTER TABLE comparisons
DROP COLUMN creator_id,
DROP COLUMN creator_name,
DROP COLUMN creator_avatar_large,
DROP COLUMN creator_avatar_medium,
DROP COLUMN creator_episodes_watched,
DROP COLUMN creator_minutes_watched,
DROP COLUMN creator_chapters_read,
DROP COLUMN creator_mean_score,
DROP COLUMN comparator_id,
DROP COLUMN comparator_name,
DROP COLUMN comparator_avatar_large,
DROP COLUMN comparator_avatar_medium,
DROP COLUMN comparator_episodes_watched,
DROP COLUMN comparator_minutes_watched,
DROP COLUMN comparator_chapters_read,
DROP COLUMN comparator_mean_score;
-- +goose StatementEnd
