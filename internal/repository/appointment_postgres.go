package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"laps/internal/domain"
)

type AppointmentRepo struct {
	db *pgxpool.Pool
}

func NewAppointmentRepository(db *pgxpool.Pool) *AppointmentRepo {
	return &AppointmentRepo{
		db: db,
	}
}

func (r *AppointmentRepo) Create(ctx context.Context, clientID int64, dto domain.CreateAppointmentDTO) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	checkQuery := `
		SELECT COUNT(*) 
		FROM appointments 
		WHERE specialist_id = $1 
		AND appointment_date = $2
		AND status != 'cancelled'
	`

	var count int
	err = tx.QueryRow(ctx, checkQuery, dto.SpecialistID, dto.AppointmentDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка проверки доступности слота: %w", err)
	}

	if count > 0 {
		return 0, errors.New("выбранный слот времени уже занят")
	}

	var price float64
	priceQuery := `
		SELECT CASE 
			WHEN $1 = 'primary' THEN primary_consult_price 
			WHEN $1 = 'secondary' THEN secondary_consult_price 
			ELSE primary_consult_price 
		END 
		FROM specialists 
		WHERE id = $2
	`
	err = tx.QueryRow(ctx, priceQuery, dto.ConsultationType, dto.SpecialistID).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения цены консультации: %w", err)
	}

	if price <= 0 {
		return 0, fmt.Errorf("некорректная цена консультации: %f", price)
	}

	query := `
		INSERT INTO appointments (client_id, specialist_id, specialization_id, appointment_date, status, consultation_type, communication_method, price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id
	`

	now := time.Now()
	var id int64
	err = tx.QueryRow(ctx, query,
		clientID,
		dto.SpecialistID,
		dto.SpecializationID,
		dto.AppointmentDate,
		domain.AppointmentStatusPending,
		dto.ConsultationType,
		dto.CommunicationMethod,
		price,
		now,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("ошибка создания записи на прием: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return id, nil
}

func (r *AppointmentRepo) GetByID(ctx context.Context, id int64) (*domain.Appointment, error) {
	query := `
		SELECT a.id, a.client_id, a.specialist_id, a.specialization_id, a.price, a.appointment_date, a.status, a.consultation_type, a.communication_method, a.created_at, a.updated_at,
		       u.first_name AS user_first_name, u.last_name AS user_last_name,
		       s.type AS specialist_type,
		       su.first_name AS specialist_first_name, su.last_name AS specialist_last_name
		FROM appointments a
		JOIN users u ON a.client_id = u.id
		JOIN specialists s ON a.specialist_id = s.id
		JOIN users su ON s.user_id = su.id
		WHERE a.id = $1
	`

	var appointment domain.Appointment
	var userFirstName, userLastName, specialistFirstName, specialistLastName string
	var specialistType domain.SpecialistType

	err := r.db.QueryRow(ctx, query, id).Scan(
		&appointment.ID,
		&appointment.ClientID,
		&appointment.SpecialistID,
		&appointment.SpecializationID,
		&appointment.Price,
		&appointment.AppointmentDate,
		&appointment.Status,
		&appointment.ConsultationType,
		&appointment.CommunicationMethod,
		&appointment.CreatedAt,
		&appointment.UpdatedAt,
		&userFirstName,
		&userLastName,
		&specialistType,
		&specialistFirstName,
		&specialistLastName,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("запись на прием с ID %d не найдена: %w", id, errors.New("not found"))
		}
		return nil, fmt.Errorf("ошибка получения записи на прием: %w", err)
	}

	return &appointment, nil
}

func (r *AppointmentRepo) UpdateStatus(ctx context.Context, id int64, status domain.AppointmentStatus) error {
	query := `
		UPDATE appointments
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса записи: %w", err)
	}

	return nil
}

func (r *AppointmentRepo) Update(ctx context.Context, id int64, dto domain.UpdateAppointmentDTO) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	if dto.AppointmentDate != nil {
		var currentAppointmentDate time.Time
		var specialistID int64

		query := `SELECT specialist_id, appointment_date FROM appointments WHERE id = $1`
		err := tx.QueryRow(ctx, query, id).Scan(&specialistID, &currentAppointmentDate)
		if err != nil {
			return fmt.Errorf("ошибка получения текущих данных записи: %w", err)
		}

		checkQuery := `
			SELECT COUNT(*) 
			FROM appointments 
			WHERE specialist_id = $1 
			AND appointment_date = $2
			AND id != $3
			AND status != 'cancelled'
		`

		var count int
		err = tx.QueryRow(ctx, checkQuery, specialistID, dto.AppointmentDate, id).Scan(&count)
		if err != nil {
			return fmt.Errorf("ошибка проверки доступности слота: %w", err)
		}

		if count > 0 {
			return errors.New("выбранный слот времени уже занят")
		}
	}

	var updateFields []string
	var args []interface{}

	argCount := 1

	if dto.AppointmentDate != nil {
		updateFields = append(updateFields, fmt.Sprintf("appointment_date = $%d", argCount))
		args = append(args, dto.AppointmentDate)
		argCount++
	}

	if dto.Status != nil {
		updateFields = append(updateFields, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *dto.Status)
		argCount++
	}

	if dto.PaymentID != nil {
		updateFields = append(updateFields, fmt.Sprintf("payment_id = $%d", argCount))
		args = append(args, *dto.PaymentID)
		argCount++
	}

	updateFields = append(updateFields, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++

	if len(updateFields) == 1 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE appointments 
		SET %s 
		WHERE id = $%d
	`, strings.Join(updateFields, ", "), argCount)

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ошибка обновления записи на прием: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return nil
}

