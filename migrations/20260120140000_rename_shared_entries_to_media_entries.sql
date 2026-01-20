-- +goose Up
-- +goose StatementBegin
-- Rename shared_entries to media_entries and add flags for which user has the media
ALTER TABLE shared_entries RENAME TO media_entries;

-- Add columns to indicate which user has this media
ALTER TABLE media_entries
ADD COLUMN in_creator_list BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN in_comparator_list BOOLEAN NOT NULL DEFAULT true;

-- For existing data, all entries are shared so both flags are true (already set by DEFAULT)

-- Add index for efficient recommendation queries
CREATE INDEX idx_media_entries_in_creator_list ON media_entries(in_creator_list);
CREATE INDEX idx_media_entries_in_comparator_list ON media_entries(in_comparator_list);

-- Drop the old index name and recreate with new name
DROP INDEX IF EXISTS idx_shared_entries_comparison_id;
DROP INDEX IF EXISTS idx_shared_entries_media_type;

CREATE INDEX idx_media_entries_comparison_id ON media_entries(comparison_id);
CREATE INDEX idx_media_entries_media_type ON media_entries(media_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove new columns
ALTER TABLE media_entries
DROP COLUMN in_creator_list,
DROP COLUMN in_comparator_list;

-- Rename back to shared_entries
ALTER TABLE media_entries RENAME TO shared_entries;

-- Recreate old indexes
DROP INDEX IF EXISTS idx_media_entries_comparison_id;
DROP INDEX IF EXISTS idx_media_entries_media_type;
DROP INDEX IF EXISTS idx_media_entries_in_creator_list;
DROP INDEX IF EXISTS idx_media_entries_in_comparator_list;

CREATE INDEX idx_shared_entries_comparison_id ON shared_entries(comparison_id);
CREATE INDEX idx_shared_entries_media_type ON shared_entries(media_type);
-- +goose StatementEnd
