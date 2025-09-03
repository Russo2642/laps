package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"laps/internal/domain"
	"laps/internal/repository"
)

type AppointmentServiceImpl struct {
	repo           repository.AppointmentRepository
	specialistRepo repository.SpecialistRepository
	userRepo       repository.UserRepository
	chatService    ChatService
	logger         *zap.Logger
}

func NewAppointmentService(
	repo repository.AppointmentRepository,
	specialistRepo repository.SpecialistRepository,
	userRepo repository.UserRepository,
	chatService ChatService,
	logger *zap.Logger,
) *AppointmentServiceImpl {
	return &AppointmentServiceImpl{
		repo:           repo,
		specialistRepo: specialistRepo,
		userRepo:       userRepo,
		chatService:    chatService,
		logger:         logger,
	}
}

func (s *AppointmentServiceImpl) Create(ctx context.Context, clientID int64, dto domain.CreateAppointmentDTO) (int64, error) {
	_, err := s.userRepo.GetByID(ctx, clientID)
	if err != nil {
		s.logger.Error("клиент не найден при создании записи", zap.Int64("clientID", clientID), zap.Error(err))
		return 0, errors.New("клиент не найден")
	}

	_, err = s.specialistRepo.GetByID(ctx, dto.SpecialistID)
	if err != nil {
		s.logger.Error("специалист не найден при создании записи", zap.Int64("specialistID", dto.SpecialistID), zap.Error(err))
		return 0, errors.New("специалист не найден")
	}

	dateStr := dto.AppointmentDate.Format("2006-01-02")
	timeStr := dto.AppointmentDate.Format("15:04")

	freeSlots, err := s.repo.GetFreeSlots(ctx, dto.SpecialistID, dateStr)
	if err != nil {
		s.logger.Error("ошибка получения свободных слотов", zap.Error(err))
		return 0, errors.New("ошибка при проверке доступности времени")
	}

	timeIsAvailable := false
	for _, slot := range freeSlots {
		if slot == timeStr {
			timeIsAvailable = true
			break
		}
	}

	if !timeIsAvailable {
		s.logger.Error("выбранное время недоступно", zap.String("time", timeStr))
		return 0, errors.New("выбранное время недоступно")
	}

	id, err := s.repo.Create(ctx, clientID, dto)
	if err != nil {
		s.logger.Error("ошибка создания записи", zap.Error(err))
		return 0, errors.New("ошибка при создании записи")
	}

	// Create chat session automatically for this appointment
	chatDTO := domain.CreateChatSessionDTO{
		AppointmentID:    id,
		ClientID:         clientID,
		SpecialistID:     dto.SpecialistID,
		SpecializationID: 0, // Will be set by chat service from appointment or specialist
		Status:           domain.ChatSessionStatusPending,
	}

	_, err = s.chatService.CreateChatSession(ctx, chatDTO)
	if err != nil {
		s.logger.Error("ошибка создания чат-сессии для записи", 
			zap.Int64("appointmentID", id), 
			zap.Error(err))
		// Don't fail the appointment creation if chat creation fails
		// Just log the error and continue
	}

	return id, nil
}

func (s *AppointmentServiceImpl) GetByID(ctx context.Context, id int64) (*domain.Appointment, error) {
	appointment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("ошибка получения записи", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("запись не найдена")
	}
	return appointment, nil
}

func (s *AppointmentServiceImpl) Update(ctx context.Context, id int64, dto domain.UpdateAppointmentDTO) error {
	appointment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("запись для обновления не найдена", zap.Int64("id", id), zap.Error(err))
		return errors.New("запись не найдена")
	}

	if dto.AppointmentDate != nil {
		dateStr := dto.AppointmentDate.Format("2006-01-02")
		timeStr := dto.AppointmentDate.Format("15:04")

		freeSlots, err := s.repo.GetFreeSlots(ctx, appointment.SpecialistID, dateStr)
		if err != nil {
			s.logger.Error("ошибка получения свободных слотов", zap.Error(err))
			return errors.New("ошибка при проверке доступности времени")
		}

		timeIsAvailable := false
		for _, slot := range freeSlots {
			if slot == timeStr {
				timeIsAvailable = true
				break
			}
		}

		if !timeIsAvailable {
			s.logger.Error("выбранное время недоступно", zap.String("time", timeStr))
			return errors.New("выбранное время недоступно")
		}
	}

	err = s.repo.Update(ctx, id, dto)
	if err != nil {
		s.logger.Error("ошибка обновления записи", zap.Int64("id", id), zap.Error(err))
		return errors.New("ошибка при обновлении записи")
	}

	return nil
}

