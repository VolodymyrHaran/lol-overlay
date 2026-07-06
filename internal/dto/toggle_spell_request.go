package dto

type ToggleSpellRequest struct {
	GameName string `json:"gameName"`
	TagLine  string `json:"tagLine"`
	Spell    string `json:"spell"`
}
