package service

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"laps/internal/domain"
	"laps/internal/repository"
	"laps/internal/storage"
)

type SpecialistServiceImpl struct {
	repo        repository.SpecialistRepository
	userRepo    repository.UserRepository
	specRepo    repository.SpecializationRepository
	fileStorage storage.FileStorage
	logger      *zap.Logger
}

func NewSpecialistService(
	repo repository.SpecialistRepository,
	userRepo repository.UserRepository,
	specRepo repository.SpecializationRepository,
	fileStorage storage.FileStorage,
	logger *zap.Logger,
) *SpecialistServiceImpl {
	return &SpecialistServiceImpl{
		repo:        repo,
		userRepo:    userRepo,
		specRepo:    specRepo,
		fileStorage: fileStorage,
		logger:      logger,
	}
}

func (s *SpecialistServiceImpl) Create(ctx context.Context, userID int64, dto domain.CreateSpecialistDTO) (int64, error) {
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("пользователь не найден при создании специалиста", zap.Int64("userID", userID), zap.Error(err))
		return 0, errors.New("пользователь не найден")
	}

	_, err = s.repo.GetByUserID(ctx, userID)
	if err == nil {
		s.logger.Error("пользователь уже зарегистрирован как специалист", zap.Int64("userID", userID))
		return 0, errors.New("пользователь уже зарегистрирован как специалист")
	}

	if dto.SpecializationID != nil {
		_, err = s.specRepo.GetByID(ctx, *dto.SpecializationID)
		if err != nil {
			s.logger.Error("указанная специализация не найдена",
				zap.Int64("specializationID", *dto.SpecializationID),
				zap.Error(err))
			return 0, errors.New("указанная специализация не найдена")
		}
	}

	id, err := s.repo.Create(ctx, userID, dto)
	if err != nil {
		s.logger.Error("ошибка создания специалиста", zap.Error(err))
		return 0, errors.New("ошибка при создании специалиста")
	}

	if len(dto.Education) > 0 {
		for _, educationDTO := range dto.Education {
			_, err := s.repo.AddEducation(ctx, id, educationDTO)
			if err != nil {
				s.logger.Error("ошибка добавления образования", zap.Error(err))
			}
		}
	}

	if len(dto.WorkExperience) > 0 {
		for _, workExpDTO := range dto.WorkExperience {
			_, err := s.repo.AddWorkExperience(ctx, id, workExpDTO)
			if err != nil {
				s.logger.Error("ошибка добавления опыта работы", zap.Error(err))
			}
		}
	}

	if len(dto.ProfilePhoto) > 0 {
		err = s.UploadProfilePhoto(ctx, id, dto.ProfilePhoto, "profile.jpg")
		if err != nil {
			s.logger.Error("ошибка загрузки фото профиля", zap.Int64("specialistID", id), zap.Error(err))
		}
	}

	return id, nil
}

func (s *SpecialistServiceImpl) GetByID(ctx context.Context, id int64) (*domain.Specialist, error) {
	specialist, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("ошибка получения специалиста", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("специалист не найден")
	}
	return specialist, nil
}

func (s *SpecialistServiceImpl) GetByUserID(ctx context.Context, userID int64) (*domain.Specialist, error) {
	specialist, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("ошибка получения специалиста по ID пользователя", zap.Int64("userID", userID), zap.Error(err))
		return nil, errors.New("специалист не найден")
	}
	return specialist, nil
}

func (s *SpecialistServiceImpl) Update(ctx context.Context, id int64, dto domain.UpdateSpecialistDTO) error {
	specialist, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("специалист для обновления не найден", zap.Int64("id", id), zap.Error(err))
		return errors.New("специалист не найден")
	}

	if dto.SpecializationID != nil {
		_, err := s.specRepo.GetByID(ctx, *dto.SpecializationID)
		if err != nil {
			s.logger.Error("указанная специализация не найдена",
				zap.Int64("specializationID", *dto.SpecializationID),
				zap.Error(err))
			return errors.New("указанная специализация не найдена")
		}
	}

	s.logger.Debug("обновление специалиста",
		zap.Int64("id", id),
		zap.Int64("userID", specialist.UserID),
		zap.Any("data", dto))

	err = s.repo.Update(ctx, id, dto)
	if err != nil {
		s.logger.Error("ошибка обновления специалиста", zap.Int64("id", id), zap.Error(err))
		return errors.New("ошибка при обновлении специалиста")
	}

	return nil
}

func (s *SpecialistServiceImpl) Delete(ctx context.Context, id int64) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("специалист для удаления не найден", zap.Int64("id", id), zap.Error(err))
		return errors.New("специалист не найден")
	}

	err = s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error("ошибка удаления специалиста", zap.Int64("id", id), zap.Error(err))
		return errors.New("ошибка при удалении специалиста")
	}

	return nil
}

