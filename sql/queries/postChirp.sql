-- name: PostChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,         -- first param: body from client request
    $2          -- second param: user id from request
)
RETURNING *;    -- return all values for http response
