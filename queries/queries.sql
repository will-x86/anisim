-- name: GetAllComparisons :many
SELECT * FROM comparisons
ORDER BY comparison_date DESC;

-- name: GetComparison :one
SELECT * FROM comparisons
WHERE id = $1;

-- name: CreateComparison :one
INSERT INTO comparisons (creator_username, comparator_username)
VALUES ($1, $2)
RETURNING *;
