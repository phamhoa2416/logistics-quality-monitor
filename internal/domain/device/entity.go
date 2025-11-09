package device

import (
	"time"

	"github.com/google/uuid"
)

// Device represents a device entity in the domain
type Device struct {
	ID                uuid.UUID
	HardwareUID       string
	DeviceName        *string
	Model             *string
	OwnerShipperID    *uuid.UUID
	CurrentShipmentID *uuid.UUID
	Status            DeviceStatus
	FirmwareVersion   *string
	BatteryLevel      *int
	TotalTrips        int
	LastSeenAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// DeviceStatus represents the status of a device
type DeviceStatus string

const (
	StatusAvailable   DeviceStatus = "available"
	StatusInTransit   DeviceStatus = "in_transit"
	StatusMaintenance DeviceStatus = "maintenance"
	StatusRetired     DeviceStatus = "retired"
)

// IsOnline checks if the device is online (last seen within 5 minutes)
func (d *Device) IsOnline() bool {
	if d.LastSeenAt == nil {
		return false
	}
	return time.Since(*d.LastSeenAt) < 5*time.Minute
}
