-- +goose Up
CREATE TABLE users (
    id UUID primary key,
    created_at TIMESTAMP not null,
    updated_at TIMESTAMP not null,
    email text not null unique
);

-- +goose Down
DROP TABLE users;
