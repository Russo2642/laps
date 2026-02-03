package rest

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"laps/internal/domain"
)

// @Summary Получить список специалистов
// @Description Возвращает список специалистов с фильтрацией и пагинацией
// @Tags Специалисты
// @Accept json
// @Produce json
// @Param limit query int false "Лимит записей на странице (по умолчанию 20)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Param type query string false "Тип специалиста (lawyer, psychologist, nutritionist, trainer)"
// @Param specialization_id query integer false "ID специализации"
// @Param date query string false "Дата для получения свободных слотов (YYYY-MM-DD)"
// @Success 200 {object} paginatedResponse "Список специалистов с пагинацией"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Router /specialists [get]
func (h *Handler) getSpecialists(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	var specialistType *domain.SpecialistType
	if typeStr := c.Query("type"); typeStr != "" {
		t := domain.SpecialistType(typeStr)
		specialistType = &t
	}

	var specializationID *int64
	if specializationIDStr := c.Query("specialization_id"); specializationIDStr != "" {
		id, err := strconv.ParseInt(specializationIDStr, 10, 64)
		if err == nil {
			specializationID = &id
		} else {
			h.logger.Warn("неверный формат specialization_id", zap.Error(err))
		}
	}

	specialists, total, err := h.services.Specialist.List(c.Request.Context(), specialistType, specializationID, limit, offset)
	if err != nil {
		h.logger.Error("ошибка при получении списка специалистов", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка при получении списка специалистов")
		return
	}

	date := c.Query("date")
	if date != "" {
		// Проверка формата даты
		_, err := time.Parse("2006-01-02", date)
		if err != nil {
			h.logger.Warn("неверный формат даты", zap.Error(err))
			badRequestResponse(c, "неверный формат даты, ожидается YYYY-MM-DD")
			return
		}

		// Получаем свободные слоты для каждого специалиста
		for i, specialist := range specialists {
			slots, err := h.services.Schedule.GenerateTimeSlots(c.Request.Context(), specialist.ID, date)
			if err != nil {
				h.logger.Error("ошибка получения свободных слотов для специалиста",
					zap.Int64("specialistID", specialist.ID), zap.Error(err))
				// Пропускаем ошибку для конкретного специалиста, чтобы не влиять на общий список
				continue
			}
			specialists[i].FreeSlots = slots
		}
	}

	page := offset/limit + 1
	paginatedSuccessResponse(c, specialists, total, page, limit)
}

// @Summary Получить специалиста по ID
// @Description Возвращает информацию о специалисте по указанному ID
// @Tags Специалисты
// @Accept json
// @Produce json
// @Param id path int true "ID специалиста"
// @Success 200 {object} domain.Specialist "Данные специалиста"
// @Failure 400 {object} errorResponseBody "Неверный формат ID"
// @Failure 404 {object} errorResponseBody "Специалист не найден"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Router /specialists/{id} [get]
func (h *Handler) getSpecialistByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "неверный формат ID")
		return
	}

	specialist, err := h.services.Specialist.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка при получении специалиста", zap.Int64("id", id), zap.Error(err))
		notFoundResponse(c, "специалист не найден")
		return
	}

	successResponse(c, http.StatusOK, specialist)
}

