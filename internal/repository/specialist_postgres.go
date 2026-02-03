package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"laps/internal/domain"
)

type SpecialistRepo struct {
	db *pgxpool.Pool
}

func NewSpecialistRepository(db *pgxpool.Pool) *SpecialistRepo {
	return &SpecialistRepo{
		db: db,
	}
}

func (r *SpecialistRepo) GetDB() *pgxpool.Pool {
	return r.db
}

func (r *SpecialistRepo) Create(ctx context.Context, userID int64, dto domain.CreateSpecialistDTO) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	var specializationIDValue interface{}
	if dto.SpecializationID != nil {
		specializationIDValue = *dto.SpecializationID
	} else {
		specializationIDValue = nil
	}

	query := `
		INSERT INTO specialists (
			user_id, 
			specialization_id,
			experience, 
			description, 
			experience_years, 
			association_member, 
			primary_consult_price, 
			secondary_consult_price,
			profile_photo_url, 
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id
	`

	now := time.Now()
	var id int64
	err = tx.QueryRow(ctx, query,
		userID,
		specializationIDValue,
		dto.Experience,
		dto.Description,
		dto.ExperienceYears,
		dto.AssociationMember,
		dto.PrimaryConsultPrice,
		dto.SecondaryConsultPrice,
		"",
		now,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("ошибка создания специалиста: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return id, nil
}

func (r *SpecialistRepo) GetByID(ctx context.Context, id int64) (*domain.Specialist, error) {
	query := `
		SELECT s.id, s.user_id, s.experience, s.description, 
		       s.experience_years, s.association_member, s.rating, s.reviews_count, 
		       s.recommendation_rate, s.primary_consult_price, s.secondary_consult_price, 
		       s.is_verified, s.profile_photo_url, s.created_at, s.updated_at,
		       s.specialization_id,
			   u.id, u.email, u.phone, u.first_name, u.last_name, u.middle_name, u.role, u.created_at, u.updated_at,
			   sp.name, sp.type
		FROM specialists s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN specializations sp ON s.specialization_id = sp.id
		WHERE s.id = $1
	`

	var specialist domain.Specialist
	var user domain.User
	var specializationID *int64
	var specializationName *string
	var specializationType *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&specialist.ID,
		&specialist.UserID,
		&specialist.Experience,
		&specialist.Description,
		&specialist.ExperienceYears,
		&specialist.AssociationMember,
		&specialist.Rating,
		&specialist.ReviewsCount,
		&specialist.RecommendationRate,
		&specialist.PrimaryConsultPrice,
		&specialist.SecondaryConsultPrice,
		&specialist.IsVerified,
		&specialist.ProfilePhotoURL,
		&specialist.CreatedAt,
		&specialist.UpdatedAt,
		&specializationID,
		&user.ID,
		&user.Email,
		&user.Phone,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&specializationName,
		&specializationType,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("специалист с id %d не найден", id)
		}
		return nil, fmt.Errorf("ошибка получения специалиста: %w", err)
	}

	specialist.User = user
	specialist.SpecializationID = specializationID
	if specializationName != nil {
		specialist.Specialization = *specializationName
	}
	if specializationType != nil {
		t := domain.SpecialistType(*specializationType)
		specialist.SpecializationType = &t
	}

	specialist.Education, err = r.GetEducationBySpecialistID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения образования: %w", err)
	}

	specialist.WorkExperience, err = r.GetWorkExperienceBySpecialistID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения опыта работы: %w", err)
	}

	return &specialist, nil
}

func (r *SpecialistRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Specialist, error) {
	query := `
		SELECT id FROM specialists WHERE user_id = $1
	`

	var specialistID int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&specialistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("специалист с user_id %d не найден", userID)
		}
		return nil, fmt.Errorf("ошибка получения ID специалиста: %w", err)
	}

	return r.GetByID(ctx, specialistID)
}

