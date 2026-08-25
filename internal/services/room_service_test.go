package services

import (
	"context"
	"lol-timer/internal/constants"
	"lol-timer/internal/models"
	"lol-timer/internal/repositories"
	"net/http"
	"testing"
	"time"
)

type noopEventPublisher struct{}

func (noopEventPublisher) Publish(
	subject string,
	data []byte,
) error {
	return nil
}

func newTestRoomService(t *testing.T) *RoomService {
	t.Helper()

	repository := repositories.NewInMemoryRoomRepository()

	return NewRoomService(repository, newTestChampionService(), noopEventPublisher{})
}

func mustCreateRoom(
	t *testing.T,
	service *RoomService,
	roomId string,
) *models.Room {
	t.Helper()

	room, err := service.CreateRoom(testContext(), roomId)
	if err != nil {
		t.Fatalf("create room %q: %v", roomId, err)
	}

	if room == nil {
		t.Fatalf("expected room %q to exist", roomId)
	}

	return room
}

func mustGetRoom(
	t *testing.T,
	service *RoomService,
	roomId string,
) *models.Room {
	t.Helper()

	room, err := service.GetRoomSnapshot(testContext(), roomId)
	if err != nil {
		t.Fatalf("get room %q: %v", roomId, err)
	}

	if room == nil {
		t.Fatalf("expected room %q to exist", roomId)
	}

	return room
}

func mustGetRooms(
	t *testing.T,
	service *RoomService,
) []*models.Room {
	t.Helper()

	rooms, err := service.GetRoomSnapshots(testContext())
	if err != nil {
		t.Fatalf("get room snapshots: %v", err)
	}

	return rooms
}

func TestCreateRoomCreatesRoom(t *testing.T) {
	service := newTestRoomService(t)

	const roomId = "game-123-team-100"

	room := mustCreateRoom(t, service, roomId)

	if room.Id != roomId {
		t.Errorf(
			"expected room ID %q, got %q",
			roomId,
			room.Id,
		)
	}

	if room.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be initialized")
	}
}

func TestCreateRoomReturnsExistingRoom(t *testing.T) {
	repository := repositories.NewInMemoryRoomRepository()
	service := NewRoomService(
		repository,
		newTestChampionService(),
		noopEventPublisher{},
	)

	firstRoom := mustCreateRoom(t, service, "room-1")
	secondRoom := mustCreateRoom(t, service, "room-1")

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

	rooms := mustGetRooms(t, service)

	if len(rooms) != 1 {
		t.Errorf(
			"expected 1 room in repository, got %d",
			len(rooms),
		)
	}
}
func TestGetRoomReturnsNilForUnknownRoom(t *testing.T) {
	service := newTestRoomService(t)

	room, err := service.GetRoomSnapshot(testContext(), "unknown")
	if err != nil {
		t.Fatalf("get unknown room: %v", err)
	}

	if room != nil {
		t.Errorf(
			"expected nil for unknown room, got %+v",
			room,
		)
	}
}

func TestAddPlayerAddsPlayerWithDefaults(t *testing.T) {
	service := newTestRoomService(t)
	mustCreateRoom(t, service, "room-1")

	player := models.Player{
		GameName:   "Player",
		TagLine:    "EUW",
		Champion:   "Ahri",
		ChampionId: 103,
	}

	ok, err := service.AddPlayer(testContext(), "room-1", player)
	if err != nil {
		t.Fatalf("add player: %v", err)
	}
	if !ok {
		t.Fatal("expected AddPlayer to succeed")
	}

	room := mustGetRoom(t, service, "room-1")

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

	ok, err := service.AddPlayer(testContext(), "unknown", models.Player{})
	if err != nil {
		t.Fatalf("add player: %v", err)
	}
	if ok {
		t.Error("expected AddPlayer to fail for unknown room")
	}
}

