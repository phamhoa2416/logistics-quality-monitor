package model

import (
	"time"

	"github.com/google/uuid"
	user "logistics-quality-monitor/internal/user/model"
)

type DeviceResponse struct {
	ID                uuid.UUID    `json:"id"`
	HardwareUID       string       `json:"hardware_uid"`
	DeviceName        *string      `json:"device_name"`
	Model             *string      `json:"model"`
	OwnerShipperID    *uuid.UUID   `json:"owner_shipper_id"`
	CurrentShipmentID *uuid.UUID   `json:"current_shipment_id"`
	Status            DeviceStatus `json:"status"`
	FirmwareVersion   *string      `json:"firmware_version"`
	BatteryLevel      *int         `json:"battery_level"`
	TotalTrips        int          `json:"total_trips"`
	LastSeenAt        *time.Time   `json:"last_seen_at"`
	IsOnline          bool         `json:"is_online"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	OwnerShipper      *user.User   `json:"owner_shipper,omitempty"`
}

type DeviceListResponse struct {
	Devices    []DeviceResponse `json:"devices"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

type BulkOperationResponse struct {
	SuccessCount int         `json:"success_count"`
	FailedCount  int         `json:"failed_count"`
	Errors       []BulkError `json:"errors,omitempty"`
}

type BulkError struct {
	DeviceID uuid.UUID `json:"device_id"`
	Error    string    `json:"error"`
}

func (d *Device) ToResponse() *DeviceResponse {
	isOnline := false
	if d.LastSeenAt != nil {
		isOnline = time.Since(*d.LastSeenAt) < 5*time.Minute
	}

	return &DeviceResponse{
		ID:                d.ID,
		HardwareUID:       d.HardwareUID,
		DeviceName:        d.DeviceName,
		Model:             d.Model,
		OwnerShipperID:    d.OwnerShipperID,
		CurrentShipmentID: d.CurrentShipmentID,
		Status:            d.Status,
		FirmwareVersion:   d.FirmwareVersion,
		BatteryLevel:      d.BatteryLevel,
		TotalTrips:        d.TotalTrips,
		LastSeenAt:        d.LastSeenAt,
		IsOnline:          isOnline,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
		OwnerShipper:      d.OwnerShipper,
	}
}
