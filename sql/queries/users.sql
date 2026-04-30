-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password) 
VALUES (
  gen_random_uuid(),
  now(),
  now(),
  $1,
  $2
)
RETURNING *;

-- name: GetPasswordHash :one
SELECT hashed_password FROM users WHERE email=$1;

-- name: GetUserInfo :one
SELECT id, created_at, updated_at, email, is_chirpy_red FROM users WHERE email=$1;

-- name: UpdateCreds :exec
UPDATE users SET email = $1, hashed_password = $2, updated_at = now() WHERE id=$3;

-- name: UpgradeUserToRed :exec
UPDATE users SET is_chirpy_red = TRUE, updated_at = now() WHERE id=$1;

-- name: DeleteUser :exec
DELETE FROM users;
