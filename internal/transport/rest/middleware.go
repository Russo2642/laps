package rest

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"laps/internal/domain"
)

const (
	authorizationHeader = "Authorization"
	userCtx             = "user"
	userIDCtx           = "user_id"
	userRoleCtx         = "user_role"
)

func (h *Handler) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		path := c.Request.URL.Path
		method := c.Request.Method
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()

		logger := h.logger.With(
			zap.String("path", path),
			zap.String("method", method),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", ip),
			zap.String("user-agent", userAgent),
		)

		if status >= 500 {
			logger.Error("server error")
		} else if status >= 400 {
			logger.Warn("client error")
		} else {
			logger.Info("request processed")
		}
	}
}

func (h *Handler) errorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				h.logger.Error("request error", zap.Error(err))
			}
		}
	}
}

func (h *Handler) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			// Check if origin is in allowed origins list
			allowed := false
			for _, allowedOrigin := range h.config.CORS.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			
			if allowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
				c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Authorization")
				c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			}
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(authorizationHeader)
		if header == "" {
			errorResponse(c, http.StatusUnauthorized, "пустой заголовок авторизации")
			c.Abort()
			return
		}

		headerParts := strings.Split(header, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			errorResponse(c, http.StatusUnauthorized, "неверный формат заголовка авторизации")
			c.Abort()
			return
		}

		token := headerParts[1]
		userID, userRole, err := h.services.Auth.ParseToken(c.Request.Context(), token)
		if err != nil {
			errorResponse(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}

		c.Set(userIDCtx, userID)
		c.Set(userRoleCtx, userRole)

		c.Next()
	}
}

func (h *Handler) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(userRoleCtx)
		if !exists {
			errorResponse(c, http.StatusUnauthorized, "пользователь не авторизован")
			c.Abort()
			return
		}

		role, ok := userRole.(domain.UserRole)
		if !ok || role != domain.UserRoleAdmin {
			errorResponse(c, http.StatusForbidden, "доступ запрещен")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (h *Handler) specialistMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(userRoleCtx)
		if !exists {
			errorResponse(c, http.StatusUnauthorized, "пользователь не авторизован")
			c.Abort()
			return
		}

		role, ok := userRole.(domain.UserRole)
		if !ok || role != domain.UserRoleSpecialist {
			errorResponse(c, http.StatusForbidden, "доступ запрещен, требуется роль специалиста")
			c.Abort()
			return
		}

		c.Next()
	}
}

func getUserID(c *gin.Context) (int64, error) {
	userID, exists := c.Get(userIDCtx)
	if !exists {
		return 0, errors.New("пользователь не авторизован")
	}

	id, ok := userID.(int64)
	if !ok {
		return 0, errors.New("некорректный ID пользователя")
	}

	return id, nil
}

func getUserRole(c *gin.Context) (domain.UserRole, error) {
	userRole, exists := c.Get(userRoleCtx)
	if !exists {
		return "", errors.New("пользователь не авторизован")
	}

	role, ok := userRole.(domain.UserRole)
	if !ok {
		return "", errors.New("некорректная роль пользователя")
	}

	return role, nil
}
