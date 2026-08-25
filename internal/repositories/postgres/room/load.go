package room

import (
	"context"
	"errors"
	"fmt"
	"lol-timer/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func loadRoom(
	ctx context.Context,
	pool *pgxpool.Pool,
	roomID string,
) (*models.Room, bool, error) {
	var room models.Room

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			last_updated
		FROM rooms
		WHERE id = $1
		`,
		roomID,
	).Scan(
		&room.Id,
		&room.LastUpdated,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf(
			"select room %q: %w",
			roomID,
			err,
		)
	}

	players, playerIndexes, err := loadPlayers(
		ctx,
		pool,
		roomID,
	)
	if err != nil {
		return nil, false, err
	}

	if err := loadSpells(
		ctx,
		pool,
		roomID,
		players,
		playerIndexes,
	); err != nil {
		return nil, false, err
	}

	room.Players = players

	return &room, true, nil
}

func loadPlayers(
	ctx context.Context,
	pool *pgxpool.Pool,
	roomID string,
) ([]models.Player, map[int64]int, error) {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			id,
			game_name,
			tag_line,
			champion,
			champion_image,
			champion_id,
			summoner_spell_haste
		FROM players
		WHERE room_id = $1
		ORDER BY id
		`,
		roomID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"select players for room %q: %w",
			roomID,
			err,
		)
	}
	defer rows.Close()

	players := make([]models.Player, 0)
	playerIndexes := make(map[int64]int)

	for rows.Next() {
		var playerID int64
		var player models.Player

		if err := rows.Scan(
			&playerID,
			&player.GameName,
			&player.TagLine,
			&player.Champion,
			&player.ChampionImage,
			&player.ChampionId,
			&player.SummonerSpellHaste,
		); err != nil {
			return nil, nil, fmt.Errorf(
				"scan player for room %q: %w",
				roomID,
				err,
			)
		}

		player.Spells = make([]models.SummonerSpell, 0)

		playerIndexes[playerID] = len(players)
		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf(
			"iterate players for room %q: %w",
			roomID,
			err,
		)
	}

	return players, playerIndexes, nil
}

func loadSpells(
	ctx context.Context,
	pool *pgxpool.Pool,
	roomID string,
	players []models.Player,
	playerIndexes map[int64]int,
) error {
	rows, err := pool.Query(
		ctx,
		`
		SELECT
			s.player_id,
			s.name,
			s.is_ready,
			s.base_cooldown,
			s.remaining_cooldown,
			s.cooldown_end_time
		FROM summoner_spells AS s
		INNER JOIN players AS p
			ON p.id = s.player_id
		WHERE p.room_id = $1
		ORDER BY s.id
		`,
		roomID,
	)
	if err != nil {
		return fmt.Errorf(
			"select spells for room %q: %w",
			roomID,
			err,
		)
	}
	defer rows.Close()

	for rows.Next() {
		var playerID int64
		var cooldownEndTime *time.Time
		var spell models.SummonerSpell

		if err := rows.Scan(
			&playerID,
			&spell.Name,
			&spell.IsReady,
			&spell.BaseCooldown,
			&spell.RemainingCooldown,
			&cooldownEndTime,
		); err != nil {
			return fmt.Errorf(
				"scan spell for room %q: %w",
				roomID,
				err,
			)
		}

		if cooldownEndTime != nil {
			spell.CooldownEndTime = *cooldownEndTime
		}

		playerIndex, exists := playerIndexes[playerID]
		if !exists {
			return fmt.Errorf(
				"player %d not found while loading room %q",
				playerID,
				roomID,
			)
		}

		players[playerIndex].Spells = append(
			players[playerIndex].Spells,
			spell,
		)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf(
			"iterate spells for room %q: %w",
			roomID,
			err,
		)
	}

	return nil
}
