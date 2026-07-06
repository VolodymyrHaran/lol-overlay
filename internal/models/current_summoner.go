package models

type CurrentSummoner struct {
	GameName      string `json:"gameName"`
	TagLine       string `json:"tagLine"`
	Puuid         string `json:"puuid"`
	SummonerLevel int    `json:"summonerLevel"`
}
