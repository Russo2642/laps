package domain

import (
	"time"
)

// ChatSessionStatus represents the status of a chat session
type ChatSessionStatus string

const (
	ChatSessionStatusPending ChatSessionStatus = "pending"
	ChatSessionStatusActive  ChatSessionStatus = "active"
	ChatSessionStatusEnded   ChatSessionStatus = "ended"
)

// MessageType represents the type of a chat message
type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeFile   MessageType = "file"
	MessageTypeSystem MessageType = "system"
)

// ChatSession represents a chat session between a client and specialist
type ChatSession struct {
	ID               int64             `json:"id" db:"id"`
	AppointmentID    int64             `json:"appointment_id" db:"appointment_id"`
	ClientID         int64             `json:"client_id" db:"client_id"`
	SpecialistID     int64             `json:"specialist_id" db:"specialist_id"`
	SpecializationID int64             `json:"specialization_id" db:"specialization_id"`
	Status           ChatSessionStatus `json:"status" db:"status"`
	StartedAt        *time.Time        `json:"started_at,omitempty" db:"started_at"`
	EndedAt          *time.Time        `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
	
	// Optional fields populated by joins
	ClientName         *string `json:"client_name,omitempty" db:"client_name"`
	ClientPhone        *string `json:"client_phone,omitempty" db:"client_phone"`
	SpecialistName     *string `json:"specialist_name,omitempty" db:"specialist_name"`
	SpecialistPhone    *string `json:"specialist_phone,omitempty" db:"specialist_phone"`
	SpecializationName *string `json:"specialization_name,omitempty" db:"specialization_name"`
}

// ChatMessage represents a message in a chat session
type ChatMessage struct {
	ID        int64       `json:"id" db:"id"`
	SessionID int64       `json:"session_id" db:"session_id"`
	SenderID  int64       `json:"sender_id" db:"sender_id"`
	Type      MessageType `json:"message_type" db:"message_type"`
	Content   string      `json:"content" db:"content"`
	FileURL   *string     `json:"file_url,omitempty" db:"file_url"`
	FileName  *string     `json:"file_name,omitempty" db:"file_name"`
	FileSize  *int64      `json:"file_size,omitempty" db:"file_size"`
	IsRead    bool        `json:"is_read" db:"is_read"`
	ReadAt    *time.Time  `json:"read_at,omitempty" db:"read_at"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
	
	// Optional fields populated by joins
	SenderName  *string `json:"sender_name,omitempty" db:"sender_name"`
	SenderRole  *string `json:"sender_role,omitempty" db:"sender_role"`
}

// ChatParticipant represents a participant in a chat session
type ChatParticipant struct {
	ID        int64      `json:"id" db:"id"`
	SessionID int64      `json:"session_id" db:"session_id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	Role      string     `json:"role" db:"role"`
	JoinedAt  time.Time  `json:"joined_at" db:"joined_at"`
	LeftAt    *time.Time `json:"left_at,omitempty" db:"left_at"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateChatSessionDTO represents the data required to create a chat session
type CreateChatSessionDTO struct {
	AppointmentID    int64             `json:"appointment_id" binding:"required"`
	ClientID         int64             `json:"client_id" binding:"required"`
	SpecialistID     int64             `json:"specialist_id" binding:"required"`
	SpecializationID int64             `json:"specialization_id" binding:"required"`
	Status           ChatSessionStatus `json:"status,omitempty"`
}

// CreateChatMessageDTO represents the data required to create a chat message
type CreateChatMessageDTO struct {
	SessionID int64       `json:"session_id" binding:"required"`
	SenderID  int64       `json:"sender_id" binding:"required"`
	Type      MessageType `json:"message_type" binding:"required"`
	Content   string      `json:"content" binding:"required"`
	FileURL   *string     `json:"file_url,omitempty"`
	FileName  *string     `json:"file_name,omitempty"`
	FileSize  *int64      `json:"file_size,omitempty"`
}

// UpdateChatSessionDTO represents the data that can be updated for a chat session
type UpdateChatSessionDTO struct {
	Status    *ChatSessionStatus `json:"status,omitempty"`
	StartedAt *time.Time         `json:"started_at,omitempty"`
	EndedAt   *time.Time         `json:"ended_at,omitempty"`
}

// ChatSessionFilter represents filters for querying chat sessions
type ChatSessionFilter struct {
	ClientID         *int64             `json:"client_id"`
	SpecialistID     *int64             `json:"specialist_id"`
	SpecializationID *int64             `json:"specialization_id"`
	Status           *ChatSessionStatus `json:"status"`
	AppointmentID    *int64             `json:"appointment_id"`
	Limit            int                `json:"limit"`
	Offset           int                `json:"offset"`
}

// ChatMessageFilter represents filters for querying chat messages
type ChatMessageFilter struct {
	SessionID *int64      `json:"session_id"`
	SenderID  *int64      `json:"sender_id"`
	Type      *MessageType `json:"message_type"`
	IsRead    *bool       `json:"is_read"`
	Limit     int         `json:"limit"`
	Offset    int         `json:"offset"`
}