-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,         -- email from client request
    $2          -- passworc from client request
)
RETURNING *;    -- return all values for http response
