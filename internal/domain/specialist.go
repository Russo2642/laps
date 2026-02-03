package domain

import (
	"time"
)

type SpecialistType string

const (
	SpecialistTypeLawyer       SpecialistType = "lawyer"
	SpecialistTypePsychologist SpecialistType = "psychologist"
	SpecialistTypeNutritionist SpecialistType = "nutritionist"
	SpecialistTypeTrainer      SpecialistType = "trainer"
)

func (t SpecialistType) IsValid() bool {
	return t == SpecialistTypeLawyer ||
		t == SpecialistTypePsychologist ||
		t == SpecialistTypeNutritionist ||
		t == SpecialistTypeTrainer
}

type Specialist struct {
	ID                    int64           `json:"id"`
	UserID                int64           `json:"user_id"`
	Specialization        string          `json:"specialization"`
	SpecializationID      *int64          `json:"specialization_id"`
	SpecializationType    *SpecialistType `json:"specialization_type,omitempty"`
	Experience            int             `json:"experience"`
	Description           string          `json:"description"`
	ExperienceYears       int             `json:"experience_years"`
	Education             []Education     `json:"education"`
	WorkExperience        []WorkPlace     `json:"work_experience"`
	AssociationMember     bool            `json:"association_member"`
	Rating                float64         `json:"rating"`
	ReviewsCount          int             `json:"reviews_count"`
	RecommendationRate    int             `json:"recommendation_rate"`
	PrimaryConsultPrice   float64         `json:"primary_consult_price"`
	SecondaryConsultPrice float64         `json:"secondary_consult_price"`
	IsVerified            bool            `json:"is_verified"`
	ProfilePhotoURL       string          `json:"profile_photo_url"`
	FreeSlots             []string        `json:"free_slots,omitempty"`
	User                  User            `json:"user"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type Education struct {
	ID             int64     `json:"id"`
	SpecialistID   int64     `json:"specialist_id"`
	Institution    string    `json:"institution"`
	Specialization string    `json:"specialization"`
	Degree         string    `json:"degree"`
	GraduationYear int       `json:"graduation_year"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WorkPlace struct {
	ID           int64     `json:"id"`
	SpecialistID int64     `json:"specialist_id"`
	Company      string    `json:"company"`
	Position     string    `json:"position"`
	StartYear    int       `json:"start_year"`
	EndYear      *int      `json:"end_year"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateSpecialistDTO struct {
	UserID                int64               `json:"user_id,omitempty"`
	SpecializationID      *int64              `json:"specialization_id" binding:"omitempty"`
	Experience            int                 `json:"experience,omitempty" binding:"min=0"`
	Description           string              `json:"description,omitempty"`
	ExperienceYears       int                 `json:"experience_years,omitempty"`
	AssociationMember     bool                `json:"association_member,omitempty"`
	PrimaryConsultPrice   float64             `json:"primary_consult_price,omitempty" binding:"min=0"`
	SecondaryConsultPrice float64             `json:"secondary_consult_price,omitempty" binding:"min=0"`
	ProfilePhoto          []byte              `json:"-"`
	Education             []EducationDTO      `json:"education,omitempty"`
	WorkExperience        []WorkExperienceDTO `json:"work_experience,omitempty"`
}

type UpdateSpecialistDTO struct {
	SpecializationID      *int64   `json:"specialization_id"`
	Experience            *int     `json:"experience" binding:"omitempty,min=0"`
	Description           *string  `json:"description"`
	ExperienceYears       *int     `json:"experience_years"`
	AssociationMember     *bool    `json:"association_member"`
	PrimaryConsultPrice   *float64 `json:"primary_consult_price" binding:"omitempty,min=0"`
	SecondaryConsultPrice *float64 `json:"secondary_consult_price" binding:"omitempty,min=0"`
	ProfilePhoto          []byte   `json:"-"`
}

type EducationDTO struct {
	Institution    string `json:"institution" binding:"required"`
	Specialization string `json:"specialization" binding:"required"`
	Degree         string `json:"degree" binding:"required"`
	GraduationYear int    `json:"graduation_year" binding:"required"`
}

type WorkExperienceDTO struct {
	Company     string `json:"company" binding:"required"`
	Position    string `json:"position" binding:"required"`
	StartYear   int    `json:"start_year" binding:"required"`
	EndYear     *int   `json:"end_year"`
	Description string `json:"description"`
}
