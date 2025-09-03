package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"laps/config"

	"laps/internal/domain"
	"laps/internal/repository"
	"laps/internal/storage"
)

type Deps struct {
	Repos       *repository.Repositories
	FileStorage storage.FileStorage
	Config      *config.Config
	Logger      *zap.Logger
}

type Services struct {
	User           UserService
	Auth           AuthService
	Specialist     SpecialistService
	Specialization SpecializationService
	Schedule       ScheduleService
	Appointment    AppointmentService
	Review         ReviewService
	Education      EducationService
	WorkExperience WorkExperienceService
	Chat           ChatService
}

func NewServices(deps Deps) *Services {
	// Create chat service first since appointment service depends on it
	chatService := NewChatService(deps.Repos)
	
	return &Services{
		User:           NewUserService(deps.Repos.User, deps.Logger),
		Auth:           NewAuthService(deps.Repos.Auth, deps.Repos.User, deps.Config.JWT, deps.Logger),
		Specialist:     NewSpecialistService(deps.Repos.Specialist, deps.Repos.User, deps.Repos.Specialization, deps.FileStorage, deps.Logger),
		Specialization: NewSpecializationService(deps.Repos.Specialization, deps.Logger),
		Schedule:       NewScheduleService(deps.Repos.Schedule, deps.Repos.Specialist, deps.Logger),
		Appointment:    NewAppointmentService(deps.Repos.Appointment, deps.Repos.Specialist, deps.Repos.User, chatService, deps.Logger),
		Review:         NewReviewService(deps.Repos.Review, deps.Repos.Specialist, deps.Repos.User, deps.Repos.Appointment, deps.Logger),
		Education:      NewEducationService(deps.Repos.Specialist, deps.Logger),
		WorkExperience: NewWorkExperienceService(deps.Repos.Specialist, deps.Logger),
		Chat:           chatService,
	}
}

type UserService interface {
	Create(ctx context.Context, dto domain.CreateUserDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, id int64, dto domain.UpdateUserDTO) error
	UpdatePassword(ctx context.Context, id int64, dto domain.PasswordUpdateDTO) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
}

type AuthService interface {
	Register(ctx context.Context, dto domain.RegisterRequest) (int64, error)
	Login(ctx context.Context, dto domain.LoginRequest, userAgent, ip string) (*domain.Tokens, error)
	RefreshTokens(ctx context.Context, refreshToken, userAgent, ip string) (*domain.Tokens, error)
	Logout(ctx context.Context, refreshToken string) error
	ParseToken(ctx context.Context, token string) (int64, domain.UserRole, error)
}

type SpecialistService interface {
	Create(ctx context.Context, userID int64, dto domain.CreateSpecialistDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Specialist, error)
	GetByUserID(ctx context.Context, userID int64) (*domain.Specialist, error)
	Update(ctx context.Context, id int64, dto domain.UpdateSpecialistDTO) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, specialistType *domain.SpecialistType, specializationID *int64, limit, offset int) ([]domain.Specialist, int, error)

	AddSpecialization(ctx context.Context, specialistID, specializationID int64) error
	RemoveSpecialization(ctx context.Context, specialistID, specializationID int64) error
	GetSpecializationsBySpecialistID(ctx context.Context, specialistID int64) ([]domain.Specialization, error)

	UploadProfilePhoto(ctx context.Context, specialistID int64, photo []byte, filename string) error
	DeleteProfilePhoto(ctx context.Context, specialistID int64) error
}

type EducationService interface {
	AddEducation(ctx context.Context, specialistID int64, dto domain.EducationDTO) (int64, error)
	UpdateEducation(ctx context.Context, id int64, dto domain.EducationDTO) error
	DeleteEducation(ctx context.Context, id int64) error
	GetEducationBySpecialistID(ctx context.Context, specialistID int64) ([]domain.Education, error)
	GetEducationByID(ctx context.Context, id int64) (*domain.Education, error)
}

