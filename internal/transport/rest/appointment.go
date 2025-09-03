package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"laps/internal/domain"
)

// @Summary Создать запись на консультацию
// @Description Создает новую запись на консультацию к специалисту
// @Tags Записи
// @Accept json
// @Produce json
// @Param input body domain.CreateAppointmentDTO true "Данные для записи на консультацию"
// @Success 201 {object} map[string]interface{} "ID созданной записи"
// @Failure 400 {object} errorResponseBody "Ошибка валидации или выбранное время недоступно"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /appointments [post]
func (h *Handler) createAppointment(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.logger.Warn("ошибка получения ID пользователя", zap.Error(err))
		unauthorizedResponse(c)
		return
	}

	var req domain.CreateAppointmentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("неверный формат данных", zap.Error(err))
		badRequestResponse(c, "неверный формат данных")
		return
	}

	id, err := h.services.Appointment.Create(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Error("ошибка создания записи на консультацию", zap.Error(err))
		badRequestResponse(c, "ошибка создания записи на консультацию")
		return
	}

	createdResponse(c, gin.H{"id": id})
}

// @Summary Получить запись по ID
// @Description Возвращает информацию о записи на консультацию по указанному ID
// @Tags Записи
// @Accept json
// @Produce json
// @Param id path int true "ID записи"
// @Success 200 {object} domain.Appointment "Данные записи"
// @Failure 400 {object} errorResponseBody "Неверный формат ID"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Запись не найдена"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /appointments/{id} [get]
func (h *Handler) getAppointmentByID(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.logger.Warn("ошибка получения ID пользователя", zap.Error(err))
		unauthorizedResponse(c)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Warn("неверный формат ID", zap.Error(err))
		badRequestResponse(c, "неверный формат ID")
		return
	}

	appointment, err := h.services.Appointment.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка получения записи", zap.Error(err), zap.Int64("id", id))
		notFoundResponse(c, "запись не найдена")
		return
	}

	userRole, _ := getUserRole(c)
	specialist, err := h.services.Specialist.GetByUserID(c.Request.Context(), userID)
	isSpecialist := err == nil && specialist != nil

	if appointment.ClientID != userID &&
		(isSpecialist && specialist.ID != appointment.SpecialistID) &&
		userRole != domain.UserRoleAdmin {
		h.logger.Warn("попытка несанкционированного доступа", zap.Int64("userID", userID))
		forbiddenResponse(c)
		return
	}

	successResponse(c, http.StatusOK, appointment)
}

// @Summary Обновить запись
// @Description Обновляет информацию о записи на консультацию
// @Tags Записи
// @Accept json
// @Produce json
// @Param id path int true "ID записи"
// @Param input body domain.UpdateAppointmentDTO true "Данные для обновления записи"
// @Success 200 {object} messageResponseType "Сообщение об успешном обновлении"
// @Failure 400 {object} errorResponseBody "Ошибка валидации или выбранное время недоступно"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Запись не найдена"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /appointments/{id} [put]
func (h *Handler) updateAppointment(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.logger.Warn("ошибка получения ID пользователя", zap.Error(err))
		unauthorizedResponse(c)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Warn("неверный формат ID", zap.Error(err))
		badRequestResponse(c, "неверный формат ID")
		return
	}

	appointment, err := h.services.Appointment.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка получения записи", zap.Error(err), zap.Int64("id", id))
		notFoundResponse(c, "запись не найдена")
		return
	}

	userRole, _ := getUserRole(c)
	specialist, err := h.services.Specialist.GetByUserID(c.Request.Context(), userID)
	isSpecialist := err == nil && specialist != nil

	if appointment.ClientID != userID &&
		(isSpecialist && specialist.ID != appointment.SpecialistID) &&
		userRole != domain.UserRoleAdmin {
		h.logger.Warn("попытка несанкционированного доступа", zap.Int64("userID", userID))
		forbiddenResponse(c)
		return
	}

	var req domain.UpdateAppointmentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("неверный формат данных", zap.Error(err))
		badRequestResponse(c, "неверный формат данных")
		return
	}

	err = h.services.Appointment.Update(c.Request.Context(), id, req)
	if err != nil {
		h.logger.Error("ошибка обновления записи", zap.Error(err))
		badRequestResponse(c, "ошибка обновления записи")
		return
	}

	messageResponse(c, http.StatusOK, "запись успешно обновлена")
}

