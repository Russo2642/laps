package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"laps/internal/domain"
)

type Repositories struct {
	User           UserRepository
	Specialist     SpecialistRepository
	Appointment    AppointmentRepository
	Review         ReviewRepository
	Specialization SpecializationRepository
	Auth           AuthRepository
	Schedule       ScheduleRepository
	Chat           ChatRepository
}

func NewRepositories(db *pgxpool.Pool) *Repositories {
	return &Repositories{
		User:           NewUserRepository(db),
		Auth:           NewAuthRepository(db),
		Specialization: NewSpecializationRepository(db),
		Specialist:     NewSpecialistRepository(db),
		Appointment:    NewAppointmentRepository(db),
		Review:         NewReviewRepository(db),
		Schedule:       NewScheduleRepository(db),
		Chat:           NewChatRepository(db),
	}
}

type UserRepository interface {
	Create(ctx context.Context, user domain.CreateUserDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByPhone(ctx context.Context, phone string) (*domain.User, error)
	Update(ctx context.Context, id int64, user domain.UpdateUserDTO) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
}

type SpecialistRepository interface {
	Create(ctx context.Context, userID int64, specialist domain.CreateSpecialistDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Specialist, error)
	GetByUserID(ctx context.Context, userID int64) (*domain.Specialist, error)
	Update(ctx context.Context, id int64, specialist domain.UpdateSpecialistDTO) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, specialistType *domain.SpecialistType, specializationID *int64, limit, offset int) ([]domain.Specialist, error)
	CountByFilter(ctx context.Context, specialistType *domain.SpecialistType, specializationID *int64) (int, error)

	UpdateProfilePhoto(ctx context.Context, id int64, photoURL string) error

	AddEducation(ctx context.Context, specialistID int64, education domain.EducationDTO) (int64, error)
	UpdateEducation(ctx context.Context, id int64, education domain.EducationDTO) error
	DeleteEducation(ctx context.Context, id int64) error
	GetEducationBySpecialistID(ctx context.Context, specialistID int64) ([]domain.Education, error)
	GetEducationByID(ctx context.Context, id int64) (*domain.Education, error)

	AddWorkExperience(ctx context.Context, specialistID int64, workExperience domain.WorkExperienceDTO) (int64, error)
	UpdateWorkExperience(ctx context.Context, id int64, workExperience domain.WorkExperienceDTO) error
	DeleteWorkExperience(ctx context.Context, id int64) error
	GetWorkExperienceBySpecialistID(ctx context.Context, specialistID int64) ([]domain.WorkPlace, error)
	GetWorkExperienceByID(ctx context.Context, id int64) (*domain.WorkPlace, error)

	AddSpecialization(ctx context.Context, specialistID, specializationID int64) error
	RemoveSpecialization(ctx context.Context, specialistID, specializationID int64) error
	GetSpecializationsBySpecialistID(ctx context.Context, specialistID int64) ([]domain.Specialization, error)
	GetDB() *pgxpool.Pool
}

type AppointmentRepository interface {
	Create(ctx context.Context, clientID int64, appointment domain.CreateAppointmentDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Appointment, error)
	Update(ctx context.Context, id int64, appointment domain.UpdateAppointmentDTO) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter domain.AppointmentFilter) ([]domain.Appointment, error)
	CountByFilter(ctx context.Context, filter domain.AppointmentFilter) (int, error)
	GetFreeSlots(ctx context.Context, specialistID int64, date string) ([]string, error)
}

type ReviewRepository interface {
	Create(ctx context.Context, clientID int64, review domain.CreateReviewDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Review, error)
	Update(ctx context.Context, id int64, dto domain.UpdateReviewDTO) error
	Delete(ctx context.Context, id int64) error
	GetBySpecialistID(ctx context.Context, specialistID int64, limit, offset int) ([]domain.Review, error)
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]domain.Review, error)
	CountBySpecialistID(ctx context.Context, specialistID int64) (int, error)
	CountByFilter(ctx context.Context, filter domain.ReviewFilter) (int, error)
	List(ctx context.Context, filter domain.ReviewFilter) ([]domain.Review, error)
	CreateReply(ctx context.Context, userID int64, reviewID int64, reply domain.CreateReplyDTO) (int64, error)
	GetReplyByID(ctx context.Context, id int64) (*domain.Reply, error)
	DeleteReply(ctx context.Context, id int64) error
	GetRepliesByReviewID(ctx context.Context, reviewID int64) ([]domain.Reply, error)
}

type SpecializationRepository interface {
	Create(ctx context.Context, specialization domain.CreateSpecializationDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Specialization, error)
	Update(ctx context.Context, id int64, specialization domain.UpdateSpecializationDTO) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter domain.SpecializationFilter) ([]domain.Specialization, error)
	CountByFilter(ctx context.Context, filter domain.SpecializationFilter) (int, error)
}

type AuthRepository interface {
	CreateSession(ctx context.Context, session domain.Session) error
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteSessionsByUserID(ctx context.Context, userID int64) error
}

type ScheduleRepository interface {
	Create(ctx context.Context, schedule domain.Schedule) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Schedule, error)
	Update(ctx context.Context, schedule domain.Schedule) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter domain.ScheduleFilter) ([]domain.Schedule, int, error)
	GetBySpecialistAndDate(ctx context.Context, specialistID int64, date time.Time) (*domain.Schedule, error)
}

type ChatRepository interface {
	// Chat Sessions
	CreateChatSession(ctx context.Context, dto domain.CreateChatSessionDTO) (*domain.ChatSession, error)
	GetChatSessionByID(ctx context.Context, id int64) (*domain.ChatSession, error)
	GetChatSessionByAppointmentID(ctx context.Context, appointmentID int64) (*domain.ChatSession, error)
	ListChatSessions(ctx context.Context, filter domain.ChatSessionFilter) ([]domain.ChatSession, error)
	CountChatSessions(ctx context.Context, filter domain.ChatSessionFilter) (int64, error)
	UpdateChatSession(ctx context.Context, id int64, dto domain.UpdateChatSessionDTO) (*domain.ChatSession, error)
	
	// Chat Messages
	CreateChatMessage(ctx context.Context, dto domain.CreateChatMessageDTO) (*domain.ChatMessage, error)
	ListChatMessages(ctx context.Context, filter domain.ChatMessageFilter) ([]domain.ChatMessage, error)
	CountChatMessages(ctx context.Context, filter domain.ChatMessageFilter) (int64, error)
	MarkMessagesAsRead(ctx context.Context, sessionID int64, userID int64) error
	GetUnreadMessageCount(ctx context.Context, sessionID int64, userID int64) (int64, error)
}
