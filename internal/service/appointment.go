package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"laps/internal/domain"
	"laps/internal/repository"
	"laps/internal/utils"
)

type AppointmentServiceImpl struct {
	repo           repository.AppointmentRepository
	specialistRepo repository.SpecialistRepository
	userRepo       repository.UserRepository
	scheduleRepo   repository.ScheduleRepository
	googleCalendar GoogleCalendarService
	logger         *zap.Logger
}

func NewAppointmentService(
	repo repository.AppointmentRepository,
	specialistRepo repository.SpecialistRepository,
	userRepo repository.UserRepository,
	scheduleRepo repository.ScheduleRepository,
	googleCalendar GoogleCalendarService,
	logger *zap.Logger,
) *AppointmentServiceImpl {
	return &AppointmentServiceImpl{
		repo:           repo,
		scheduleRepo:   scheduleRepo,
		specialistRepo: specialistRepo,
		userRepo:       userRepo,
		googleCalendar: googleCalendar,
		logger:         logger,
	}
}

func (s *AppointmentServiceImpl) Create(ctx context.Context, clientID int64, dto domain.CreateAppointmentDTO) (int64, error) {
	client, err := s.userRepo.GetByID(ctx, clientID)
	if err != nil {
		s.logger.Error("клиент не найден при создании записи", zap.Int64("clientID", clientID), zap.Error(err))
		return 0, errors.New("клиент не найден")
	}

	specialist, err := s.specialistRepo.GetByID(ctx, dto.SpecialistID)
	if err != nil {
		s.logger.Error("специалист не найден при создании записи", zap.Int64("specialistID", dto.SpecialistID), zap.Error(err))
		return 0, errors.New("специалист не найден")
	}

	specialistUser, err := s.userRepo.GetByID(ctx, specialist.UserID)
	if err != nil {
		s.logger.Error("пользователь специалиста не найден", zap.Int64("userID", specialist.UserID), zap.Error(err))
		return 0, errors.New("данные специалиста не найдены")
	}

	dateStr := dto.AppointmentDate.Format("2006-01-02")
	timeStr := dto.AppointmentDate.Format("15:04")

	schedule, err := s.scheduleRepo.GetBySpecialistAndDate(ctx, dto.SpecialistID, dto.AppointmentDate)
	if err != nil {
		s.logger.Error("ошибка получения расписания", zap.Error(err))
		return 0, errors.New("ошибка при проверке доступности времени")
	}

	if schedule == nil {
		s.logger.Error("у специалиста нет расписания на выбранную дату", zap.String("date", dateStr))
		return 0, errors.New("специалист не работает в выбранную дату")
	}

	startTime, _ := parseTimeOfDay(schedule.StartTime)
	endTime, _ := parseTimeOfDay(schedule.EndTime)
	appointmentTime, _ := parseTimeOfDay(timeStr)

	if appointmentTime.Before(startTime) || !appointmentTime.Before(endTime) {
		s.logger.Error("время записи вне рабочих часов", zap.String("time", timeStr))
		return 0, errors.New("выбранное время вне рабочих часов специалиста")
	}

	for _, excludeTime := range schedule.ExcludeTimes {
		if excludeTime == timeStr {
			s.logger.Error("время записи в списке исключений", zap.String("time", timeStr))
			return 0, errors.New("выбранное время недоступно")
		}
	}

	busySlots, err := s.repo.GetBusySlots(ctx, dto.SpecialistID, dateStr)
	if err != nil {
		s.logger.Error("ошибка получения занятых слотов", zap.Error(err))
		return 0, errors.New("ошибка при проверке доступности времени")
	}

	if busySlots[timeStr] {
		s.logger.Error("выбранное время уже занято", zap.String("time", timeStr))
		return 0, errors.New("выбранное время уже занято")
	}

	if dto.AppointmentDate.Before(time.Now()) {
		s.logger.Error("попытка записи на прошедшее время",
			zap.Time("appointment_date", dto.AppointmentDate),
			zap.Time("current_time", time.Now()))
		return 0, errors.New("нельзя записаться на прошедшее время")
	}

	id, err := s.repo.Create(ctx, clientID, dto)
	if err != nil {
		s.logger.Error("ошибка создания записи", zap.Error(err))
		return 0, errors.New("ошибка при создании записи")
	}

	if dto.CommunicationMethod == domain.CommunicationMethodVideo {
		appointment, err := s.repo.GetByID(ctx, id)
		if err != nil {
			s.logger.Error("не удалось получить созданную запись", zap.Int64("id", id), zap.Error(err))
			return id, nil
		}

		clientName := fmt.Sprintf("%s %s", client.FirstName, client.LastName)
		specialistName := fmt.Sprintf("%s %s", specialistUser.FirstName, specialistUser.LastName)

		meetLink, eventID, err := s.googleCalendar.CreateMeetingEvent(
			ctx,
			appointment,
			client.Email,
			specialistUser.Email,
			clientName,
			specialistName,
		)

		if err != nil {
			s.logger.Error("не удалось создать Google Meet событие",
				zap.Int64("appointment_id", id),
				zap.Error(err),
			)
		} else {
			err = s.repo.UpdateMeetInfo(ctx, id, meetLink, eventID)
			if err != nil {
				s.logger.Error("не удалось сохранить информацию о Google Meet",
					zap.Int64("appointment_id", id),
					zap.Error(err),
				)
			} else {
				s.logger.Info("Google Meet событие успешно создано и сохранено",
					zap.Int64("appointment_id", id),
					zap.String("meet_link", meetLink),
					zap.String("event_id", eventID),
				)
			}
		}
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

		schedule, err := s.scheduleRepo.GetBySpecialistAndDate(ctx, appointment.SpecialistID, *dto.AppointmentDate)
		if err != nil {
			s.logger.Error("ошибка получения расписания", zap.Error(err))
			return errors.New("ошибка при проверке доступности времени")
		}

		if schedule == nil {
			s.logger.Error("у специалиста нет расписания на выбранную дату", zap.String("date", dateStr))
			return errors.New("специалист не работает в выбранную дату")
		}

		startTime, _ := parseTimeOfDay(schedule.StartTime)
		endTime, _ := parseTimeOfDay(schedule.EndTime)
		appointmentTime, _ := parseTimeOfDay(timeStr)

		if appointmentTime.Before(startTime) || !appointmentTime.Before(endTime) {
			s.logger.Error("время записи вне рабочих часов", zap.String("time", timeStr))
			return errors.New("выбранное время вне рабочих часов специалиста")
		}

		for _, excludeTime := range schedule.ExcludeTimes {
			if excludeTime == timeStr {
				s.logger.Error("время записи в списке исключений", zap.String("time", timeStr))
				return errors.New("выбранное время недоступно")
			}
		}

		busySlots, err := s.repo.GetBusySlots(ctx, appointment.SpecialistID, dateStr)
		if err != nil {
			s.logger.Error("ошибка получения занятых слотов", zap.Error(err))
			return errors.New("ошибка при проверке доступности времени")
		}

		currentTimeStr := appointment.AppointmentDate.Format("15:04")
		if timeStr != currentTimeStr && busySlots[timeStr] {
			s.logger.Error("выбранное время уже занято", zap.String("time", timeStr))
			return errors.New("выбранное время уже занято")
		}

		if dto.AppointmentDate.Before(time.Now()) {
			s.logger.Error("попытка записи на прошедшее время",
				zap.Time("appointment_date", *dto.AppointmentDate),
				zap.Time("current_time", time.Now()))
			return errors.New("нельзя записаться на прошедшее время")
		}
	}

	err = s.repo.Update(ctx, id, dto)
	if err != nil {
		s.logger.Error("ошибка обновления записи", zap.Int64("id", id), zap.Error(err))
		return errors.New("ошибка при обновлении записи")
	}

	if appointment.IsOnline && appointment.MeetEventID != nil && dto.AppointmentDate != nil {
		client, err := s.userRepo.GetByID(ctx, appointment.ClientID)
		if err == nil {
			specialist, err := s.specialistRepo.GetByID(ctx, appointment.SpecialistID)
			if err == nil {
				specialistUser, err := s.userRepo.GetByID(ctx, specialist.UserID)
				if err == nil {
					updatedAppointment := *appointment
					updatedAppointment.AppointmentDate = *dto.AppointmentDate

					clientName := fmt.Sprintf("%s %s", client.FirstName, client.LastName)
					specialistName := fmt.Sprintf("%s %s", specialistUser.FirstName, specialistUser.LastName)

					err = s.googleCalendar.UpdateMeetingEvent(
						ctx,
						*appointment.MeetEventID,
						&updatedAppointment,
						client.Email,
						specialistUser.Email,
						clientName,
						specialistName,
					)
					if err != nil {
						s.logger.Error("не удалось обновить Google Meet событие",
							zap.Int64("appointment_id", id),
							zap.String("event_id", *appointment.MeetEventID),
							zap.Error(err),
						)
					} else {
						s.logger.Info("Google Meet событие успешно обновлено",
							zap.Int64("appointment_id", id),
							zap.String("event_id", *appointment.MeetEventID),
						)
					}
				}
			}
		}
	}

	return nil
}

func (s *AppointmentServiceImpl) Cancel(ctx context.Context, id int64) error {
	appointment, err := s.repo.GetByID(ctx, id)
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

	if appointment.IsOnline && appointment.MeetEventID != nil {
		err = s.googleCalendar.DeleteMeetingEvent(ctx, *appointment.MeetEventID)
		if err != nil {
			s.logger.Error("не удалось удалить Google Meet событие",
				zap.Int64("appointment_id", id),
				zap.String("event_id", *appointment.MeetEventID),
				zap.Error(err),
			)
		} else {
			s.logger.Info("Google Meet событие успешно удалено",
				zap.Int64("appointment_id", id),
				zap.String("event_id", *appointment.MeetEventID),
			)
		}
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

func parseTimeOfDay(timeStr string) (time.Time, error) {
	return utils.ParseTimeOfDay(timeStr)
}
