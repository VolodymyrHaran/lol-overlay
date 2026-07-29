package services

import (
	"fmt"
	"log"
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
) (*models.Room, error) {
	room, exists, err := s.repository.Get(roomId)
	if err != nil {
		return nil, fmt.Errorf("get room %q: %w", roomId, err)
	}

	if !exists {
		return nil, nil
	}

	return room, nil
}

func (s *RoomService) GetRoomSnapshots() (
	[]*models.Room,
	error,
) {
	rooms, err := s.repository.GetAll()
	if err != nil {
		return nil, fmt.Errorf("get all rooms: %w", err)
	}

	return rooms, nil
}

func (s *RoomService) CreateRoom(
	roomId string,
) (*models.Room, error) {
	if roomId == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existingRoom, exists, err := s.repository.Get(roomId)
	if err != nil {
		return nil, fmt.Errorf(
			"check existing room %q: %w",
			roomId,
			err,
		)
	}

	if exists {
		return existingRoom, nil
	}

	room := &models.Room{
		Id:          roomId,
		LastUpdated: time.Now(),
	}

	if err := s.repository.Save(room); err != nil {
		return nil, fmt.Errorf(
			"save room %q: %w",
			roomId,
			err,
		)
	}

	return room.Clone(), nil
}

func (s *RoomService) AddPlayer(
	roomId string,
	player models.Player,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists, err := s.repository.Get(roomId)
	if err != nil {
		return false, fmt.Errorf(
			"get room %q for adding player: %w",
			roomId,
			err,
		)
	}

	if !exists {
		return false, nil
	}

	existingPlayer := FindPlayerByRiotId(
		room,
		player.GameName,
		player.TagLine,
	)

	if existingPlayer != nil {
		existingPlayer.Champion = player.Champion
		existingPlayer.ChampionId = player.ChampionId

		if err := s.saveRoom(room); err != nil {
			return false, err
		}

		return true, nil
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

	if err := s.saveRoom(room); err != nil {
		return false, err
	}

	return true, nil
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

func (s *RoomService) RefreshCooldowns() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms, err := s.repository.GetAll()
	if err != nil {
		return fmt.Errorf(
			"get rooms for cooldown refresh: %w",
			err,
		)
	}

	for _, room := range rooms {
		refreshRoomCooldowns(room)

		if err := s.repository.Save(room); err != nil {
			return fmt.Errorf(
				"save room %q after cooldown refresh: %w",
				room.Id,
				err,
			)
		}
	}

	return nil
}

func refreshRoomCooldowns(room *models.Room) {
	if room == nil {
		return
	}

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
			if err := s.RefreshCooldowns(); err != nil {
				log.Printf(
					"refresh cooldowns error: %v",
					err,
				)
			}
		}
	}()
}

func (s *RoomService) SyncFromChampSelect(
	roomId string,
	session *models.ChampSelectSession,
) (bool, error) {
	if roomId == "" || session == nil {
		return false, nil
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
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists, err := s.repository.Get(roomId)
	if err != nil {
		return false, fmt.Errorf(
			"get room %q for replacing players: %w",
			roomId,
			err,
		)
	}

	if !exists {
		return false, nil
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

	if err := s.saveRoom(room); err != nil {
		return false, err
	}

	return true, nil
}

func (s *RoomService) ToggleSpellByRiotId(
	roomId string,
	gameName string,
	tagLine string,
	spellName string,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists, err := s.repository.Get(roomId)
	if err != nil {
		return false, fmt.Errorf(
			"get room %q for toggling spell: %w",
			roomId,
			err,
		)
	}

	if !exists {
		return false, nil
	}

	player := FindPlayerByRiotId(
		room,
		gameName,
		tagLine,
	)
	if player == nil {
		return false, nil
	}

	if !TogglePlayerSpell(player, spellName) {
		return false, nil
	}

	if err := s.saveRoom(room); err != nil {
		return false, err
	}

	return true, nil
}

func (s *RoomService) StartRoomCleanup() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.cleanupOldRooms(); err != nil {
				log.Printf(
					"room cleanup error: %v",
					err,
				)
			}
		}
	}()
}

func (s *RoomService) cleanupOldRooms() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms, err := s.repository.GetAll()
	if err != nil {
		return fmt.Errorf(
			"get rooms for cleanup: %w",
			err,
		)
	}

	now := time.Now()

	for _, room := range rooms {
		if now.Sub(room.LastUpdated) <=
			constants.RoomExpirationDuration {
			continue
		}

		if err := s.repository.Delete(room.Id); err != nil {
			return fmt.Errorf(
				"delete expired room %q: %w",
				room.Id,
				err,
			)
		}
	}

	return nil
}

func (s *RoomService) saveRoom(
	room *models.Room,
) error {
	if room == nil {
		return nil
	}

	room.LastUpdated = time.Now()

	if err := s.repository.Save(room); err != nil {
		return fmt.Errorf(
			"save room %q: %w",
			room.Id,
			err,
		)
	}

	return nil
}
