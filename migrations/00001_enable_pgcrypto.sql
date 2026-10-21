-- +goose Up
-- pgcrypto provides gen_random_uuid(), used for every table's primary key
-- going forward (users, folders, files, ...).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose Down
DROP EXTENSION IF EXISTS pgcrypto;
