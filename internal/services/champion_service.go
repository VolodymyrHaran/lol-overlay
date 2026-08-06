package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	dataDragonVersionsURL = "https://ddragon.leagueoflegends.com/api/versions.json"
	dataDragonCDNURL      = "https://ddragon.leagueoflegends.com/cdn"
)

type ChampionInfo struct {
	ID       int
	Name     string
	ImageURL string
}

type ChampionService struct {
	mu        sync.RWMutex
	champions map[int]ChampionInfo
	client    *http.Client
	version   string
}

type dataDragonChampionResponse struct {
	Data map[string]dataDragonChampion `json:"data"`
}

type dataDragonChampion struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Image struct {
		Full string `json:"full"`
	} `json:"image"`
}

func NewChampionService() *ChampionService {
	return &ChampionService{
		champions: make(map[int]ChampionInfo),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *ChampionService) Load(
	ctx context.Context,
) error {
	version, err := s.loadLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf(
			"load Data Dragon version: %w",
			err,
		)
	}

	champions, err := s.loadChampions(ctx, version)
	if err != nil {
		return fmt.Errorf(
			"load Data Dragon champions: %w",
			err,
		)
	}

	s.mu.Lock()
	s.champions = champions
	s.version = version
	s.mu.Unlock()

	return nil
}

func (s *ChampionService) Get(
	championID int,
) ChampionInfo {
	s.mu.RLock()
	champion, exists := s.champions[championID]
	s.mu.RUnlock()

	if exists {
		return champion
	}

	return ChampionInfo{
		ID:   championID,
		Name: "Unknown",
	}
}

func (s *ChampionService) Version() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.version
}

func (s *ChampionService) loadLatestVersion(
	ctx context.Context,
) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		dataDragonVersionsURL,
		nil,
	)
	if err != nil {
		return "", err
	}

	response, err := s.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"unexpected status: %s",
			response.Status,
		)
	}

	var versions []string

	if err := json.NewDecoder(response.Body).
		Decode(&versions); err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf(
			"Data Dragon returned no versions",
		)
	}

	return versions[0], nil
}

func (s *ChampionService) loadChampions(
	ctx context.Context,
	version string,
) (map[int]ChampionInfo, error) {
	url := fmt.Sprintf(
		"%s/%s/data/en_US/champion.json",
		dataDragonCDNURL,
		version,
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected status: %s",
			response.Status,
		)
	}

	var payload dataDragonChampionResponse

	if err := json.NewDecoder(response.Body).
		Decode(&payload); err != nil {
		return nil, err
	}

	champions := make(
		map[int]ChampionInfo,
		len(payload.Data),
	)

	for _, champion := range payload.Data {
		championID, err := strconv.Atoi(champion.Key)
		if err != nil {
			continue
		}

		champions[championID] = ChampionInfo{
			ID:   championID,
			Name: champion.Name,
			ImageURL: fmt.Sprintf(
				"%s/%s/img/champion/%s",
				dataDragonCDNURL,
				version,
				champion.Image.Full,
			),
		}
	}

	return champions, nil
}
