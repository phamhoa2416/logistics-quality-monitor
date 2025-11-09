package handler

import (
	"logistics-quality-monitor/internal/usecase/device"
	"logistics-quality-monitor/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DeviceHandler struct {
	service *device.Service
}

func NewDeviceHandler(service *device.Service) *DeviceHandler {
	return &DeviceHandler{service: service}
}

func (h *DeviceHandler) RegisterRoutes(router *gin.RouterGroup) {
	devices := router.Group("/devices")
	{
		// Public/Shipper routes
		devices.GET("", h.ListDevices)
		devices.GET("/:id", h.GetDevice)
		devices.GET("/hardware/:uid", h.GetDeviceByHardwareUID)
		devices.GET("/available", h.GetAvailableDevices)
	}
}

func (h *DeviceHandler) RegisterAdminRoutes(router *gin.RouterGroup) {
	devices := router.Group("/devices")
	{
		// Admin-only routes
		devices.POST("/create", h.CreateDevice)
		devices.PUT("/:id", h.UpdateDevice)
		devices.DELETE("/:id", h.DeleteDevice)
		devices.POST("/:id/assign-owner", h.AssignOwner)
		devices.POST("/:id/unassign-owner", h.UnassignOwner)
		devices.PUT("/:id/status", h.UpdateStatus)
		devices.PUT("/:id/battery", h.UpdateBattery)
		devices.POST("/bulk-assign", h.BulkAssignOwner)
		devices.GET("/statistics", h.GetStatistics)
	}
}

func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var req device.CreateDeviceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.service.CreateDevice(c.Request.Context(), &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Device created successfully", device)
}

func (h *DeviceHandler) GetDevice(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid device ID")
		return
	}

	device, err := h.service.GetDevice(c.Request.Context(), deviceID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Device retrieved successfully", device)
}

func (h *DeviceHandler) GetDeviceByHardwareUID(c *gin.Context) {
	hardwareUID := c.Param("uid")
	if hardwareUID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Hardware UID required")
		return
	}

	device, err := h.service.GetDeviceByHardwareUID(c.Request.Context(), hardwareUID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Device retrieved successfully", device)
}

func (h *DeviceHandler) ListDevices(c *gin.Context) {
	var filter device.DeviceFilterRequest

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid query parameters")
		return
	}

	devices, err := h.service.ListDevices(c.Request.Context(), &filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Devices retrieved successfully", devices)
}

func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid device ID")
		return
	}

	var req device.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.service.UpdateDevice(c.Request.Context(), deviceID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Device updated successfully", device)
}

func (h *DeviceHandler) AssignOwner(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid device ID")
		return
	}

	var req device.AssignOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.service.AssignOwner(c.Request.Context(), deviceID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Owner assigned successfully", device)
}

func (h *DeviceHandler) UnassignOwner(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid device ID")
		return
	}

	var req device.UnassignOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.service.UnassignOwner(c.Request.Context(), deviceID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Owner unassigned successfully", device)
}

func (h *DeviceHandler) UpdateStatus(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid device ID")
		return
	}

	var req device.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.service.UpdateStatus(c.Request.Context(), deviceID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Status updated successfully", device)
}

func (h *DeviceHandler) UpdateBattery(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid device ID")
		return
	}

	var req device.UpdateBatteryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	device, err := h.service.UpdateBattery(c.Request.Context(), deviceID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Battery updated successfully", device)
}

func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid device ID")
		return
	}

	if err := h.service.DeleteDevice(c.Request.Context(), deviceID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Device retired successfully", nil)
}

func (h *DeviceHandler) BulkAssignOwner(c *gin.Context) {
	var req device.BulkAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.service.BulkAssignOwner(c.Request.Context(), &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Bulk assignment completed", result)
}

func (h *DeviceHandler) GetStatistics(c *gin.Context) {
	stats, err := h.service.GetStatistics(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Statistics retrieved successfully", stats)
}

func (h *DeviceHandler) GetAvailableDevices(c *gin.Context) {
	var shipperID *uuid.UUID
	if shipperIDStr := c.Query("shipper_id"); shipperIDStr != "" {
		id, err := uuid.Parse(shipperIDStr)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipper ID")
			return
		}
		shipperID = &id
	}

	devices, err := h.service.GetAvailableDevices(c.Request.Context(), shipperID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Available devices retrieved", devices)
}