func TestAddPlayerDoesNotCreateDuplicate(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom(testContext(), "room-1")

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

	service.AddPlayer(testContext(), "room-1", firstPlayer)
	service.AddPlayer(testContext(), "room-1", secondPlayer)

	room := mustGetRoom(t, service, "room-1")

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
	service.CreateRoom(testContext(), "room-1")

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

	service.ReplacePlayers(testContext(), "room-1", oldPlayers)

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

	ok, err := service.ReplacePlayers(testContext(), "room-1", newPlayers)
	if err != nil {
		t.Fatalf("replace players: %v", err)
	}

	if ok {
		t.Fatal("expected preserved state not to update room")
	}

	room := mustGetRoom(t, service, "room-1")
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
	service.CreateRoom(testContext(), "room-1")

	service.ReplacePlayers(testContext(), "room-1", []models.Player{
		{GameName: "Player1", TagLine: "EUW"},
		{GameName: "Player2", TagLine: "EUW"},
	})

	service.ReplacePlayers(testContext(), "room-1", []models.Player{
		{GameName: "Player2", TagLine: "EUW"},
	})

	room := mustGetRoom(t, service, "room-1")

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
	service.CreateRoom(testContext(), "room-1")

	service.ReplacePlayers(testContext(), "room-1", []models.Player{
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

	ok, err := service.ToggleSpellByRiotId(
		testContext(),
		"room-1",
		"Player",
		"EUW",
		"Flash",
	)
	if err != nil {
		t.Fatalf("toggle spell: %v", err)
	}

	if !ok {
		t.Fatal("expected toggle to succeed")
	}

	room := mustGetRoom(t, service, "room-1")
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
	service.CreateRoom(testContext(), "room-1")

	ok, err := service.ToggleSpellByRiotId(
		testContext(),
		"room-1",
		"Unknown",
		"EUW",
		"Flash",
	)
	if err != nil {
		t.Fatalf("toggle spell: %v", err)
	}
	if ok {
		t.Error("expected false for unknown player")
	}
}

func TestRefreshCooldownsMarksExpiredSpellReady(t *testing.T) {
	service := newTestRoomService(t)
	service.CreateRoom(testContext(), "room-1")

	service.ReplacePlayers(testContext(), "room-1", []models.Player{
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

	service.RefreshCooldowns(testContext())

	room := mustGetRoom(t, service, "room-1")
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
	service.CreateRoom(testContext(), "room-1")

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

	ok, err := service.SyncFromChampSelect(testContext(), "room-1", session)
	if err != nil {
		t.Fatalf("sync from champ select: %v", err)
	}
	if !ok {
		t.Fatal("expected synchronization to succeed")
	}

	room := mustGetRoom(t, service, "room-1")
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
	service.CreateRoom(testContext(), "room-1")

	ok, err := service.SyncFromChampSelect(testContext(), "room-1", nil)
	if err != nil {
		t.Fatalf("sync from champ select: %v", err)
	}

	if ok {
		t.Error("expected false for nil session")
	}
}

func TestSyncFromChampSelectReturnsFalseForEmptyRoomId(t *testing.T) {
	service := newTestRoomService(t)

	session := &models.ChampSelectSession{}

	ok, err := service.SyncFromChampSelect(testContext(), "", session)
	if err != nil {
		t.Fatalf("sync from champ select: %v", err)
	}

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

	ok, err := service.SyncFromChampSelect(testContext(), "unknown-room", session)
	if err != nil {
		t.Fatalf("sync from champ select: %v", err)
	}

	if ok {
		t.Error("expected false for unknown room")
	}
}

func testContext() context.Context {
	return context.Background()
}

func TestCurrentRoomID(t *testing.T) {
	service := newTestRoomService(t)

	if actual := service.GetCurrentRoomID(); actual != "" {
		t.Fatalf(
			"expected empty current room, got %q",
			actual,
		)
	}

	service.SetCurrentRoomID("room-1")

	if actual := service.GetCurrentRoomID(); actual != "room-1" {
		t.Fatalf(
			"expected current room %q, got %q",
			"room-1",
			actual,
		)
	}

	service.ClearCurrentRoomID("another-room")

	if actual := service.GetCurrentRoomID(); actual != "room-1" {
		t.Fatalf(
			"expected current room to remain %q, got %q",
			"room-1",
			actual,
		)
	}

	service.ClearCurrentRoomID("room-1")

	if actual := service.GetCurrentRoomID(); actual != "" {
		t.Fatalf(
			"expected current room to be cleared, got %q",
			actual,
		)
	}
}

func newTestChampionService() *ChampionService {
	return &ChampionService{
		champions: map[int]ChampionInfo{
			103: {
				ID:       103,
				Name:     "Ahri",
				ImageURL: "https://example.com/Ahri.png",
			},
			222: {
				ID:       222,
				Name:     "Jinx",
				ImageURL: "https://example.com/Jinx.png",
			},
		},
		client: &http.Client{},
	}
}

func TestReplacePlayersReturnsFalseWhenPlayersUnchanged(
	t *testing.T,
) {
	service := newTestRoomService(t)
	mustCreateRoom(t, service, "room-1")

	players := []models.Player{
		{
			GameName:   "Player",
			TagLine:    "EUW",
			Champion:   "Ahri",
			ChampionId: 103,
		},
	}

	updated, err := service.ReplacePlayers(
		testContext(),
		"room-1",
		players,
	)
	if err != nil {
		t.Fatalf("first replace players: %v", err)
	}

	if !updated {
		t.Fatal("expected first replacement to update room")
	}

	updated, err = service.ReplacePlayers(
		testContext(),
		"room-1",
		players,
	)
	if err != nil {
		t.Fatalf("second replace players: %v", err)
	}

	if updated {
		t.Fatal("expected unchanged players not to update room")
	}
}
