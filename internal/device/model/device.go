package model

import (
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID                uuid.UUID    `json:"id"`
	HardwareUID       string       `json:"hardware_uid"`
	DeviceName        *string      `json:"device_name,omitempty"`
	Model             *string      `json:"model,omitempty"`
	OwnerShipperID    *uuid.UUID   `json:"owner_shipper_id,omitempty"`
	CurrentShipmentID *uuid.UUID   `json:"current_shipment_id,omitempty"`
	Status            DeviceStatus `json:"status"`
	FirmwareVersion   *string      `json:"firmware_version,omitempty"`
	BatteryLevel      *int         `json:"battery_level,omitempty"`
	TotalTrips        int          `json:"total_trips,omitempty"`
	LastSeenAt        *time.Time   `json:"last_seen_at,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`

	OwnerShipper *OwnerInfo `json:"owner_shipper,omitempty"`
}

type OwnerInfo struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	PhoneNumber *string   `json:"phone_number"`
}
