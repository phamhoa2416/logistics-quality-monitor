package handler

import (
	"cargo-tracker/internal/usecase/shipment"
	"cargo-tracker/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShipmentHandler struct {
	service *shipment.Service
}

func NewShipmentHandler(service *shipment.Service) *ShipmentHandler {
	return &ShipmentHandler{service: service}
}

func (h *ShipmentHandler) RegisterRoutes(router *gin.RouterGroup) {
	shipments := router.Group("/shipments")
	{
		// Public routes
		shipments.GET("", h.ListShipments)
		shipments.GET("/:id", h.GetShipment)
		shipments.GET("/statistics", h.GetStatistics)
	}
}

func (h *ShipmentHandler) RegisterCustomerRoutes(router *gin.RouterGroup) {
	shipments := router.Group("/shipments")
	{
		// Customer routes
		shipments.POST("/create-demand", h.CreateDemand)
		//shipments.PUT("/:id", h.UpdateShipment)
		shipments.POST("/:id/cancel", h.CancelShipment)
		//shipments.POST("/:id/rate", h.RateDelivery)
	}
}

func (h *ShipmentHandler) RegisterProviderRoutes(router *gin.RouterGroup) {
	shipments := router.Group("/shipments")
	{
		// Provider routes
		shipments.POST("/:id/post-order", h.PostOrder)
	}
}

func (h *ShipmentHandler) RegisterShipperRoutes(router *gin.RouterGroup) {
	shipments := router.Group("/shipments")
	{
		// Shipper routes
		shipments.POST("/:id/accept", h.AcceptOrder)
		shipments.POST("/:id/confirm-rules", h.ConfirmRules)
		shipments.POST("/:id/start-shipping", h.StartShipping)
		shipments.POST("/:id/complete", h.CompleteDelivery)
		shipments.POST("/:id/report-issue", h.ReportIssue)
	}
}

func (h *ShipmentHandler) CreateDemand(c *gin.Context) {
	var req shipment.CreateDemandRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get customer ID from context (set by auth middleware)
	customerID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	customerUUID, ok := customerID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID format")
		return
	}

	result, err := h.service.CreateDemand(c.Request.Context(), customerUUID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Demand created successfully", result)
}

func (h *ShipmentHandler) PostOrder(c *gin.Context) {
	userRole := c.MustGet("role").(string)

	if userRole != "provider" {
		utils.ErrorResponse(c, http.StatusForbidden, "Only providers can post orders")
		return
	}

	shipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	var req shipment.PostOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get provider ID from context
	providerID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	providerUUID, ok := providerID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID format")
		return
	}

	result, err := h.service.PostOrder(c.Request.Context(), shipmentID, providerUUID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Order posted successfully", result)
}

func (h *ShipmentHandler) AcceptOrder(c *gin.Context) {
	shipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	var req shipment.AcceptOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get shipper ID from context
	shipperID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	shipperUUID, ok := shipperID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID format")
		return
	}

	result, err := h.service.AcceptOrder(c.Request.Context(), shipmentID, shipperUUID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Order accepted successfully", result)
}

func (h *ShipmentHandler) ConfirmRules(c *gin.Context) {
	shipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	// Get shipper ID from context
	shipperID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	shipperUUID, ok := shipperID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID format")
		return
	}

	result, err := h.service.ConfirmRules(c.Request.Context(), shipmentID, shipperUUID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Rules confirmed successfully", result)
}

func (h *ShipmentHandler) StartShipping(c *gin.Context) {
	userRole := c.MustGet("role").(string)

	if userRole != "shipper" {
		utils.ErrorResponse(c, http.StatusForbidden, "Only shippers can start shipping")
		return
	}

	shipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	var req shipment.StartShippingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get shipper ID from context
	shipperID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	shipperUUID, ok := shipperID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID format")
		return
	}

	result, err := h.service.StartShipping(c.Request.Context(), shipmentID, shipperUUID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Shipping started successfully", result)
}

func (h *ShipmentHandler) CompleteDelivery(c *gin.Context) {
	shipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	var req shipment.CompleteDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get shipper ID from context
	shipperID, exists := c.Get("userID")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	shipperUUID, ok := shipperID.(uuid.UUID)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID format")
		return
	}

	result, err := h.service.CompleteDelivery(c.Request.Context(), shipperUUID, shipmentID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Delivery completed successfully", result)
}

func (h *ShipmentHandler) ReportIssue(c *gin.Context) {
	shipmentID, err := uuid.Parse(c.Param("id"))
	reporterID := c.MustGet("userID").(uuid.UUID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	var req shipment.ReportIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.service.ReportIssue(c.Request.Context(), reporterID, shipmentID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Issue reported successfully", result)
}

func (h *ShipmentHandler) CancelShipment(c *gin.Context) {
	shipmentID, err := uuid.Parse(c.Param("id"))
	userID := c.MustGet("userID").(uuid.UUID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	var req shipment.CancelShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.service.CancelShipment(c.Request.Context(), userID, shipmentID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Shipment cancelled successfully", result)
}

//func (h *ShipmentHandler) UpdateShipment(c *gin.Context) {
//	shipmentID, err := uuid.Parse(c.Param("id"))
//	if err != nil {
//		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
//		return
//	}
//
//	var req shipment.UpdateShipmentRequest
//	if err := c.ShouldBindJSON(&req); err != nil {
//		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
//		return
//	}
//
//	// TODO: Implement UpdateShipment in service
//	utils.ErrorResponse(c, http.StatusNotImplemented, "Update shipment not yet implemented")
//}
//
//func (h *ShipmentHandler) RateDelivery(c *gin.Context) {
//	shipmentID, err := uuid.Parse(c.Param("id"))
//	if err != nil {
//		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
//		return
//	}
//
//	var req shipment.RateDeliveryRequest
//	if err := c.ShouldBindJSON(&req); err != nil {
//		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
//		return
//	}
//
//	// TODO: Implement RateDelivery in service
//	utils.ErrorResponse(c, http.StatusNotImplemented, "Rate delivery not yet implemented")
//}
//

func (h *ShipmentHandler) GetShipment(c *gin.Context) {
	shipmentID, err := uuid.Parse(c.Param("id"))
	userID := c.MustGet("userID").(uuid.UUID)

	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	result, err := h.service.GetShipment(c.Request.Context(), userID, shipmentID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Shipment retrieved successfully", result)
}

func (h *ShipmentHandler) ListShipments(c *gin.Context) {
	var filter shipment.ShipmentFilterRequest
	userID := c.MustGet("userID").(uuid.UUID)
	userRole := c.MustGet("role").(string)

	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid query parameters")
		return
	}

	result, err := h.service.ListShipments(c.Request.Context(), userID, userRole, &filter)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Shipments retrieved successfully", result)
}

func (h *ShipmentHandler) GetStatistics(c *gin.Context) {
	result, err := h.service.GetStatistics(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Statistics retrieved successfully", result)
}
