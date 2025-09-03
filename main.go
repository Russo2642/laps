package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"laps/config"
	_ "laps/docs"
	"laps/internal/repository"
	"laps/internal/service"
	"laps/internal/storage"
	"laps/internal/transport/rest"
	"laps/internal/transport/websocket"
	"laps/pkg/database"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title LAPS API
// @version 1.0
// @description API для записи к специалистам
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 94.247.129.222:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatal("Не удалось загрузить конфигурацию", zap.Error(err))
	}

	db, err := database.NewPostgresDB(cfg.Postgres)
	if err != nil {
		logger.Fatal("Не удалось подключиться к БД", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Запуск миграций базы данных")
	if err := database.RunMigrations(db, "./migrations", logger); err != nil {
		logger.Fatal("Ошибка при выполнении миграций", zap.Error(err))
	}
	logger.Info("Миграции успешно выполнены")

	var fileStorage storage.FileStorage
	if cfg.S3.Endpoint != "" {
		s3Storage, err := storage.NewS3Storage(cfg.S3, logger)
		if err != nil {
			logger.Fatal("Не удалось инициализировать S3 хранилище", zap.Error(err))
		}
		fileStorage = s3Storage
		logger.Info("S3 хранилище успешно инициализировано", zap.String("endpoint", cfg.S3.Endpoint))
	} else {
		logger.Warn("S3 хранилище не настроено, функции загрузки файлов будут недоступны")
		// Можно использовать заглушку или локальное хранилище, если S3 не настроено
		// В данном случае просто пропускаем
	}

	repos := repository.NewRepositories(db)

	services := service.NewServices(service.Deps{
		Repos:       repos,
		Logger:      logger,
		Config:      cfg,
		FileStorage: fileStorage,
	})

	// Initialize WebSocket signaling hub
	signalingHub := websocket.NewSignalingHub(logger, services)
	go signalingHub.Run()

	handler := rest.NewHandler(services, logger, cfg, signalingHub)

	router := gin.Default()

	handler.InitRoutes(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	router.GET("/swagger.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.File("./docs/swagger.json")
	})

	srv := &http.Server{
		Addr:    ":" + cfg.HTTP.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Ошибка запуска сервера", zap.Error(err))
		}
	}()

	logger.Info("Сервер запущен", zap.String("addr", srv.Addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Выключение сервера...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Ошибка при остановке сервера", zap.Error(err))
	}

	logger.Info("Сервер успешно остановлен")
}
