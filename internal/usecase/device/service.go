package device

import (
	"context"
	domainDevice "logistics-quality-monitor/internal/domain/device"
	domainUser "logistics-quality-monitor/internal/domain/user"
	"logistics-quality-monitor/internal/logger"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service implements device use cases
type Service struct {
	deviceRepo domainDevice.Repository
	userRepo   domainUser.Repository
}

// NewService creates a new device service
func NewService(deviceRepo domainDevice.Repository, userRepo domainUser.Repository) *Service {
	return &Service{
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
	}
}

func (s *Service) CreateDevice(ctx context.Context, req *CreateDeviceRequest) (*DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Check if device already exists
	existingDevice, _ := s.deviceRepo.GetByHardwareUID(ctx, req.HardwareUID)
	if existingDevice != nil {
		return nil, appErrors.NewAppError("DEVICE_EXISTS", "Device with this hardware UID already exists", nil)
	}

	// Validate owner if provided
	if req.OwnerShipperID != nil {
		if err := ValidateShipperOwner(ctx, s.userRepo, *req.OwnerShipperID); err != nil {
			return nil, err
		}
	}

	// Create domain entity
	device := &domainDevice.Device{
		HardwareUID:     req.HardwareUID,
		DeviceName:      req.DeviceName,
		Model:           req.Model,
		OwnerShipperID:  req.OwnerShipperID,
		FirmwareVersion: req.FirmwareVersion,
		Status:          domainDevice.StatusAvailable,
		TotalTrips:      0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Save device
	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, err
	}

	// Get created device
	createdDevice, err := s.deviceRepo.GetByID(ctx, device.ID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device created",
		zap.String("device_id", createdDevice.ID.String()),
		zap.String("hardware_uid", createdDevice.HardwareUID),
		zap.String("event", "device_created"),
	)

	return ToDeviceResponse(createdDevice), nil
}

func (s *Service) GetDevice(ctx context.Context, deviceID uuid.UUID) (*DeviceResponse, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return ToDeviceResponse(device), nil
}

func (s *Service) GetDeviceByHardwareUID(ctx context.Context, hardwareUID string) (*DeviceResponse, error) {
	device, err := s.deviceRepo.GetByHardwareUID(ctx, hardwareUID)
	if err != nil {
		return nil, err
	}

	return ToDeviceResponse(device), nil
}

func (s *Service) ListDevices(ctx context.Context, filter *DeviceFilterRequest) (*DeviceListResponse, error) {
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

	// Convert to domain filter
	domainFilter := ToDomainFilter(filter)

	// Get devices from repository
	devices, total, err := s.deviceRepo.List(ctx, domainFilter)
	if err != nil {
		return nil, err
	}

	// Convert to response DTOs
	deviceResponses := make([]DeviceResponse, len(devices))
	for i, device := range devices {
		deviceResponses[i] = *ToDeviceResponse(device)
	}

	// Calculate total pages
	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &DeviceListResponse{
		Devices:    deviceResponses,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) UpdateDevice(ctx context.Context, deviceID uuid.UUID, req *UpdateDeviceRequest) (*DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get existing device
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// Business rule: Cannot update device status while in transit
	if device.Status == domainDevice.StatusInTransit && req.Status != nil && *req.Status != string(domainDevice.StatusInTransit) {
		return nil, appErrors.NewAppError("DEVICE_IN_USE", "Cannot update device status while in transit", nil)
	}

	// Update fields
	if req.DeviceName != nil {
		device.DeviceName = req.DeviceName
	}
	if req.Model != nil {
		device.Model = req.Model
	}
	if req.FirmwareVersion != nil {
		device.FirmwareVersion = req.FirmwareVersion
	}
	if req.Status != nil {
		newStatus := domainDevice.DeviceStatus(*req.Status)
		if err := ValidateDeviceStatus(device.Status, newStatus); err != nil {
			return nil, err
		}
		device.Status = newStatus
	}
	device.UpdatedAt = time.Now()

	// Save updates
	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, err
	}

	// Get updated device
	updatedDevice, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device updated",
		zap.String("device_id", updatedDevice.ID.String()),
		zap.String("hardware_uid", updatedDevice.HardwareUID),
		zap.String("event", "device_updated"),
	)

	return ToDeviceResponse(updatedDevice), nil
}

func (s *Service) AssignOwner(ctx context.Context, deviceID uuid.UUID, req *AssignOwnerRequest) (*DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Validate shipper
	if err := ValidateShipperOwner(ctx, s.userRepo, req.OwnerShipperID); err != nil {
		return nil, err
	}

	// Get device
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// Business rule: Cannot assign owner while device is in transit
	if device.Status == domainDevice.StatusInTransit {
		return nil, appErrors.NewAppError("DEVICE_IN_USE", "Cannot assign owner while device is in transit", nil)
	}

	// Assign owner
	if err := s.deviceRepo.AssignOwner(ctx, deviceID, req.OwnerShipperID); err != nil {
		return nil, err
	}

	// Get updated device
	updatedDevice, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device assigned to shipper",
		zap.String("device_id", deviceID.String()),
		zap.String("shipper_id", req.OwnerShipperID.String()),
		zap.String("event", "device_assigned"),
	)

	return ToDeviceResponse(updatedDevice), nil
}

