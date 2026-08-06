package services

import (
	"lol-timer/internal/models"
	"time"
)

func GetSpellName(spellId int) string {
	switch spellId {
	case 1:
		return "Cleanse"
	case 3:
		return "Exhaust"
	case 4:
		return "Flash"
	case 6:
		return "Ghost"
	case 7:
		return "Heal"
	case 11:
		return "Smite"
	case 12:
		return "Teleport"
	case 14:
		return "Ignite"
	case 21:
		return "Barrier"
	default:
		return "Unknown"
	}
}
func GetSpellCooldown(spellId int) int {
	switch spellId {
	case 4:
		return 300 // Flash
	case 14:
		return 180 // Ignite
	case 7:
		return 240 // Heal
	case 3:
		return 240 // Exhaust
	case 6:
		return 240 // Ghost
	case 12:
		return 300 // Teleport
	case 11:
		return 15 // Smite
	case 21:
		return 180 // Barrier
	case 1:
		return 240 // Cleanse
	default:
		return 300
	}
}

func CalculateCooldown(baseCooldown int, summonerSpellHaste int) int {
	return baseCooldown * 100 / (100 + summonerSpellHaste)
}

func TogglePlayerSpell(
	player *models.Player,
	spellName string,
) bool {
	spell := player.FindSpell(spellName)
	if spell == nil {
		return false
	}

	if !spell.IsReady {
		spell.IsReady = true
		spell.CooldownEndTime = time.Time{}
		spell.RemainingCooldown = 0
		return true
	}

	cooldown := CalculateCooldown(
		spell.BaseCooldown,
		player.SummonerSpellHaste,
	)

	spell.IsReady = false
	spell.RemainingCooldown = cooldown
	spell.CooldownEndTime = time.Now().UTC().Add(
		time.Duration(cooldown) * time.Second,
	)

	return true
}

func CopySpellState(
	oldPlayer *models.Player,
	newPlayer *models.Player,
) {
	for si := range newPlayer.Spells {
		newSpell := &newPlayer.Spells[si]
		oldSpell := oldPlayer.FindSpell(newSpell.Name)

		if oldSpell != nil {
			newSpell.IsReady = oldSpell.IsReady
			newSpell.CooldownEndTime = oldSpell.CooldownEndTime
			newSpell.RemainingCooldown = oldSpell.RemainingCooldown
		}
	}
}
