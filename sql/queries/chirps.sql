-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
  gen_random_uuid(),
  now(),
  now(),
  $1,
  $2
)
RETURNING *;

-- name: GetAllUserChirps :many
SELECT * FROM chirps ORDER BY created_at ASC;

-- name: GetUserChirps :many
SELECT * FROM chirps WHERE user_id=$1 ORDER BY created_at ASC;
-- name: GetChirpByID :one
SELECT * FROM chirps WHERE id=$1;

-- name: DeleteChirpByID :exec
DELETE FROM chirps WHERE id=$1;
