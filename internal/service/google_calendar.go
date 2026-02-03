package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"laps/config"
	"laps/internal/domain"
)

type GoogleCalendarService interface {
	CreateMeetingEvent(ctx context.Context, appointment *domain.Appointment, clientEmail, specialistEmail, clientName, specialistName string) (meetLink, eventID string, err error)
	UpdateMeetingEvent(ctx context.Context, eventID string, appointment *domain.Appointment, clientEmail, specialistEmail, clientName, specialistName string) error
	DeleteMeetingEvent(ctx context.Context, eventID string) error
}

type googleCalendarService struct {
	service    *calendar.Service
	calendarID string
	enabled    bool
	logger     *zap.Logger
}

func NewGoogleCalendarService(cfg config.GoogleCalendarConfig, logger *zap.Logger) (GoogleCalendarService, error) {
	if !cfg.Enabled {
		logger.Info("Google Calendar интеграция отключена")
		return &googleCalendarService{
			enabled: false,
			logger:  logger,
		}, nil
	}

	if cfg.CredentialsJSON == "" {
		return nil, fmt.Errorf("GOOGLE_CALENDAR_CREDENTIALS_JSON не установлен")
	}

	ctx := context.Background()

	service, err := calendar.NewService(ctx, option.WithCredentialsJSON([]byte(cfg.CredentialsJSON)))
	if err != nil {
		return nil, fmt.Errorf("не удалось создать Google Calendar сервис: %w", err)
	}

	calendarID := cfg.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	logger.Info("Google Calendar сервис успешно инициализирован",
		zap.String("calendar_id", calendarID),
	)

	return &googleCalendarService{
		service:    service,
		calendarID: calendarID,
		enabled:    true,
		logger:     logger,
	}, nil
}

func (s *googleCalendarService) CreateMeetingEvent(
	ctx context.Context,
	appointment *domain.Appointment,
	clientEmail, specialistEmail, clientName, specialistName string,
) (meetLink, eventID string, err error) {
	if !s.enabled {
		return "", "", fmt.Errorf("Google Calendar интеграция отключена")
	}

	consultationType := "Первичная"
	if appointment.ConsultationType == domain.ConsultationTypeSecondary {
		consultationType = "Вторичная"
	}

	summary := fmt.Sprintf("%s консультация: %s и %s", consultationType, clientName, specialistName)
	description := fmt.Sprintf(
		"Консультация в системе LAPS\n\n"+
			"Клиент: %s (%s)\n"+
			"Специалист: %s (%s)\n"+
			"Тип: %s консультация\n"+
			"Цена: %.2f руб.\n\n"+
			"Встреча будет проходить онлайн через Google Meet.",
		clientName, clientEmail,
		specialistName, specialistEmail,
		consultationType,
		appointment.Price,
	)

	startTime := appointment.AppointmentDate
	endTime := startTime.Add(60 * time.Minute)

	meetLink = fmt.Sprintf("https://meet.jit.si/laps-appointment-%d-%d", appointment.ID, time.Now().Unix())

	descriptionWithLink := fmt.Sprintf("%s\n\nСсылка на видеовстречу: %s", description, meetLink)

	event := &calendar.Event{
		Summary:     summary,
		Description: descriptionWithLink,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "Asia/Almaty",
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "Asia/Almaty",
		},
	}

	createdEvent, err := s.service.Events.Insert(s.calendarID, event).
		SendUpdates("none").
		Context(ctx).
		Do()

	if err != nil {
		s.logger.Error("Ошибка создания события в Google Calendar",
			zap.Error(err),
			zap.Int64("appointment_id", appointment.ID),
		)
		return "", "", fmt.Errorf("не удалось создать событие: %w", err)
	}

	s.logger.Info("Событие Google Calendar успешно создано",
		zap.Int64("appointment_id", appointment.ID),
		zap.String("event_id", createdEvent.Id),
		zap.String("meet_link", meetLink),
	)

	return meetLink, createdEvent.Id, nil
}

func (s *googleCalendarService) UpdateMeetingEvent(
	ctx context.Context,
	eventID string,
	appointment *domain.Appointment,
	clientEmail, specialistEmail, clientName, specialistName string,
) error {
	if !s.enabled {
		return fmt.Errorf("Google Calendar интеграция отключена")
	}

	if eventID == "" {
		return fmt.Errorf("event_id не может быть пустым")
	}

	event, err := s.service.Events.Get(s.calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("не удалось получить событие: %w", err)
	}

	startTime := appointment.AppointmentDate
	endTime := startTime.Add(60 * time.Minute)

	event.Start = &calendar.EventDateTime{
		DateTime: startTime.Format(time.RFC3339),
		TimeZone: "Europe/Moscow",
	}
	event.End = &calendar.EventDateTime{
		DateTime: endTime.Format(time.RFC3339),
		TimeZone: "Europe/Moscow",
	}

	_, err = s.service.Events.Update(s.calendarID, eventID, event).
		SendUpdates("none").
		Context(ctx).
		Do()

	if err != nil {
		s.logger.Error("Ошибка обновления события в Google Calendar",
			zap.Error(err),
			zap.String("event_id", eventID),
		)
		return fmt.Errorf("не удалось обновить событие: %w", err)
	}

	s.logger.Info("Событие Google Calendar успешно обновлено",
		zap.String("event_id", eventID),
		zap.Int64("appointment_id", appointment.ID),
	)

	return nil
}

func (s *googleCalendarService) DeleteMeetingEvent(ctx context.Context, eventID string) error {
	if !s.enabled {
		return fmt.Errorf("Google Calendar интеграция отключена")
	}

	if eventID == "" {
		return fmt.Errorf("event_id не может быть пустым")
	}

	err := s.service.Events.Delete(s.calendarID, eventID).
		SendUpdates("none").
		Context(ctx).
		Do()

	if err != nil {
		s.logger.Error("Ошибка удаления события из Google Calendar",
			zap.Error(err),
			zap.String("event_id", eventID),
		)
		return fmt.Errorf("не удалось удалить событие: %w", err)
	}

	s.logger.Info("Событие Google Calendar успешно удалено",
		zap.String("event_id", eventID),
	)

	return nil
}
