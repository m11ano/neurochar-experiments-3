-- +goose Up

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Таблица задач
CREATE TABLE "task" (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name               TEXT NOT NULL,
    method                  TEXT NOT NULL,
    is_processed            BOOLEAN NOT NULL DEFAULT FALSE,
    result                  TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_task_method_is_processed ON "task" (method, is_processed);

-- +goose Down
DROP INDEX IF EXISTS idx_task_method_is_processed;
DROP TABLE IF EXISTS "task";