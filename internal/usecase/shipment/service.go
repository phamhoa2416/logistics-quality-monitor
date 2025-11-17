package shipment

//
import (
	"context"
	"fmt"
	domainDevice "logistics-quality-monitor/internal/domain/device"
	domainShipment "logistics-quality-monitor/internal/domain/shipment"
	domainUser "logistics-quality-monitor/internal/domain/user"
	"logistics-quality-monitor/internal/logger"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service implements shipment use cases
type Service struct {
	shipmentRepo domainShipment.Repository
	userRepo     domainUser.Repository
	deviceRepo   domainDevice.Repository
}

// NewService creates a new shipment service
func NewService(
	shipmentRepo domainShipment.Repository,
	userRepo domainUser.Repository,
	deviceRepo domainDevice.Repository,
) *Service {
	return &Service{
		shipmentRepo: shipmentRepo,
		userRepo:     userRepo,
		deviceRepo:   deviceRepo,
	}
}

// Step 1: Customer creates demand

func (s *Service) CreateDemand(ctx context.Context, customerID uuid.UUID, req *CreateDemandRequest) (*ShipmentResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Validate parties
	if err := ValidateParties(ctx, s.userRepo, customerID, req.ProviderID, nil); err != nil {
		return nil, err
	}

	// Validate time range
	if err := ValidateTimeRange(req.EstimatedPickupAt, req.EstimatedDeliveryAt); err != nil {
		return nil, err
	}

	// Create domain entity
	shipment := &domainShipment.Shipment{
		CustomerID:          customerID,
		ProviderID:          req.ProviderID,
		Status:              domainShipment.StatusDemandCreated,
		GoodsDescription:    req.GoodsDescription,
		GoodsValue:          req.GoodsValue,
		GoodsWeight:         req.GoodsWeight,
		PickupAddress:       req.PickupAddress,
		DeliveryAddress:     req.DeliveryAddress,
		EstimatedPickupAt:   req.EstimatedPickupAt,
		EstimatedDeliveryAt: req.EstimatedDeliveryAt,
		CustomerNotes:       req.CustomerNotes,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Save shipment
	if err := s.shipmentRepo.Create(ctx, shipment); err != nil {
		return nil, err
	}

	// Get created shipment
	createdShipment, err := s.shipmentRepo.GetByID(ctx, shipment.ID)
	if err != nil {
		return nil, err
	}

	logger.Info("Shipment demand created",
		zap.String("shipment_id", createdShipment.ID.String()),
		zap.String("customer_id", customerID.String()),
		zap.String("provider_id", req.ProviderID.String()),
		zap.String("event", "shipment_demand_created"),
	)

	rules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, createdShipment.ID)
	return ToShipmentResponse(createdShipment, rules), nil
}

// Step 2: Provider posts order to marketplace with quality rules

func (s *Service) PostOrder(ctx context.Context, shipmentID, providerID uuid.UUID, req *PostOrderRequest) (*ShipmentResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Validate shipping rules
	if err := ValidateShippingRules(req); err != nil {
		return nil, err
	}

	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Verify provider owns this shipment
	if shipment.ProviderID != providerID {
		return nil, appErrors.NewAppError("UNAUTHORIZED", "Provider does not own this shipment", nil)
	}

	// Validate status transition
	if err := ValidateStatusTransition(shipment.Status, domainShipment.StatusOrderPosted); err != nil {
		return nil, err
	}

	// Create shipping rules
	rules := &domainShipment.ShippingRules{
		ShipmentID:            shipmentID,
		ReportCycleSec:        req.ReportCycleSec,
		TempMin:               req.TempMin,
		TempMax:               req.TempMax,
		HumidityMin:           req.HumidityMin,
		HumidityMax:           req.HumidityMax,
		LightMax:              req.LightMax,
		TiltMaxAngle:          req.TiltMaxAngle,
		ImpactThresholdG:      req.ImpactThresholdG,
		EnablePredictiveAlert: req.EnablePredictiveAlert,
		AlertBufferTimeMin:    req.AlertBufferTimeMin,
		SetByProviderID:       providerID,
		SetAt:                 time.Now(),
	}

	if err := s.shipmentRepo.CreateRules(ctx, rules); err != nil {
		return nil, err
	}

	// Update shipment status
	if err := s.shipmentRepo.UpdateStatus(ctx, shipment.ID, domainShipment.StatusOrderPosted); err != nil {
		return nil, err
	}

	// Get updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Order posted to marketplace",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("provider_id", providerID.String()),
		zap.String("event", "order_posted"),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

// Step 3: Shipper accepts order from marketplace

func (s *Service) AcceptOrder(ctx context.Context, shipmentID, shipperID uuid.UUID, req *AcceptOrderRequest) (*ShipmentResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if err := ValidateStatusTransition(shipment.Status, domainShipment.StatusShippingAssigned); err != nil {
		return nil, err
	}

	// Validate device
	if err := ValidateDevice(ctx, s.deviceRepo, req.DeviceID, shipperID); err != nil {
		return nil, err
	}

	// Assign shipper
	if err := s.shipmentRepo.AssignShipper(ctx, shipmentID, shipperID); err != nil {
		return nil, err
	}

	// Assign device
	if err := s.shipmentRepo.AssignDevice(ctx, shipmentID, req.DeviceID); err != nil {
		return nil, err
	}

	// Get rules
	rules, err := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, appErrors.NewAppError("RULES_NOT_FOUND", "Shipping rules not found", err)
	}

	// Validate business rules
	if err := ValidateBusinessRules(shipment, rules, domainShipment.StatusShippingAssigned); err != nil {
		return nil, err
	}

	// Update device status
	if err := s.deviceRepo.UpdateStatus(ctx, req.DeviceID, domainDevice.StatusInTransit); err != nil {
		return nil, fmt.Errorf("failed to update device status: %w", err)
	}

	// Update shipment
	shipment.ShipperID = &shipperID
	shipment.LinkedDeviceID = &req.DeviceID
	shipment.Status = domainShipment.StatusShippingAssigned
	shipment.UpdatedAt = time.Now()
	if err := s.shipmentRepo.Update(ctx, shipment); err != nil {
		return nil, err
	}

	// Get updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Order accepted by shipper",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("shipper_id", shipperID.String()),
		zap.String("device_id", req.DeviceID.String()),
		zap.String("event", "order_accepted"),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

// Step 4: Shipper confirms rules

func (s *Service) ConfirmRules(ctx context.Context, shipmentID, shipperID uuid.UUID) (*ShipmentResponse, error) {
	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Verify shipper owns this shipment
	if shipment.ShipperID == nil || *shipment.ShipperID != shipperID {
		return nil, appErrors.ErrUnauthorized
	}

	// Confirm rules
	if err := s.shipmentRepo.ConfirmRules(ctx, shipmentID, shipperID); err != nil {
		return nil, err
	}

	// Get updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Rules confirmed by shipper",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("shipper_id", shipperID.String()),
		zap.String("event", "rules_confirmed"),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

// Step 5: Shipper starts shipping

func (s *Service) StartShipping(ctx context.Context, shipmentID, shipperID uuid.UUID, req *StartShippingRequest) (*ShipmentResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Verify shipper owns this shipment
	if shipment.ShipperID == nil || *shipment.ShipperID != shipperID {
		return nil, appErrors.NewAppError("UNAUTHORIZED", "Shipper does not own this shipment", nil)
	}

	// Validate status transition
	if err := ValidateStatusTransition(shipment.Status, domainShipment.StatusInTransit); err != nil {
		return nil, err
	}

	// Get rules
	rules, err := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	if err != nil {
		return nil, appErrors.NewAppError("RULES_NOT_FOUND", "Shipping rules not found", err)
	}

	// Validate business rules
	if err := ValidateBusinessRules(shipment, rules, domainShipment.StatusInTransit); err != nil {
		return nil, err
	}

	// Update shipment
	pickupTime := time.Now()
	if req.ActualPickupAt != nil {
		pickupTime = *req.ActualPickupAt
	}
	shipment.ActualPickupAt = &pickupTime
	shipment.Status = domainShipment.StatusInTransit
	shipment.UpdatedAt = time.Now()
	if err := s.shipmentRepo.Update(ctx, shipment); err != nil {
		return nil, err
	}

	// Get updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Shipping started",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("shipper_id", shipperID.String()),
		zap.String("event", "shipping_started"),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

// Step 6: Complete delivery

func (s *Service) CompleteDelivery(ctx context.Context, shipperID, shipmentID uuid.UUID, req *CompleteDeliveryRequest) (*ShipmentResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Verify shipper owns this shipment
	if shipment.ShipperID == nil || *shipment.ShipperID != shipperID {
		return nil, appErrors.ErrUnauthorized
	}

	// Validate status transition
	if err := ValidateStatusTransition(shipment.Status, domainShipment.StatusCompleted); err != nil {
		return nil, err
	}

	// Update shipment
	deliveryTime := time.Now()
	if req.ActualDeliveryAt != nil {
		deliveryTime = *req.ActualDeliveryAt
	}

	if err := s.shipmentRepo.SetActualDelivery(ctx, shipmentID, deliveryTime, req.CompletionNotes); err != nil {
		return nil, err
	}

	// Update status
	if err := s.shipmentRepo.UpdateStatus(ctx, shipmentID, domainShipment.StatusCompleted); err != nil {
		return nil, err
	}

	// Update device status back to available
	if shipment.LinkedDeviceID != nil {
		if err := s.deviceRepo.UpdateStatus(ctx, *shipment.LinkedDeviceID, domainDevice.StatusAvailable); err != nil {
			logger.Warn("Failed to update device status",
				zap.String("device_id", shipment.LinkedDeviceID.String()),
				zap.Error(err),
			)
		}
	}

	// Get updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Delivery completed",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("event", "delivery_completed"),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

// Customer rates delivery

func (s *Service) RateDelivery(ctx context.Context, customerID, shipmentID uuid.UUID, req *RateDeliveryRequest) (*ShipmentResponse, error) {
	// Validate input
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Verify customer owns this shipment
	if shipment.CustomerID != customerID {
		return nil, appErrors.ErrUnauthorized
	}

	// Verify status
	if shipment.Status != domainShipment.StatusCompleted {
		return nil, appErrors.NewAppError("INVALID_STATUS", "Can only rate completed deliveries", nil)
	}

	// Set rating
	if err := s.shipmentRepo.SetCustomerRating(ctx, shipmentID, req.Rating, req.Feedback); err != nil {
		return nil, err
	}

	// Fetch updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Delivery rated",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("customer_id", customerID.String()),
		zap.Int("rating", req.Rating),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)

	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

// Reporting issues

func (s *Service) ReportIssue(ctx context.Context, reporterID, shipmentID uuid.UUID, req *ReportIssueRequest) (*ShipmentResponse, error) {
	// Validate input
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}
	// Verify reporter is involved in shipment
	isInvolved := shipment.CustomerID == reporterID ||
		shipment.ProviderID == reporterID ||
		(shipment.ShipperID != nil && *shipment.ShipperID == reporterID)

	if !isInvolved {
		return nil, appErrors.ErrUnauthorized
	}

	// Validate status transition
	if err := ValidateStatusTransition(shipment.Status, domainShipment.StatusIssueReported); err != nil {
		return nil, err
	}

	// Update shipment
	if err := s.shipmentRepo.UpdateStatus(ctx, shipmentID, domainShipment.StatusIssueReported); err != nil {
		return nil, err
	}

	// Get updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Issue reported",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("issue_type", req.IssueType),
		zap.String("severity", req.Severity),
		zap.String("event", "issue_reported"),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

func (s *Service) CancelShipment(ctx context.Context, userID, shipmentID uuid.UUID, req *CancelShipmentRequest) (*ShipmentResponse, error) {
	// Validate input
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get shipment
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Verify user is involved in shipment
	isInvolved := shipment.CustomerID == userID ||
		shipment.ProviderID == userID ||
		(shipment.ShipperID != nil && *shipment.ShipperID == userID)

	if !isInvolved {
		return nil, appErrors.ErrUnauthorized
	}

	// Validate status transition
	if err := ValidateStatusTransition(shipment.Status, domainShipment.StatusCancelled); err != nil {
		return nil, err
	}

	// Cannot cancel if in transit
	if shipment.Status == domainShipment.StatusInTransit {
		return nil, appErrors.NewAppError("CANNOT_CANCEL", "Cannot cancel shipment in transit", nil)
	}

	// Update shipment
	if err := s.shipmentRepo.UpdateStatus(ctx, shipmentID, domainShipment.StatusCancelled); err != nil {
		return nil, err
	}

	// Update device status back to available if assigned
	if shipment.LinkedDeviceID != nil {
		if err := s.deviceRepo.UpdateStatus(ctx, *shipment.LinkedDeviceID, domainDevice.StatusAvailable); err != nil {
			logger.Warn("Failed to update device status",
				zap.String("device_id", shipment.LinkedDeviceID.String()),
				zap.Error(err),
			)
		}
	}

	// Get updated shipment
	updatedShipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	logger.Info("Shipment cancelled",
		zap.String("shipment_id", shipmentID.String()),
		zap.String("reason", req.Reason),
		zap.String("event", "shipment_cancelled"),
	)

	updatedRules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	return ToShipmentResponse(updatedShipment, updatedRules), nil
}

func (s *Service) GetShipment(ctx context.Context, userID, shipmentID uuid.UUID) (*ShipmentDetailResponse, error) {
	shipment, err := s.shipmentRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return nil, err
	}

	// Verify user has access
	isAuthorized := shipment.CustomerID == userID ||
		shipment.ProviderID == userID ||
		(shipment.ShipperID != nil && *shipment.ShipperID == userID)

	if !isAuthorized {
		// Check if user is admin
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user.Role != "admin" {
			return nil, appErrors.ErrUnauthorized
		}
	}

	rules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipmentID)
	response := ToShipmentResponse(shipment, rules)

	return &ShipmentDetailResponse{
		ShipmentResponse: response,
		Rules:            toShippingRulesResponse(rules),
	}, nil
}

func (s *Service) ListShipments(ctx context.Context, userID uuid.UUID, userRole string, filter *ShipmentFilterRequest) (*ShipmentListResponse, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	if userRole != "admin" {
		switch userRole {
		case "customer":
			filter.CustomerID = &userID
		case "provider":
			filter.ProviderID = &userID
		case "shipper":
			filter.ShipperID = &userID
		}
	}

	// Convert to domain filter
	domainFilter := ToDomainFilter(filter)

	// Get shipments from repository
	shipments, total, err := s.shipmentRepo.List(ctx, domainFilter)
	if err != nil {
		return nil, err
	}

	// Convert to response DTOs
	shipmentResponses := make([]ShipmentResponse, len(shipments))
	for i, shipment := range shipments {
		rules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipment.ID)
		shipmentResponses[i] = *ToShipmentResponse(shipment, rules)
	}

	// Calculate total pages
	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &ShipmentListResponse{
		Shipments:  shipmentResponses,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) GetMarketplaceListings(ctx context.Context, page, pageSize int) (*ShipmentListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	shipments, total, err := s.shipmentRepo.GetMarketplaceListings(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	// Convert to response
	shipmentResponses := make([]ShipmentResponse, len(shipments))
	for i, shipment := range shipments {
		rules, _ := s.shipmentRepo.GetRulesByShipmentID(ctx, shipment.ID)
		shipmentResponses[i] = *ToShipmentResponse(shipment, rules)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ShipmentListResponse{
		Shipments:  shipmentResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) GetStatistics(ctx context.Context) (*ShipmentStatisticsResponse, error) {
	stats, err := s.shipmentRepo.GetStatistics(ctx)
	if err != nil {
		return nil, err
	}

	return ToStatisticsResponse(stats), nil
}

// Helper function
func toShippingRulesResponse(rules *domainShipment.ShippingRules) *ShippingRulesResponse {
	if rules == nil {
		return nil
	}
	return &ShippingRulesResponse{
		ID:                    rules.ID,
		ShipmentID:            rules.ShipmentID,
		ReportCycleSec:        rules.ReportCycleSec,
		TempMin:               rules.TempMin,
		TempMax:               rules.TempMax,
		HumidityMin:           rules.HumidityMin,
		HumidityMax:           rules.HumidityMax,
		LightMax:              rules.LightMax,
		TiltMaxAngle:          rules.TiltMaxAngle,
		ImpactThresholdG:      rules.ImpactThresholdG,
		EnablePredictiveAlert: rules.EnablePredictiveAlert,
		AlertBufferTimeMin:    rules.AlertBufferTimeMin,
		SetByProviderID:       rules.SetByProviderID,
		ConfirmedByShipperID:  rules.ConfirmedByShipperID,
		SetAt:                 rules.SetAt,
		ConfirmedAt:           rules.ConfirmedAt,
	}
}
