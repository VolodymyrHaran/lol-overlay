package models

import "time"

type SummonerSpell struct {
	Name              string    `json:"name"`
	IsReady           bool      `json:"isReady"`
	BaseCooldown      int       `json:"baseCooldown"`
	CooldownEndTime   time.Time `json:"cooldownEndTime"`
	RemainingCooldown int       `json:"remainingCooldown"`
}
