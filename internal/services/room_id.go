package services

import (
	"fmt"
	"lol-timer/internal/models"
)

func BuildRoomId(session *models.ChampSelectSession) string {
	if session == nil {
		return ""
	}

	teamId := 0

	for _, player := range session.MyTeam {
		if player.CellId == session.LocalPlayerCellId {
			teamId = player.Team
			break
		}
	}

	return fmt.Sprintf("%d-%d", session.GameId, teamId)
}