func (r *AppointmentRepo) Delete(ctx context.Context, id int64) error {
	return r.UpdateStatus(ctx, id, domain.AppointmentStatusCancelled)
}

func (r *AppointmentRepo) GetByUserID(ctx context.Context, userID int64, filter domain.AppointmentFilter) ([]domain.Appointment, error) {
	conditions := []string{"a.client_id = $1"}
	args := []interface{}{userID}
	argCount := 2

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argCount))
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.appointment_date >= $%d", argCount))
		args = append(args, filter.StartDate)
		argCount++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.appointment_date <= $%d", argCount))
		args = append(args, filter.EndDate)
		argCount++
	}

	if filter.SpecialistID != nil {
		conditions = append(conditions, fmt.Sprintf("a.specialist_id = $%d", argCount))
		args = append(args, *filter.SpecialistID)
		argCount++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	args = append(args, filter.Limit, filter.Offset)

	query := fmt.Sprintf(`
		SELECT a.id, a.client_id, a.specialist_id, a.specialization_id, a.appointment_date, a.status, a.consultation_type, a.communication_method, a.created_at, a.updated_at,
		       u.first_name AS user_first_name, u.last_name AS user_last_name,
		       s.type AS specialist_type,
		       su.first_name AS specialist_first_name, su.last_name AS specialist_last_name
		FROM appointments a
		JOIN users u ON a.client_id = u.id
		JOIN specialists s ON a.specialist_id = s.id
		JOIN users su ON s.user_id = su.id
		%s
		ORDER BY a.appointment_date DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	appointments := make([]domain.Appointment, 0)
	for rows.Next() {
		var appointment domain.Appointment
		var userFirstName, userLastName, specialistFirstName, specialistLastName string
		var specialistType domain.SpecialistType

		if err := rows.Scan(
			&appointment.ID,
			&appointment.ClientID,
			&appointment.SpecialistID,
			&appointment.SpecializationID,
			&appointment.AppointmentDate,
			&appointment.Status,
			&appointment.ConsultationType,
			&appointment.CommunicationMethod,
			&appointment.CreatedAt,
			&appointment.UpdatedAt,
			&userFirstName,
			&userLastName,
			&specialistType,
			&specialistFirstName,
			&specialistLastName,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки записи: %w", err)
		}

		appointments = append(appointments, appointment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return appointments, nil
}

func (r *AppointmentRepo) GetBySpecialistID(ctx context.Context, specialistID int64, filter domain.AppointmentFilter) ([]domain.Appointment, error) {
	conditions := []string{"a.specialist_id = $1"}
	args := []interface{}{specialistID}
	argCount := 2

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argCount))
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.appointment_date >= $%d", argCount))
		args = append(args, filter.StartDate)
		argCount++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.appointment_date <= $%d", argCount))
		args = append(args, filter.EndDate)
		argCount++
	}

	if filter.ClientID != nil {
		conditions = append(conditions, fmt.Sprintf("a.client_id = $%d", argCount))
		args = append(args, *filter.ClientID)
		argCount++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	args = append(args, filter.Limit, filter.Offset)

	query := fmt.Sprintf(`
		SELECT a.id, a.client_id, a.specialist_id, a.specialization_id, a.appointment_date, a.status, a.consultation_type, a.communication_method, a.created_at, a.updated_at,
		       u.first_name AS user_first_name, u.last_name AS user_last_name,
		       s.type AS specialist_type,
		       su.first_name AS specialist_first_name, su.last_name AS specialist_last_name
		FROM appointments a
		JOIN users u ON a.client_id = u.id
		JOIN specialists s ON a.specialist_id = s.id
		JOIN users su ON s.user_id = su.id
		%s
		ORDER BY a.appointment_date DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	appointments := make([]domain.Appointment, 0)
	for rows.Next() {
		var appointment domain.Appointment
		var userFirstName, userLastName, specialistFirstName, specialistLastName string
		var specialistType domain.SpecialistType

		if err := rows.Scan(
			&appointment.ID,
			&appointment.ClientID,
			&appointment.SpecialistID,
			&appointment.SpecializationID,
			&appointment.AppointmentDate,
			&appointment.Status,
			&appointment.ConsultationType,
			&appointment.CommunicationMethod,
			&appointment.CreatedAt,
			&appointment.UpdatedAt,
			&userFirstName,
			&userLastName,
			&specialistType,
			&specialistFirstName,
			&specialistLastName,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки записи: %w", err)
		}

		appointments = append(appointments, appointment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return appointments, nil
}

func (r *AppointmentRepo) GetFreeSlots(ctx context.Context, specialistID int64, date string) ([]string, error) {
	query := `
		SELECT TO_CHAR(appointment_date, 'HH24:MI') as time_slot
		FROM appointments 
		WHERE specialist_id = $1 
		AND DATE(appointment_date) = $2
		AND status != 'cancelled'
	`

	rows, err := r.db.Query(ctx, query, specialistID, date)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения занятых слотов: %w", err)
	}
	defer rows.Close()

	busySlots := make(map[string]bool)
	for rows.Next() {
		var slot string
		if err := rows.Scan(&slot); err != nil {
			return nil, fmt.Errorf("ошибка сканирования слотов: %w", err)
		}
		busySlots[slot] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке результатов: %w", err)
	}

	allSlots := []string{
		"09:00", "10:00", "11:00", "12:00", "13:00", "14:00", "15:00", "16:00", "17:00",
	}

	var freeSlots []string
	for _, slot := range allSlots {
		if !busySlots[slot] {
			freeSlots = append(freeSlots, slot)
		}
	}

	return freeSlots, nil
}

func (r *AppointmentRepo) CountByFilter(ctx context.Context, filter domain.AppointmentFilter) (int, error) {
	baseQuery := `
		SELECT COUNT(*)
		FROM appointments
	`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.ClientID != nil {
		conditions = append(conditions, fmt.Sprintf("client_id = $%d", argCount))
		args = append(args, *filter.ClientID)
		argCount++
	}

	if filter.SpecialistID != nil {
		conditions = append(conditions, fmt.Sprintf("specialist_id = $%d", argCount))
		args = append(args, *filter.SpecialistID)
		argCount++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.ExcludeStatus != nil {
		conditions = append(conditions, fmt.Sprintf("status != $%d", argCount))
		args = append(args, *filter.ExcludeStatus)
		argCount++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("appointment_date >= $%d", argCount))
		args = append(args, filter.StartDate)
		argCount++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("appointment_date <= $%d", argCount))
		args = append(args, filter.EndDate)
		argCount++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчета записей: %w", err)
	}

	return count, nil
}

func (r *AppointmentRepo) List(ctx context.Context, filter domain.AppointmentFilter) ([]domain.Appointment, error) {
	baseQuery := `
		SELECT a.id, a.client_id, a.specialist_id, a.specialization_id, a.price, a.appointment_date, a.status, a.consultation_type, a.communication_method, a.created_at, a.updated_at,
		       u.first_name AS user_first_name, u.last_name AS user_last_name,
		       s.type AS specialist_type,
		       su.first_name AS specialist_first_name, su.last_name AS specialist_last_name
		FROM appointments a
		JOIN users u ON a.client_id = u.id
		JOIN specialists s ON a.specialist_id = s.id
		JOIN users su ON s.user_id = su.id
	`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.SpecialistID != nil {
		conditions = append(conditions, fmt.Sprintf("a.specialist_id = $%d", argCount))
		args = append(args, *filter.SpecialistID)
		argCount++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argCount))
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.ExcludeStatus != nil {
		conditions = append(conditions, fmt.Sprintf("a.status != $%d", argCount))
		args = append(args, *filter.ExcludeStatus)
		argCount++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.appointment_date >= $%d", argCount))
		args = append(args, filter.StartDate)
		argCount++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("a.appointment_date <= $%d", argCount))
		args = append(args, filter.EndDate)
		argCount++
	}

	if filter.ClientID != nil {
		conditions = append(conditions, fmt.Sprintf("a.client_id = $%d", argCount))
		args = append(args, *filter.ClientID)
		argCount++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY a.appointment_date DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполnения запроса: %w", err)
	}
	defer rows.Close()

	var appointments []domain.Appointment
	for rows.Next() {
		var appointment domain.Appointment
		var userFirstName, userLastName, specialistFirstName, specialistLastName string
		var specialistType domain.SpecialistType

		if err := rows.Scan(
			&appointment.ID,
			&appointment.ClientID,
			&appointment.SpecialistID,
			&appointment.SpecializationID,
			&appointment.Price,
			&appointment.AppointmentDate,
			&appointment.Status,
			&appointment.ConsultationType,
			&appointment.CommunicationMethod,
			&appointment.CreatedAt,
			&appointment.UpdatedAt,
			&userFirstName,
			&userLastName,
			&specialistType,
			&specialistFirstName,
			&specialistLastName,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования результатов: %w", err)
		}

		appointments = append(appointments, appointment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке результатов: %w", err)
	}

	return appointments, nil
}
