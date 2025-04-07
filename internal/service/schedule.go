package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"

	"laps/internal/domain"
	"laps/internal/repository"
)

type ScheduleServiceImpl struct {
	repo           repository.ScheduleRepository
	specialistRepo repository.SpecialistRepository
	logger         *zap.Logger
}

func NewScheduleService(
	repo repository.ScheduleRepository,
	specialistRepo repository.SpecialistRepository,
	logger *zap.Logger,
) *ScheduleServiceImpl {
	return &ScheduleServiceImpl{
		repo:           repo,
		specialistRepo: specialistRepo,
		logger:         logger,
	}
}

func (s *ScheduleServiceImpl) Create(ctx context.Context, specialistID int64, dto domain.CreateScheduleDTO) (int64, error) {
	_, err := s.specialistRepo.GetByID(ctx, specialistID)
	if err != nil {
		s.logger.Error("ошибка при получении специалиста", zap.Error(err))
		return 0, errors.New("специалист не найден")
	}

	if dto.SlotTime < 10 || dto.SlotTime > 120 {
		s.logger.Error("недопустимая длительность слота", zap.Int("slot_time", dto.SlotTime))
		return 0, errors.New("длительность слота должна быть от 10 до 120 минут")
	}

	// Создаем расписание на неделю вперед
	now := time.Now()
	startDate := now.AddDate(0, 0, -int(now.Weekday())+1) // Начало текущей недели
	var lastID int64

	for i := 0; i < 7; i++ {
		currentDate := startDate.AddDate(0, 0, i)
		var daySchedule *domain.DaySchedule

		switch i {
		case 0:
			daySchedule = dto.WeekSchedule.Monday
		case 1:
			daySchedule = dto.WeekSchedule.Tuesday
		case 2:
			daySchedule = dto.WeekSchedule.Wednesday
		case 3:
			daySchedule = dto.WeekSchedule.Thursday
		case 4:
			daySchedule = dto.WeekSchedule.Friday
		case 5:
			daySchedule = dto.WeekSchedule.Saturday
		case 6:
			daySchedule = dto.WeekSchedule.Sunday
		}

		if daySchedule != nil && len(daySchedule.WorkTime) > 0 {
			for _, slot := range daySchedule.WorkTime {
				_, err = time.Parse("15:04", slot.StartTime)
				if err != nil {
					s.logger.Error("неверный формат времени начала", zap.Error(err))
					return 0, errors.New("неверный формат времени начала")
				}

				_, err = time.Parse("15:04", slot.EndTime)
				if err != nil {
					s.logger.Error("неверный формат времени окончания", zap.Error(err))
					return 0, errors.New("неверный формат времени окончания")
				}

				schedule := domain.Schedule{
					SpecialistID: specialistID,
					Date:         currentDate,
					StartTime:    slot.StartTime,
					EndTime:      slot.EndTime,
					SlotTime:     dto.SlotTime,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				id, err := s.repo.Create(ctx, schedule)
				if err != nil {
					s.logger.Error("ошибка создания расписания", zap.Error(err))
					return 0, fmt.Errorf("ошибка создания расписания: %w", err)
				}
				lastID = id
			}
		}
	}

	return lastID, nil
}

func (s *ScheduleServiceImpl) GetByID(ctx context.Context, id int64) (*domain.Schedule, error) {
	schedule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("ошибка получения расписания", zap.Error(err))
		return nil, fmt.Errorf("ошибка получения расписания: %w", err)
	}
	return schedule, nil
}

func (s *ScheduleServiceImpl) Update(ctx context.Context, id int64, dto domain.UpdateScheduleDTO) error {
	schedule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("ошибка получения расписания", zap.Error(err))
		return fmt.Errorf("ошибка получения расписания: %w", err)
	}

	if dto.StartTime != nil {
		schedule.StartTime = *dto.StartTime
	}
	if dto.EndTime != nil {
		schedule.EndTime = *dto.EndTime
	}
	if dto.SlotTime != nil {
		schedule.SlotTime = *dto.SlotTime
	}
	if dto.ExcludeTimes != nil {
		schedule.ExcludeTimes = *dto.ExcludeTimes
	}

	schedule.UpdatedAt = time.Now()

	err = s.repo.Update(ctx, *schedule)
	if err != nil {
		s.logger.Error("ошибка обновления расписания", zap.Error(err))
		return fmt.Errorf("ошибка обновления расписания: %w", err)
	}

	return nil
}

func (s *ScheduleServiceImpl) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error("ошибка удаления расписания", zap.Error(err))
		return fmt.Errorf("ошибка удаления расписания: %w", err)
	}
	return nil
}

func (s *ScheduleServiceImpl) List(ctx context.Context, filter domain.ScheduleFilter) ([]domain.Schedule, int, error) {
	schedules, total, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("ошибка получения списка расписаний", zap.Error(err))
		return nil, 0, fmt.Errorf("ошибка получения списка расписаний: %w", err)
	}
	return schedules, total, nil
}

func (s *ScheduleServiceImpl) GetBySpecialistAndDate(ctx context.Context, specialistID int64, dateStr string) (*domain.Schedule, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		s.logger.Error("неверный формат даты", zap.Error(err))
		return nil, errors.New("неверный формат даты")
	}

	schedule, err := s.repo.GetBySpecialistAndDate(ctx, specialistID, date)
	if err != nil {
		s.logger.Error("ошибка получения расписания", zap.Error(err))
		return nil, fmt.Errorf("ошибка получения расписания: %w", err)
	}

	return schedule, nil
}

func (s *ScheduleServiceImpl) GenerateTimeSlots(ctx context.Context, specialistID int64, dateStr string) ([]string, error) {
	schedule, err := s.GetBySpecialistAndDate(ctx, specialistID, dateStr)
	if err != nil {
		return nil, err
	}

	if schedule == nil {
		return []string{}, nil
	}

	startTime, _ := time.Parse("15:04", schedule.StartTime)
	endTime, _ := time.Parse("15:04", schedule.EndTime)

	excludedSlots := make(map[string]bool)
	for _, excludeTime := range schedule.ExcludeTimes {
		excludedSlots[excludeTime] = true
	}

	var slots []string
	currentTime := startTime
	duration := time.Duration(schedule.SlotTime) * time.Minute

	for currentTime.Before(endTime) {
		timeStr := currentTime.Format("15:04")

		if !excludedSlots[timeStr] {
			slots = append(slots, timeStr)
		}

		currentTime = currentTime.Add(duration)
	}

	sort.Strings(slots)

	return slots, nil
}
