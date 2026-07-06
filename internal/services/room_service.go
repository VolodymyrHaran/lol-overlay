package services

import (
	"lol-timer/internal/constants"
	"lol-timer/internal/models"
	"sync"
	"time"
)

type RoomService struct {
	mu    sync.Mutex
	rooms map[string]*models.Room
}

func NewRoomService() *RoomService {
	return &RoomService{
		rooms: make(map[string]*models.Room),
	}
}

func (s *RoomService) GetRoom(roomId string) *models.Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.rooms[roomId]
}

func (s *RoomService) GetRooms() []*models.Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms := make([]*models.Room, 0, len(s.rooms))

	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

func (s *RoomService) CreateRoom(roomId string) *models.Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, exists := s.rooms[roomId]; exists {
		return room
	}

	room := &models.Room{
		Id:          roomId,
		LastUpdated: time.Now(),
	}

	s.rooms[roomId] = room
	return room
}

func (s *RoomService) AddPlayer(roomId string, player models.Player) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists := s.rooms[roomId]
	if !exists {
		return false
	}

	for i := range room.Players {
		existingPlayer := &room.Players[i]

		if existingPlayer.GameName == player.GameName &&
			existingPlayer.TagLine == player.TagLine {

			existingPlayer.Champion = player.Champion
			existingPlayer.ChampionId = player.ChampionId

			return true
		}
	}

	player.SummonerSpellHaste = constants.DefaultSummonerSpellHaste
	if len(player.Spells) == 0 {
		player.Spells = []models.SummonerSpell{
			{Name: "Flash", IsReady: true, BaseCooldown: 300},
			{Name: "Ignite", IsReady: true, BaseCooldown: 180},
		}
	}

	room.Players = append(room.Players, player)

	return true
}

func FindPlayerByRiotId(
	room *models.Room,
	gameName string,
	tagLine string,
) *models.Player {
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
	s.refreshCooldowns()
}

func (s *RoomService) refreshCooldowns() {
	for _, room := range s.rooms {
		for pi := range room.Players {
			for si := range room.Players[pi].Spells {
				spell := &room.Players[pi].Spells[si]

				if spell.IsReady || spell.CooldownEndTime.IsZero() {
					spell.RemainingCooldown = 0
					continue
				}

				remaining := int(time.Until(spell.CooldownEndTime).Seconds())

				if remaining <= 0 {
					spell.IsReady = true
					spell.RemainingCooldown = 0
					spell.CooldownEndTime = time.Time{}
				} else {
					spell.RemainingCooldown = remaining
				}
			}
		}
	}
}

func (s *RoomService) StartCooldownUpdater() {
	go func() {
		ticker := time.NewTicker(time.Second)

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

	players := make([]models.Player, 0, len(session.MyTeam))

	for _, member := range session.MyTeam {
		player := models.Player{
			GameName:           member.GameName,
			TagLine:            member.TagLine,
			ChampionId:         member.ChampionId,
			SummonerSpellHaste: constants.DefaultSummonerSpellHaste,
			Spells: []models.SummonerSpell{
				{
					Name:         GetSpellName(member.Spell1Id),
					IsReady:      true,
					BaseCooldown: GetSpellCooldown(member.Spell1Id),
				},
				{
					Name:         GetSpellName(member.Spell2Id),
					IsReady:      true,
					BaseCooldown: GetSpellCooldown(member.Spell2Id),
				},
			},
			Champion: GetChampionName(member.ChampionId),
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

	room, exists := s.rooms[roomId]
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
	room.LastUpdated = time.Now()

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

	room, exists := s.rooms[roomId]
	if !exists {
		return false
	}

	player := FindPlayerByRiotId(room, gameName, tagLine)
	if player == nil {
		return false
	}

	return TogglePlayerSpell(player, spellName)
}

func (s *RoomService) StartRoomCleanup() {
	go func() {
		ticker := time.NewTicker(time.Minute)

		for range ticker.C {
			s.cleanupOldRooms()
		}
	}()
}

func (s *RoomService) cleanupOldRooms() {
	s.mu.Lock()
	defer s.mu.Unlock()

	expirationTime := constants.RoomExpirationDuration
	now := time.Now()

	for roomId, room := range s.rooms {
		if now.Sub(room.LastUpdated) > expirationTime {
			delete(s.rooms, roomId)
		}
	}
}
