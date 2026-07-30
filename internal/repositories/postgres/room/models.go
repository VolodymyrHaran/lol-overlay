package room

import "time"

type roomRow struct {
	ID          string
	LastUpdated time.Time
}

type playerRow struct {
	ID                 int64
	RoomID             string
	GameName           string
	TagLine            string
	Champion           string
	ChampionID         int
	SummonerSpellHaste int
}

type spellRow struct {
	PlayerID          int64
	Name              string
	IsReady           bool
	BaseCooldown      int
	RemainingCooldown int
	CooldownEndTime   *time.Time
}
