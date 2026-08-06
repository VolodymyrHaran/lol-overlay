ALTER TABLE summoner_spells
ALTER COLUMN cooldown_end_time
TYPE TIMESTAMP
USING cooldown_end_time AT TIME ZONE 'UTC';