func (s *AppointmentServiceImpl) Cancel(ctx context.Context, id int64) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("запись для отмены не найдена", zap.Int64("id", id), zap.Error(err))
		return errors.New("запись не найдена")
	}

	dto := domain.UpdateAppointmentDTO{
		Status: PointerTo(domain.AppointmentStatusCancelled),
	}

	err = s.repo.Update(ctx, id, dto)
	if err != nil {
		s.logger.Error("ошибка отмены записи", zap.Int64("id", id), zap.Error(err))
		return errors.New("ошибка при отмене записи")
	}

	// Archive the chat session when appointment is cancelled
	err = s.chatService.ArchiveChatSession(ctx, id)
	if err != nil {
		s.logger.Error("ошибка архивации чат-сессии при отмене записи", 
			zap.Int64("appointmentID", id), 
			zap.Error(err))
		// Don't fail the cancellation if chat archiving fails
		// Just log the error and continue
	}

	return nil
}

func (s *AppointmentServiceImpl) List(ctx context.Context, filter domain.AppointmentFilter) ([]domain.Appointment, int, error) {
	appointments, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("ошибка получения списка записей", zap.Error(err))
		return nil, 0, errors.New("ошибка при получении списка записей")
	}

	count, err := s.repo.CountByFilter(ctx, filter)
	if err != nil {
		s.logger.Error("ошибка получения количества записей", zap.Error(err))
		return appointments, 0, nil
	}

	for i, appointment := range appointments {
		user, err := s.userRepo.GetByID(ctx, appointment.ClientID)
		if err != nil {
			s.logger.Warn("не удалось получить данные пользователя",
				zap.Int64("clientID", appointment.ClientID),
				zap.Error(err))
			continue
		}

		appt := appointments[i]
		appt.ClientName = user.FirstName + " " + user.LastName
		if user.MiddleName != "" {
			appt.ClientName += " " + user.MiddleName
		}
		appt.ClientPhone = user.Phone

		specialist, err := s.specialistRepo.GetByID(ctx, appointment.SpecialistID)
		if err != nil {
			s.logger.Warn("не удалось получить данные специалиста",
				zap.Int64("specialistID", appointment.SpecialistID),
				zap.Error(err))
		} else {
			specialistUser, err := s.userRepo.GetByID(ctx, specialist.UserID)
			if err != nil {
				s.logger.Warn("не удалось получить данные пользователя специалиста",
					zap.Int64("specialistUserID", specialist.UserID),
					zap.Error(err))
			} else {
				appt.SpecialistName = specialistUser.FirstName + " " + specialistUser.LastName
				if specialistUser.MiddleName != "" {
					appt.SpecialistName += " " + specialistUser.MiddleName
				}
				appt.SpecialistPhone = specialistUser.Phone
			}
		}

		appointments[i] = appt
	}

	return appointments, count, nil
}

func (s *AppointmentServiceImpl) GetFreeSlots(ctx context.Context, specialistID int64, date string) ([]string, error) {
	slots, err := s.repo.GetFreeSlots(ctx, specialistID, date)
	if err != nil {
		s.logger.Error("ошибка получения свободных слотов", zap.Error(err))
		return nil, err
	}
	return slots, nil
}

func (s *AppointmentServiceImpl) CheckConsultationType(ctx context.Context, clientID int64, specialistID int64) (domain.ConsultationType, error) {
	filter := domain.AppointmentFilter{
		ClientID:     &clientID,
		SpecialistID: &specialistID,
	}

	appointments, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("ошибка при проверке истории записей", zap.Error(err))
		return "", fmt.Errorf("ошибка при проверке истории записей: %w", err)
	}

	hasActiveAppointments := false
	for _, appointment := range appointments {
		if appointment.Status != domain.AppointmentStatusCancelled {
			hasActiveAppointments = true
			break
		}
	}

	if !hasActiveAppointments {
		return domain.ConsultationTypePrimary, nil
	}

	return domain.ConsultationTypeSecondary, nil
}

func PointerTo[T any](v T) *T {
	return &v
}
