CREATE TABLE outbox_events
(
    id UUID PRIMARY KEY,

    subject TEXT NOT NULL,
    payload JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    published_at TIMESTAMPTZ,

    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,

    CONSTRAINT outbox_events_attempt_count_non_negative
        CHECK (attempt_count >= 0)
);

CREATE INDEX idx_outbox_events_pending
ON outbox_events(available_at, created_at)
WHERE published_at IS NULL;

CREATE INDEX idx_outbox_events_published_at
ON outbox_events(published_at)
WHERE published_at IS NOT NULL;