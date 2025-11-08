package validator

import (
	"context"
	"fmt"
	"logistics-quality-monitor/internal/device/model"
	"logistics-quality-monitor/internal/user/repository"
	appErrors "logistics-quality-monitor/pkg/errors"

	"github.com/google/uuid"
)

func ValidateShipperOwner(ctx context.Context, userRepository repository.UserRepository, shipperID uuid.UUID) error {
	user, err := userRepository.GetUserByID(ctx, shipperID)
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

func ValidateDeviceStatus(currentStatus, newStatus model.DeviceStatus) error {
	validTransitions := map[model.DeviceStatus][]model.DeviceStatus{
		model.StatusAvailable:   {model.StatusInTransit, model.StatusMaintenance, model.StatusRetired},
		model.StatusInTransit:   {model.StatusAvailable, model.StatusMaintenance, model.StatusRetired},
		model.StatusMaintenance: {model.StatusAvailable, model.StatusRetired},
		model.StatusRetired:     {},
	}

	allowedStatus, exists := validTransitions[newStatus]
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