// @Summary Отменить запись
// @Description Отменяет запись на консультацию
// @Tags Записи
// @Accept json
// @Produce json
// @Param id path int true "ID записи"
// @Success 200 {object} messageResponseType "Сообщение об успешной отмене"
// @Failure 400 {object} errorResponseBody "Неверный формат ID или ошибка отмены"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Запись не найдена"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /appointments/{id} [delete]
func (h *Handler) cancelAppointment(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.logger.Warn("ошибка получения ID пользователя", zap.Error(err))
		unauthorizedResponse(c)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Warn("неверный формат ID", zap.Error(err))
		badRequestResponse(c, "неверный формат ID")
		return
	}

	appointment, err := h.services.Appointment.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка получения записи", zap.Error(err), zap.Int64("id", id))
		notFoundResponse(c, "запись не найдена")
		return
	}

	userRole, _ := getUserRole(c)
	specialist, err := h.services.Specialist.GetByUserID(c.Request.Context(), userID)
	isSpecialist := err == nil && specialist != nil

	if appointment.ClientID != userID &&
		(isSpecialist && specialist.ID != appointment.SpecialistID) &&
		userRole != domain.UserRoleAdmin {
		h.logger.Warn("попытка несанкционированного доступа", zap.Int64("userID", userID))
		forbiddenResponse(c)
		return
	}

	err = h.services.Appointment.Cancel(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка отмены записи", zap.Error(err))
		badRequestResponse(c, "ошибка отмены записи")
		return
	}

	messageResponse(c, http.StatusOK, "запись успешно отменена")
}

// @Summary Получить список записей
// @Description Возвращает список записей на консультации с фильтрацией и пагинацией
// @Tags Записи
// @Accept json
// @Produce json
// @Param limit query int false "Лимит записей на странице (по умолчанию 20)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Param client_id query int false "ID клиента (только для админов)"
// @Param specialist_id query int false "ID специалиста (только для админов)"
// @Param status query string false "Статус записи"
// @Param start_date query string false "Начальная дата (YYYY-MM-DD)"
// @Param end_date query string false "Конечная дата (YYYY-MM-DD)"
// @Success 200 {object} paginatedResponse "Список записей с пагинацией"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /appointments [get]
func (h *Handler) getAppointments(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.logger.Warn("ошибка получения ID пользователя", zap.Error(err))
		unauthorizedResponse(c)
		return
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
		Limit:  limit,
		Offset: offset,
	}

	specialist, err := h.services.Specialist.GetByUserID(c.Request.Context(), userID)
	isSpecialist := err == nil && specialist != nil

	if clientIDStr := c.Query("client_id"); clientIDStr != "" {
		clientID, err := strconv.ParseInt(clientIDStr, 10, 64)
		if err == nil {
			filter.ClientID = &clientID
		}
	}

	if specialistIDStr := c.Query("specialist_id"); specialistIDStr != "" {
		specialistID, err := strconv.ParseInt(specialistIDStr, 10, 64)
		if err == nil {
			filter.SpecialistID = &specialistID
		}
	}

	if filter.ClientID == nil && filter.SpecialistID == nil {
		if isSpecialist {
			filter.SpecialistID = &specialist.ID
		} else {
			filter.ClientID = &userID
		}
	}

	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.AppointmentStatus(statusStr)
		filter.Status = &status
	}

	if excludeStatusStr := c.Query("exclude_status"); excludeStatusStr != "" {
		excludeStatus := domain.AppointmentStatus(excludeStatusStr)
		filter.ExcludeStatus = &excludeStatus
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			filter.StartDate = &startDate
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			filter.EndDate = &endDate
		}
	}

	appointments, total, err := h.services.Appointment.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("ошибка получения списка записей", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка получения списка записей")
		return
	}

	page := offset/limit + 1
	paginatedSuccessResponse(c, appointments, total, page, limit)
}

// @Summary Проверить тип консультации
// @Description Проверяет, является ли консультация первичной или вторичной для клиента у указанного специалиста
// @Tags Записи
// @Accept json
// @Produce json
// @Param specialist_id query int true "ID специалиста"
// @Success 200 {object} map[string]string "Тип консультации"
// @Failure 400 {object} errorResponseBody "Ошибка валидации"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /appointments/check-pay [get]
func (h *Handler) checkConsultationType(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		h.logger.Warn("ошибка получения ID пользователя", zap.Error(err))
		unauthorizedResponse(c)
		return
	}

	specialistIDStr := c.Query("specialist_id")
	if specialistIDStr == "" {
		h.logger.Warn("не указан ID специалиста")
		badRequestResponse(c, "не указан ID специалиста")
		return
	}

	specialistID, err := strconv.ParseInt(specialistIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("неверный формат ID специалиста", zap.Error(err))
		badRequestResponse(c, "неверный формат ID специалиста")
		return
	}

	consultationType, err := h.services.Appointment.CheckConsultationType(c.Request.Context(), userID, specialistID)
	if err != nil {
		h.logger.Error("ошибка при определении типа консультации", zap.Error(err))
		internalServerErrorResponse(c)
		return
	}

	successResponse(c, http.StatusOK, gin.H{
		"consultation_type": string(consultationType),
	})
}
