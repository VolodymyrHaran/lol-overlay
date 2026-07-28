package services

import (
	"lol-timer/internal/constants"
	"lol-timer/internal/models"
	"lol-timer/internal/repositories"
	"testing"
	"time"
)

func TestCreateRoomCreatesRoom(t *testing.T) {
	service := newTestRoomService(t)

	room := service.CreateRoom("game-123-team-100")

	if room == nil {
		t.Fatal("expected room to be created")
	}

	if room.Id != "game-123-team-100" {
		t.Errorf(
			"expected room ID %q, got %q",
			"game-123-team-100",
			room.Id,
		)
	}

	if room.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be initialized")
	}
}

func TestCreateRoomReturnsExistingRoom(t *testing.T) {
	repository := repositories.NewInMemoryRoomRepository()
	service := NewRoomService(repository)

	firstRoom := service.CreateRoom("room-1")
	secondRoom := service.CreateRoom("room-1")

	if firstRoom == nil || secondRoom == nil {
		t.Fatal("expected both room results to be non-nil")
	}

	if firstRoom.Id != secondRoom.Id {
		t.Errorf(
			"expected the same room ID, got %q and %q",
			firstRoom.Id,
			secondRoom.Id,
		)
	}

	if firstRoom == secondRoom {
		t.Error("expected independent room snapshots")
	}

	rooms := service.GetRoomSnapshots()

	if len(rooms) != 1 {
		t.Errorf(
			"expected 1 room in repository, got %d",
			len(rooms),
		)
	}
}
func TestGetRoomReturnsNilForUnknownRoom(t *testing.T) {
	service := newTestRoomService(t)

	room := service.GetRoomSnapshot("unknown")

	if room != nil {
		t.Error("expected nil for unknown room")
	}
}

func TestAddPlayerAddsPlayerWithDefaults(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	player := models.Player{
		GameName:   "Player",
		TagLine:    "EUW",
		Champion:   "Ahri",
		ChampionId: 103,
	}

	ok := service.AddPlayer("room-1", player)
	if !ok {
		t.Fatal("expected AddPlayer to succeed")
	}

	room := service.GetRoomSnapshot("room-1")

	if len(room.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(room.Players))
	}

	addedPlayer := room.Players[0]

	if addedPlayer.SummonerSpellHaste != constants.DefaultSummonerSpellHaste {
		t.Errorf(
			"expected haste %d, got %d",
			constants.DefaultSummonerSpellHaste,
			addedPlayer.SummonerSpellHaste,
		)
	}

	if len(addedPlayer.Spells) != 2 {
		t.Fatalf("expected 2 default spells, got %d", len(addedPlayer.Spells))
	}

	if addedPlayer.Spells[0].Name != "Flash" {
		t.Errorf(
			"expected first spell Flash, got %q",
			addedPlayer.Spells[0].Name,
		)
	}

	if addedPlayer.Spells[1].Name != "Ignite" {
		t.Errorf(
			"expected second spell Ignite, got %q",
			addedPlayer.Spells[1].Name,
		)
	}
}

func TestAddPlayerReturnsFalseForUnknownRoom(t *testing.T) {
	service := newTestRoomService(t)

	ok := service.AddPlayer("unknown", models.Player{})

	if ok {
		t.Error("expected AddPlayer to fail for unknown room")
	}
}

func TestAddPlayerDoesNotCreateDuplicate(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	firstPlayer := models.Player{
		GameName:   "Player",
		TagLine:    "EUW",
		Champion:   "Ahri",
		ChampionId: 103,
	}

	secondPlayer := models.Player{
		GameName:   "Player",
		TagLine:    "EUW",
		Champion:   "Jinx",
		ChampionId: 222,
	}

	service.AddPlayer("room-1", firstPlayer)
	service.AddPlayer("room-1", secondPlayer)

	room := service.GetRoomSnapshot("room-1")

	if len(room.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(room.Players))
	}

	player := room.Players[0]

	if player.Champion != "Jinx" {
		t.Errorf(
			"expected champion to be updated to Jinx, got %q",
			player.Champion,
		)
	}

	if player.ChampionId != 222 {
		t.Errorf(
			"expected champion ID 222, got %d",
			player.ChampionId,
		)
	}
}

