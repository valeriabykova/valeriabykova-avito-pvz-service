-- +goose Up
CREATE TABLE product (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    reception_id UUID NOT NULL,
    type TEXT NOT NULL, 
    datetime TIMESTAMP
);

-- +goose Down
DROP TABLE product;