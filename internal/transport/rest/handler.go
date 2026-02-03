package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"laps/config"
	"laps/internal/domain"
	"laps/internal/service"
)

type Handler struct {
	services *service.Services
	logger   *zap.Logger
	config   *config.Config
}

func NewHandler(services *service.Services, logger *zap.Logger, config *config.Config) *Handler {
	return &Handler{
		services: services,
		logger:   logger,
		config:   config,
	}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	router.Use(h.corsMiddleware())

	router.Use(h.loggerMiddleware())

	router.Use(h.errorMiddleware())

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.register)
			auth.POST("/login", h.login)
			auth.POST("/refresh", h.refreshTokens)
			auth.POST("/logout", h.logout)
		}

		users := api.Group("/users")
		users.Use(h.authMiddleware())
		{
			users.GET("/me", h.getCurrentUser)
			users.GET("/:id", h.getUserByID)
			users.PUT("/:id", h.updateUser)
			users.PUT("/:id/password", h.updatePassword)

			users.POST("", h.adminMiddleware(), h.createUser)
			users.GET("", h.adminMiddleware(), h.getUsers)
			users.DELETE("/:id", h.adminMiddleware(), h.deleteUser)
		}
		specialists := api.Group("/specialists")
		{
			specialists.GET("", h.getSpecialists)
			specialists.GET("/:id", h.getSpecialistByID)
			specialists.GET("/:id/reviews", h.getSpecialistReviewsRedirect)
			specialists.GET("/me", h.authMiddleware(), h.getMySpecialistProfile)

			specialists.POST("", h.authMiddleware(), h.createSpecialist)
			specialists.PUT("/:id", h.authMiddleware(), h.updateSpecialist)
			specialists.DELETE("/:id", h.authMiddleware(), h.deleteSpecialist)

			specialists.PUT("/:id/education/:eduId", h.authMiddleware(), h.updateSpecialistEducation)
			specialists.DELETE("/:id/education/:eduId", h.authMiddleware(), h.deleteSpecialistEducation)

			specialists.PUT("/:id/work-experience/:expId", h.authMiddleware(), h.updateSpecialistWorkExperience)
			specialists.DELETE("/:id/work-experience/:expId", h.authMiddleware(), h.deleteSpecialistWorkExperience)

			specialists.POST("/:id/specializations/:specId", h.authMiddleware(), h.addSpecialistSpecialization)
			specialists.DELETE("/:id/specializations/:specId", h.authMiddleware(), h.removeSpecialistSpecialization)

			specialists.GET("/specialist-actions/appointments", h.authMiddleware(), h.specialistMiddleware(), h.getSpecialistAppointments)

			specialists.POST("/:id/photo", h.authMiddleware(), h.uploadSpecialistPhoto)
			specialists.DELETE("/:id/photo", h.authMiddleware(), h.deleteSpecialistPhoto)

			specialists.POST("/:id/work-experience", h.authMiddleware(), h.addWorkExperienceToSpecialist)
			specialists.POST("/:id/education", h.authMiddleware(), h.addEducationToSpecialist)
		}

		h.initScheduleRoutes(api)

		appointments := api.Group("/appointments")
		appointments.Use(h.authMiddleware())
		{
			appointments.POST("", h.createAppointment)
			appointments.GET("/:id", h.getAppointmentByID)
			appointments.PUT("/:id", h.updateAppointment)
			appointments.DELETE("/:id", h.cancelAppointment)
			appointments.GET("", h.getAppointments)
			appointments.GET("/check-pay", h.checkConsultationType)
		}

		reviews := api.Group("/reviews")
		{
			reviews.GET("", h.getReviews)
			reviews.GET("/:id", h.getReviewByID)
			reviews.GET("/:id/replies", h.getReviewReplies)

			reviews.POST("", h.authMiddleware(), h.createReview)
			reviews.DELETE("/:id", h.authMiddleware(), h.deleteReview)
			reviews.POST("/:id/replies", h.authMiddleware(), h.createReviewReply)
			reviews.DELETE("/replies/:replyId", h.authMiddleware(), h.deleteReviewReply)
		}

		specializations := api.Group("/specializations")
		{
			specializations.GET("", h.getSpecializations)
			specializations.GET("/:id", h.getSpecializationByID)

			specializations.POST("", h.authMiddleware(), h.adminMiddleware(), h.createSpecialization)
			specializations.PUT("/:id", h.authMiddleware(), h.adminMiddleware(), h.updateSpecialization)
			specializations.DELETE("/:id", h.authMiddleware(), h.adminMiddleware(), h.deleteSpecialization)
		}

		education := api.Group("/education")
		{
			education.GET("", h.getEducation)
			education.GET("/:id", h.getEducationByID)

			education.POST("", h.authMiddleware(), h.addEducation)
			education.PUT("/:id", h.authMiddleware(), h.updateEducation)
			education.DELETE("/:id", h.authMiddleware(), h.deleteEducation)
		}

		workExperience := api.Group("/work-experience")
		{
			workExperience.GET("", h.getWorkExperience)
			workExperience.GET("/:id", h.getWorkExperienceByID)

			workExperience.POST("", h.authMiddleware(), h.addWorkExperience)
			workExperience.PUT("/:id", h.authMiddleware(), h.updateWorkExperience)
			workExperience.DELETE("/:id", h.authMiddleware(), h.deleteWorkExperience)
		}
	}
}

func (h *Handler) initScheduleRoutes(api *gin.RouterGroup) {
	schedules := api.Group("/schedules")
	{
		schedules.GET("/free-slots", h.getFreeSlots)
		schedules.GET("/week", h.getScheduleWeek)
		schedules.GET("", h.getSchedules)
		schedules.GET("/:id", h.getScheduleByID)

		schedules.POST("", h.authMiddleware(), h.specialistMiddleware(), h.createSchedule)
		schedules.PUT("", h.authMiddleware(), h.specialistMiddleware(), h.updateSchedule)
		schedules.DELETE("/:id", h.authMiddleware(), h.specialistMiddleware(), h.deleteSchedule)
	}
}

func (h *Handler) getSpecialistAppointments(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	specialist, err := h.services.Specialist.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("ошибка при получении данных специалиста", zap.Error(err))
		notFoundResponse(c, "профиль специалиста не найден")
		return
	}

	statusStr := c.DefaultQuery("status", "")
	var status *domain.AppointmentStatus
	if statusStr != "" {
		appStatus := domain.AppointmentStatus(statusStr)
		status = &appStatus
	}

	dateFrom := c.DefaultQuery("date_from", "")
	var startDate *time.Time
	if dateFrom != "" {
		parsedDate, err := time.Parse("2006-01-02", dateFrom)
		if err == nil {
			startDate = &parsedDate
		}
	}

	dateTo := c.DefaultQuery("date_to", "")
	var endDate *time.Time
	if dateTo != "" {
		parsedDate, err := time.Parse("2006-01-02", dateTo)
		if err == nil {
			parsedDate = parsedDate.Add(24 * time.Hour).Add(-time.Second)
			endDate = &parsedDate
		}
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 0 {
		limit = 20
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	filter := domain.AppointmentFilter{
		SpecialistID: &specialist.ID,
		Status:       status,
		StartDate:    startDate,
		EndDate:      endDate,
		Limit:        limit,
		Offset:       offset,
	}

	appointments, total, err := h.services.Appointment.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("ошибка при получении записей", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка при получении записей")
		return
	}

	page := offset/limit + 1

	paginatedSuccessResponse(c, appointments, total, page, limit)
}
