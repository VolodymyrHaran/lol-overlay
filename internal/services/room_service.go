package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"lol-timer/internal/constants"
	"lol-timer/internal/models"
	"lol-timer/internal/repositories"
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
	ctx context.Context,
	roomID string,
) (*models.Room, error) {
	room, exists, err := s.repository.Get(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf(
			"get room %q: %w",
			roomID,
			err,
		)
	}

	if !exists {
		return nil, nil
	}

	return room, nil
}

func (s *RoomService) GetRoomSnapshots(
	ctx context.Context,
) ([]*models.Room, error) {
	rooms, err := s.repository.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all rooms: %w", err)
	}

	return rooms, nil
}

func (s *RoomService) CreateRoom(
	ctx context.Context,
	roomID string,
) (*models.Room, error) {
	if roomID == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existingRoom, exists, err := s.repository.Get(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf(
			"check existing room %q: %w",
			roomID,
			err,
		)
	}

	if exists {
		return existingRoom, nil
	}

	room := &models.Room{
		Id:          roomID,
		LastUpdated: time.Now(),
	}

	if err := s.repository.Save(ctx, room); err != nil {
		return nil, fmt.Errorf(
			"save room %q: %w",
			roomID,
			err,
		)
	}

	return room.Clone(), nil
}

func (s *RoomService) AddPlayer(
	ctx context.Context,
	roomID string,
	player models.Player,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists, err := s.repository.Get(ctx, roomID)
	if err != nil {
		return false, fmt.Errorf(
			"get room %q for adding player: %w",
			roomID,
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

		if err := s.saveRoom(ctx, room); err != nil {
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

	if err := s.saveRoom(ctx, room); err != nil {
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

func (s *RoomService) RefreshCooldowns(
	ctx context.Context,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms, err := s.repository.GetAll(ctx)
	if err != nil {
		return fmt.Errorf(
			"get rooms for cooldown refresh: %w",
			err,
		)
	}

	for _, room := range rooms {
		if err := ctx.Err(); err != nil {
			return err
		}

		refreshRoomCooldowns(room)

		if err := s.repository.Save(ctx, room); err != nil {
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

func (s *RoomService) StartCooldownUpdater(
	ctx context.Context,
) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := s.RefreshCooldowns(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}

					log.Printf(
						"refresh cooldowns error: %v",
						err,
					)
				}
			}
		}
	}()
}

func (s *RoomService) SyncFromChampSelect(
	ctx context.Context,
	roomID string,
	session *models.ChampSelectSession,
) (bool, error) {
	if roomID == "" || session == nil {
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

	return s.ReplacePlayers(ctx, roomID, players)
}

func (s *RoomService) ReplacePlayers(
	ctx context.Context,
	roomID string,
	players []models.Player,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists, err := s.repository.Get(ctx, roomID)
	if err != nil {
		return false, fmt.Errorf(
			"get room %q for replacing players: %w",
			roomID,
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

	if err := s.saveRoom(ctx, room); err != nil {
		return false, err
	}

	return true, nil
}

func (s *RoomService) ToggleSpellByRiotId(
	ctx context.Context,
	roomID string,
	gameName string,
	tagLine string,
	spellName string,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists, err := s.repository.Get(ctx, roomID)
	if err != nil {
		return false, fmt.Errorf(
			"get room %q for toggling spell: %w",
			roomID,
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

	if err := s.saveRoom(ctx, room); err != nil {
		return false, err
	}

	return true, nil
}

func (s *RoomService) StartRoomCleanup(
	ctx context.Context,
) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := s.cleanupOldRooms(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}

					log.Printf(
						"room cleanup error: %v",
						err,
					)
				}
			}
		}
	}()
}

func (s *RoomService) cleanupOldRooms(
	ctx context.Context,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms, err := s.repository.GetAll(ctx)
	if err != nil {
		return fmt.Errorf(
			"get rooms for cleanup: %w",
			err,
		)
	}

	now := time.Now()

	for _, room := range rooms {
		if err := ctx.Err(); err != nil {
			return err
		}

		if now.Sub(room.LastUpdated) <=
			constants.RoomExpirationDuration {
			continue
		}

		if err := s.repository.Delete(ctx, room.Id); err != nil {
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
	ctx context.Context,
	room *models.Room,
) error {
	if room == nil {
		return nil
	}

	room.LastUpdated = time.Now()

	if err := s.repository.Save(ctx, room); err != nil {
		return fmt.Errorf(
			"save room %q: %w",
			room.Id,
			err,
		)
	}

	return nil
}
