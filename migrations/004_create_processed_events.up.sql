CREATE TABLE processed_events
(
    consumer_name TEXT NOT NULL,
    event_id UUID NOT NULL,
    subject TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (consumer_name, event_id)
);