package rest

import (
	"net/http"
	"strconv"

	"laps/internal/domain"
	"laps/internal/service"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService service.ChatService
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// @Summary Create chat session
// @Description Create a new chat session for an appointment
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.CreateChatSessionDTO true "Chat session data"
// @Success 201 {object} successResponse{data=domain.ChatSession}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /chat/sessions [post]
func (h *ChatHandler) CreateChatSession(c *gin.Context) {
	var dto domain.CreateChatSessionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		badRequestResponse(c, "Invalid request body: " + err.Error())
		return
	}

	session, err := h.chatService.CreateChatSession(c.Request.Context(), dto)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	createdResponse(c, session)
}

// @Summary Get chat session by ID
// @Description Get a specific chat session by ID
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path int true "Chat session ID"
// @Success 200 {object} successResponse{data=domain.ChatSession}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /chat/sessions/{id} [get]
func (h *ChatHandler) GetChatSession(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}
	
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "Invalid session ID")
		return
	}

	session, err := h.chatService.GetChatSessionByID(c.Request.Context(), id, userID)
	if err != nil {
		notFoundResponse(c, err.Error())
		return
	}

	successResponse(c, http.StatusOK, session)
}

// @Summary Get chat session by appointment ID
// @Description Get a chat session by appointment ID
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param appointment_id path int true "Appointment ID"
// @Success 200 {object} successResponse{data=domain.ChatSession}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /chat/sessions/appointment/{appointment_id} [get]
func (h *ChatHandler) GetChatSessionByAppointment(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}
	
	appointmentID, err := strconv.ParseInt(c.Param("appointment_id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "Invalid appointment ID")
		return
	}

	session, err := h.chatService.GetChatSessionByAppointmentID(c.Request.Context(), appointmentID, userID)
	if err != nil {
		notFoundResponse(c, err.Error())
		return
	}

	successResponse(c, http.StatusOK, session)
}

// @Summary List chat sessions
// @Description List chat sessions for the authenticated user
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param specialization_id query int false "Filter by specialization ID"
// @Param status query string false "Filter by status" Enums(pending,active,ended)
// @Param limit query int false "Limit number of results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} paginatedSuccessResponse{data=[]domain.ChatSession}
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /chat/sessions [get]
func (h *ChatHandler) ListChatSessions(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	var filter domain.ChatSessionFilter

	if specializationIDStr := c.Query("specialization_id"); specializationIDStr != "" {
		specializationID, err := strconv.ParseInt(specializationIDStr, 10, 64)
		if err != nil {
			badRequestResponse(c, "Invalid specialization_id")
			return
		}
		filter.SpecializationID = &specializationID
	}

	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.ChatSessionStatus(statusStr)
		filter.Status = &status
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filter.Limit = limit
	filter.Offset = offset

	sessions, totalCount, err := h.chatService.ListChatSessions(c.Request.Context(), userID, filter)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	page := (offset / limit) + 1
	paginatedSuccessResponse(c, sessions, int(totalCount), page, limit)
}

// @Summary Update chat session
// @Description Update a chat session (e.g., change status)
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Chat session ID"
// @Param request body domain.UpdateChatSessionDTO true "Update data"
// @Success 200 {object} successResponse{data=domain.ChatSession}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /chat/sessions/{id} [patch]
func (h *ChatHandler) UpdateChatSession(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}
	
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "Invalid session ID")
		return
	}

	var dto domain.UpdateChatSessionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		badRequestResponse(c, "Invalid request body: " + err.Error())
		return
	}

	session, err := h.chatService.UpdateChatSession(c.Request.Context(), id, dto, userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	successResponse(c, http.StatusOK, session)
}

// @Summary Send message
// @Description Send a message in a chat session
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.CreateChatMessageDTO true "Message data"
// @Success 201 {object} successResponse{data=domain.ChatMessage}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /chat/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	var dto domain.CreateChatMessageDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		badRequestResponse(c, "Invalid request body: " + err.Error())
		return
	}

	// Ensure sender ID matches authenticated user
	dto.SenderID = userID

	message, err := h.chatService.CreateChatMessage(c.Request.Context(), dto, userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	createdResponse(c, message)
}

// @Summary Get messages
// @Description Get messages for a chat session
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param session_id path int true "Chat session ID"
// @Param message_type query string false "Filter by message type" Enums(text,image,file,system)
// @Param limit query int false "Limit number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} paginatedSuccessResponse{data=[]domain.ChatMessage}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /chat/sessions/{session_id}/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}
	
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "Invalid session ID")
		return
	}

	var filter domain.ChatMessageFilter

	if messageTypeStr := c.Query("message_type"); messageTypeStr != "" {
		messageType := domain.MessageType(messageTypeStr)
		filter.Type = &messageType
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filter.Limit = limit
	filter.Offset = offset

	messages, totalCount, err := h.chatService.ListChatMessages(c.Request.Context(), sessionID, userID, filter)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	page := (offset / limit) + 1
	paginatedSuccessResponse(c, messages, int(totalCount), page, limit)
}

// @Summary Mark messages as read
// @Description Mark all unread messages in a session as read
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param session_id path int true "Chat session ID"
// @Success 200 {object} successResponse{data=string}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /chat/sessions/{session_id}/read [post]
func (h *ChatHandler) MarkMessagesAsRead(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}
	
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "Invalid session ID")
		return
	}

	err = h.chatService.MarkMessagesAsRead(c.Request.Context(), sessionID, userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	successResponse(c, http.StatusOK, "Messages marked as read")
}

// @Summary Get unread message count
// @Description Get count of unread messages in a session
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param session_id path int true "Chat session ID"
// @Success 200 {object} successResponse{data=int64}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /chat/sessions/{session_id}/unread [get]
func (h *ChatHandler) GetUnreadMessageCount(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}
	
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		badRequestResponse(c, "Invalid session ID")
		return
	}

	count, err := h.chatService.GetUnreadMessageCount(c.Request.Context(), sessionID, userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	successResponse(c, http.StatusOK, count)
}

// @Summary Get user chat summary
// @Description Get summary of user's chat sessions with unread counts
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Success 200 {object} successResponse{data=map[string]interface{}}
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /chat/summary [get]
func (h *ChatHandler) GetChatSummary(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		unauthorizedResponse(c)
		return
	}

	summary, err := h.chatService.GetUserChatSummary(c.Request.Context(), userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	successResponse(c, http.StatusOK, summary)
}