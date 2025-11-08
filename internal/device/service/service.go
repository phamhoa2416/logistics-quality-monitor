package service

import (
	"context"
	"logistics-quality-monitor/internal/config"
	"logistics-quality-monitor/internal/device/model"
	"logistics-quality-monitor/internal/device/repository"
	deviceValidator "logistics-quality-monitor/internal/device/validator"
	"logistics-quality-monitor/internal/logger"
	userRepository "logistics-quality-monitor/internal/user/repository"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DeviceService struct {
	repo     *repository.DeviceRepository
	userRepo userRepository.UserRepository
	config   *config.Config
}

func NewService(repo *repository.DeviceRepository, userRepo userRepository.UserRepository, config *config.Config) *DeviceService {
	return &DeviceService{
		repo:     repo,
		userRepo: userRepo,
		config:   config,
	}
}

func (s *DeviceService) CreateDevice(ctx context.Context, request *model.CreateDeviceRequest) (*model.DeviceResponse, error) {
	if err := utils.ValidateStruct(request); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	existingDevice, _ := s.repo.GetDeviceByHardwareUID(ctx, request.HardwareUID)
	if existingDevice != nil {
		return nil, appErrors.NewAppError("DEVICE_EXISTS", "Device with this hardware UID already exists", nil)
	}

	if request.OwnerShipperID != nil {
		if err := deviceValidator.ValidateShipperOwner(ctx, s.userRepo, *request.OwnerShipperID); err != nil {
			return nil, err
		}
	}

	device := &model.Device{
		HardwareUID:     request.HardwareUID,
		DeviceName:      request.DeviceName,
		Model:           request.Model,
		OwnerShipperID:  request.OwnerShipperID,
		FirmwareVersion: request.FirmwareVersion,
	}

	if err := s.repo.CreateDevice(ctx, device); err != nil {
		return nil, err
	}

	createdDevice, err := s.repo.GetDeviceByID(ctx, device.ID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device created",
		zap.String("device_id", createdDevice.ID.String()),
		zap.String("hardware_uid", createdDevice.HardwareUID),
		zap.String("event", "device_created"),
	)

	return createdDevice.ToResponse(), nil
}

func (s *DeviceService) GetDevice(ctx context.Context, deviceID uuid.UUID) (*model.DeviceResponse, error) {
	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return device.ToResponse(), nil
}

func (s *DeviceService) GetDeviceByHardwareUID(ctx context.Context, hardwareUID string) (*model.DeviceResponse, error) {
	device, err := s.repo.GetDeviceByHardwareUID(ctx, hardwareUID)
	if err != nil {
		return nil, err
	}

	return device.ToResponse(), nil
}

func (s *DeviceService) ListDevices(ctx context.Context, filter *model.DeviceFilterRequest) (*model.DeviceListResponse, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}

	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	devices, total, err := s.repo.ListDevices(ctx, filter)
	if err != nil {
		return nil, err
	}

	deviceResponses := make([]model.DeviceResponse, len(devices))
	for i, device := range devices {
		deviceResponses[i] = *device.ToResponse()
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &model.DeviceListResponse{
		Devices:    deviceResponses,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *DeviceService) UpdateDevice(ctx context.Context, deviceID uuid.UUID, req *model.UpdateDeviceRequest) (*model.DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if device.Status == model.StatusInTransit && req.Status != nil && *req.Status != string(model.StatusInTransit) {
		return nil, appErrors.NewAppError("DEVICE_IN_USE", "Cannot update device status while in transit", nil)
	}

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
		newStatus := model.DeviceStatus(*req.Status)
		if err := deviceValidator.ValidateDeviceStatus(device.Status, newStatus); err != nil {
			return nil, err
		}
		device.Status = newStatus
	}

	if err := s.repo.UpdateDevice(ctx, device); err != nil {
		return nil, err
	}

	updatedDevice, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device updated",
		zap.String("device_id", updatedDevice.ID.String()),
		zap.String("hardware_uid", updatedDevice.HardwareUID),
		zap.String("event", "device_updated"),
	)

	return updatedDevice.ToResponse(), nil
}

func (s *DeviceService) AssignOwner(ctx context.Context, deviceID uuid.UUID, req *model.AssignOwnerRequest) (*model.DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := deviceValidator.ValidateShipperOwner(ctx, s.userRepo, req.OwnerShipperID); err != nil {
		return nil, err
	}

	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if device.Status == model.StatusInTransit {
		return nil, appErrors.NewAppError("DEVICE_IN_USE", "Cannot assign owner while device is in transit", nil)
	}

	if err := s.repo.AssignOwner(ctx, deviceID, req.OwnerShipperID); err != nil {
		return nil, err
	}

	updatedDevice, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device assigned to shipper",
		zap.String("device_id", deviceID.String()),
		zap.String("shipper_id", req.OwnerShipperID.String()),
		zap.String("event", "device_assigned"),
	)

	return updatedDevice.ToResponse(), nil
}