func (s *Service) UnassignOwner(ctx context.Context, deviceID uuid.UUID, req *UnassignOwnerRequest) (*DeviceResponse, error) {
	// Get device
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// Business rule: Cannot unassign owner while device is in transit
	if device.Status == domainDevice.StatusInTransit {
		return nil, appErrors.NewAppError("DEVICE_IN_USE", "Cannot unassign owner while device is in transit", nil)
	}

	// Business rule: Device must have an owner
	if device.OwnerShipperID == nil {
		return nil, appErrors.NewAppError("NO_OWNER", "Device has no owner", nil)
	}

	// Unassign owner
	if err := s.deviceRepo.UnassignOwner(ctx, deviceID); err != nil {
		return nil, err
	}

	// Get updated device
	updatedDevice, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device unassigned",
		zap.String("device_id", deviceID.String()),
		zap.String("reason", req.Reason),
		zap.String("event", "device_unassigned"),
	)

	return ToDeviceResponse(updatedDevice), nil
}

func (s *Service) UpdateStatus(ctx context.Context, deviceID uuid.UUID, req *UpdateStatusRequest) (*DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get device
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if err := ValidateDeviceStatus(device.Status, req.Status); err != nil {
		return nil, err
	}

	// Update status
	if err := s.deviceRepo.UpdateStatus(ctx, deviceID, req.Status); err != nil {
		return nil, err
	}

	// Get updated device
	updatedDevice, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device status changed",
		zap.String("device_id", deviceID.String()),
		zap.String("old_status", string(device.Status)),
		zap.String("new_status", string(req.Status)),
		zap.String("reason", req.Reason),
		zap.String("event", "device_status_changed"),
	)

	return ToDeviceResponse(updatedDevice), nil
}

func (s *Service) UpdateBattery(ctx context.Context, deviceID uuid.UUID, req *UpdateBatteryRequest) (*DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Update battery
	if err := s.deviceRepo.UpdateBattery(ctx, deviceID, req.BatteryLevel); err != nil {
		return nil, err
	}

	// Update last seen
	_ = s.deviceRepo.UpdateLastSeen(ctx, deviceID)

	// Get updated device
	updatedDevice, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return ToDeviceResponse(updatedDevice), nil
}

func (s *Service) DeleteDevice(ctx context.Context, deviceID uuid.UUID) error {
	// Get device
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return err
	}

	// Business rule: Cannot delete device in transit
	if device.Status == domainDevice.StatusInTransit {
		return appErrors.NewAppError("DEVICE_IN_USE", "Cannot delete device in transit", nil)
	}

	// Business rule: Cannot delete device assigned to shipment
	if device.CurrentShipmentID != nil {
		return appErrors.NewAppError("DEVICE_IN_USE", "Cannot delete device assigned to shipment", nil)
	}

	// Delete device (marks as retired)
	if err := s.deviceRepo.Delete(ctx, deviceID); err != nil {
		return err
	}

	logger.Info("Device marked as retired",
		zap.String("device_id", deviceID.String()),
		zap.String("event", "device_retired"),
	)

	return nil
}

func (s *Service) BulkAssignOwner(ctx context.Context, req *BulkAssignRequest) (*BulkOperationResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Validate shipper
	if err := ValidateShipperOwner(ctx, s.userRepo, req.OwnerShipperID); err != nil {
		return nil, err
	}

	response := &BulkOperationResponse{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       []BulkError{},
	}

	// Process each device
	for _, deviceID := range req.DeviceIDs {
		device, err := s.deviceRepo.GetByID(ctx, deviceID)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, BulkError{
				DeviceID: deviceID,
				Error:    err.Error(),
			})
			continue
		}

		// Business rule: Cannot assign owner while device is in transit
		if device.Status == domainDevice.StatusInTransit {
			response.FailedCount++
			response.Errors = append(response.Errors, BulkError{
				DeviceID: deviceID,
				Error:    "Cannot assign owner while device is in transit",
			})
			continue
		}

		// Assign owner
		err = s.deviceRepo.AssignOwner(ctx, deviceID, req.OwnerShipperID)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, BulkError{
				DeviceID: deviceID,
				Error:    err.Error(),
			})
		} else {
			response.SuccessCount++
		}
	}

	logger.Info("Bulk assignment completed",
		zap.Int("success_count", response.SuccessCount),
		zap.Int("failed_count", response.FailedCount),
		zap.String("event", "bulk_assignment_completed"),
	)

	return response, nil
}

func (s *Service) GetStatistics(ctx context.Context) (*DeviceStatisticsResponse, error) {
	stats, err := s.deviceRepo.GetStatistics(ctx)
	if err != nil {
		return nil, err
	}

	return ToStatisticsResponse(stats), nil
}

func (s *Service) GetAvailableDevices(ctx context.Context, shipperID *uuid.UUID) ([]DeviceResponse, error) {
	filter := &DeviceFilterRequest{
		Status:   (*domainDevice.DeviceStatus)(utils.StringPtr(string(domainDevice.StatusAvailable))),
		PageSize: 100,
	}

	if shipperID != nil {
		filter.OwnerShipperID = shipperID
	}

	devices, _, err := s.deviceRepo.List(ctx, ToDomainFilter(filter))
	if err != nil {
		return nil, err
	}

	responses := make([]DeviceResponse, len(devices))
	for i, device := range devices {
		responses[i] = *ToDeviceResponse(device)
	}

	return responses, nil
}
