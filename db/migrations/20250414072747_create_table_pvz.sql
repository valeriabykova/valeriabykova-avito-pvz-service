-- +goose Up
CREATE TABLE pvz (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    registration_date TIMESTAMP,
    city text
);

-- +goose Down
DROP TABLE pvz;