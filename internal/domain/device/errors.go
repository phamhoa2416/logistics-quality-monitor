package device

import "errors"

var (
	ErrDeviceNotFound          = errors.New("device not found")
	ErrDeviceAlreadyExists     = errors.New("device already exists")
	ErrDeviceInUse             = errors.New("device is in use")
	ErrInvalidStatus           = errors.New("invalid device status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrNoOwner                 = errors.New("device has no owner")
	ErrAssignmentFailed        = errors.New("assignment failed")
	ErrUnassignmentFailed      = errors.New("unassignment failed")
)
