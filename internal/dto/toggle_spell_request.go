package dto

type ToggleSpellRequest struct {
	GameName string `json:"gameName" example:"PlayerOne"`
	TagLine  string `json:"tagLine" example:"EUW"`
	Spell    string `json:"spell" example:"Flash"`
}
