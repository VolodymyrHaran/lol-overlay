package services

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

func (s *LolClientService) GetChampionName(championId int) string {
	switch championId {
	case 103:
		return "Ahri"
	case 222:
		return "Jinx"
	case 64:
		return "Lee Sin"
	default:
		return "Unknown"
	}
}

func (s *LolClientService) StartChampSelectSync(
	roomService *RoomService,
) {
	go func() {
		for {
			phase, err := s.GetGameflowPhase()
			if err != nil {
				fmt.Println("LCU phase error:", err)
				time.Sleep(2 * time.Second)
				continue
			}

			fmt.Println("Current phase:", phase)

			if phase == "ChampSelect" {
				session, err := s.GetChampSelectSession()
				if err != nil {
					fmt.Println("ChampSelect session error:", err)
					time.Sleep(2 * time.Second)
					continue
				}

				roomId := BuildRoomId(session)
				fmt.Println("RoomId:", roomId)

				if roomId != "" {
					roomService.CreateRoom(roomId)
					roomService.SyncFromChampSelect(roomId, session)

					fmt.Println("Synced players:", len(session.MyTeam))
				}
			}

			time.Sleep(2 * time.Second)
		}
	}()
}