func (s *DeviceService) UnassignOwner(ctx context.Context, deviceID uuid.UUID, req *model.UnassignOwnerRequest) (*model.DeviceResponse, error) {
	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if device.Status == model.StatusInTransit {
		return nil, appErrors.NewAppError("DEVICE_IN_USE", "Cannot unassign owner while device is in transit", nil)
	}

	if device.OwnerShipperID == nil {
		return nil, appErrors.NewAppError("NO_OWNER", "Device has no owner", nil)
	}

	if err := s.repo.UnassignOwner(ctx, deviceID); err != nil {
		return nil, err
	}

	updatedDevice, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	logger.Info("Device unassigned",
		zap.String("device_id", deviceID.String()),
		zap.String("reason", req.Reason),
		zap.String("event", "device_unassigned"),
	)

	return updatedDevice.ToResponse(), nil
}

func (s *DeviceService) UpdateStatus(ctx context.Context, deviceID uuid.UUID, req *model.UpdateStatusRequest) (*model.DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if err := deviceValidator.ValidateDeviceStatus(device.Status, req.Status); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, deviceID, req.Status); err != nil {
		return nil, err
	}

	updatedDevice, err := s.repo.GetDeviceByID(ctx, deviceID)
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

	return updatedDevice.ToResponse(), nil
}

func (s *DeviceService) UpdateBattery(ctx context.Context, deviceID uuid.UUID, req *model.UpdateBatteryRequest) (*model.DeviceResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := s.repo.UpdateBattery(ctx, deviceID, req.BatteryLevel); err != nil {
		return nil, err
	}

	_ = s.repo.UpdateLastSeen(ctx, deviceID)

	updatedDevice, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return updatedDevice.ToResponse(), nil
}

func (s *DeviceService) DeleteDevice(ctx context.Context, deviceID uuid.UUID) error {
	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return err
	}

	if device.Status == model.StatusInTransit {
		return appErrors.NewAppError("DEVICE_IN_USE", "Cannot delete device in transit", nil)
	}

	if device.CurrentShipmentID != nil {
		return appErrors.NewAppError("DEVICE_IN_USE", "Cannot delete device assigned to shipment", nil)
	}

	if err := s.repo.DeleteDevice(ctx, deviceID); err != nil {
		return err
	}

	logger.Info("Device marked as retired",
		zap.String("device_id", deviceID.String()),
		zap.String("event", "device_retired"),
	)

	return nil
}

func (s *DeviceService) BulkAssignOwner(ctx context.Context, req *model.BulkAssignRequest) (*model.BulkOperationResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := deviceValidator.ValidateShipperOwner(ctx, s.userRepo, req.OwnerShipperID); err != nil {
		return nil, err
	}

	response := &model.BulkOperationResponse{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       []model.BulkError{},
	}

	for _, deviceID := range req.DeviceIDs {
		device, err := s.repo.GetDeviceByID(ctx, deviceID)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, model.BulkError{
				DeviceID: deviceID,
				Error:    err.Error(),
			})
			continue
		}

		if device.Status == model.StatusInTransit {
			response.FailedCount++
			response.Errors = append(response.Errors, model.BulkError{
				DeviceID: deviceID,
				Error:    "Cannot assign owner while device is in transit",
			})
			continue
		}

		err = s.repo.AssignOwner(ctx, deviceID, req.OwnerShipperID)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, model.BulkError{
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

func (s *DeviceService) GetStatistics(ctx context.Context) (*model.DeviceStatistics, error) {
	return s.repo.GetStatistics(ctx)
}

func (s *DeviceService) GetAvailableDevices(ctx context.Context, shipperID *uuid.UUID) ([]model.DeviceResponse, error) {
	filter := &model.DeviceFilterRequest{
		Status:   (*model.DeviceStatus)(utils.StringPtr(string(model.StatusAvailable))),
		PageSize: 100,
	}

	if shipperID != nil {
		filter.OwnerShipperID = shipperID
	}

	devices, _, err := s.repo.ListDevices(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]model.DeviceResponse, len(devices))
	for i, device := range devices {
		responses[i] = *device.ToResponse()
	}

	return responses, nil
}
