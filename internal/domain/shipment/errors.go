package shipment

import "errors"

var (
	ErrShipmentNotFound        = errors.New("shipment not found")
	ErrShipmentAlreadyExists   = errors.New("shipment already exists")
	ErrInvalidStatus           = errors.New("invalid shipment status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrShipperRequired         = errors.New("shipper is required")
	ErrDeviceRequired          = errors.New("device is required")
	ErrRulesRequired           = errors.New("shipping rules are required")
	ErrRulesNotConfirmed       = errors.New("rules not confirmed by shipper")
	ErrShipmentInTransit       = errors.New("shipment is in transit")
	ErrShipmentCompleted       = errors.New("shipment is already completed")
	ErrShipmentCancelled       = errors.New("shipment is cancelled")
	ErrInvalidParties          = errors.New("invalid parties")
	ErrDeviceUnavailable       = errors.New("device is unavailable")
)
