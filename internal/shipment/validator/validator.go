package validator

import (
	"context"
	"github.com/google/uuid"
	"logistics-quality-monitor/internal/infrastructure/database/postgres"
	"logistics-quality-monitor/internal/shipment/models"
	appErrors "logistics-quality-monitor/pkg/errors"
	"time"
)

// ValidateParties validates customer, provider, and shipper
func ValidateParties(ctx context.Context, userRepo postgres.UserRepository, customerID, providerID uuid.UUID, shipperID *uuid.UUID) error {
	// Validate customer
	customer, err := userRepo.GetByID(ctx, customerID)
	if err != nil {
		return appErrors.ErrUserNotFound
	}
	if customer.Role != "customer" {
		return appErrors.NewAppError("INVALID_ROLE", "Customer must have 'customer' role", nil)
	}
	if !customer.IsActive {
		return appErrors.ErrUserInactive
	}

	// Validate provider
	provider, err := userRepo.GetByID(ctx, providerID)
	if err != nil {
		return appErrors.ErrUserNotFound
	}
	if provider.Role != "provider" {
		return appErrors.NewAppError("INVALID_ROLE", "Provider must have 'provider' role", nil)
	}
	if !provider.IsActive {
		return appErrors.ErrUserInactive
	}

	// Ensure customer and provider are different
	if customerID == providerID {
		return appErrors.NewAppError("SAME_PARTY", "Customer and provider must be different users", nil)
	}

	// Validate shipper if provided
	if shipperID != nil {
		shipper, err := userRepo.GetByID(ctx, *shipperID)
		if err != nil {
			return appErrors.ErrUserNotFound
		}
		if shipper.Role != "shipper" {
			return appErrors.NewAppError("INVALID_ROLE", "Shipper must have 'shipper' role", nil)
		}
		if !shipper.IsActive {
			return appErrors.ErrUserInactive
		}

		// Ensure shipper is different from customer and provider
		if *shipperID == customerID || *shipperID == providerID {
			return appErrors.NewAppError("SAME_PARTY", "Shipper must be different from customer and provider", nil)
		}
	}

	return nil
}

// ValidateDevice validates device assignment
//func ValidateDevice(ctx context.Context, deviceRepo postgres.DeviceRepository, deviceID uuid.UUID, shipperID uuid.UUID) error {
//	device, err := deviceRepo.GetByID(ctx, deviceID)
//	if err != nil {
//		return appErrors.NewAppError("DEVICE_NOT_FOUND", "Device not found", err)
//	}
//
//	// Check if device is available
//	available, err := deviceRepo.IsDeviceAvailable(ctx, deviceID)
//	if err != nil {
//		return err
//	}
//	if !available {
//		return appErrors.NewAppError("DEVICE_UNAVAILABLE", "Device is not available for assignment", nil)
//	}
//
//	return nil
//}

// ValidateShippingRules validates quality control rules
func ValidateShippingRules(rules *models.PostOrderRequest) error {
	// Temperature range check
	if rules.TempMin != nil && rules.TempMax != nil {
		if *rules.TempMin >= *rules.TempMax {
			return appErrors.NewAppError("INVALID_RULES", "Temperature minimum must be less than maximum", nil)
		}
	}

	// Humidity range check
	if rules.HumidityMin != nil && rules.HumidityMax != nil {
		if *rules.HumidityMin >= *rules.HumidityMax {
			return appErrors.NewAppError("INVALID_RULES", "Humidity minimum must be less than maximum", nil)
		}
	}

	// Report cycle validation
	if rules.ReportCycleSec < 10 || rules.ReportCycleSec > 300 {
		return appErrors.NewAppError("INVALID_RULES", "Report cycle must be between 10 and 300 seconds", nil)
	}

	return nil
}

// ValidateTimeRange validates pickup and delivery times
func ValidateTimeRange(pickupTime, deliveryTime *time.Time) error {
	if pickupTime == nil || deliveryTime == nil {
		return nil // Optional fields
	}

	if deliveryTime.Before(*pickupTime) {
		return appErrors.NewAppError("INVALID_TIME", "Delivery time must be after pickup time", nil)
	}

	return nil
}
