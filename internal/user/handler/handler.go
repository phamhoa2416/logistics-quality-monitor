package handler

import (
	"errors"
	"net/http"

	"logistics-quality-monitor/internal/logger"
	"logistics-quality-monitor/internal/middleware"
	"logistics-quality-monitor/internal/user/model"
	"logistics-quality-monitor/internal/user/service"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserHandler struct {
	service *service.UserService
}

func NewHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup) {
	user := router.Group("/user")
	{
		user.POST("/register", h.Register)
		user.POST("/login", h.Login)
		user.POST("/forgot-password", h.ForgotPassword)
		user.POST("/reset-password", h.ResetPassword)
		user.POST("/refresh", h.RefreshToken)
		user.POST("/revoke", h.RevokeToken)
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var request model.RegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	request.Email = utils.SanitizeEmail(request.Email)
	request.Username = utils.SanitizeString(request.Username)
	request.FullName = utils.SanitizeString(request.FullName)
	if request.PhoneNumber != nil {
		sanitized := utils.SanitizeString(*request.PhoneNumber)
		request.PhoneNumber = &sanitized
	}
	if request.Address != nil {
		sanitized := utils.SanitizeString(*request.Address)
		request.Address = &sanitized
	}

	authResponse, err := h.service.Register(c.Request.Context(), &request)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "User registered successfully", authResponse)
}

func (h *UserHandler) Login(c *gin.Context) {
	var request model.LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	request.Email = utils.SanitizeEmail(request.Email)

	authResponse, err := h.service.Login(c.Request.Context(), &request)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Login successful", authResponse)
}

func (h *UserHandler) ForgotPassword(c *gin.Context) {
	var request model.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	request.Email = utils.SanitizeEmail(request.Email)

	if err := h.service.ForgotPassword(c.Request.Context(), &request); err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "If the email exists, a reset link has been sent", nil)
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req model.ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), &req); err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password reset successfully", nil)
}

func (h *UserHandler) RefreshToken(c *gin.Context) {
	refreshToken := c.GetHeader("Authorization")
	if refreshToken == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Refresh token required")
		return
	}

	if len(refreshToken) > 7 && refreshToken[:7] == "Bearer " {
		refreshToken = refreshToken[7:]
	}

	tokenPair, err := h.service.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token refreshed successfully", tokenPair)
}

func (h *UserHandler) RevokeToken(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user identifier")
		return
	}

	refreshToken := c.GetHeader("Authorization")
	if refreshToken == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Refresh token required")
		return
	}

	if len(refreshToken) > 7 && refreshToken[:7] == "Bearer " {
		refreshToken = refreshToken[7:]
	}

	if err := h.service.RevokeToken(c.Request.Context(), userUUID, refreshToken); err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token revoked successfully", nil)
}

func respondWithError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, appErrors.ErrUserAlreadyExists):
		utils.ErrorResponse(c, http.StatusConflict, err.Error())
	case errors.Is(err, appErrors.ErrInvalidCredentials),
		errors.Is(err, appErrors.ErrInvalidToken),
		errors.Is(err, appErrors.ErrTokenInvalid),
		errors.Is(err, appErrors.ErrTokenExpired),
		errors.Is(err, appErrors.ErrUnauthorized):
		utils.ErrorResponse(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, appErrors.ErrUserInactive),
		errors.Is(err, appErrors.ErrInsufficientPermissions):
		utils.ErrorResponse(c, http.StatusForbidden, err.Error())
	case errors.Is(err, appErrors.ErrUserNotFound):
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
	default:
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code {
			case "VALIDATION_ERROR", "WEAK_PASSWORD":
				utils.ErrorResponse(c, http.StatusBadRequest, appErr.Message)
			default:
				utils.ErrorResponse(c, http.StatusBadRequest, appErr.Message)
			}
			return
		}

		requestID := middleware.GetRequestID(c)
		logger.Error("Internal server error",
			zap.String("request_id", requestID),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.Error(err),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Internal server error")
	}
}
