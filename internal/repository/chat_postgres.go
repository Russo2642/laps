package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"laps/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepositoryImpl struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) *ChatRepositoryImpl {
	return &ChatRepositoryImpl{db: db}
}

// Chat Sessions

func (r *ChatRepositoryImpl) CreateChatSession(ctx context.Context, dto domain.CreateChatSessionDTO) (*domain.ChatSession, error) {
	query := `
		INSERT INTO chat_sessions (appointment_id, client_id, specialist_id, specialization_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, appointment_id, client_id, specialist_id, specialization_id, status, started_at, ended_at, created_at, updated_at`

	status := dto.Status
	if status == "" {
		status = domain.ChatSessionStatusPending
	}

	var session domain.ChatSession
	err := r.db.QueryRow(ctx, query, dto.AppointmentID, dto.ClientID, dto.SpecialistID, dto.SpecializationID, status).Scan(
		&session.ID,
		&session.AppointmentID,
		&session.ClientID,
		&session.SpecialistID,
		&session.SpecializationID,
		&session.Status,
		&session.StartedAt,
		&session.EndedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	return &session, err
}

func (r *ChatRepositoryImpl) GetChatSessionByID(ctx context.Context, id int64) (*domain.ChatSession, error) {
	query := `
		SELECT 
			cs.id, cs.appointment_id, cs.client_id, cs.specialist_id, cs.specialization_id, 
			cs.status, cs.started_at, cs.ended_at, cs.created_at, cs.updated_at,
			CONCAT(uc.first_name, ' ', uc.last_name) as client_name, uc.phone as client_phone,
			CONCAT(us.first_name, ' ', us.last_name) as specialist_name, us.phone as specialist_phone,
			sp.name as specialization_name
		FROM chat_sessions cs
		LEFT JOIN users uc ON cs.client_id = uc.id
		LEFT JOIN specialists s ON cs.specialist_id = s.id
		LEFT JOIN users us ON s.user_id = us.id
		LEFT JOIN specializations sp ON cs.specialization_id = sp.id
		WHERE cs.id = $1`

	var session domain.ChatSession
	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.AppointmentID,
		&session.ClientID,
		&session.SpecialistID,
		&session.SpecializationID,
		&session.Status,
		&session.StartedAt,
		&session.EndedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.ClientName,
		&session.ClientPhone,
		&session.SpecialistName,
		&session.SpecialistPhone,
		&session.SpecializationName,
	)

	return &session, err
}

func (r *ChatRepositoryImpl) GetChatSessionByAppointmentID(ctx context.Context, appointmentID int64) (*domain.ChatSession, error) {
	query := `
		SELECT 
			cs.id, cs.appointment_id, cs.client_id, cs.specialist_id, cs.specialization_id, 
			cs.status, cs.started_at, cs.ended_at, cs.created_at, cs.updated_at,
			CONCAT(uc.first_name, ' ', uc.last_name) as client_name, uc.phone as client_phone,
			CONCAT(us.first_name, ' ', us.last_name) as specialist_name, us.phone as specialist_phone,
			sp.name as specialization_name
		FROM chat_sessions cs
		LEFT JOIN users uc ON cs.client_id = uc.id
		LEFT JOIN specialists s ON cs.specialist_id = s.id
		LEFT JOIN users us ON s.user_id = us.id
		LEFT JOIN specializations sp ON cs.specialization_id = sp.id
		WHERE cs.appointment_id = $1`

	var session domain.ChatSession
	err := r.db.QueryRow(ctx, query, appointmentID).Scan(
		&session.ID,
		&session.AppointmentID,
		&session.ClientID,
		&session.SpecialistID,
		&session.SpecializationID,
		&session.Status,
		&session.StartedAt,
		&session.EndedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.ClientName,
		&session.ClientPhone,
		&session.SpecialistName,
		&session.SpecialistPhone,
		&session.SpecializationName,
	)

	return &session, err
}

