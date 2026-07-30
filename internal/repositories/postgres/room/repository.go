package room

import (
	"context"
	"fmt"
	"lol-timer/internal/database"
	"lol-timer/internal/models"
	"lol-timer/internal/repositories"
	"time"

	"github.com/jackc/pgx/v5"
)

type RoomRepository struct {
	db *database.Postgres
}

var _ repositories.RoomRepository = (*RoomRepository)(nil)

func NewRoomRepository(
	db *database.Postgres,
) *RoomRepository {
	return &RoomRepository{
		db: db,
	}
}

func (r *RoomRepository) Save(
	ctx context.Context,
	room *models.Room,
) error {
	if room == nil {
		return nil
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin room save transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := saveRoom(ctx, tx, room); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit room save transaction: %w",
			err,
		)
	}

	return nil
}

func saveRoom(
	ctx context.Context,
	tx pgx.Tx,
	room *models.Room,
) error {
	_, err := tx.Exec(
		ctx,
		`
		INSERT INTO rooms (
			id,
			last_updated
		)
		VALUES ($1, $2)
		ON CONFLICT (id)
		DO UPDATE SET
			last_updated = EXCLUDED.last_updated
		`,
		room.Id,
		room.LastUpdated,
	)
	if err != nil {
		return fmt.Errorf(
			"upsert room %q: %w",
			room.Id,
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`DELETE FROM players WHERE room_id = $1`,
		room.Id,
	)
	if err != nil {
		return fmt.Errorf(
			"delete old players for room %q: %w",
			room.Id,
			err,
		)
	}

	for i := range room.Players {
		player := &room.Players[i]

		if err := savePlayer(
			ctx,
			tx,
			room.Id,
			player,
		); err != nil {
			return err
		}
	}

	return nil
}

func savePlayer(
	ctx context.Context,
	tx pgx.Tx,
	roomId string,
	player *models.Player,
) error {
	var playerId int64

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO players (
			room_id,
			game_name,
			tag_line,
			champion,
			champion_id,
			summoner_spell_haste
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
		`,
		roomId,
		player.GameName,
		player.TagLine,
		player.Champion,
		player.ChampionId,
		player.SummonerSpellHaste,
	).Scan(&playerId)
	if err != nil {
		return fmt.Errorf(
			"insert player %q#%q: %w",
			player.GameName,
			player.TagLine,
			err,
		)
	}

	for i := range player.Spells {
		spell := &player.Spells[i]

		_, err := tx.Exec(
			ctx,
			`
			INSERT INTO summoner_spells (
				player_id,
				name,
				is_ready,
				base_cooldown,
				remaining_cooldown,
				cooldown_end_time
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			`,
			playerId,
			spell.Name,
			spell.IsReady,
			spell.BaseCooldown,
			spell.RemainingCooldown,
			nullableTime(spell.CooldownEndTime),
		)
		if err != nil {
			return fmt.Errorf(
				"insert spell %q for player %q#%q: %w",
				spell.Name,
				player.GameName,
				player.TagLine,
				err,
			)
		}
	}

	return nil
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}

	return value
}

func (r *RoomRepository) Get(
	ctx context.Context,
	id string,
) (*models.Room, bool, error) {
	if id == "" {
		return nil, false, nil
	}

	return loadRoom(ctx, r.db.Pool, id)
}

func (r *RoomRepository) GetAll(
	ctx context.Context,
) ([]*models.Room, error) {
	rows, err := r.db.Pool.Query(
		ctx,
		`
		SELECT id
		FROM rooms
		ORDER BY last_updated DESC
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"select room ids: %w",
			err,
		)
	}
	defer rows.Close()

	rooms := make([]*models.Room, 0)

	for rows.Next() {
		var roomID string

		if err := rows.Scan(&roomID); err != nil {
			return nil, fmt.Errorf(
				"scan room id: %w",
				err,
			)
		}

		room, exists, err := loadRoom(
			ctx,
			r.db.Pool,
			roomID,
		)
		if err != nil {
			return nil, err
		}

		if exists {
			rooms = append(rooms, room)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate rooms: %w",
			err,
		)
	}

	return rooms, nil
}

func (r *RoomRepository) Delete(
	ctx context.Context,
	id string,
) error {
	_, err := r.db.Pool.Exec(
		ctx,
		`
		DELETE FROM rooms
		WHERE id = $1
		`,
		id,
	)
	if err != nil {
		return fmt.Errorf(
			"delete room %q: %w",
			id,
			err,
		)
	}

	return nil
}
