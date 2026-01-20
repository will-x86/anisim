-- name: GetAllComparisons :many
SELECT * FROM comparisons
ORDER BY comparison_date DESC;

-- name: GetComparison :one
SELECT * FROM comparisons
WHERE id = $1;

-- name: CreateComparison :one
INSERT INTO comparisons (
    creator_username,
    comparator_username,
    creator_id,
    creator_name,
    creator_avatar_large,
    creator_avatar_medium,
    creator_episodes_watched,
    creator_minutes_watched,
    creator_chapters_read,
    creator_mean_score,
    comparator_id,
    comparator_name,
    comparator_avatar_large,
    comparator_avatar_medium,
    comparator_episodes_watched,
    comparator_minutes_watched,
    comparator_chapters_read,
    comparator_mean_score
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
RETURNING *;

-- name: CreateSharedEntry :one
INSERT INTO shared_entries (
    comparison_id,
    media_type,
    media_id,
    media_title_romaji,
    media_title_english,
    media_cover_large,
    media_cover_medium,
    creator_status,
    comparator_status,
    creator_score,
    comparator_score
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetSharedEntriesByComparison :many
SELECT * FROM shared_entries
WHERE comparison_id = $1
ORDER BY creator_status, media_title_romaji;
