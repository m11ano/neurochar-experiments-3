-- +goose Up

ALTER TABLE "task"
ADD COLUMN "processor" TEXT NOT NULL DEFAULT '',
ADD COLUMN "process_duration_ms" BIGINT NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE "task"
DROP COLUMN IF EXISTS "process_duration_ms",
DROP COLUMN IF EXISTS "processor";