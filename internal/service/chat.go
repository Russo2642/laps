package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"laps/internal/domain"
	"laps/internal/repository"
)

type ChatServiceImpl struct {
	chatRepo        repository.ChatRepository
	appointmentRepo repository.AppointmentRepository
	userRepo        repository.UserRepository
	specialistRepo  repository.SpecialistRepository
}

func NewChatService(repos *repository.Repositories) *ChatServiceImpl {
	return &ChatServiceImpl{
		chatRepo:        repos.Chat,
		appointmentRepo: repos.Appointment,
		userRepo:        repos.User,
		specialistRepo:  repos.Specialist,
	}
}

// Chat Sessions

func (s *ChatServiceImpl) CreateChatSession(ctx context.Context, dto domain.CreateChatSessionDTO) (*domain.ChatSession, error) {
	// Verify appointment exists and get specialization_id
	appointment, err := s.appointmentRepo.GetByID(ctx, dto.AppointmentID)
	if err != nil {
		return nil, fmt.Errorf("appointment not found: %w", err)
	}

	// Ensure the client and specialist IDs match the appointment
	if appointment.ClientID != dto.ClientID {
		return nil, errors.New("client ID does not match appointment")
	}
	if appointment.SpecialistID != dto.SpecialistID {
		return nil, errors.New("specialist ID does not match appointment")
	}

	// Set specialization_id from appointment if not provided
	if dto.SpecializationID == 0 {
		if appointment.SpecializationID != nil {
			dto.SpecializationID = *appointment.SpecializationID
		} else {
			// Fallback: get specialist's primary specialization
			specializations, err := s.specialistRepo.GetSpecializationsBySpecialistID(ctx, dto.SpecialistID)
			if err != nil || len(specializations) == 0 {
				return nil, errors.New("no specialization found for specialist")
			}
			dto.SpecializationID = specializations[0].ID
		}
	}

	// Check if chat session already exists for this appointment
	existingSession, err := s.chatRepo.GetChatSessionByAppointmentID(ctx, dto.AppointmentID)
	if err == nil {
		return existingSession, nil
	}

	// Create new chat session
	return s.chatRepo.CreateChatSession(ctx, dto)
}

func (s *ChatServiceImpl) GetChatSessionByID(ctx context.Context, id int64, userID int64) (*domain.ChatSession, error) {
	session, err := s.chatRepo.GetChatSessionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this chat session
	hasAccess := false
	
	// Check if user is the client
	if session.ClientID == userID {
		hasAccess = true
	} else {
		// Check if user is the specialist by looking up their specialist record
		specialist, err := s.specialistRepo.GetByUserID(ctx, userID)
		if err == nil && specialist.ID == session.SpecialistID {
			hasAccess = true
		}
	}
	
	if !hasAccess {
		return nil, errors.New("access denied to chat session")
	}

	return session, nil
}

