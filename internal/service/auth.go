package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"laps/config"
	"laps/internal/domain"
	"laps/internal/repository"
)

type tokenClaims struct {
	jwt.RegisteredClaims
	UserID int64           `json:"user_id"`
	Role   domain.UserRole `json:"role"`
}

type AuthServiceImpl struct {
	authRepo       repository.AuthRepository
	userRepo       repository.UserRepository
	specialistRepo repository.SpecialistRepository
	jwtConfig      config.JWTConfig
	logger         *zap.Logger
}

func NewAuthService(
	authRepo repository.AuthRepository,
	userRepo repository.UserRepository,
	specialistRepo repository.SpecialistRepository,
	jwtConfig config.JWTConfig,
	logger *zap.Logger,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		authRepo:       authRepo,
		userRepo:       userRepo,
		specialistRepo: specialistRepo,
		jwtConfig:      jwtConfig,
		logger:         logger,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, dto domain.RegisterRequest) (int64, error) {
	if dto.Role == domain.UserRoleSpecialist && dto.SpecializationID == nil {
		return 0, errors.New("для регистрации специалиста необходимо указать специализацию")
	}

	existingUser, err := s.userRepo.GetByEmail(ctx, dto.Email)
	if err == nil && existingUser != nil {
		return 0, errors.New("пользователь с таким email уже существует")
	}

	existingUser, err = s.userRepo.GetByPhone(ctx, dto.Phone)
	if err == nil && existingUser != nil {
		return 0, errors.New("пользователь с таким телефоном уже существует")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("ошибка при хешировании пароля", zap.Error(err))
		return 0, errors.New("ошибка при регистрации пользователя")
	}

	createUserDTO := domain.CreateUserDTO{
		FirstName:  dto.FirstName,
		LastName:   dto.LastName,
		MiddleName: dto.MiddleName,
		Email:      dto.Email,
		Phone:      dto.Phone,
		Password:   string(hashedPassword),
		Role:       dto.Role,
	}

	db := s.specialistRepo.GetDB()
	tx, err := db.Begin(ctx)
	if err != nil {
		s.logger.Error("ошибка начала транзакции", zap.Error(err))
		return 0, errors.New("ошибка при регистрации пользователя")
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO users (first_name, last_name, middle_name, email, phone, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id
	`
	now := time.Now()
	var userID int64
	err = tx.QueryRow(ctx, query,
		createUserDTO.FirstName,
		createUserDTO.LastName,
		createUserDTO.MiddleName,
		createUserDTO.Email,
		createUserDTO.Phone,
		createUserDTO.Password,
		createUserDTO.Role,
		true,
		now,
	).Scan(&userID)

	if err != nil {
		s.logger.Error("ошибка при создании пользователя", zap.Error(err))
		return 0, errors.New("ошибка при регистрации пользователя")
	}

	if dto.Role == domain.UserRoleSpecialist {
		specialistQuery := `
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

		var specialistID int64
		err = tx.QueryRow(ctx, specialistQuery,
			userID,
			dto.SpecializationID,
			0,     // experience
			"",    // description
			0,     // experience_years
			false, // association_member
			0.0,   // primary_consult_price
			0.0,   // secondary_consult_price
			"",    // profile_photo_url
			now,
		).Scan(&specialistID)

		if err != nil {
			s.logger.Error("ошибка при создании профиля специалиста",
				zap.Int64("userID", userID),
				zap.Error(err))
			return 0, errors.New("ошибка при создании профиля специалиста")
		}

		s.logger.Info("автоматически создан профиль специалиста при регистрации",
			zap.Int64("userID", userID),
			zap.Int64("specialistID", specialistID))
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("ошибка при коммите транзакции", zap.Error(err))
		return 0, errors.New("ошибка при регистрации пользователя")
	}

	return userID, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, dto domain.LoginRequest, userAgent, ip string) (*domain.Tokens, error) {
	var user *domain.User
	var err error

	user, err = s.userRepo.GetByEmail(ctx, dto.Login)
	if err != nil {
		user, err = s.userRepo.GetByPhone(ctx, dto.Login)
		if err != nil {
			s.logger.Error("пользователь не найден", zap.String("login", dto.Login), zap.Error(err))
			return nil, errors.New("неверный логин или пароль")
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.Password))
	if err != nil {
		s.logger.Error("неверный пароль", zap.Error(err))
		return nil, errors.New("неверный логин или пароль")
	}

	if !user.IsActive {
		return nil, errors.New("аккаунт деактивирован")
	}

	tokens, err := s.generateTokens(user.ID, user.Role)
	if err != nil {
		s.logger.Error("ошибка генерации токенов", zap.Error(err))
		return nil, errors.New("ошибка при аутентификации")
	}

	session := domain.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: tokens.RefreshToken,
		UserAgent:    userAgent,
		IP:           ip,
		ExpiresAt:    time.Now().Add(s.jwtConfig.RefreshTokenTTL),
		CreatedAt:    time.Now(),
	}

	err = s.authRepo.CreateSession(ctx, session)
	if err != nil {
		s.logger.Error("ошибка сохранения сессии", zap.Error(err))
		return nil, errors.New("ошибка при аутентификации")
	}

	return tokens, nil
}

