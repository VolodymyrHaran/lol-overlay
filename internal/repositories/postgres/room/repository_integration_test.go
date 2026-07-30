package room

import (
	"context"
	"os"
	"testing"
	"time"

	"lol-timer/internal/database"
	"lol-timer/internal/models"
)

func newTestRepository(
	t *testing.T,
) (*RoomRepository, func()) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	repository := NewRoomRepository(db)

	cleanup := func() {
		_, err := db.Pool.Exec(
			context.Background(),
			`
			TRUNCATE TABLE
				summoner_spells,
				players,
				rooms
			RESTART IDENTITY
			CASCADE
			`,
		)
		if err != nil {
			t.Errorf("truncate test database: %v", err)
		}

		db.Close()
	}

	_, err = db.Pool.Exec(
		ctx,
		`
		TRUNCATE TABLE
			summoner_spells,
			players,
			rooms
		RESTART IDENTITY
		CASCADE
		`,
	)
	if err != nil {
		db.Close()
		t.Fatalf("prepare test database: %v", err)
	}

	return repository, cleanup
}

func TestRoomRepositorySaveAndGet(t *testing.T) {
	repository, cleanup := newTestRepository(t)
	defer cleanup()

	cooldownEndTime := time.Now().
		Add(120 * time.Second).
		UTC().
		Truncate(time.Microsecond)

	expected := &models.Room{
		Id: "integration-room-1",
		Players: []models.Player{
			{
				GameName:           "PlayerOne",
				TagLine:            "EUW",
				Champion:           "Ahri",
				ChampionId:         103,
				SummonerSpellHaste: 18,
				Spells: []models.SummonerSpell{
					{
						Name:              "Flash",
						IsReady:           false,
						BaseCooldown:      300,
						RemainingCooldown: 120,
						CooldownEndTime:   cooldownEndTime,
					},
					{
						Name:              "Ignite",
						IsReady:           true,
						BaseCooldown:      180,
						RemainingCooldown: 0,
					},
				},
			},
		},
		LastUpdated: time.Now().
			UTC().
			Truncate(time.Microsecond),
	}

	if err := repository.Save(context.Background(), expected); err != nil {
		t.Fatalf("save room: %v", err)
	}

	actual, exists, err := repository.Get(context.Background(), expected.Id)
	if err != nil {
		t.Fatalf("get room: %v", err)
	}

	if !exists {
		t.Fatal("expected room to exist")
	}

	if actual.Id != expected.Id {
		t.Errorf(
			"expected room ID %q, got %q",
			expected.Id,
			actual.Id,
		)
	}

	if !actual.LastUpdated.Equal(expected.LastUpdated) {
		t.Errorf(
			"expected LastUpdated %v, got %v",
			expected.LastUpdated,
			actual.LastUpdated,
		)
	}

	if len(actual.Players) != 1 {
		t.Fatalf(
			"expected 1 player, got %d",
			len(actual.Players),
		)
	}

	player := actual.Players[0]

	if player.GameName != "PlayerOne" {
		t.Errorf(
			"expected game name PlayerOne, got %q",
			player.GameName,
		)
	}

	if player.TagLine != "EUW" {
		t.Errorf(
			"expected tag line EUW, got %q",
			player.TagLine,
		)
	}

	if player.Champion != "Ahri" {
		t.Errorf(
			"expected champion Ahri, got %q",
			player.Champion,
		)
	}

	if player.ChampionId != 103 {
		t.Errorf(
			"expected champion ID 103, got %d",
			player.ChampionId,
		)
	}

	if player.SummonerSpellHaste != 18 {
		t.Errorf(
			"expected haste 18, got %d",
			player.SummonerSpellHaste,
		)
	}

	if len(player.Spells) != 2 {
		t.Fatalf(
			"expected 2 spells, got %d",
			len(player.Spells),
		)
	}

	flash := player.FindSpell("Flash")
	if flash == nil {
		t.Fatal("expected Flash to exist")
	}

	if flash.IsReady {
		t.Error("expected Flash to be on cooldown")
	}

	if flash.RemainingCooldown != 120 {
		t.Errorf(
			"expected remaining cooldown 120, got %d",
			flash.RemainingCooldown,
		)
	}

	if !flash.CooldownEndTime.Equal(cooldownEndTime) {
		t.Errorf(
			"expected cooldown end %v, got %v",
			cooldownEndTime,
			flash.CooldownEndTime,
		)
	}

	ignite := player.FindSpell("Ignite")
	if ignite == nil {
		t.Fatal("expected Ignite to exist")
	}

	if !ignite.IsReady {
		t.Error("expected Ignite to be ready")
	}

	if !ignite.CooldownEndTime.IsZero() {
		t.Errorf(
			"expected zero Ignite cooldown end, got %v",
			ignite.CooldownEndTime,
		)
	}
}
func TestRoomRepositorySaveReplacesPlayersAndSpells(t *testing.T) {
	repository, cleanup := newTestRepository(t)
	defer cleanup()

	room := &models.Room{
		Id: "integration-room-update",
		Players: []models.Player{
			{
				GameName:           "PlayerOne",
				TagLine:            "EUW",
				Champion:           "Ahri",
				ChampionId:         103,
				SummonerSpellHaste: 18,
				Spells: []models.SummonerSpell{
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
				},
			},
		},
		LastUpdated: time.Now().UTC().Truncate(time.Microsecond),
	}

	if err := repository.Save(context.Background(), room); err != nil {
		t.Fatalf("save initial room: %v", err)
	}

	room.Players = []models.Player{
		{
			GameName:           "PlayerTwo",
			TagLine:            "EUNE",
			Champion:           "Jinx",
			ChampionId:         222,
			SummonerSpellHaste: 10,
			Spells: []models.SummonerSpell{
				{
					Name:              "Heal",
					IsReady:           false,
					BaseCooldown:      240,
					RemainingCooldown: 100,
					CooldownEndTime: time.Now().
						Add(100 * time.Second).
						UTC().
						Truncate(time.Microsecond),
				},
			},
		},
	}
	room.LastUpdated = time.Now().UTC().Truncate(time.Microsecond)

	if err := repository.Save(context.Background(), room); err != nil {
		t.Fatalf("save updated room: %v", err)
	}

	actual, exists, err := repository.Get(context.Background(), room.Id)
	if err != nil {
		t.Fatalf("get updated room: %v", err)
	}

	if !exists {
		t.Fatal("expected updated room to exist")
	}

	if len(actual.Players) != 1 {
		t.Fatalf(
			"expected 1 player after replacement, got %d",
			len(actual.Players),
		)
	}

	player := actual.Players[0]

	if player.GameName != "PlayerTwo" {
		t.Errorf(
			"expected PlayerTwo, got %q",
			player.GameName,
		)
	}

	if player.FindSpell("Flash") != nil {
		t.Error("expected old Flash spell to be removed")
	}

	if player.FindSpell("Ignite") != nil {
		t.Error("expected old Ignite spell to be removed")
	}

	if player.FindSpell("Heal") == nil {
		t.Error("expected Heal spell to exist")
	}
}
func TestRoomRepositoryGetReturnsFalseForUnknownRoom(t *testing.T) {
	repository, cleanup := newTestRepository(t)
	defer cleanup()

	room, exists, err := repository.Get(context.Background(), "unknown-room")
	if err != nil {
		t.Fatalf("get unknown room: %v", err)
	}

	if exists {
		t.Error("expected room not to exist")
	}

	if room != nil {
		t.Errorf("expected nil room, got %+v", room)
	}
}
func TestRoomRepositoryGetAll(t *testing.T) {
	repository, cleanup := newTestRepository(t)
	defer cleanup()

	older := &models.Room{
		Id:          "older-room",
		LastUpdated: time.Now().Add(-time.Minute).UTC().Truncate(time.Microsecond),
	}

	newer := &models.Room{
		Id:          "newer-room",
		LastUpdated: time.Now().UTC().Truncate(time.Microsecond),
	}

	if err := repository.Save(context.Background(), older); err != nil {
		t.Fatalf("save older room: %v", err)
	}

	if err := repository.Save(context.Background(), newer); err != nil {
		t.Fatalf("save newer room: %v", err)
	}

	rooms, err := repository.GetAll(context.Background())
	if err != nil {
		t.Fatalf("get all rooms: %v", err)
	}

	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms, got %d", len(rooms))
	}

	if rooms[0].Id != "newer-room" {
		t.Errorf(
			"expected newer-room first, got %q",
			rooms[0].Id,
		)
	}

	if rooms[1].Id != "older-room" {
		t.Errorf(
			"expected older-room second, got %q",
			rooms[1].Id,
		)
	}
}
func TestRoomRepositoryDelete(t *testing.T) {
	repository, cleanup := newTestRepository(t)
	defer cleanup()

	room := &models.Room{
		Id:          "room-to-delete",
		LastUpdated: time.Now().UTC().Truncate(time.Microsecond),
		Players: []models.Player{
			{
				GameName:           "Player",
				TagLine:            "EUW",
				Champion:           "Ahri",
				ChampionId:         103,
				SummonerSpellHaste: 18,
				Spells: []models.SummonerSpell{
					{
						Name:         "Flash",
						IsReady:      true,
						BaseCooldown: 300,
					},
				},
			},
		},
	}

	if err := repository.Save(context.Background(), room); err != nil {
		t.Fatalf("save room: %v", err)
	}

	if err := repository.Delete(context.Background(), room.Id); err != nil {
		t.Fatalf("delete room: %v", err)
	}

	actual, exists, err := repository.Get(context.Background(), room.Id)
	if err != nil {
		t.Fatalf("get deleted room: %v", err)
	}

	if exists {
		t.Error("expected room to be deleted")
	}

	if actual != nil {
		t.Errorf("expected nil room, got %+v", actual)
	}
}
