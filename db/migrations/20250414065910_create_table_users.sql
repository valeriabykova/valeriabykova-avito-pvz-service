-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE users (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    email text UNIQUE,
    role text,
    password_hash text
);

-- +goose Down
DROP TABLE users;