type WorkExperienceService interface {
	AddWorkExperience(ctx context.Context, specialistID int64, dto domain.WorkExperienceDTO) (int64, error)
	UpdateWorkExperience(ctx context.Context, id int64, dto domain.WorkExperienceDTO) error
	DeleteWorkExperience(ctx context.Context, id int64) error
	GetWorkExperienceBySpecialistID(ctx context.Context, specialistID int64) ([]domain.WorkPlace, error)
	GetWorkExperienceByID(ctx context.Context, id int64) (*domain.WorkPlace, error)
}

type SpecializationService interface {
	Create(ctx context.Context, dto domain.CreateSpecializationDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Specialization, error)
	Update(ctx context.Context, id int64, dto domain.UpdateSpecializationDTO) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter domain.SpecializationFilter) ([]domain.Specialization, int, error)
}

type ScheduleService interface {
	Create(ctx context.Context, specialistID int64, dto domain.CreateScheduleDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Schedule, error)
	Update(ctx context.Context, specialistID int64, dto domain.UpdateScheduleDTO) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter domain.ScheduleFilter) ([]domain.Schedule, int, error)
	GetBySpecialistAndDate(ctx context.Context, specialistID int64, date string) (*domain.Schedule, error)
	GenerateTimeSlots(ctx context.Context, specialistID int64, date string) ([]string, error)
	GetWeekSchedule(ctx context.Context, specialistID int64, startDate time.Time) (*domain.WeekSchedule, int, error)
}

type AppointmentService interface {
	Create(ctx context.Context, clientID int64, dto domain.CreateAppointmentDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Appointment, error)
	Update(ctx context.Context, id int64, dto domain.UpdateAppointmentDTO) error
	Cancel(ctx context.Context, id int64) error
	List(ctx context.Context, filter domain.AppointmentFilter) ([]domain.Appointment, int, error)
	GetFreeSlots(ctx context.Context, specialistID int64, date string) ([]string, error)
	CheckConsultationType(ctx context.Context, clientID int64, specialistID int64) (domain.ConsultationType, error)
}

type ReviewService interface {
	Create(ctx context.Context, clientID int64, dto domain.CreateReviewDTO) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Review, error)
	Update(ctx context.Context, id int64, dto domain.UpdateReviewDTO) error
	Delete(ctx context.Context, id int64) error
	GetBySpecialistID(ctx context.Context, specialistID int64, limit, offset int) ([]domain.Review, int, error)
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]domain.Review, error)
	List(ctx context.Context, filter domain.ReviewFilter) ([]domain.Review, int, error)
	CreateReply(ctx context.Context, userID int64, reviewID int64, reply domain.CreateReplyDTO) (int64, error)
	GetReplyByID(ctx context.Context, id int64) (*domain.Reply, error)
	DeleteReply(ctx context.Context, replyID int64) error
	GetRepliesByReviewID(ctx context.Context, reviewID int64) ([]domain.Reply, error)
}

type ChatService interface {
	// Chat Sessions
	CreateChatSession(ctx context.Context, dto domain.CreateChatSessionDTO) (*domain.ChatSession, error)
	GetChatSessionByID(ctx context.Context, id int64, userID int64) (*domain.ChatSession, error)
	GetChatSessionByAppointmentID(ctx context.Context, appointmentID int64, userID int64) (*domain.ChatSession, error)
	ListChatSessions(ctx context.Context, userID int64, filter domain.ChatSessionFilter) ([]domain.ChatSession, int64, error)
	UpdateChatSession(ctx context.Context, id int64, dto domain.UpdateChatSessionDTO, userID int64) (*domain.ChatSession, error)
	ArchiveChatSession(ctx context.Context, appointmentID int64) error
	
	// Chat Messages
	CreateChatMessage(ctx context.Context, dto domain.CreateChatMessageDTO, userID int64) (*domain.ChatMessage, error)
	ListChatMessages(ctx context.Context, sessionID int64, userID int64, filter domain.ChatMessageFilter) ([]domain.ChatMessage, int64, error)
	MarkMessagesAsRead(ctx context.Context, sessionID int64, userID int64) error
	GetUnreadMessageCount(ctx context.Context, sessionID int64, userID int64) (int64, error)
	GetUserChatSummary(ctx context.Context, userID int64) (map[string]interface{}, error)
}
