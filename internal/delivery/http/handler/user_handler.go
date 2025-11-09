package handler

import (
	"errors"
	"logistics-quality-monitor/internal/usecase/user"
	"net/http"

	"logistics-quality-monitor/internal/logger"
	"logistics-quality-monitor/internal/middleware"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserHandler struct {
	service *user.Service
}

func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/user")
	{
		userGroup.POST("/register", h.Register)
		userGroup.POST("/login", h.Login)
		userGroup.POST("/forgot-password", h.ForgotPassword)
		userGroup.POST("/reset-password", h.ResetPassword)
		userGroup.POST("/refresh", h.RefreshToken)
		userGroup.POST("/revoke", h.RevokeToken)
	}
}

func (h *UserHandler) RegisterAdminRoutes(router *gin.RouterGroup) {
	admin := router.Group("")
	{
		admin.GET("/users", h.GetAllUsers)
		admin.DELETE("/users/:user_id", h.DeleteUser)
	}
}

func (h *UserHandler) RegisterProfileRoutes(router *gin.RouterGroup) {
	profile := router.Group("/profile")
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
		profile.POST("/change-password", h.ChangePassword)
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req user.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize input
	req.Email = utils.SanitizeEmail(req.Email)
	req.Username = utils.SanitizeString(req.Username)
	req.FullName = utils.SanitizeString(req.FullName)
	if req.PhoneNumber != nil {
		sanitized := utils.SanitizeString(*req.PhoneNumber)
		req.PhoneNumber = &sanitized
	}
	if req.Address != nil {
		sanitized := utils.SanitizeString(*req.Address)
		req.Address = &sanitized
	}

	authResponse, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "User registered successfully", authResponse)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req user.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Email = utils.SanitizeEmail(req.Email)

	authResponse, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Login successful", authResponse)
}

func (h *UserHandler) ForgotPassword(c *gin.Context) {
	var req user.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Email = utils.SanitizeEmail(req.Email)

	if err := h.service.ForgotPassword(c.Request.Context(), &req); err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "If the email exists, a reset link has been sent", nil)
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req user.ResetPasswordRequest

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

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	users, err := h.service.GetAllUsers(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get users")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", users)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = h.service.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User deleted successfully", nil)
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

func (h *UserHandler) GetProfile(c *gin.Context) {
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

	profile, err := h.service.GetProfile(c.Request.Context(), userUUID)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Profile retrieved successfully", profile)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req user.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize input
	if req.FullName != nil {
		sanitized := utils.SanitizeString(*req.FullName)
		req.FullName = &sanitized
	}
	if req.PhoneNumber != nil {
		sanitized := utils.SanitizePhone(*req.PhoneNumber)
		req.PhoneNumber = &sanitized
	}
	if req.Address != nil {
		sanitized := utils.SanitizeText(*req.Address)
		req.Address = &sanitized
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user identifier")
		return
	}

	profile, err := h.service.UpdateProfile(c.Request.Context(), userUUID, &req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Profile updated successfully", profile)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req user.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user identifier")
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userUUID, &req); err != nil {
		respondWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password changed successfully", nil)
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