func (r *SpecialistRepo) Update(ctx context.Context, id int64, dto domain.UpdateSpecialistDTO) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	query := "UPDATE specialists SET "
	var setClauses []string
	var args []interface{}
	argIndex := 1

	if dto.Experience != nil {
		setClauses = append(setClauses, fmt.Sprintf("experience = $%d", argIndex))
		args = append(args, *dto.Experience)
		argIndex++
	}

	if dto.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *dto.Description)
		argIndex++
	}

	if dto.ExperienceYears != nil {
		setClauses = append(setClauses, fmt.Sprintf("experience_years = $%d", argIndex))
		args = append(args, *dto.ExperienceYears)
		argIndex++
	}

	if dto.AssociationMember != nil {
		setClauses = append(setClauses, fmt.Sprintf("association_member = $%d", argIndex))
		args = append(args, *dto.AssociationMember)
		argIndex++
	}

	if dto.PrimaryConsultPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("primary_consult_price = $%d", argIndex))
		args = append(args, *dto.PrimaryConsultPrice)
		argIndex++
	}

	if dto.SecondaryConsultPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("secondary_consult_price = $%d", argIndex))
		args = append(args, *dto.SecondaryConsultPrice)
		argIndex++
	}

	if dto.SpecializationID != nil {
		setClauses = append(setClauses, fmt.Sprintf("specialization_id = $%d", argIndex))
		args = append(args, *dto.SpecializationID)
		argIndex++
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	if len(setClauses) == 1 {
		return nil
	}

	query += strings.Join(setClauses, ", ")
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, id)

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ошибка обновления специалиста: %w", err)
	}

	updateRatingQuery := `
		UPDATE specialists
		SET rating = (
			SELECT COALESCE(AVG(rating), 0) FROM reviews WHERE specialist_id = $1
		)
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateRatingQuery, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления рейтинга специалиста: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM specialists WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления специалиста: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) List(ctx context.Context, specialistType *domain.SpecialistType, specializationID *int64, limit, offset int) ([]domain.Specialist, error) {
	baseQuery := `
		SELECT s.id, s.user_id, s.experience, s.description, 
		       s.experience_years, s.association_member, s.rating, s.reviews_count, 
		       s.recommendation_rate, s.primary_consult_price, s.secondary_consult_price, 
		       s.is_verified, s.profile_photo_url, s.created_at, s.updated_at, s.specialization_id,
			   u.id, u.email, u.phone, u.first_name, u.last_name, u.middle_name, u.role, 
			   u.is_active, u.created_at, u.updated_at,
               sp.name, sp.type
		FROM specialists s
		JOIN users u ON s.user_id = u.id
        LEFT JOIN specializations sp ON s.specialization_id = sp.id
	`

	var whereClauseConditions []string
	var args []interface{}
	argIndex := 1

	if specialistType != nil {
		whereClauseConditions = append(whereClauseConditions, fmt.Sprintf("sp.type = $%d", argIndex))
		args = append(args, *specialistType)
		argIndex++
	}

	if specializationID != nil {
		whereClauseConditions = append(whereClauseConditions, fmt.Sprintf("s.specialization_id = $%d", argIndex))
		args = append(args, *specializationID)
		argIndex++
	}

	var whereClause string
	if len(whereClauseConditions) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauseConditions, " AND ")
	}

	orderLimitClause := fmt.Sprintf(" ORDER BY s.id LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	query := baseQuery + whereClause + orderLimitClause

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var specialists []domain.Specialist
	for rows.Next() {
		var specialist domain.Specialist
		var user domain.User
		var isActive bool
		var specializationName *string
		var specializationType *string

		err := rows.Scan(
			&specialist.ID,
			&specialist.UserID,
			&specialist.Experience,
			&specialist.Description,
			&specialist.ExperienceYears,
			&specialist.AssociationMember,
			&specialist.Rating,
			&specialist.ReviewsCount,
			&specialist.RecommendationRate,
			&specialist.PrimaryConsultPrice,
			&specialist.SecondaryConsultPrice,
			&specialist.IsVerified,
			&specialist.ProfilePhotoURL,
			&specialist.CreatedAt,
			&specialist.UpdatedAt,
			&specialist.SpecializationID,
			&user.ID,
			&user.Email,
			&user.Phone,
			&user.FirstName,
			&user.LastName,
			&user.MiddleName,
			&user.Role,
			&isActive,
			&user.CreatedAt,
			&user.UpdatedAt,
			&specializationName,
			&specializationType,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}

		user.IsActive = isActive
		specialist.User = user
		if specializationName != nil {
			specialist.Specialization = *specializationName
		}
		if specializationType != nil {
			t := domain.SpecialistType(*specializationType)
			specialist.SpecializationType = &t
		}

		specialists = append(specialists, specialist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка обработки результатов: %w", err)
	}

	for i, specialist := range specialists {
		education, err := r.GetEducationBySpecialistID(ctx, specialist.ID)
		if err == nil {
			specialists[i].Education = education
		}

		workExperience, err := r.GetWorkExperienceBySpecialistID(ctx, specialist.ID)
		if err == nil {
			specialists[i].WorkExperience = workExperience
		}
	}

	return specialists, nil
}

func (r *SpecialistRepo) CountByFilter(ctx context.Context, specialistType *domain.SpecialistType, specializationID *int64) (int, error) {
	baseQuery := `
		SELECT COUNT(*)
		FROM specialists s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN specializations sp ON s.specialization_id = sp.id
	`

	var whereClauseConditions []string
	var args []interface{}
	argIndex := 1

	if specialistType != nil {
		whereClauseConditions = append(whereClauseConditions, fmt.Sprintf("sp.type = $%d", argIndex))
		args = append(args, *specialistType)
		argIndex++
	}

	if specializationID != nil {
		whereClauseConditions = append(whereClauseConditions, fmt.Sprintf("s.specialization_id = $%d", argIndex))
		args = append(args, *specializationID)
		argIndex++
	}

	var whereClause string
	if len(whereClauseConditions) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauseConditions, " AND ")
	}

	query := baseQuery + whereClause

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчёта специалистов: %w", err)
	}

	return count, nil
}

func (r *SpecialistRepo) AddEducation(ctx context.Context, specialistID int64, education domain.EducationDTO) (int64, error) {
	query := `
		INSERT INTO education (
			specialist_id, institution, specialization, degree, graduation_year, 
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id
	`

	now := time.Now()
	var id int64
	err := r.db.QueryRow(ctx, query,
		specialistID,
		education.Institution,
		education.Specialization,
		education.Degree,
		education.GraduationYear,
		now,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("ошибка добавления образования: %w", err)
	}

	return id, nil
}

func (r *SpecialistRepo) UpdateEducation(ctx context.Context, id int64, education domain.EducationDTO) error {
	query := `
		UPDATE education
		SET institution = $1,
		    specialization = $2,
		    degree = $3,
		    graduation_year = $4,
		    updated_at = $5
		WHERE id = $6
	`

	_, err := r.db.Exec(ctx, query,
		education.Institution,
		education.Specialization,
		education.Degree,
		education.GraduationYear,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления образования: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) DeleteEducation(ctx context.Context, id int64) error {
	query := `DELETE FROM education WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления образования: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) GetEducationBySpecialistID(ctx context.Context, specialistID int64) ([]domain.Education, error) {
	query := `
		SELECT id, specialist_id, institution, specialization, degree, graduation_year, 
		       created_at, updated_at
		FROM education
		WHERE specialist_id = $1
		ORDER BY graduation_year DESC
	`

	rows, err := r.db.Query(ctx, query, specialistID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения образования: %w", err)
	}
	defer rows.Close()

	education := make([]domain.Education, 0)
	for rows.Next() {
		var edu domain.Education
		if err := rows.Scan(
			&edu.ID,
			&edu.SpecialistID,
			&edu.Institution,
			&edu.Specialization,
			&edu.Degree,
			&edu.GraduationYear,
			&edu.CreatedAt,
			&edu.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки образования: %w", err)
		}
		education = append(education, edu)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return education, nil
}

func (r *SpecialistRepo) GetEducationByID(ctx context.Context, id int64) (*domain.Education, error) {
	query := `
		SELECT id, specialist_id, institution, specialization, degree, graduation_year, 
		       created_at, updated_at
		FROM education
		WHERE id = $1
		LIMIT 1
	`

	var edu domain.Education
	err := r.db.QueryRow(ctx, query, id).Scan(
		&edu.ID,
		&edu.SpecialistID,
		&edu.Institution,
		&edu.Specialization,
		&edu.Degree,
		&edu.GraduationYear,
		&edu.CreatedAt,
		&edu.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("образование с ID %d не найдено", id)
		}
		return nil, fmt.Errorf("ошибка получения образования: %w", err)
	}

	return &edu, nil
}