func (s *ChatServiceImpl) GetChatSessionByAppointmentID(ctx context.Context, appointmentID int64, userID int64) (*domain.ChatSession, error) {
	session, err := s.chatRepo.GetChatSessionByAppointmentID(ctx, appointmentID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this chat session
	hasAccess := false
	
	// Check if user is the client
	if session.ClientID == userID {
		hasAccess = true
	} else {
		// Check if user is the specialist by looking up their specialist record
		specialist, err := s.specialistRepo.GetByUserID(ctx, userID)
		if err == nil && specialist.ID == session.SpecialistID {
			hasAccess = true
		}
	}
	
	if !hasAccess {
		return nil, errors.New("access denied to chat session")
	}

	return session, nil
}

func (s *ChatServiceImpl) ListChatSessions(ctx context.Context, userID int64, filter domain.ChatSessionFilter) ([]domain.ChatSession, int64, error) {
	// Get user to determine role
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("user not found: %w", err)
	}

	// Set appropriate filter based on user role
	if user.Role == domain.UserRoleClient {
		filter.ClientID = &userID
	} else if user.Role == domain.UserRoleSpecialist {
		// For specialists, we need to find their specialist_id, not use their user_id
		specialist, err := s.specialistRepo.GetByUserID(ctx, userID)
		if err != nil {
			return nil, 0, fmt.Errorf("specialist not found for user_id %d: %w", userID, err)
		}
		filter.SpecialistID = &specialist.ID
	} else {
		return nil, 0, errors.New("invalid user role for chat access")
	}

	sessions, err := s.chatRepo.ListChatSessions(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.chatRepo.CountChatSessions(ctx, filter)
	if err != nil {
		return sessions, 0, err
	}

	return sessions, count, nil
}

func (s *ChatServiceImpl) UpdateChatSession(ctx context.Context, id int64, dto domain.UpdateChatSessionDTO, userID int64) (*domain.ChatSession, error) {
	// Get existing session to verify access
	session, err := s.GetChatSessionByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Business logic for status transitions
	if dto.Status != nil {
		switch *dto.Status {
		case domain.ChatSessionStatusActive:
			if session.Status == domain.ChatSessionStatusPending {
				now := time.Now()
				dto.StartedAt = &now
			}
		case domain.ChatSessionStatusEnded:
			if session.Status == domain.ChatSessionStatusActive {
				now := time.Now()
				dto.EndedAt = &now
			}
		}
	}

	return s.chatRepo.UpdateChatSession(ctx, id, dto)
}

func (s *ChatServiceImpl) ArchiveChatSession(ctx context.Context, appointmentID int64) error {
	session, err := s.chatRepo.GetChatSessionByAppointmentID(ctx, appointmentID)
	if err != nil {
		// If no session exists, nothing to archive
		return nil
	}

	now := time.Now()
	dto := domain.UpdateChatSessionDTO{
		Status:  &[]domain.ChatSessionStatus{domain.ChatSessionStatusEnded}[0],
		EndedAt: &now,
	}

	_, err = s.chatRepo.UpdateChatSession(ctx, session.ID, dto)
	return err
}

// Chat Messages

func (s *ChatServiceImpl) CreateChatMessage(ctx context.Context, dto domain.CreateChatMessageDTO, userID int64) (*domain.ChatMessage, error) {
	// Verify user has access to the chat session
	session, err := s.GetChatSessionByID(ctx, dto.SessionID, userID)
	if err != nil {
		return nil, err
	}

	// Ensure sender ID matches the authenticated user
	if dto.SenderID != userID {
		return nil, errors.New("sender ID must match authenticated user")
	}

	// Validate that the user is either client or specialist in this session
	hasAccess := false
	
	// Check if user is the client
	if session.ClientID == userID {
		hasAccess = true
	} else {
		// Check if user is the specialist by looking up their specialist record
		specialist, err := s.specialistRepo.GetByUserID(ctx, userID)
		if err == nil && specialist.ID == session.SpecialistID {
			hasAccess = true
		}
	}
	
	if !hasAccess {
		return nil, errors.New("user not authorized to send messages in this session")
	}

	// Auto-activate session if it's pending and this is the first message
	if session.Status == domain.ChatSessionStatusPending {
		now := time.Now()
		updateDTO := domain.UpdateChatSessionDTO{
			Status:    &[]domain.ChatSessionStatus{domain.ChatSessionStatusActive}[0],
			StartedAt: &now,
		}
		_, err = s.chatRepo.UpdateChatSession(ctx, session.ID, updateDTO)
		if err != nil {
			return nil, fmt.Errorf("failed to activate chat session: %w", err)
		}
	}

	return s.chatRepo.CreateChatMessage(ctx, dto)
}

func (s *ChatServiceImpl) ListChatMessages(ctx context.Context, sessionID int64, userID int64, filter domain.ChatMessageFilter) ([]domain.ChatMessage, int64, error) {
	// Verify user has access to the chat session
	_, err := s.GetChatSessionByID(ctx, sessionID, userID)
	if err != nil {
		return nil, 0, err
	}

	// Set session ID in filter
	filter.SessionID = &sessionID

	messages, err := s.chatRepo.ListChatMessages(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.chatRepo.CountChatMessages(ctx, filter)
	if err != nil {
		return messages, 0, err
	}

	return messages, count, nil
}

func (s *ChatServiceImpl) MarkMessagesAsRead(ctx context.Context, sessionID int64, userID int64) error {
	// Verify user has access to the chat session
	_, err := s.GetChatSessionByID(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	return s.chatRepo.MarkMessagesAsRead(ctx, sessionID, userID)
}

func (s *ChatServiceImpl) GetUnreadMessageCount(ctx context.Context, sessionID int64, userID int64) (int64, error) {
	// Verify user has access to the chat session
	_, err := s.GetChatSessionByID(ctx, sessionID, userID)
	if err != nil {
		return 0, err
	}

	return s.chatRepo.GetUnreadMessageCount(ctx, sessionID, userID)
}

func (s *ChatServiceImpl) GetUserChatSummary(ctx context.Context, userID int64) (map[string]interface{}, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	filter := domain.ChatSessionFilter{
		Status: &[]domain.ChatSessionStatus{domain.ChatSessionStatusActive}[0],
		Limit:  100,
		Offset: 0,
	}

	sessions, totalCount, err := s.ListChatSessions(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	// Calculate unread messages for each session
	var totalUnread int64
	sessionSummaries := make([]map[string]interface{}, 0, len(sessions))

	for _, session := range sessions {
		unreadCount, err := s.GetUnreadMessageCount(ctx, session.ID, userID)
		if err != nil {
			unreadCount = 0
		}
		totalUnread += unreadCount

		sessionSummaries = append(sessionSummaries, map[string]interface{}{
			"session_id":         session.ID,
			"appointment_id":     session.AppointmentID,
			"specialization_id":  session.SpecializationID,
			"specialization_name": session.SpecializationName,
			"other_party_name":   getOtherPartyName(&session, userID),
			"unread_count":       unreadCount,
			"created_at":         session.CreatedAt,
			"updated_at":         session.UpdatedAt,
		})
	}

	return map[string]interface{}{
		"user_role":        user.Role,
		"total_sessions":   totalCount,
		"active_sessions":  len(sessions),
		"total_unread":     totalUnread,
		"sessions":         sessionSummaries,
	}, nil
}

// Helper function to get the other party's name in a chat
func getOtherPartyName(session *domain.ChatSession, userID int64) *string {
	if session.ClientID == userID {
		return session.SpecialistName
	}
	return session.ClientName
}