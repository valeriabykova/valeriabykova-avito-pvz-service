-- +goose Up
CREATE TABLE reception (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    pvz_id UUID NOT NULL,
    status text NOT NULL,
    datetime TIMESTAMP
);

-- +goose Down
DROP TABLE reception;