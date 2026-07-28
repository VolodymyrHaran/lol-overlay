package models

import "time"

type Room struct {
	Id          string    `json:"id"`
	Players     []Player  `json:"players"`
	LastUpdated time.Time `json:"lastUpdated"`
}

func (r *Room) Clone() *Room {
	if r == nil {
		return nil
	}

	roomCopy := *r

	roomCopy.Players = make([]Player, len(r.Players))

	for i := range r.Players {
		roomCopy.Players[i] = r.Players[i]

		roomCopy.Players[i].Spells = make(
			[]SummonerSpell,
			len(r.Players[i].Spells),
		)

		copy(
			roomCopy.Players[i].Spells,
			r.Players[i].Spells,
		)
	}

	return &roomCopy
}
