package domain

import (
	"time"
)

type AppointmentStatus string

const (
	AppointmentStatusPending   AppointmentStatus = "pending"
	AppointmentStatusPaid      AppointmentStatus = "paid"
	AppointmentStatusCompleted AppointmentStatus = "completed"
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
)

type ConsultationType string

const (
	ConsultationTypePrimary   ConsultationType = "primary"
	ConsultationTypeSecondary ConsultationType = "secondary"
)

type CommunicationMethod string

const (
	CommunicationMethodPhone     CommunicationMethod = "phone"
	CommunicationMethodWhatsApp  CommunicationMethod = "whatsapp"
	CommunicationMethodVideoCall CommunicationMethod = "video_call"
)

type Appointment struct {
	ID                  int64               `json:"id"`
	ClientID            int64               `json:"client_id"`
	SpecialistID        int64               `json:"specialist_id"`
	ConsultationType    ConsultationType    `json:"consultation_type"`
	SpecializationID    *int64              `json:"specialization_id"`
	Price               float64             `json:"price"`
	AppointmentDate     time.Time           `json:"appointment_date"`
	Status              AppointmentStatus   `json:"status"`
	PaymentID           *string             `json:"payment_id"`
	CommunicationMethod CommunicationMethod `json:"communication_method"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
	ClientName          string              `json:"client_name,omitempty"`
	ClientPhone         string              `json:"client_phone,omitempty"`
	SpecialistName      string              `json:"specialist_name,omitempty"`
	SpecialistPhone     string              `json:"specialist_phone,omitempty"`
}

type CreateAppointmentDTO struct {
	SpecialistID        int64               `json:"specialist_id" binding:"required"`
	ConsultationType    ConsultationType    `json:"consultation_type" binding:"required,oneof=primary secondary"`
	SpecializationID    *int64              `json:"specialization_id"`
	AppointmentDate     time.Time           `json:"appointment_date" binding:"required"`
	CommunicationMethod CommunicationMethod `json:"communication_method" binding:"required,oneof=phone whatsapp video_call"`
}

type UpdateAppointmentDTO struct {
	Status          *AppointmentStatus `json:"status" binding:"omitempty,oneof=pending paid completed cancelled"`
	AppointmentDate *time.Time         `json:"appointment_date"`
	PaymentID       *string            `json:"payment_id"`
}

type AppointmentFilter struct {
	ClientID      *int64             `json:"client_id"`
	SpecialistID  *int64             `json:"specialist_id"`
	Status        *AppointmentStatus `json:"status"`
	ExcludeStatus *AppointmentStatus `json:"exclude_status"`
	StartDate     *time.Time         `json:"start_date"`
	EndDate       *time.Time         `json:"end_date"`
	Limit         int                `json:"limit"`
	Offset        int                `json:"offset"`
}
