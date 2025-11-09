package device

import (
	"context"
	"fmt"
	domainDevice "logistics-quality-monitor/internal/domain/device"
	domainUser "logistics-quality-monitor/internal/domain/user"
	appErrors "logistics-quality-monitor/pkg/errors"

	"github.com/google/uuid"
)

// ValidateShipperOwner validates that the shipper ID is valid and active
func ValidateShipperOwner(ctx context.Context, userRepo domainUser.Repository, shipperID uuid.UUID) error {
	user, err := userRepo.GetByID(ctx, shipperID)
	if err != nil {
		return appErrors.ErrUserNotFound
	}

	if user.Role != "shipper" {
		return appErrors.NewAppError("INVALID_ROLE", "Owner must be a shipper", nil)
	}

	if !user.IsActive {
		return appErrors.ErrUserInactive
	}

	return nil
}

// ValidateDeviceStatus validates device status transitions
func ValidateDeviceStatus(currentStatus, newStatus domainDevice.DeviceStatus) error {
	validTransitions := map[domainDevice.DeviceStatus][]domainDevice.DeviceStatus{
		domainDevice.StatusAvailable:   {domainDevice.StatusInTransit, domainDevice.StatusMaintenance, domainDevice.StatusRetired},
		domainDevice.StatusInTransit:   {domainDevice.StatusAvailable, domainDevice.StatusMaintenance, domainDevice.StatusRetired},
		domainDevice.StatusMaintenance: {domainDevice.StatusAvailable, domainDevice.StatusRetired},
		domainDevice.StatusRetired:     {},
	}

	allowedStatus, exists := validTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("invalid current status: %s", currentStatus)
	}

	for _, allowed := range allowedStatus {
		if newStatus == allowed {
			return nil
		}
	}

	return appErrors.NewAppError(
		"INVALID_STATUS_TRANSITION",
		fmt.Sprintf("Cannot transition from %s to %s", currentStatus, newStatus),
		nil)
}