func TestReplacePlayersPreservesCooldownState(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	cooldownEnd := time.Now().Add(120 * time.Second)

	oldPlayers := []models.Player{
		{
			GameName: "Player",
			TagLine:  "EUW",
			Spells: []models.SummonerSpell{
				{
					Name:              "Flash",
					IsReady:           false,
					BaseCooldown:      300,
					RemainingCooldown: 120,
					CooldownEndTime:   cooldownEnd,
				},
			},
		},
	}

	service.ReplacePlayers("room-1", oldPlayers)

	newPlayers := []models.Player{
		{
			GameName: "Player",
			TagLine:  "EUW",
			Spells: []models.SummonerSpell{
				{
					Name:         "Flash",
					IsReady:      true,
					BaseCooldown: 300,
				},
			},
		},
	}

	ok := service.ReplacePlayers("room-1", newPlayers)
	if !ok {
		t.Fatal("expected ReplacePlayers to succeed")
	}

	room := service.GetRoomSnapshot("room-1")
	spell := room.Players[0].FindSpell("Flash")

	if spell == nil {
		t.Fatal("expected Flash to exist")
	}

	if spell.IsReady {
		t.Error("expected active cooldown to be preserved")
	}

	if spell.RemainingCooldown != 120 {
		t.Errorf(
			"expected remaining cooldown 120, got %d",
			spell.RemainingCooldown,
		)
	}

	if !spell.CooldownEndTime.Equal(cooldownEnd) {
		t.Error("expected CooldownEndTime to be preserved")
	}
}

func TestReplacePlayersRemovesMissingPlayers(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	service.ReplacePlayers("room-1", []models.Player{
		{GameName: "Player1", TagLine: "EUW"},
		{GameName: "Player2", TagLine: "EUW"},
	})

	service.ReplacePlayers("room-1", []models.Player{
		{GameName: "Player2", TagLine: "EUW"},
	})

	room := service.GetRoomSnapshot("room-1")

	if len(room.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(room.Players))
	}

	if room.Players[0].GameName != "Player2" {
		t.Errorf(
			"expected Player2 to remain, got %q",
			room.Players[0].GameName,
		)
	}
}

func TestToggleSpellByRiotId(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	service.ReplacePlayers("room-1", []models.Player{
		{
			GameName:           "Player",
			TagLine:            "EUW",
			SummonerSpellHaste: 0,
			Spells: []models.SummonerSpell{
				{
					Name:         "Flash",
					IsReady:      true,
					BaseCooldown: 300,
				},
			},
		},
	})

	ok := service.ToggleSpellByRiotId(
		"room-1",
		"Player",
		"EUW",
		"Flash",
	)

	if !ok {
		t.Fatal("expected toggle to succeed")
	}

	room := service.GetRoomSnapshot("room-1")
	spell := room.Players[0].FindSpell("Flash")

	if spell == nil {
		t.Fatal("expected Flash to exist")
	}

	if spell.IsReady {
		t.Error("expected Flash to be on cooldown")
	}

	if spell.CooldownEndTime.IsZero() {
		t.Error("expected CooldownEndTime to be set")
	}
}

func TestToggleSpellByRiotIdReturnsFalseForUnknownPlayer(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	ok := service.ToggleSpellByRiotId(
		"room-1",
		"Unknown",
		"EUW",
		"Flash",
	)

	if ok {
		t.Error("expected false for unknown player")
	}
}

