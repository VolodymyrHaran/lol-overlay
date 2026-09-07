package services

import (
	"context"
	"testing"
)

func TestChampionServiceFind(
	t *testing.T,
) {
	service := newChampionCatalogTestService()

	champion, exists := service.Find(103)
	if !exists {
		t.Fatal("expected champion to exist")
	}

	if champion.ID != 103 {
		t.Errorf(
			"expected champion ID 103, got %d",
			champion.ID,
		)
	}

	if champion.Name != "Ahri" {
		t.Errorf(
			"expected champion name %q, got %q",
			"Ahri",
			champion.Name,
		)
	}

	_, exists = service.Find(999)
	if exists {
		t.Fatal(
			"expected unknown champion not to exist",
		)
	}
}

func TestChampionServiceGetReturnsUnknown(
	t *testing.T,
) {
	service := newChampionCatalogTestService()

	champion, err := service.Get(
		context.Background(),
		999,
	)
	if err != nil {
		t.Fatalf(
			"get unknown champion: %v",
			err,
		)
	}

	if champion.ID != 999 {
		t.Errorf(
			"expected champion ID 999, got %d",
			champion.ID,
		)
	}

	if champion.Name != "Unknown" {
		t.Errorf(
			"expected unknown champion name, got %q",
			champion.Name,
		)
	}

	if champion.ImageURL != "" {
		t.Errorf(
			"expected empty image URL, got %q",
			champion.ImageURL,
		)
	}
}

func TestChampionServiceListReturnsSortedCopy(
	t *testing.T,
) {
	service := newChampionCatalogTestService()

	champions := service.List()

	if len(champions) != 2 {
		t.Fatalf(
			"expected two champions, got %d",
			len(champions),
		)
	}

	if champions[0].ID != 103 {
		t.Errorf(
			"expected first champion ID 103, got %d",
			champions[0].ID,
		)
	}

	if champions[1].ID != 222 {
		t.Errorf(
			"expected second champion ID 222, got %d",
			champions[1].ID,
		)
	}

	champions[0].Name = "Changed"

	stored, exists := service.Find(103)
	if !exists {
		t.Fatal("expected champion to exist")
	}

	if stored.Name != "Ahri" {
		t.Errorf(
			"expected stored champion not to change, got %q",
			stored.Name,
		)
	}
}

func TestChampionServiceVersion(
	t *testing.T,
) {
	service := newChampionCatalogTestService()

	if actual := service.Version(); actual != "16.18.1" {
		t.Errorf(
			"expected version %q, got %q",
			"16.18.1",
			actual,
		)
	}
}

func newChampionCatalogTestService() *ChampionService {
	return &ChampionService{
		champions: map[int]ChampionInfo{
			222: {
				ID:       222,
				Name:     "Jinx",
				ImageURL: "https://example.com/Jinx.png",
			},
			103: {
				ID:       103,
				Name:     "Ahri",
				ImageURL: "https://example.com/Ahri.png",
			},
		},
		version: "16.18.1",
	}
}