func (s *SpecialistServiceImpl) List(ctx context.Context, specialistType *domain.SpecialistType, specializationID *int64, limit, offset int) ([]domain.Specialist, int, error) {
	if specialistType != nil && !specialistType.IsValid() {
		s.logger.Error("некорректный тип специалиста", zap.String("type", string(*specialistType)))
		return nil, 0, errors.New("некорректный тип специалиста")
	}

	if specializationID != nil {
		_, err := s.specRepo.GetByID(ctx, *specializationID)
		if err != nil {
			s.logger.Error("указанная специализация не найдена",
				zap.Int64("specializationID", *specializationID),
				zap.Error(err))
			return nil, 0, errors.New("указанная специализация не найдена")
		}
	}

	total, err := s.repo.CountByFilter(ctx, specialistType, specializationID)
	if err != nil {
		s.logger.Error("ошибка подсчета количества специалистов", zap.Error(err))
		return nil, 0, errors.New("ошибка при получении списка специалистов")
	}

	specialists, err := s.repo.List(ctx, specialistType, specializationID, limit, offset)
	if err != nil {
		s.logger.Error("ошибка получения списка специалистов", zap.Error(err))
		return nil, 0, errors.New("ошибка при получении списка специалистов")
	}

	return specialists, total, nil
}

func (s *SpecialistServiceImpl) AddSpecialization(ctx context.Context, specialistID, specializationID int64) error {
	_, err := s.repo.GetByID(ctx, specialistID)
	if err != nil {
		s.logger.Error("специалист не найден при добавлении специализации", zap.Int64("specialistID", specialistID), zap.Error(err))
		return errors.New("специалист не найден")
	}

	_, err = s.specRepo.GetByID(ctx, specializationID)
	if err != nil {
		s.logger.Error("специализация не найдена", zap.Int64("specializationID", specializationID), zap.Error(err))
		return errors.New("специализация не найдена")
	}

	err = s.repo.AddSpecialization(ctx, specialistID, specializationID)
	if err != nil {
		s.logger.Error("ошибка добавления специализации", zap.Error(err))
		return errors.New("ошибка при добавлении специализации")
	}

	return nil
}

func (s *SpecialistServiceImpl) RemoveSpecialization(ctx context.Context, specialistID, specializationID int64) error {
	_, err := s.repo.GetByID(ctx, specialistID)
	if err != nil {
		s.logger.Error("специалист не найден при удалении специализации", zap.Int64("specialistID", specialistID), zap.Error(err))
		return errors.New("специалист не найден")
	}

	err = s.repo.RemoveSpecialization(ctx, specialistID, specializationID)
	if err != nil {
		s.logger.Error("ошибка удаления специализации", zap.Error(err))
		return errors.New("ошибка при удалении специализации")
	}

	return nil
}

func (s *SpecialistServiceImpl) GetSpecializationsBySpecialistID(ctx context.Context, specialistID int64) ([]domain.Specialization, error) {
	_, err := s.repo.GetByID(ctx, specialistID)
	if err != nil {
		s.logger.Error("специалист не найден при получении специализаций", zap.Int64("specialistID", specialistID), zap.Error(err))
		return nil, errors.New("специалист не найден")
	}

	specializations, err := s.repo.GetSpecializationsBySpecialistID(ctx, specialistID)
	if err != nil {
		s.logger.Error("ошибка получения специализаций", zap.Error(err))
		return nil, errors.New("ошибка при получении специализаций")
	}

	return specializations, nil
}

func (s *SpecialistServiceImpl) UploadProfilePhoto(ctx context.Context, specialistID int64, photo []byte, filename string) error {
	_, err := s.repo.GetByID(ctx, specialistID)
	if err != nil {
		s.logger.Error("специалист не найден при загрузке фото", zap.Int64("specialistID", specialistID), zap.Error(err))
		return errors.New("специалист не найден")
	}

	if len(photo) == 0 {
		s.logger.Error("пустой файл фотографии", zap.Int64("specialistID", specialistID))
		return errors.New("пустой файл фотографии")
	}

	photoURL, err := s.fileStorage.UploadFile(ctx, photo, filename)
	if err != nil {
		s.logger.Error("ошибка загрузки фото в хранилище", zap.Int64("specialistID", specialistID), zap.Error(err))
		return errors.New("ошибка загрузки фотографии")
	}

	err = s.repo.UpdateProfilePhoto(ctx, specialistID, photoURL)
	if err != nil {
		s.logger.Error("ошибка обновления URL фото в БД", zap.Int64("specialistID", specialistID), zap.Error(err))

		deleteErr := s.fileStorage.DeleteFile(ctx, photoURL)
		if deleteErr != nil {
			s.logger.Error("ошибка удаления фото после неудачного обновления URL",
				zap.String("photoURL", photoURL), zap.Error(deleteErr))
		}

		return errors.New("ошибка сохранения информации о фотографии")
	}

	return nil
}

func (s *SpecialistServiceImpl) DeleteProfilePhoto(ctx context.Context, specialistID int64) error {
	specialist, err := s.repo.GetByID(ctx, specialistID)
	if err != nil {
		s.logger.Error("специалист не найден при удалении фото", zap.Int64("specialistID", specialistID), zap.Error(err))
		return errors.New("специалист не найден")
	}

	if specialist.ProfilePhotoURL == "" {
		return nil
	}

	err = s.fileStorage.DeleteFile(ctx, specialist.ProfilePhotoURL)
	if err != nil {
		s.logger.Error("ошибка удаления фото из хранилища",
			zap.String("photoURL", specialist.ProfilePhotoURL), zap.Error(err))
	}

	err = s.repo.UpdateProfilePhoto(ctx, specialistID, "")
	if err != nil {
		s.logger.Error("ошибка обновления URL фото в БД при удалении",
			zap.Int64("specialistID", specialistID), zap.Error(err))
		return errors.New("ошибка удаления информации о фотографии")
	}

	return nil
}