func (s *AuthServiceImpl) RefreshTokens(ctx context.Context, refreshToken, userAgent, ip string) (*domain.Tokens, error) {
	session, err := s.authRepo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		s.logger.Error("ошибка получения сессии", zap.Error(err))
		return nil, errors.New("недействительный refresh token")
	}

	if session.ExpiresAt.Before(time.Now()) {
		s.authRepo.DeleteSession(ctx, session.ID)
		return nil, errors.New("refresh token истек")
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		s.logger.Error("пользователь не найден", zap.Int64("userId", session.UserID), zap.Error(err))
		return nil, errors.New("пользователь не найден")
	}

	if !user.IsActive {
		return nil, errors.New("аккаунт деактивирован")
	}

	err = s.authRepo.DeleteSession(ctx, session.ID)
	if err != nil {
		s.logger.Warn("ошибка удаления старой сессии", zap.Error(err))
	}

	tokens, err := s.generateTokens(user.ID, user.Role)
	if err != nil {
		s.logger.Error("ошибка генерации токенов", zap.Error(err))
		return nil, errors.New("ошибка при обновлении токенов")
	}

	newSession := domain.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		RefreshToken: tokens.RefreshToken,
		UserAgent:    userAgent,
		IP:           ip,
		ExpiresAt:    time.Now().Add(s.jwtConfig.RefreshTokenTTL),
		CreatedAt:    time.Now(),
	}

	err = s.authRepo.CreateSession(ctx, newSession)
	if err != nil {
		s.logger.Error("ошибка сохранения новой сессии", zap.Error(err))
		return nil, errors.New("ошибка при обновлении токенов")
	}

	return tokens, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.authRepo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		s.logger.Warn("сессия не найдена при выходе", zap.Error(err))
	}

	err = s.authRepo.DeleteSession(ctx, session.ID)
	if err != nil {
		s.logger.Error("ошибка удаления сессии", zap.Error(err))
		return errors.New("ошибка при выходе")
	}

	return nil
}

func (s *AuthServiceImpl) ParseToken(ctx context.Context, tokenString string) (int64, domain.UserRole, error) {
	token, err := jwt.ParseWithClaims(tokenString, &tokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(s.jwtConfig.SigningKey), nil
	})

	if err != nil {
		return 0, "", fmt.Errorf("ошибка парсинга токена: %w", err)
	}

	claims, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return 0, "", errors.New("недействительный токен")
	}

	return claims.UserID, claims.Role, nil
}

func (s *AuthServiceImpl) generateTokens(userID int64, role domain.UserRole) (*domain.Tokens, error) {
	accessTokenClaims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtConfig.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
		Role:   role,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.jwtConfig.SigningKey))
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи access token: %w", err)
	}

	refreshTokenClaims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtConfig.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
		Role:   role,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtConfig.SigningKey))
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи refresh token: %w", err)
	}

	return &domain.Tokens{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}
