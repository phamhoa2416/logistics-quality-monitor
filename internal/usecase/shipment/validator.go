package shipment

import (
	"context"
	"fmt"
	domainDevice "logistics-quality-monitor/internal/domain/device"
	domainShipment "logistics-quality-monitor/internal/domain/shipment"
	domainUser "logistics-quality-monitor/internal/domain/user"
	appErrors "logistics-quality-monitor/pkg/errors"
	"time"

	"github.com/google/uuid"
)

// ValidateStatusTransition checks if status transition is allowed
func ValidateStatusTransition(currentStatus, newStatus domainShipment.ShipmentStatus) error {
	validTransitions := map[domainShipment.ShipmentStatus][]domainShipment.ShipmentStatus{
		domainShipment.StatusDemandCreated: {
			domainShipment.StatusOrderPosted,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusOrderPosted: {
			domainShipment.StatusShippingAssigned,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusShippingAssigned: {
			domainShipment.StatusInTransit,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusInTransit: {
			domainShipment.StatusCompleted,
			domainShipment.StatusIssueReported,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusIssueReported: {
			domainShipment.StatusInTransit,
			domainShipment.StatusCompleted,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusCompleted: {
			// Terminal state - no transitions
		},
		domainShipment.StatusCancelled: {
			// Terminal state - no transitions
		},
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return appErrors.NewAppError(
			"INVALID_STATUS",
			fmt.Sprintf("Unknown current status: %s", currentStatus),
			nil,
		)
	}

	for _, allowed := range allowedStatuses {
		if newStatus == allowed {
			return nil
		}
	}

	return appErrors.NewAppError(
		"INVALID_TRANSITION",
		fmt.Sprintf("Cannot transition from %s to %s", currentStatus, newStatus),
		nil,
	)
}

// GetAllowedTransitions returns allowed next statuses
func GetAllowedTransitions(currentStatus domainShipment.ShipmentStatus) []domainShipment.ShipmentStatus {
	validTransitions := map[domainShipment.ShipmentStatus][]domainShipment.ShipmentStatus{
		domainShipment.StatusDemandCreated: {
			domainShipment.StatusOrderPosted,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusOrderPosted: {
			domainShipment.StatusShippingAssigned,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusShippingAssigned: {
			domainShipment.StatusInTransit,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusInTransit: {
			domainShipment.StatusCompleted,
			domainShipment.StatusIssueReported,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusIssueReported: {
			domainShipment.StatusInTransit,
			domainShipment.StatusCompleted,
			domainShipment.StatusCancelled,
		},
		domainShipment.StatusCompleted: {},
		domainShipment.StatusCancelled: {},
	}
	return validTransitions[currentStatus]
}

type TransitionRules struct {
	RequiresShipper bool
	RequiresDevice  bool
	RequiresRules   bool
	RulesConfirmed  bool
	CanCancel       bool
}

var transitionRules = map[domainShipment.ShipmentStatus]TransitionRules{
	domainShipment.StatusDemandCreated: {
		RequiresShipper: false,
		RequiresDevice:  false,
		RequiresRules:   false,
		RulesConfirmed:  false,
		CanCancel:       true,
	},
	domainShipment.StatusOrderPosted: {
		RequiresShipper: false,
		RequiresDevice:  false,
		RequiresRules:   true,
		RulesConfirmed:  false,
		CanCancel:       true,
	},
	domainShipment.StatusShippingAssigned: {
		RequiresShipper: true,
		RequiresDevice:  true,
		RequiresRules:   true,
		RulesConfirmed:  false,
		CanCancel:       true,
	},
	domainShipment.StatusInTransit: {
		RequiresShipper: true,
		RequiresDevice:  true,
		RequiresRules:   true,
		RulesConfirmed:  true,
		CanCancel:       false,
	},
	domainShipment.StatusIssueReported: {
		RequiresShipper: true,
		RequiresDevice:  true,
		RequiresRules:   true,
		RulesConfirmed:  true,
		CanCancel:       true,
	},
}

// ValidateBusinessRules checks if shipment meets requirements for status
func ValidateBusinessRules(shipment *domainShipment.Shipment, rules *domainShipment.ShippingRules, targetStatus domainShipment.ShipmentStatus) error {
	statusRules, exists := transitionRules[targetStatus]
	if !exists {
		return nil // No specific rules for this status
	}

	if statusRules.RequiresShipper && shipment.ShipperID == nil {
		return appErrors.NewAppError(
			"SHIPPER_REQUIRED",
			fmt.Sprintf("Shipper is required for status %s", targetStatus),
			nil,
		)
	}

	if statusRules.RequiresDevice && shipment.LinkedDeviceID == nil {
		return appErrors.NewAppError(
			"DEVICE_REQUIRED",
			fmt.Sprintf("Device is required for status %s", targetStatus),
			nil,
		)
	}

	if statusRules.RequiresRules && rules == nil {
		return appErrors.NewAppError(
			"RULES_REQUIRED",
			fmt.Sprintf("Shipping rules are required for status %s", targetStatus),
			nil,
		)
	}

	if statusRules.RulesConfirmed && (rules == nil || rules.ConfirmedByShipperID == nil) {
		return appErrors.NewAppError(
			"RULES_NOT_CONFIRMED",
			"Shipper must confirm quality rules before starting transit",
			nil,
		)
	}

	return nil
}

// ValidateParties validates customer, provider, and shipper
func ValidateParties(ctx context.Context, userRepo domainUser.Repository, customerID, providerID uuid.UUID, shipperID *uuid.UUID) error {
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
func ValidateDevice(ctx context.Context, deviceRepo domainDevice.Repository, deviceID uuid.UUID, shipperID uuid.UUID) error {
	device, err := deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return appErrors.NewAppError("DEVICE_NOT_FOUND", "Device not found", err)
	}

	// Check if device is available
	if device.Status != domainDevice.StatusAvailable {
		return appErrors.NewAppError("DEVICE_UNAVAILABLE", "Device is not available for assignment", nil)
	}

	// Check if device has owner and it matches shipper
	if device.OwnerShipperID != nil && *device.OwnerShipperID != shipperID {
		return appErrors.NewAppError("DEVICE_OWNER_MISMATCH", "Device owner does not match shipper", nil)
	}

	return nil
}

// ValidateShippingRules validates quality control rules
func ValidateShippingRules(rules *PostOrderRequest) error {
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
