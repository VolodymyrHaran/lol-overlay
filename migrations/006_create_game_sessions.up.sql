CREATE TABLE game_sessions
(
    game_id BIGINT PRIMARY KEY,
    room_id TEXT NOT NULL,

    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT game_sessions_ended_after_started
        CHECK (
            ended_at IS NULL
            OR ended_at >= started_at
        )
);

CREATE INDEX idx_game_sessions_room_id
ON game_sessions(room_id);

CREATE INDEX idx_game_sessions_started_at
ON game_sessions(started_at);