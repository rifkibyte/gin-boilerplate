CREATE TABLE IF NOT EXISTS files (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename   VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size       BIGINT NOT NULL,
    path       VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);