func (r *ChatRepositoryImpl) ListChatSessions(ctx context.Context, filter domain.ChatSessionFilter) ([]domain.ChatSession, error) {
	var conditions []string
	var args []interface{}
	argCount := 1

	baseQuery := `
		SELECT 
			cs.id, cs.appointment_id, cs.client_id, cs.specialist_id, cs.specialization_id, 
			cs.status, cs.started_at, cs.ended_at, cs.created_at, cs.updated_at,
			CONCAT(uc.first_name, ' ', uc.last_name) as client_name, uc.phone as client_phone,
			CONCAT(us.first_name, ' ', us.last_name) as specialist_name, us.phone as specialist_phone,
			sp.name as specialization_name
		FROM chat_sessions cs
		LEFT JOIN users uc ON cs.client_id = uc.id
		LEFT JOIN specialists s ON cs.specialist_id = s.id
		LEFT JOIN users us ON s.user_id = us.id
		LEFT JOIN specializations sp ON cs.specialization_id = sp.id`

	if filter.ClientID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.client_id = $%d", argCount))
		args = append(args, *filter.ClientID)
		argCount++
	}

	if filter.SpecialistID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.specialist_id = $%d", argCount))
		args = append(args, *filter.SpecialistID)
		argCount++
	}

	if filter.SpecializationID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.specialization_id = $%d", argCount))
		args = append(args, *filter.SpecializationID)
		argCount++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("cs.status = $%d", argCount))
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.AppointmentID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.appointment_id = $%d", argCount))
		args = append(args, *filter.AppointmentID)
		argCount++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY cs.created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
		argCount++
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.ChatSession
	for rows.Next() {
		var session domain.ChatSession
		err := rows.Scan(
			&session.ID,
			&session.AppointmentID,
			&session.ClientID,
			&session.SpecialistID,
			&session.SpecializationID,
			&session.Status,
			&session.StartedAt,
			&session.EndedAt,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.ClientName,
			&session.ClientPhone,
			&session.SpecialistName,
			&session.SpecialistPhone,
			&session.SpecializationName,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (r *ChatRepositoryImpl) CountChatSessions(ctx context.Context, filter domain.ChatSessionFilter) (int64, error) {
	var conditions []string
	var args []interface{}
	argCount := 1

	baseQuery := "SELECT COUNT(*) FROM chat_sessions cs"

	if filter.ClientID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.client_id = $%d", argCount))
		args = append(args, *filter.ClientID)
		argCount++
	}

	if filter.SpecialistID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.specialist_id = $%d", argCount))
		args = append(args, *filter.SpecialistID)
		argCount++
	}

	if filter.SpecializationID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.specialization_id = $%d", argCount))
		args = append(args, *filter.SpecializationID)
		argCount++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("cs.status = $%d", argCount))
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.AppointmentID != nil {
		conditions = append(conditions, fmt.Sprintf("cs.appointment_id = $%d", argCount))
		args = append(args, *filter.AppointmentID)
		argCount++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int64
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *ChatRepositoryImpl) UpdateChatSession(ctx context.Context, id int64, dto domain.UpdateChatSessionDTO) (*domain.ChatSession, error) {
	var setParts []string
	var args []interface{}
	argCount := 1

	if dto.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *dto.Status)
		argCount++
	}

	if dto.StartedAt != nil {
		setParts = append(setParts, fmt.Sprintf("started_at = $%d", argCount))
		args = append(args, *dto.StartedAt)
		argCount++
	}

	if dto.EndedAt != nil {
		setParts = append(setParts, fmt.Sprintf("ended_at = $%d", argCount))
		args = append(args, *dto.EndedAt)
		argCount++
	}

	if len(setParts) == 0 {
		return r.GetChatSessionByID(ctx, id)
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE chat_sessions 
		SET %s
		WHERE id = $%d
		RETURNING id, appointment_id, client_id, specialist_id, specialization_id, status, started_at, ended_at, created_at, updated_at`,
		strings.Join(setParts, ", "), argCount)

	var session domain.ChatSession
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&session.ID,
		&session.AppointmentID,
		&session.ClientID,
		&session.SpecialistID,
		&session.SpecializationID,
		&session.Status,
		&session.StartedAt,
		&session.EndedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	return &session, err
}

// Chat Messages

func (r *ChatRepositoryImpl) CreateChatMessage(ctx context.Context, dto domain.CreateChatMessageDTO) (*domain.ChatMessage, error) {
	query := `
		INSERT INTO chat_messages (session_id, sender_id, message_type, content, file_url, file_name, file_size)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, session_id, sender_id, message_type, content, file_url, file_name, file_size, is_read, read_at, created_at, updated_at`

	var message domain.ChatMessage
	err := r.db.QueryRow(ctx, query, dto.SessionID, dto.SenderID, dto.Type, dto.Content, dto.FileURL, dto.FileName, dto.FileSize).Scan(
		&message.ID,
		&message.SessionID,
		&message.SenderID,
		&message.Type,
		&message.Content,
		&message.FileURL,
		&message.FileName,
		&message.FileSize,
		&message.IsRead,
		&message.ReadAt,
		&message.CreatedAt,
		&message.UpdatedAt,
	)

	return &message, err
}

func (r *ChatRepositoryImpl) ListChatMessages(ctx context.Context, filter domain.ChatMessageFilter) ([]domain.ChatMessage, error) {
	var conditions []string
	var args []interface{}
	argCount := 1

	baseQuery := `
		SELECT 
			cm.id, cm.session_id, cm.sender_id, cm.message_type, cm.content, 
		       cm.file_url, cm.file_name, cm.file_size, cm.is_read, cm.read_at, 
		       cm.created_at, cm.updated_at,
			CONCAT(u.first_name, ' ', u.last_name) as sender_name,
			CASE 
				WHEN cs.client_id = cm.sender_id THEN 'client'
				WHEN cs.specialist_id = cm.sender_id THEN 'specialist'
				ELSE 'system'
			END as sender_role
		FROM chat_messages cm
		LEFT JOIN users u ON cm.sender_id = u.id
		LEFT JOIN chat_sessions cs ON cm.session_id = cs.id`

	if filter.SessionID != nil {
		conditions = append(conditions, fmt.Sprintf("cm.session_id = $%d", argCount))
		args = append(args, *filter.SessionID)
		argCount++
	}

	if filter.SenderID != nil {
		conditions = append(conditions, fmt.Sprintf("cm.sender_id = $%d", argCount))
		args = append(args, *filter.SenderID)
		argCount++
	}

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("cm.message_type = $%d", argCount))
		args = append(args, *filter.Type)
		argCount++
	}

	if filter.IsRead != nil {
		conditions = append(conditions, fmt.Sprintf("cm.is_read = $%d", argCount))
		args = append(args, *filter.IsRead)
		argCount++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY cm.created_at ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
		argCount++
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.ChatMessage
	for rows.Next() {
		var message domain.ChatMessage
		err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.SenderID,
			&message.Type,
			&message.Content,
			&message.FileURL,
			&message.FileName,
			&message.FileSize,
			&message.IsRead,
			&message.ReadAt,
			&message.CreatedAt,
			&message.UpdatedAt,
			&message.SenderName,
			&message.SenderRole,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (r *ChatRepositoryImpl) CountChatMessages(ctx context.Context, filter domain.ChatMessageFilter) (int64, error) {
	var conditions []string
	var args []interface{}
	argCount := 1

	baseQuery := "SELECT COUNT(*) FROM chat_messages cm"

	if filter.SessionID != nil {
		conditions = append(conditions, fmt.Sprintf("cm.session_id = $%d", argCount))
		args = append(args, *filter.SessionID)
		argCount++
	}

	if filter.SenderID != nil {
		conditions = append(conditions, fmt.Sprintf("cm.sender_id = $%d", argCount))
		args = append(args, *filter.SenderID)
		argCount++
	}

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("cm.message_type = $%d", argCount))
		args = append(args, *filter.Type)
		argCount++
	}

	if filter.IsRead != nil {
		conditions = append(conditions, fmt.Sprintf("cm.is_read = $%d", argCount))
		args = append(args, *filter.IsRead)
		argCount++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int64
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *ChatRepositoryImpl) MarkMessagesAsRead(ctx context.Context, sessionID int64, userID int64) error {
	query := `
		UPDATE chat_messages 
		SET is_read = true, read_at = NOW(), updated_at = NOW()
		WHERE session_id = $1 AND sender_id != $2 AND is_read = false`

	_, err := r.db.Exec(ctx, query, sessionID, userID)
	return err
}

func (r *ChatRepositoryImpl) GetUnreadMessageCount(ctx context.Context, sessionID int64, userID int64) (int64, error) {
	query := `
		SELECT COUNT(*) 
		FROM chat_messages 
		WHERE session_id = $1 AND sender_id != $2 AND is_read = false`

	var count int64
	err := r.db.QueryRow(ctx, query, sessionID, userID).Scan(&count)
	return count, err
} 