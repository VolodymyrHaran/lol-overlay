CREATE TABLE rooms
(
    id TEXT PRIMARY KEY,

    last_updated TIMESTAMP NOT NULL
);

CREATE TABLE players
(
    id BIGSERIAL PRIMARY KEY,

    room_id TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,

    game_name TEXT NOT NULL,

    tag_line TEXT NOT NULL,

    champion TEXT NOT NULL,

    champion_id INTEGER NOT NULL,

    summoner_spell_haste INTEGER NOT NULL
);

CREATE TABLE summoner_spells
(
    id BIGSERIAL PRIMARY KEY,

    player_id BIGINT NOT NULL REFERENCES players(id) ON DELETE CASCADE,

    name TEXT NOT NULL,

    is_ready BOOLEAN NOT NULL,

    base_cooldown INTEGER NOT NULL,

    remaining_cooldown INTEGER NOT NULL,

    cooldown_end_time TIMESTAMP
);

CREATE INDEX idx_players_room
ON players(room_id);

CREATE INDEX idx_spells_player
ON summoner_spells(player_id);