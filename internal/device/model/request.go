package model

import "github.com/google/uuid"

type CreateDeviceRequest struct {
	HardwareUID     string     `json:"hardware_uid" validate:"required,min=5,max=255"`
	DeviceName      *string    `json:"device_name" validate:"omitempty,min=2,max=100"`
	Model           *string    `json:"model" validate:"omitempty,max=50"`
	OwnerShipperID  *uuid.UUID `json:"owner_shipper_id" validate:"omitempty,uuid"`
	FirmwareVersion *string    `json:"firmware_version" validate:"omitempty,max=50"`
}

type UpdateDeviceRequest struct {
	DeviceName      *string `json:"device_name" validate:"omitempty,min=2,max=100"`
	Model           *string `json:"model" validate:"omitempty,max=50"`
	FirmwareVersion *string `json:"firmware_version" validate:"omitempty,max=50"`
	Status          *string `json:"status" validate:"omitempty,oneof=available in_transit maintenance retired"`
}

type AssignOwnerRequest struct {
	OwnerShipperID uuid.UUID `json:"owner_shipper_id" validate:"required,uuid"`
}

type UnassignOwnerRequest struct {
	Reason string `json:"reason" validate:"omitempty,max=500"`
}

type UpdateStatusRequest struct {
	Status DeviceStatus `json:"status" validate:"required,oneof=available in_transit maintenance retired"`
	Reason string       `json:"reason" validate:"omitempty,max=500"`
}

type UpdateBatteryRequest struct {
	BatteryLevel int `json:"battery_level" validate:"required,min=0,max=100"`
}

type BulkAssignRequest struct {
	DeviceIDs      []uuid.UUID `json:"device_ids" validate:"required,min=1,dive,uuid"`
	OwnerShipperID uuid.UUID   `json:"owner_shipper_id" validate:"required,uuid"`
}

type DeviceFilterRequest struct {
	Status         *DeviceStatus `form:"status"`
	OwnerShipperID *uuid.UUID    `form:"owner_shipper_id"`
	MinBattery     *int          `form:"min_battery" validate:"omitempty,min=0,max=100"`
	MaxBattery     *int          `form:"max_battery" validate:"omitempty,min=0,max=100"`
	IsOffline      *bool         `form:"is_offline"`
	Search         string        `form:"search"`
	Page           int           `form:"page" validate:"omitempty,min=1"`
	PageSize       int           `form:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy         string        `form:"sort_by" validate:"omitempty,oneof=created_at updated_at battery_level total_trips last_seen_at"`
	SortOrder      string        `form:"sort_order" validate:"omitempty,oneof=asc desc"`
}
