package domain

import (
	"time"
)

type Specialization struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        SpecialistType `json:"type"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type SpecialistSpecialization struct {
	SpecialistID     int64     `json:"specialist_id"`
	SpecializationID int64     `json:"specialization_id"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateSpecializationDTO struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description" binding:"required"`
	Type        SpecialistType `json:"type" binding:"required,oneof=lawyer psychologist nutritionist trainer"`
	IsActive    bool           `json:"is_active"`
}

type UpdateSpecializationDTO struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type SpecializationFilter struct {
	Type         *SpecialistType `json:"type"`
	IsActive     *bool           `json:"is_active"`
	SearchTerm   *string         `json:"search_term"`
	SpecialistID *int64          `json:"specialist_id"`
	Limit        int             `json:"limit"`
	Offset       int             `json:"offset"`
}
