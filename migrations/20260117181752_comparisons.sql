-- +goose Up
-- +goose StatementBegin
CREATE table comparisons (
    id SERIAL PRIMARY KEY,
    creator_username TEXT NOT NULL,
    comparator_username TEXT NOT NULL,
    comparison_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP table comparisons;
-- +goose StatementEnd
