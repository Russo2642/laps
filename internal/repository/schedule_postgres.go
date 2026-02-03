package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"laps/internal/domain"
)

type ScheduleRepo struct {
	db *pgxpool.Pool
}

func NewScheduleRepository(db *pgxpool.Pool) ScheduleRepository {
	return &ScheduleRepo{db: db}
}

func (r *ScheduleRepo) Create(ctx context.Context, schedule domain.Schedule) (int64, error) {
	var id int64

	query := `
		INSERT INTO schedules (
			specialist_id, day_of_week, start_time, end_time, slot_time, exclude_times, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := r.db.QueryRow(
		ctx,
		query,
		schedule.SpecialistID,
		schedule.DayOfWeek,
		schedule.StartTime,
		schedule.EndTime,
		schedule.SlotTime,
		schedule.ExcludeTimes,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("ошибка создания расписания: %w", err)
	}

	return id, nil
}

func (r *ScheduleRepo) GetByID(ctx context.Context, id int64) (*domain.Schedule, error) {
	query := `
		SELECT id, specialist_id, day_of_week, start_time, end_time, slot_time, exclude_times, created_at, updated_at
		FROM schedules
		WHERE id = $1
	`

	var schedule domain.Schedule
	err := r.db.QueryRow(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.SpecialistID,
		&schedule.DayOfWeek,
		&schedule.StartTime,
		&schedule.EndTime,
		&schedule.SlotTime,
		&schedule.ExcludeTimes,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения расписания: %w", err)
	}

	return &schedule, nil
}

func (r *ScheduleRepo) Update(ctx context.Context, schedule domain.Schedule) error {
	query := `
		UPDATE schedules
		SET start_time = $1, end_time = $2, slot_time = $3, exclude_times = $4, updated_at = $5
		WHERE id = $6
	`

	_, err := r.db.Exec(
		ctx,
		query,
		schedule.StartTime,
		schedule.EndTime,
		schedule.SlotTime,
		schedule.ExcludeTimes,
		schedule.UpdatedAt,
		schedule.ID,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления расписания: %w", err)
	}

	return nil
}

func (r *ScheduleRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM schedules WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления расписания: %w", err)
	}

	return nil
}

func (r *ScheduleRepo) List(ctx context.Context, filter domain.ScheduleFilter) ([]domain.Schedule, int, error) {
	countQuery := `SELECT COUNT(*) FROM schedules WHERE 1=1`
	selectQuery := `
		SELECT id, specialist_id, day_of_week, start_time, end_time, slot_time, exclude_times, created_at, updated_at
		FROM schedules
		WHERE 1=1
	`

	var conditions string
	var args []interface{}
	argPos := 1

	if filter.SpecialistID != nil {
		conditions += fmt.Sprintf(" AND specialist_id = $%d", argPos)
		args = append(args, *filter.SpecialistID)
		argPos++
	}

	if filter.DayOfWeek != nil {
		conditions += fmt.Sprintf(" AND day_of_week = $%d", argPos)
		args = append(args, *filter.DayOfWeek)
		argPos++
	}

	countQuery += conditions
	selectQuery += conditions

	selectQuery += fmt.Sprintf(" ORDER BY day_of_week, start_time LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, filter.Limit, filter.Offset)

	var total int
	countArgs := args
	if argPos > 1 {
		countArgs = args[:argPos-1]
	} else {
		countArgs = []interface{}{}
	}

	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("ошибка получения количества расписаний: %w", err)
	}

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ошибка получения списка расписаний: %w", err)
	}
	defer rows.Close()

	var schedules []domain.Schedule
	for rows.Next() {
		var schedule domain.Schedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.SpecialistID,
			&schedule.DayOfWeek,
			&schedule.StartTime,
			&schedule.EndTime,
			&schedule.SlotTime,
			&schedule.ExcludeTimes,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("ошибка сканирования строки расписания: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, total, nil
}

func (r *ScheduleRepo) GetBySpecialistAndDate(ctx context.Context, specialistID int64, date time.Time) (*domain.Schedule, error) {
	dayOfWeek := int(date.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7
	}

	query := `
		SELECT id, specialist_id, day_of_week, start_time, end_time, slot_time, exclude_times, created_at, updated_at
		FROM schedules
		WHERE specialist_id = $1 AND day_of_week = $2
		LIMIT 1
	`

	var schedule domain.Schedule
	err := r.db.QueryRow(ctx, query, specialistID, dayOfWeek).Scan(
		&schedule.ID,
		&schedule.SpecialistID,
		&schedule.DayOfWeek,
		&schedule.StartTime,
		&schedule.EndTime,
		&schedule.SlotTime,
		&schedule.ExcludeTimes,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения расписания: %w", err)
	}

	return &schedule, nil
}