// @Summary Создать специалиста
// @Description Создает или обновляет профиль специалиста для пользователя (при регистрации профиль создается автоматически)
// @Tags Специалисты
// @Accept json
// @Produce json
// @Param input body domain.CreateSpecialistDTO true "Данные специалиста"
// @Success 201 {object} map[string]interface{} "ID созданного специалиста"
// @Failure 400 {object} errorResponseBody "Ошибка валидации"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specialists [post]
func (h *Handler) createSpecialist(c *gin.Context) {
	var req domain.CreateSpecialistDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("неверный формат данных", zap.Error(err))
		badRequestResponse(c, "неверный формат данных")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	userRole, err := getUserRole(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	if userRole != domain.UserRoleSpecialist && userRole != domain.UserRoleAdmin {
		forbiddenResponse(c)
		return
	}

	// Если текущий пользователь админ и в запросе указан UserID, используем его
	// Иначе используем ID текущего пользователя
	targetUserID := userID
	if userRole == domain.UserRoleAdmin && req.UserID > 0 {
		user, err := h.services.User.GetByID(c.Request.Context(), req.UserID)
		if err != nil {
			h.logger.Error("ошибка при получении пользователя", zap.Error(err))
			badRequestResponse(c, "пользователь не найден")
			return
		}

		if user.Role != domain.UserRoleSpecialist {
			badRequestResponse(c, "указанный пользователь не имеет роли специалиста")
			return
		}

		targetUserID = req.UserID
	} else {
		targetUserID = userID

		user, err := h.services.User.GetByID(c.Request.Context(), userID)
		if err != nil {
			h.logger.Error("ошибка при получении пользователя", zap.Error(err))
			errorResponse(c, http.StatusInternalServerError, "ошибка при получении данных пользователя")
			return
		}

		if user.Role != domain.UserRoleSpecialist {
			badRequestResponse(c, "у вас нет прав для создания профиля специалиста")
			return
		}
	}

	if req.SpecializationID == nil {
		badRequestResponse(c, "поле 'specialization_id' обязательно для заполнения")
		return
	}

	id, err := h.services.Specialist.Create(c.Request.Context(), targetUserID, req)
	if err != nil {
		h.logger.Error("ошибка при создании специалиста", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	createdResponse(c, map[string]interface{}{
		"id": id,
	})
}

// @Summary Обновить специалиста
// @Description Обновляет информацию о специалисте
// @Tags Специалисты
// @Accept json
// @Produce json
// @Param id path int true "ID специалиста"
// @Param input body domain.UpdateSpecialistDTO true "Новые данные специалиста"
// @Success 204 {object} nil "Данные успешно обновлены"
// @Failure 400 {object} errorResponseBody "Ошибка валидации"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Специалист не найден"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specialists/{id} [put]
func (h *Handler) updateSpecialist(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "неверный формат ID")
		return
	}

	specialist, err := h.services.Specialist.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("специалист не найден", zap.Int64("id", id), zap.Error(err))
		notFoundResponse(c, "специалист не найден")
		return
	}

	currentUserID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	userRole, err := getUserRole(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	if specialist.UserID != currentUserID && userRole != domain.UserRoleAdmin {
		forbiddenResponse(c)
		return
	}

	var req domain.UpdateSpecialistDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("неверный формат данных", zap.Error(err))
		badRequestResponse(c, "неверный формат данных")
		return
	}

	h.logger.Debug("запрос на обновление специалиста",
		zap.Int64("id", id),
		zap.Any("request", req))

	err = h.services.Specialist.Update(c.Request.Context(), id, req)
	if err != nil {
		h.logger.Error("ошибка при обновлении специалиста", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	updatedSpecialist, err := h.services.Specialist.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка при получении обновленного специалиста", zap.Error(err))
	}

	successResponse(c, http.StatusOK, updatedSpecialist)
}

// @Summary Получить отзывы о специалисте
// @Description Возвращает список отзывов о специалисте с пагинацией (перенаправляет на /reviews)
// @Tags Специалисты,Отзывы
// @Accept json
// @Produce json
// @Param id path int true "ID специалиста"
// @Param limit query int false "Лимит записей на странице (по умолчанию 10)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Success 200 {object} paginatedResponse "Список отзывов с пагинацией"
// @Failure 400 {object} errorResponseBody "Неверный формат ID"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Router /specialists/{id}/reviews [get]
func (h *Handler) getSpecialistReviewsRedirect(c *gin.Context) {
	id := c.Param("id")
	limit := c.DefaultQuery("limit", "10")
	offset := c.DefaultQuery("offset", "0")

	targetURL := fmt.Sprintf("/api/v1/reviews?specialist_id=%s&limit=%s&offset=%s", id, limit, offset)
	c.Redirect(http.StatusPermanentRedirect, targetURL)
}

// @Summary Получить профиль специалиста текущего пользователя
// @Description Возвращает профиль специалиста для текущего авторизованного пользователя
// @Tags Специалисты
// @Accept json
// @Produce json
// @Success 200 {object} domain.Specialist "Данные специалиста"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 404 {object} errorResponseBody "Профиль специалиста не найден"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specialists/me [get]
func (h *Handler) getMySpecialistProfile(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	specialist, err := h.services.Specialist.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("ошибка при получении профиля специалиста", zap.Int64("userID", userID), zap.Error(err))
		notFoundResponse(c, "профиль специалиста не найден")
		return
	}

	successResponse(c, http.StatusOK, specialist)
}