func (r *SpecialistRepo) AddWorkExperience(ctx context.Context, specialistID int64, workExperience domain.WorkExperienceDTO) (int64, error) {
	query := `
		INSERT INTO work_experience (specialist_id, company, position, start_year, end_year, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id
	`

	now := time.Now()
	var id int64
	err := r.db.QueryRow(ctx, query,
		specialistID,
		workExperience.Company,
		workExperience.Position,
		workExperience.StartYear,
		workExperience.EndYear,
		workExperience.Description,
		now,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("ошибка добавления опыта работы: %w", err)
	}

	return id, nil
}

func (r *SpecialistRepo) UpdateWorkExperience(ctx context.Context, id int64, workExperience domain.WorkExperienceDTO) error {
	query := `
		UPDATE work_experience
		SET company = $1,
		    position = $2,
		    start_year = $3,
		    end_year = $4,
		    description = $5,
		    updated_at = $6
		WHERE id = $7
	`

	_, err := r.db.Exec(ctx, query,
		workExperience.Company,
		workExperience.Position,
		workExperience.StartYear,
		workExperience.EndYear,
		workExperience.Description,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("ошибка обновления опыта работы: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) DeleteWorkExperience(ctx context.Context, id int64) error {
	query := `DELETE FROM work_experience WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления опыта работы: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) GetWorkExperienceBySpecialistID(ctx context.Context, specialistID int64) ([]domain.WorkPlace, error) {
	query := `
		SELECT id, specialist_id, company, position, start_year, end_year, description, created_at, updated_at
		FROM work_experience
		WHERE specialist_id = $1
		ORDER BY end_year DESC NULLS FIRST, start_year DESC
	`

	rows, err := r.db.Query(ctx, query, specialistID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения опыта работы: %w", err)
	}
	defer rows.Close()

	workExperience := make([]domain.WorkPlace, 0)
	for rows.Next() {
		var work domain.WorkPlace
		if err := rows.Scan(
			&work.ID,
			&work.SpecialistID,
			&work.Company,
			&work.Position,
			&work.StartYear,
			&work.EndYear,
			&work.Description,
			&work.CreatedAt,
			&work.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки опыта работы: %w", err)
		}
		workExperience = append(workExperience, work)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return workExperience, nil
}

func (r *SpecialistRepo) GetWorkExperienceByID(ctx context.Context, id int64) (*domain.WorkPlace, error) {
	query := `
		SELECT id, specialist_id, company, position, start_year, end_year, description, created_at, updated_at
		FROM work_experience
		WHERE id = $1
		LIMIT 1
	`

	var work domain.WorkPlace
	err := r.db.QueryRow(ctx, query, id).Scan(
		&work.ID,
		&work.SpecialistID,
		&work.Company,
		&work.Position,
		&work.StartYear,
		&work.EndYear,
		&work.Description,
		&work.CreatedAt,
		&work.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("опыт работы с ID %d не найден", id)
		}
		return nil, fmt.Errorf("ошибка получения опыта работы: %w", err)
	}

	return &work, nil
}

func (r *SpecialistRepo) AddSpecialization(ctx context.Context, specialistID, specializationID int64) error {
	query := `
		INSERT INTO specialist_specializations (specialist_id, specialization_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (specialist_id, specialization_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, specialistID, specializationID, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка добавления специализации: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) RemoveSpecialization(ctx context.Context, specialistID, specializationID int64) error {
	query := `
		DELETE FROM specialist_specializations
		WHERE specialist_id = $1 AND specialization_id = $2
	`

	_, err := r.db.Exec(ctx, query, specialistID, specializationID)
	if err != nil {
		return fmt.Errorf("ошибка удаления специализации: %w", err)
	}

	return nil
}

func (r *SpecialistRepo) GetSpecializationsBySpecialistID(ctx context.Context, specialistID int64) ([]domain.Specialization, error) {
	query := `
		SELECT s.id, s.name, s.description, s.type, s.is_active, s.created_at, s.updated_at
		FROM specializations s
		JOIN specialist_specializations ss ON s.id = ss.specialization_id
		WHERE ss.specialist_id = $1
		ORDER BY s.name
	`

	rows, err := r.db.Query(ctx, query, specialistID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения специализаций: %w", err)
	}
	defer rows.Close()

	specializations := make([]domain.Specialization, 0)
	for rows.Next() {
		var spec domain.Specialization
		if err := rows.Scan(
			&spec.ID,
			&spec.Name,
			&spec.Description,
			&spec.Type,
			&spec.IsActive,
			&spec.CreatedAt,
			&spec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки специализации: %w", err)
		}
		specializations = append(specializations, spec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return specializations, nil
}

func (r *SpecialistRepo) UpdateProfilePhoto(ctx context.Context, id int64, photoURL string) error {
	query := `
		UPDATE specialists
		SET profile_photo_url = $1,
		    updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, photoURL, time.Now(), id)
	if err != nil {
		return fmt.Errorf("ошибка обновления фотографии профиля: %w", err)
	}

	return nil
}
