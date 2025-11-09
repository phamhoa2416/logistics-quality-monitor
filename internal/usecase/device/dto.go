package device

import (
	"time"

	"github.com/google/uuid"
	domainDevice "logistics-quality-monitor/internal/domain/device"
)

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
	Status domainDevice.DeviceStatus `json:"status" validate:"required,oneof=available in_transit maintenance retired"`
	Reason string                    `json:"reason" validate:"omitempty,max=500"`
}

type UpdateBatteryRequest struct {
	BatteryLevel int `json:"battery_level" validate:"required,min=0,max=100"`
}

type BulkAssignRequest struct {
	DeviceIDs      []uuid.UUID `json:"device_ids" validate:"required,min=1,dive,uuid"`
	OwnerShipperID uuid.UUID   `json:"owner_shipper_id" validate:"required,uuid"`
}

type DeviceFilterRequest struct {
	Status         *domainDevice.DeviceStatus `form:"status"`
	OwnerShipperID *uuid.UUID                 `form:"owner_shipper_id"`
	MinBattery     *int                       `form:"min_battery" validate:"omitempty,min=0,max=100"`
	MaxBattery     *int                       `form:"max_battery" validate:"omitempty,min=0,max=100"`
	IsOffline      *bool                      `form:"is_offline"`
	Search         string                     `form:"search"`
	Page           int                        `form:"page" validate:"omitempty,min=1"`
	PageSize       int                        `form:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy         string                     `form:"sort_by" validate:"omitempty,oneof=created_at updated_at battery_level total_trips last_seen_at"`
	SortOrder      string                     `form:"sort_order" validate:"omitempty,oneof=asc desc"`
}

type DeviceResponse struct {
	ID                uuid.UUID                 `json:"id"`
	HardwareUID       string                    `json:"hardware_uid"`
	DeviceName        *string                   `json:"device_name"`
	Model             *string                   `json:"model"`
	OwnerShipperID    *uuid.UUID                `json:"owner_shipper_id"`
	CurrentShipmentID *uuid.UUID                `json:"current_shipment_id"`
	Status            domainDevice.DeviceStatus `json:"status"`
	FirmwareVersion   *string                   `json:"firmware_version"`
	BatteryLevel      *int                      `json:"battery_level"`
	TotalTrips        int                       `json:"total_trips"`
	LastSeenAt        *time.Time                `json:"last_seen_at"`
	IsOnline          bool                      `json:"is_online"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
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

type DeviceStatisticsResponse struct {
	TotalDevices       int          `json:"total_devices"`
	AvailableDevices   int          `json:"available_devices"`
	InTransitDevices   int          `json:"in_transit_devices"`
	MaintenanceDevices int          `json:"maintenance_devices"`
	RetiredDevices     int          `json:"retired_devices"`
	ByOwner            []OwnerStats `json:"by_owner"`
	LowBatteryDevices  int          `json:"low_battery_devices"`
	OfflineDevices     int          `json:"offline_devices"`
}

type OwnerStats struct {
	OwnerID     string `json:"owner_id"`
	OwnerName   string `json:"owner_name"`
	DeviceCount int    `json:"device_count"`
}

func ToDeviceResponse(d *domainDevice.Device) *DeviceResponse {
	if d == nil {
		return nil
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
		IsOnline:          d.IsOnline(),
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

func ToDomainFilter(req *DeviceFilterRequest) *domainDevice.Filter {
	if req == nil {
		return &domainDevice.Filter{}
	}
	return &domainDevice.Filter{
		Status:         req.Status,
		OwnerShipperID: req.OwnerShipperID,
		MinBattery:     req.MinBattery,
		MaxBattery:     req.MaxBattery,
		IsOffline:      req.IsOffline,
		Search:         req.Search,
		Page:           req.Page,
		PageSize:       req.PageSize,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
	}
}

func ToStatisticsResponse(s *domainDevice.Statistics) *DeviceStatisticsResponse {
	if s == nil {
		return nil
	}
	ownerStats := make([]OwnerStats, len(s.ByOwner))
	for i, stat := range s.ByOwner {
		ownerStats[i] = OwnerStats{
			OwnerID:     stat.OwnerID,
			OwnerName:   stat.OwnerName,
			DeviceCount: stat.DeviceCount,
		}
	}
	return &DeviceStatisticsResponse{
		TotalDevices:       s.TotalDevices,
		AvailableDevices:   s.AvailableDevices,
		InTransitDevices:   s.InTransitDevices,
		MaintenanceDevices: s.MaintenanceDevices,
		RetiredDevices:     s.RetiredDevices,
		ByOwner:            ownerStats,
		LowBatteryDevices:  s.LowBatteryDevices,
		OfflineDevices:     s.OfflineDevices,
	}
}
