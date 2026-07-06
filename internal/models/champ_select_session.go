package models

type ChampSelectSession struct {
	GameId            int64               `json:"gameId"`
	LocalPlayerCellId int                 `json:"localPlayerCellId"`
	MyTeam            []ChampSelectPlayer `json:"myTeam"`
	TheirTeam         []ChampSelectPlayer `json:"theirTeam"`
}

type ChampSelectPlayer struct {
	CellId     int    `json:"cellId"`
	ChampionId int    `json:"championId"`
	Spell1Id   int    `json:"spell1Id"`
	Spell2Id   int    `json:"spell2Id"`
	GameName   string `json:"gameName"`
	TagLine    string `json:"tagLine"`
	Puuid      string `json:"puuid"`
	Team       int    `json:"team"`
}
