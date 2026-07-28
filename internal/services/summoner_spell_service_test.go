package services

import (
	"lol-timer/internal/models"
	"lol-timer/internal/repositories"
	"testing"
	"time"
)

func TestCalculateCooldown(t *testing.T) {
	tests := []struct {
		name     string
		base     int
		haste    int
		expected int
	}{
		{
			name:     "without haste",
			base:     300,
			haste:    0,
			expected: 300,
		},
		{
			name:     "with 18 haste",
			base:     300,
			haste:    18,
			expected: 254,
		},
		{
			name:     "with 100 haste",
			base:     300,
			haste:    100,
			expected: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CalculateCooldown(tt.base, tt.haste)

			if actual != tt.expected {
				t.Fatalf(
					"expected cooldown %d, got %d",
					tt.expected,
					actual,
				)
			}
		})
	}
}

func TestGetSpellName(t *testing.T) {
	tests := []struct {
		spellID  int
		expected string
	}{
		{spellID: 4, expected: "Flash"},
		{spellID: 14, expected: "Ignite"},
		{spellID: 7, expected: "Heal"},
		{spellID: 999, expected: "Unknown"},
	}

	for _, tt := range tests {
		actual := GetSpellName(tt.spellID)

		if actual != tt.expected {
			t.Errorf(
				"spell ID %d: expected %q, got %q",
				tt.spellID,
				tt.expected,
				actual,
			)
		}
	}
}

func TestGetSpellCooldown(t *testing.T) {
	tests := []struct {
		spellID  int
		expected int
	}{
		{spellID: 4, expected: 300},
		{spellID: 14, expected: 180},
		{spellID: 11, expected: 15},
		{spellID: 999, expected: 300},
	}

	for _, tt := range tests {
		actual := GetSpellCooldown(tt.spellID)

		if actual != tt.expected {
			t.Errorf(
				"spell ID %d: expected %d, got %d",
				tt.spellID,
				tt.expected,
				actual,
			)
		}
	}
}

func TestTogglePlayerSpellStartsCooldown(t *testing.T) {
	player := models.Player{
		SummonerSpellHaste: 0,
		Spells: []models.SummonerSpell{
			{
				Name:         "Flash",
				IsReady:      true,
				BaseCooldown: 300,
			},
		},
	}

	beforeToggle := time.Now()

	ok := TogglePlayerSpell(&player, "Flash")
	if !ok {
		t.Fatal("expected spell toggle to succeed")
	}

	spell := player.FindSpell("Flash")
	if spell == nil {
		t.Fatal("expected Flash to exist")
	}

	if spell.IsReady {
		t.Error("expected Flash to be on cooldown")
	}

	expectedEnd := beforeToggle.Add(300 * time.Second)

	if spell.CooldownEndTime.Before(expectedEnd.Add(-time.Second)) ||
		spell.CooldownEndTime.After(expectedEnd.Add(time.Second)) {
		t.Errorf(
			"unexpected cooldown end time: %v",
			spell.CooldownEndTime,
		)
	}
}

func TestTogglePlayerSpellResetsCooldown(t *testing.T) {
	player := models.Player{
		Spells: []models.SummonerSpell{
			{
				Name:              "Flash",
				IsReady:           false,
				BaseCooldown:      300,
				RemainingCooldown: 120,
				CooldownEndTime:   time.Now().Add(120 * time.Second),
			},
		},
	}

	ok := TogglePlayerSpell(&player, "Flash")
	if !ok {
		t.Fatal("expected spell toggle to succeed")
	}

	spell := player.FindSpell("Flash")

	if !spell.IsReady {
		t.Error("expected Flash to be ready")
	}

	if spell.RemainingCooldown != 0 {
		t.Errorf(
			"expected remaining cooldown 0, got %d",
			spell.RemainingCooldown,
		)
	}

	if !spell.CooldownEndTime.IsZero() {
		t.Error("expected cooldown end time to be reset")
	}
}

func TestTogglePlayerSpellReturnsFalseForUnknownSpell(t *testing.T) {
	player := models.Player{}

	ok := TogglePlayerSpell(&player, "Flash")

	if ok {
		t.Error("expected false for missing spell")
	}
}
func TestGetRoomReturnsIndependentSnapshot(t *testing.T) {
	repo := repositories.NewInMemoryRoomRepository()
	service := NewRoomService(repo)

	service.CreateRoom("room-1")

	service.ReplacePlayers("room-1", []models.Player{
		{
			GameName: "Player",
			TagLine:  "EUW",
			Spells: []models.SummonerSpell{
				{
					Name:    "Flash",
					IsReady: true,
				},
			},
		},
	})

	snapshot := service.GetRoomSnapshot("room-1")
	if snapshot == nil {
		t.Fatal("expected room snapshot")
	}

	snapshot.Id = "modified"
	snapshot.Players[0].GameName = "ModifiedPlayer"
	snapshot.Players[0].Spells[0].Name = "ModifiedSpell"

	actual := service.GetRoomSnapshot("room-1")
	if actual == nil {
		t.Fatal("expected room to exist")
	}

	if actual.Id != "room-1" {
		t.Errorf("internal room ID was modified: %q", actual.Id)
	}

	if actual.Players[0].GameName != "Player" {
		t.Errorf(
			"internal player was modified: %q",
			actual.Players[0].GameName,
		)
	}

	if actual.Players[0].Spells[0].Name != "Flash" {
		t.Errorf(
			"internal spell was modified: %q",
			actual.Players[0].Spells[0].Name,
		)
	}
}
