package handler

import (
	"logistics-quality-monitor/internal/user/model"
	"logistics-quality-monitor/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) RegisterProfileRoutes(router *gin.RouterGroup) {
	profile := router.Group("/profile")
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
		profile.POST("/change-password", h.ChangePassword)
	}
}

func (h *Handler) GetProfile(c *gin.Context) {
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

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
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

func (h *Handler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req model.ChangePasswordRequest
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
