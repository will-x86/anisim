-- name: GetAllComparisons :many
SELECT * FROM comparisons
ORDER BY comparison_date DESC;

-- name: GetAllComparisonsOnePerCombo :many
SELECT * FROM (
    SELECT DISTINCT ON (creator_username, comparator_username) *
    FROM comparisons
    ORDER BY creator_username, comparator_username, comparison_date DESC
) sub
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

-- name: BatchCreateSharedEntries :copyfrom
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetSharedEntriesByComparison :many
SELECT * FROM shared_entries
WHERE comparison_id = $1
ORDER BY creator_status, media_title_romaji;

-- name: UpsertMediaCache :one
INSERT INTO media_cache (
    id, type, title_romaji, title_english, cover_image_large, cover_image_medium,
    genres, average_score, mean_score, popularity, favourites, format, status,
    season_year, season, is_adult
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type,
    title_romaji = EXCLUDED.title_romaji,
    title_english = EXCLUDED.title_english,
    cover_image_large = EXCLUDED.cover_image_large,
    cover_image_medium = EXCLUDED.cover_image_medium,
    genres = EXCLUDED.genres,
    average_score = EXCLUDED.average_score,
    mean_score = EXCLUDED.mean_score,
    popularity = EXCLUDED.popularity,
    favourites = EXCLUDED.favourites,
    format = EXCLUDED.format,
    status = EXCLUDED.status,
    season_year = EXCLUDED.season_year,
    season = EXCLUDED.season,
    is_adult = EXCLUDED.is_adult,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetOrCreateTag :one
INSERT INTO tags (name)
VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: UpsertMediaTag :exec
INSERT INTO media_tags (media_id, tag_id, rank, is_media_spoiler, is_general_spoiler)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (media_id, tag_id) DO UPDATE SET
    rank = EXCLUDED.rank,
    is_media_spoiler = EXCLUDED.is_media_spoiler,
    is_general_spoiler = EXCLUDED.is_general_spoiler;

-- name: GetMediaCache :one
SELECT * FROM media_cache
WHERE id = $1;

-- name: GetMediaTags :many
SELECT t.id, t.name, mt.rank, mt.is_media_spoiler, mt.is_general_spoiler
FROM media_tags mt
JOIN tags t ON mt.tag_id = t.id
WHERE mt.media_id = $1
ORDER BY mt.rank DESC;

-- name: EnqueueMediaForCaching :exec
INSERT INTO media_cache_queue (media_id, media_type)
VALUES ($1, $2)
ON CONFLICT (media_id) WHERE status IN ('pending', 'processing') DO NOTHING;

-- name: BatchEnqueueMediaForCaching :copyfrom
INSERT INTO media_cache_queue (media_id, media_type)
VALUES ($1, $2);

-- name: GetPendingQueueItems :many
SELECT * FROM media_cache_queue
WHERE status = 'pending'
  AND (retry_after IS NULL OR retry_after <= NOW())
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkQueueItemProcessing :exec
UPDATE media_cache_queue
SET status = 'processing', updated_at = NOW()
WHERE id = $1;

-- name: MarkQueueItemCompleted :exec
UPDATE media_cache_queue
SET status = 'completed', updated_at = NOW()
WHERE id = $1;

-- name: MarkQueueItemFailed :exec
UPDATE media_cache_queue
SET status = 'failed',
    attempts = attempts + 1,
    retry_after = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: NeedsCacheUpdate :one
SELECT
    CASE
        WHEN updated_at < NOW() - INTERVAL '24 hours' THEN true
        ELSE false
    END as needs_update
FROM media_cache
WHERE id = $1;
