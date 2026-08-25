package services

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"lol-timer/internal/models"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LolClientService struct {
	leaguePath string
}

type LcuCredentials struct {
	ProcessName string
	Pid         string
	Port        string
	Password    string
	Protocol    string
}

func NewLolClientService() *LolClientService {
	return &LolClientService{}
}

func (s *LolClientService) FindLockfile() (string, bool) {
	possiblePaths := []string{
		`C:\Riot Games\League of Legends`,
		`E:\Games\Riot Games\League of Legends`,
	}

	for _, path := range possiblePaths {
		lockfilePath := filepath.Join(path, "lockfile")

		if _, err := os.Stat(lockfilePath); err == nil {
			s.leaguePath = path
			return lockfilePath, true
		}
	}

	return "", false
}

func (s *LolClientService) ReadCredentials() (*LcuCredentials, bool) {
	lockfilePath, found := s.FindLockfile()
	if !found {
		return nil, false
	}

	content, err := os.ReadFile(lockfilePath)
	if err != nil {
		return nil, false
	}

	parts := strings.Split(string(content), ":")
	if len(parts) != 5 {
		return nil, false
	}

	return &LcuCredentials{
		ProcessName: parts[0],
		Pid:         parts[1],
		Port:        parts[2],
		Password:    parts[3],
		Protocol:    parts[4],
	}, true
}

func (s *LolClientService) GetCurrentSummoner() (*models.CurrentSummoner, error) {
	body, err := s.get("/lol-summoner/v1/current-summoner")
	if err != nil {
		return nil, err
	}

	var summoner models.CurrentSummoner

	err = json.Unmarshal(body, &summoner)
	if err != nil {
		return nil, err
	}

	return &summoner, nil
}

func (s *LolClientService) GetGameflowPhase() (string, error) {
	body, err := s.get("/lol-gameflow/v1/gameflow-phase")
	if err != nil {
		return "", err
	}

	var phase string

	err = json.Unmarshal(body, &phase)
	if err != nil {
		return "", err
	}

	return phase, nil
}

func (s *LolClientService) get(path string) ([]byte, error) {
	credentials, ok := s.ReadCredentials()
	if !ok {
		return nil, fmt.Errorf("LCU credentials not found")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	url := fmt.Sprintf(
		"https://127.0.0.1:%s%s",
		credentials.Port,
		path,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString(
		[]byte("riot:" + credentials.Password),
	)

	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (s *LolClientService) GetChampSelectSession() (*models.ChampSelectSession, error) {
	body, err := s.get("/lol-champ-select/v1/session")
	if err != nil {
		return nil, err
	}

	var session models.ChampSelectSession

	err = json.Unmarshal(body, &session)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *LolClientService) GetLocalPlayerTeam(session *models.ChampSelectSession) int {
	for _, player := range session.MyTeam {
		if player.CellId == session.LocalPlayerCellId {
			return player.Team
		}
	}

	return 0
}

func (s *LolClientService) StartChampSelectSync(
	ctx context.Context,
	roomService *RoomService,
) {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				phase, err := s.GetGameflowPhase()
				if err != nil {
					continue
				}

				if phase != "ChampSelect" {
					roomID := roomService.GetCurrentRoomID()
					if roomID != "" {
						roomService.ClearCurrentRoomID(roomID)
					}
					continue
				}

				session, err := s.GetChampSelectSession()
				if err != nil {
					continue
				}

				roomID := BuildRoomId(session)
				if roomID == "" {
					continue
				}

				if _, err := roomService.CreateRoom(
					ctx,
					roomID,
				); err != nil {
					log.Printf(
						"create room from champ select: %v",
						err,
					)
					continue
				}

				if _, err := roomService.SyncFromChampSelect(
					ctx,
					roomID,
					session,
				); err != nil {
					log.Printf(
						"sync champ select: %v",
						err,
					)
				}
			}
		}
	}()
}
