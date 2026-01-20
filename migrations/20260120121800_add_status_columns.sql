-- +goose Up
-- +goose StatementBegin
-- shared_entries have both creator and comparator status
ALTER TABLE shared_entries
ADD COLUMN creator_status TEXT,
ADD COLUMN comparator_status TEXT;

-- Copy existing creator_status 
UPDATE shared_entries SET creator_status = status WHERE creator_status IS NULL;
UPDATE shared_entries SET comparator_status = status WHERE comparator_status IS NULL;

ALTER TABLE shared_entries
ALTER COLUMN creator_status SET NOT NULL,
ALTER COLUMN comparator_status SET NOT NULL;

ALTER TABLE shared_entries DROP COLUMN status;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE shared_entries ADD COLUMN status TEXT;

UPDATE shared_entries SET status = creator_status;

ALTER TABLE shared_entries ALTER COLUMN status SET NOT NULL;

ALTER TABLE shared_entries
DROP COLUMN creator_status,
DROP COLUMN comparator_status;
-- +goose StatementEnd
