package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"laps/internal/domain"
)

// @Summary Получить список специализаций
// @Description Возвращает список специализаций с фильтрацией и пагинацией
// @Tags Специализации
// @Accept json
// @Produce json
// @Param limit query int false "Лимит записей на странице (по умолчанию 20)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Param type query string false "Тип специалиста (lawyer, psychologist, nutritionist, trainer)"
// @Param is_active query boolean false "Фильтр по активности"
// @Param search query string false "Поисковый запрос"
// @Param specialist_id query int false "ID специалиста для фильтрации специализаций"
// @Success 200 {object} paginatedResponse "Список специализаций с пагинацией"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Router /specializations [get]
func (h *Handler) getSpecializations(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	filter := domain.SpecializationFilter{
		Limit:  limit,
		Offset: offset,
	}

	if specType := c.Query("type"); specType != "" {
		specTypeEnum := domain.SpecialistType(specType)
		filter.Type = &specTypeEnum
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	if search := c.Query("search"); search != "" {
		filter.SearchTerm = &search
	}

	if specialistIDStr := c.Query("specialist_id"); specialistIDStr != "" {
		specialistID, err := strconv.ParseInt(specialistIDStr, 10, 64)
		if err == nil {
			filter.SpecialistID = &specialistID
		}
	}

	specializations, total, err := h.services.Specialization.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("ошибка получения списка специализаций", zap.Error(err))
		internalServerErrorResponse(c)
		return
	}

	page := offset/limit + 1
	paginatedSuccessResponse(c, specializations, total, page, limit)
}

// @Summary Получить специализацию по ID
// @Description Возвращает информацию о специализации по указанному ID
// @Tags Специализации
// @Accept json
// @Produce json
// @Param id path int true "ID специализации"
// @Success 200 {object} domain.Specialization "Данные специализации"
// @Failure 400 {object} errorResponseBody "Неверный формат ID"
// @Failure 404 {object} errorResponseBody "Специализация не найдена"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Router /specializations/{id} [get]
func (h *Handler) getSpecializationByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Warn("неверный формат ID", zap.Error(err))
		badRequestResponse(c, "неверный формат ID")
		return
	}

	specialization, err := h.services.Specialization.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка получения специализации", zap.Error(err), zap.Int64("id", id))
		notFoundResponse(c, "специализация не найдена")
		return
	}

	successResponse(c, http.StatusOK, specialization)
}

// @Summary Создать специализацию
// @Description Создает новую специализацию (только для администраторов)
// @Tags Специализации
// @Accept json
// @Produce json
// @Param input body domain.CreateSpecializationDTO true "Данные специализации"
// @Success 201 {object} map[string]interface{} "ID созданной специализации"
// @Failure 400 {object} errorResponseBody "Ошибка валидации"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specializations [post]
func (h *Handler) createSpecialization(c *gin.Context) {
	var req domain.CreateSpecializationDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("неверный формат данных", zap.Error(err))
		badRequestResponse(c, "неверный формат данных")
		return
	}

	id, err := h.services.Specialization.Create(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("ошибка создания специализации", zap.Error(err))
		internalServerErrorResponse(c)
		return
	}

	createdResponse(c, gin.H{"id": id})
}

// @Summary Обновить специализацию
// @Description Обновляет информацию о специализации (только для администраторов)
// @Tags Специализации
// @Accept json
// @Produce json
// @Param id path int true "ID специализации"
// @Param input body domain.UpdateSpecializationDTO true "Новые данные специализации"
// @Success 200 {object} messageResponseType "Сообщение об успешном обновлении"
// @Failure 400 {object} errorResponseBody "Ошибка валидации"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Специализация не найдена"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specializations/{id} [put]
func (h *Handler) updateSpecialization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Warn("неверный формат ID", zap.Error(err))
		badRequestResponse(c, "неверный формат ID")
		return
	}

	var req domain.UpdateSpecializationDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("неверный формат данных", zap.Error(err))
		badRequestResponse(c, "неверный формат данных")
		return
	}

	err = h.services.Specialization.Update(c.Request.Context(), id, req)
	if err != nil {
		h.logger.Error("ошибка обновления специализации", zap.Error(err), zap.Int64("id", id))
		notFoundResponse(c, "специализация не найдена или ошибка обновления")
		return
	}

	messageResponse(c, http.StatusOK, "специализация успешно обновлена")
}

// @Summary Удалить специализацию
// @Description Удаляет специализацию (только для администраторов)
// @Tags Специализации
// @Accept json
// @Produce json
// @Param id path int true "ID специализации"
// @Success 204 {object} nil "Специализация успешно удалена"
// @Failure 400 {object} errorResponseBody "Неверный формат ID"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Специализация не найдена"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specializations/{id} [delete]
func (h *Handler) deleteSpecialization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.logger.Warn("неверный формат ID", zap.Error(err))
		badRequestResponse(c, "неверный формат ID")
		return
	}

	err = h.services.Specialization.Delete(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка удаления специализации", zap.Error(err), zap.Int64("id", id))
		notFoundResponse(c, "специализация не найдена или ошибка удаления")
		return
	}

	noContentResponse(c)
}
