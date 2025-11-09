package lifecycle

import (
	"fmt"
	"logistics-quality-monitor/internal/shipment/models"
	appErrors "logistics-quality-monitor/pkg/errors"
)

// State machine for shipment status transitions
var validTransitions = map[models.ShipmentStatus][]models.ShipmentStatus{
	models.StatusDemandCreated: {
		models.StatusOrderPosted,
		models.StatusCancelled,
	},
	models.StatusOrderPosted: {
		models.StatusShippingAssigned,
		models.StatusCancelled,
	},
	models.StatusShippingAssigned: {
		models.StatusInTransit,
		models.StatusCancelled,
	},
	models.StatusInTransit: {
		models.StatusCompleted,
		models.StatusIssueReported,
		models.StatusCancelled,
	},
	models.StatusIssueReported: {
		models.StatusInTransit, // Resume after resolving issue
		models.StatusCompleted, // Complete despite issue
		models.StatusCancelled, // Cancel due to issue
	},
	models.StatusCompleted: {
		// Terminal state - no transitions
	},
	models.StatusCancelled: {
		// Terminal state - no transitions
	},
}

// ValidateStatusTransition checks if status transition is allowed
func ValidateStatusTransition(currentStatus, newStatus models.ShipmentStatus) error {
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
func GetAllowedTransitions(currentStatus models.ShipmentStatus) []models.ShipmentStatus {
	return validTransitions[currentStatus]
}

type TransitionRules struct {
	RequiresShipper bool
	RequiresDevice  bool
	RequiresRules   bool
	RulesConfirmed  bool
	CanCancel       bool
}

var transitionRules = map[models.ShipmentStatus]TransitionRules{
	models.StatusDemandCreated: {
		RequiresShipper: false,
		RequiresDevice:  false,
		RequiresRules:   false,
		RulesConfirmed:  false,
		CanCancel:       true,
	},
	models.StatusOrderPosted: {
		RequiresShipper: false,
		RequiresDevice:  false,
		RequiresRules:   true, // Rules must be set before posting
		RulesConfirmed:  false,
		CanCancel:       true,
	},
	models.StatusShippingAssigned: {
		RequiresShipper: true,
		RequiresDevice:  true, // Device must be assigned when shipper accepts
		RequiresRules:   true,
		RulesConfirmed:  false,
		CanCancel:       true,
	},
	models.StatusInTransit: {
		RequiresShipper: true,
		RequiresDevice:  true,
		RequiresRules:   true,
		RulesConfirmed:  true,  // Shipper must confirm rules before starting
		CanCancel:       false, // Cannot cancel once in transit
	},
	models.StatusIssueReported: {
		RequiresShipper: true,
		RequiresDevice:  true,
		RequiresRules:   true,
		RulesConfirmed:  true,
		CanCancel:       true,
	},
}

// ValidateBusinessRules checks if shipment meets requirements for status
func ValidateBusinessRules(shipment *models.Shipment, targetStatus models.ShipmentStatus) error {
	rules, exists := transitionRules[targetStatus]
	if !exists {
		return nil // No specific rules for this status
	}

	if rules.RequiresShipper && shipment.ShipperID == nil {
		return appErrors.NewAppError(
			"SHIPPER_REQUIRED",
			fmt.Sprintf("Shipper is required for status %s", targetStatus),
			nil,
		)
	}

	if rules.RequiresDevice && shipment.LinkedDeviceID == nil {
		return appErrors.NewAppError(
			"DEVICE_REQUIRED",
			fmt.Sprintf("Device is required for status %s", targetStatus),
			nil,
		)
	}

	if rules.RequiresRules && shipment.Rules == nil {
		return appErrors.NewAppError(
			"RULES_REQUIRED",
			fmt.Sprintf("Shipping rules are required for status %s", targetStatus),
			nil,
		)
	}

	if rules.RulesConfirmed && (shipment.Rules == nil || shipment.Rules.ConfirmedByShipperID == nil) {
		return appErrors.NewAppError(
			"RULES_NOT_CONFIRMED",
			"Shipper must confirm quality rules before starting transit",
			nil,
		)
	}

	return nil
}
