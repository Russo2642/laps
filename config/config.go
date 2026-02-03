package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Environment    string
	Name           string
	Version        string
	HTTP           HTTPConfig
	Postgres       PostgresConfig
	JWT            JWTConfig
	S3             S3Config
	GoogleCalendar GoogleCalendarConfig
}

type HTTPConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	MaxHeaderMB  int
}

type PostgresConfig struct {
	Host               string
	Port               string
	Username           string
	Password           string
	DBName             string
	SSLMode            string
	MaxConnections     int
	MaxIdleConnections int
	MaxLifetime        time.Duration
}

type JWTConfig struct {
	SigningKey      string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type S3Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
}

type GoogleCalendarConfig struct {
	Enabled             bool
	CredentialsJSON     string
	CalendarID          string
	ServiceAccountEmail string
}

func NewConfig() (*Config, error) {
	httpReadTimeout, err := time.ParseDuration(getEnv("HTTP_READ_TIMEOUT", "10s"))
	if err != nil {
		return nil, err
	}

	httpWriteTimeout, err := time.ParseDuration(getEnv("HTTP_WRITE_TIMEOUT", "10s"))
	if err != nil {
		return nil, err
	}

	postgresMaxLifetime, err := time.ParseDuration(getEnv("POSTGRES_MAX_LIFETIME", "5m"))
	if err != nil {
		return nil, err
	}

	jwtAccessTokenTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return nil, err
	}

	jwtRefreshTokenTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TOKEN_TTL", "24h"))
	if err != nil {
		return nil, err
	}

	return &Config{
		Environment: getEnv("APP_ENV", "development"),
		Name:        getEnv("APP_NAME", "laps"),
		Version:     getEnv("APP_VERSION", "1.0.0"),
		HTTP: HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			ReadTimeout:  httpReadTimeout,
			WriteTimeout: httpWriteTimeout,
			MaxHeaderMB:  getEnvAsInt("HTTP_MAX_HEADER_MB", 1),
		},
		Postgres: PostgresConfig{
			Host:               getEnv("POSTGRES_HOST", "localhost"),
			Port:               getEnv("POSTGRES_PORT", "5432"),
			Username:           getEnv("POSTGRES_USER", "postgres"),
			Password:           getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:             getEnv("POSTGRES_DB", "laps"),
			SSLMode:            getEnv("POSTGRES_SSL_MODE", "disable"),
			MaxConnections:     getEnvAsInt("POSTGRES_MAX_CONNECTIONS", 10),
			MaxIdleConnections: getEnvAsInt("POSTGRES_MAX_IDLE_CONNECTIONS", 5),
			MaxLifetime:        postgresMaxLifetime,
		},
		JWT: JWTConfig{
			SigningKey:      getEnv("JWT_SIGNING_KEY", "your_secret_key"),
			AccessTokenTTL:  jwtAccessTokenTTL,
			RefreshTokenTTL: jwtRefreshTokenTTL,
		},
		S3: S3Config{
			Endpoint:        getEnv("S3_ENDPOINT", ""),
			Region:          getEnv("S3_REGION", "us-east-1"),
			AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
			Bucket:          getEnv("S3_BUCKET", "laps"),
			UseSSL:          getEnv("S3_USE_SSL", "true") == "true",
		},
		GoogleCalendar: GoogleCalendarConfig{
			Enabled:             getEnv("GOOGLE_CALENDAR_ENABLED", "false") == "true",
			CredentialsJSON:     getEnv("GOOGLE_CALENDAR_CREDENTIALS_JSON", ""),
			CalendarID:          getEnv("GOOGLE_CALENDAR_ID", "primary"),
			ServiceAccountEmail: getEnv("GOOGLE_SERVICE_ACCOUNT_EMAIL", ""),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value := 0
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return defaultValue
	}

	return value
}