// @Summary Загрузить фотографию профиля
// @Description Загружает и устанавливает фотографию профиля специалиста
// @Tags Специалисты
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID специалиста"
// @Param photo formData file true "Файл изображения"
// @Success 200 {object} successResponseBody "Фотография успешно загружена"
// @Failure 400 {object} errorResponseBody "Неверный формат ID, отсутствует файл или он не является изображением"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Специалист не найден"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specialists/{id}/photo [post]
func (h *Handler) uploadSpecialistPhoto(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "неверный формат ID")
		return
	}

	specialist, err := h.services.Specialist.GetByID(c.Request.Context(), id)
	if err != nil {
		notFoundResponse(c, "специалист не найден")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	userRole, err := getUserRole(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	if specialist.UserID != userID && userRole != domain.UserRoleAdmin {
		forbiddenResponse(c)
		return
	}

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		h.logger.Warn("ошибка получения файла из формы", zap.Error(err))
		badRequestResponse(c, "не удалось получить файл")
		return
	}
	defer file.Close()

	const maxSize = 5 * 1024 * 1024
	if header.Size > maxSize {
		badRequestResponse(c, "файл слишком большой (максимальный размер 5 MB)")
		return
	}

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		h.logger.Error("ошибка чтения файла", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка чтения файла")
		return
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		h.logger.Error("ошибка сброса указателя файла", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка чтения файла")
		return
	}

	fileType := http.DetectContentType(buffer)
	if !strings.HasPrefix(fileType, "image/") {
		badRequestResponse(c, "файл не является изображением")
		return
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("ошибка чтения файла", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка чтения файла")
		return
	}

	err = h.services.Specialist.UploadProfilePhoto(c.Request.Context(), id, fileData, header.Filename)
	if err != nil {
		h.logger.Error("ошибка загрузки фото в хранилище", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка загрузки фотографии")
		return
	}

	successResponse(c, http.StatusOK, map[string]string{
		"message": "фотография профиля успешно загружена",
	})
}

// @Summary Удалить фотографию профиля
// @Description Удаляет фотографию профиля специалиста
// @Tags Специалисты
// @Produce json
// @Param id path int true "ID специалиста"
// @Success 200 {object} successResponseBody "Фотография успешно удалена"
// @Failure 400 {object} errorResponseBody "Неверный формат ID"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Специалист не найден"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specialists/{id}/photo [delete]
func (h *Handler) deleteSpecialistPhoto(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "неверный формат ID")
		return
	}

	specialist, err := h.services.Specialist.GetByID(c.Request.Context(), id)
	if err != nil {
		notFoundResponse(c, "специалист не найден")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	userRole, err := getUserRole(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	if specialist.UserID != userID && userRole != domain.UserRoleAdmin {
		forbiddenResponse(c)
		return
	}

	err = h.services.Specialist.DeleteProfilePhoto(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка удаления фото", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка удаления фотографии")
		return
	}

	successResponse(c, http.StatusOK, map[string]string{
		"message": "фотография профиля успешно удалена",
	})
}

// @Summary Удалить специалиста
// @Description Удаляет профиль специалиста
// @Tags Специалисты
// @Produce json
// @Param id path int true "ID специалиста"
// @Success 204 {object} nil "Профиль специалиста удален"
// @Failure 400 {object} errorResponseBody "Неверный формат ID"
// @Failure 401 {object} errorResponseBody "Не авторизован"
// @Failure 403 {object} errorResponseBody "Доступ запрещен"
// @Failure 404 {object} errorResponseBody "Специалист не найден"
// @Failure 500 {object} errorResponseBody "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /specialists/{id} [delete]
func (h *Handler) deleteSpecialist(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "неверный формат ID")
		return
	}

	specialist, err := h.services.Specialist.GetByID(c.Request.Context(), id)
	if err != nil {
		notFoundResponse(c, "специалист не найден")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	userRole, err := getUserRole(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	if specialist.UserID != userID && userRole != domain.UserRoleAdmin {
		forbiddenResponse(c)
		return
	}

	if specialist.ProfilePhotoURL != "" {
		err = h.services.Specialist.DeleteProfilePhoto(c.Request.Context(), id)
		if err != nil {
			h.logger.Error("ошибка удаления фото при удалении профиля", zap.Error(err))
		}
	}

	err = h.services.Specialist.Delete(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("ошибка удаления специалиста", zap.Error(err))
		errorResponse(c, http.StatusInternalServerError, "ошибка удаления специалиста")
		return
	}

	c.Status(http.StatusNoContent)
}
