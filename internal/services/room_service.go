package services

import (
	"lol-timer/internal/constants"
	"lol-timer/internal/models"
	"lol-timer/internal/repositories"
	"sync"
	"time"
)

type RoomService struct {
	mu         sync.Mutex
	repository repositories.RoomRepository
}

func NewRoomService(
	repository repositories.RoomRepository,
) *RoomService {
	return &RoomService{
		repository: repository,
	}
}

func (s *RoomService) GetRoomSnapshot(
	roomId string,
) *models.Room {
	room, exists := s.repository.Get(roomId)
	if !exists {
		return nil
	}

	return room
}

func (s *RoomService) GetRoomSnapshots() []*models.Room {
	return s.repository.GetAll()
}

func (s *RoomService) CreateRoom(
	roomId string,
) *models.Room {
	if roomId == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if room, exists := s.repository.Get(roomId); exists {
		return room
	}

	room := &models.Room{
		Id:          roomId,
		LastUpdated: time.Now(),
	}

	s.repository.Save(room)

	return room.Clone()
}

func (s *RoomService) AddPlayer(
	roomId string,
	player models.Player,
) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists := s.repository.Get(roomId)
	if !exists {
		return false
	}

	existingPlayer := FindPlayerByRiotId(
		room,
		player.GameName,
		player.TagLine,
	)

	if existingPlayer != nil {
		existingPlayer.Champion = player.Champion
		existingPlayer.ChampionId = player.ChampionId
		room.LastUpdated = time.Now()

		s.repository.Save(room)
		return true
	}

	player.SummonerSpellHaste =
		constants.DefaultSummonerSpellHaste

	if len(player.Spells) == 0 {
		player.Spells = []models.SummonerSpell{
			{
				Name:         "Flash",
				IsReady:      true,
				BaseCooldown: 300,
			},
			{
				Name:         "Ignite",
				IsReady:      true,
				BaseCooldown: 180,
			},
		}
	}

	room.Players = append(room.Players, player)
	s.saveRoom(room)

	return true
}

func FindPlayerByRiotId(
	room *models.Room,
	gameName string,
	tagLine string,
) *models.Player {
	if room == nil {
		return nil
	}

	for i := range room.Players {
		player := &room.Players[i]

		if player.GameName == gameName &&
			player.TagLine == tagLine {
			return player
		}
	}

	return nil
}

func (s *RoomService) RefreshCooldowns() {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms := s.repository.GetAll()

	for _, room := range rooms {
		refreshRoomCooldowns(room)
		s.repository.Save(room)
	}
}

func refreshRoomCooldowns(room *models.Room) {
	now := time.Now()

	for playerIndex := range room.Players {
		player := &room.Players[playerIndex]

		for spellIndex := range player.Spells {
			spell := &player.Spells[spellIndex]

			if spell.IsReady ||
				spell.CooldownEndTime.IsZero() {
				spell.RemainingCooldown = 0
				continue
			}

			remaining := int(
				spell.CooldownEndTime.Sub(now).Seconds(),
			)

			if remaining <= 0 {
				spell.IsReady = true
				spell.RemainingCooldown = 0
				spell.CooldownEndTime = time.Time{}
				continue
			}

			spell.RemainingCooldown = remaining
		}
	}
}

func (s *RoomService) StartCooldownUpdater() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			s.RefreshCooldowns()
		}
	}()
}

func (s *RoomService) SyncFromChampSelect(
	roomId string,
	session *models.ChampSelectSession,
) bool {
	if roomId == "" || session == nil {
		return false
	}

	players := make(
		[]models.Player,
		0,
		len(session.MyTeam),
	)

	for _, member := range session.MyTeam {
		player := models.Player{
			GameName:           member.GameName,
			TagLine:            member.TagLine,
			ChampionId:         member.ChampionId,
			Champion:           GetChampionName(member.ChampionId),
			SummonerSpellHaste: constants.DefaultSummonerSpellHaste,
			Spells: []models.SummonerSpell{
				{
					Name: GetSpellName(
						member.Spell1Id,
					),
					IsReady: true,
					BaseCooldown: GetSpellCooldown(
						member.Spell1Id,
					),
				},
				{
					Name: GetSpellName(
						member.Spell2Id,
					),
					IsReady: true,
					BaseCooldown: GetSpellCooldown(
						member.Spell2Id,
					),
				},
			},
		}

		players = append(players, player)
	}

	return s.ReplacePlayers(roomId, players)
}

func (s *RoomService) ReplacePlayers(
	roomId string,
	players []models.Player,
) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists := s.repository.Get(roomId)
	if !exists {
		return false
	}

	for i := range players {
		newPlayer := &players[i]

		oldPlayer := FindPlayerByRiotId(
			room,
			newPlayer.GameName,
			newPlayer.TagLine,
		)

		if oldPlayer != nil {
			CopySpellState(oldPlayer, newPlayer)
		}
	}

	room.Players = players
	s.saveRoom(room)

	return true
}

func (s *RoomService) ToggleSpellByRiotId(
	roomId string,
	gameName string,
	tagLine string,
	spellName string,
) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists := s.repository.Get(roomId)
	if !exists {
		return false
	}

	player := FindPlayerByRiotId(
		room,
		gameName,
		tagLine,
	)
	if player == nil {
		return false
	}

	if !TogglePlayerSpell(player, spellName) {
		return false
	}

	s.saveRoom(room)

	return true
}

func (s *RoomService) StartRoomCleanup() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.cleanupOldRooms()
		}
	}()
}

func (s *RoomService) cleanupOldRooms() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	rooms := s.repository.GetAll()

	for _, room := range rooms {
		if now.Sub(room.LastUpdated) >
			constants.RoomExpirationDuration {
			s.repository.Delete(room.Id)
		}
	}
}
func (s *RoomService) saveRoom(room *models.Room) {
	if room == nil {
		return
	}

	room.LastUpdated = time.Now()
	s.repository.Save(room)
}
