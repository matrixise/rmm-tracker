-- +goose Up
CREATE TABLE IF NOT EXISTS tracker_metadata (
    id          INT PRIMARY KEY DEFAULT 1,
    last_run_at TIMESTAMPTZ,
    succeeded   BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT single_row CHECK (id = 1)
);

INSERT INTO tracker_metadata (id, last_run_at, succeeded)
VALUES (1, NULL, false)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS tracker_metadata;
