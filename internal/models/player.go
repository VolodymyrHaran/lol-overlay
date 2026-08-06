package models

type Player struct {
	Champion           string          `json:"champion"`
	ChampionImage      string          `json:"championImage"`
	GameName           string          `json:"gameName"`
	TagLine            string          `json:"tagLine"`
	ChampionId         int             `json:"championId"`
	SummonerSpellHaste int             `json:"summonerSpellHaste"`
	Spells             []SummonerSpell `json:"spells"`
}

func (p *Player) FindSpell(spellName string) *SummonerSpell {
	for i := range p.Spells {
		if p.Spells[i].Name == spellName {
			return &p.Spells[i]
		}
	}

	return nil
}
