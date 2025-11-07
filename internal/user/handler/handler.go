package handler

import (
	"errors"
	"log"
	"net/http"

	"logistics-quality-monitor/internal/user/model"
	"logistics-quality-monitor/internal/user/service"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/user")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
		auth.POST("/refresh", h.RefreshToken)
	}
}

func (h *Handler) Register(c *gin.Context) {
	var request model.RegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	authResponse, err := h.service.Register(c.Request.Context(), &request)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "User registered successfully", authResponse)
}

func (h *Handler) Login(c *gin.Context) {
	var request model.LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	authResponse, err := h.service.Login(c.Request.Context(), &request)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Login successful", authResponse)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var request model.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.ForgotPassword(c.Request.Context(), &request); err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "If the email exists, a reset link has been sent", nil)
}

func (h *Handler) ResetPassword(c *gin.Context) {
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

func (h *Handler) RefreshToken(c *gin.Context) {
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

		log.Printf("internal error: %v", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Internal server error")
	}
}