func TestRefreshCooldownsMarksExpiredSpellReady(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	service.ReplacePlayers("room-1", []models.Player{
		{
			GameName: "Player",
			TagLine:  "EUW",
			Spells: []models.SummonerSpell{
				{
					Name:              "Flash",
					IsReady:           false,
					BaseCooldown:      300,
					RemainingCooldown: 1,
					CooldownEndTime:   time.Now().Add(-time.Second),
				},
			},
		},
	})

	service.RefreshCooldowns()

	room := service.GetRoomSnapshot("room-1")
	spell := room.Players[0].FindSpell("Flash")

	if spell == nil {
		t.Fatal("expected Flash to exist")
	}

	if !spell.IsReady {
		t.Error("expected expired spell to become ready")
	}

	if spell.RemainingCooldown != 0 {
		t.Errorf(
			"expected remaining cooldown 0, got %d",
			spell.RemainingCooldown,
		)
	}

	if !spell.CooldownEndTime.IsZero() {
		t.Error("expected CooldownEndTime to be reset")
	}
}
func TestSyncFromChampSelectCreatesPlayersFromMyTeam(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	session := &models.ChampSelectSession{
		GameId:            123,
		LocalPlayerCellId: 1,
		MyTeam: []models.ChampSelectPlayer{
			{
				CellId:     1,
				ChampionId: 103,
				Spell1Id:   4,
				Spell2Id:   14,
				GameName:   "PlayerOne",
				TagLine:    "EUW",
				Puuid:      "puuid-1",
				Team:       100,
			},
			{
				CellId:     2,
				ChampionId: 222,
				Spell1Id:   4,
				Spell2Id:   7,
				GameName:   "PlayerTwo",
				TagLine:    "EUW",
				Puuid:      "puuid-2",
				Team:       100,
			},
		},
	}

	ok := service.SyncFromChampSelect("room-1", session)
	if !ok {
		t.Fatal("expected synchronization to succeed")
	}

	room := service.GetRoomSnapshot("room-1")
	if room == nil {
		t.Fatal("expected room to exist")
	}

	if len(room.Players) != 2 {
		t.Fatalf(
			"expected 2 players, got %d",
			len(room.Players),
		)
	}

	firstPlayer := room.Players[0]

	if firstPlayer.GameName != "PlayerOne" {
		t.Errorf(
			"expected game name PlayerOne, got %q",
			firstPlayer.GameName,
		)
	}

	if firstPlayer.TagLine != "EUW" {
		t.Errorf(
			"expected tag line EUW, got %q",
			firstPlayer.TagLine,
		)
	}

	if firstPlayer.ChampionId != 103 {
		t.Errorf(
			"expected champion ID 103, got %d",
			firstPlayer.ChampionId,
		)
	}

	if firstPlayer.Champion != "Ahri" {
		t.Errorf(
			"expected champion Ahri, got %q",
			firstPlayer.Champion,
		)
	}

	if firstPlayer.SummonerSpellHaste != constants.DefaultSummonerSpellHaste {
		t.Errorf(
			"expected haste %d, got %d",
			constants.DefaultSummonerSpellHaste,
			firstPlayer.SummonerSpellHaste,
		)
	}

	if len(firstPlayer.Spells) != 2 {
		t.Fatalf(
			"expected 2 spells, got %d",
			len(firstPlayer.Spells),
		)
	}

	if firstPlayer.Spells[0].Name != "Flash" {
		t.Errorf(
			"expected first spell Flash, got %q",
			firstPlayer.Spells[0].Name,
		)
	}

	if firstPlayer.Spells[0].BaseCooldown != 300 {
		t.Errorf(
			"expected Flash cooldown 300, got %d",
			firstPlayer.Spells[0].BaseCooldown,
		)
	}

	if firstPlayer.Spells[1].Name != "Ignite" {
		t.Errorf(
			"expected second spell Ignite, got %q",
			firstPlayer.Spells[1].Name,
		)
	}

	if firstPlayer.Spells[1].BaseCooldown != 180 {
		t.Errorf(
			"expected Ignite cooldown 180, got %d",
			firstPlayer.Spells[1].BaseCooldown,
		)
	}
}
func TestSyncFromChampSelectReturnsFalseForNilSession(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom("room-1")

	ok := service.SyncFromChampSelect("room-1", nil)

	if ok {
		t.Error("expected false for nil session")
	}
}

func TestSyncFromChampSelectReturnsFalseForEmptyRoomId(t *testing.T) {
	service := newTestRoomService(t)

	session := &models.ChampSelectSession{}

	ok := service.SyncFromChampSelect("", session)

	if ok {
		t.Error("expected false for empty room ID")
	}
}

func TestSyncFromChampSelectReturnsFalseForUnknownRoom(t *testing.T) {
	service := newTestRoomService(t)

	session := &models.ChampSelectSession{
		MyTeam: []models.ChampSelectPlayer{
			{
				GameName: "Player",
				TagLine:  "EUW",
			},
		},
	}

	ok := service.SyncFromChampSelect("unknown-room", session)

	if ok {
		t.Error("expected false for unknown room")
	}
}

func newTestRoomService(t *testing.T) *RoomService {
	t.Helper()

	repository := repositories.NewInMemoryRoomRepository()
	return NewRoomService(repository)
}
