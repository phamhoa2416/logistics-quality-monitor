package device

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for device repository operations
type Repository interface {
	Create(ctx context.Context, device *Device) error
	GetByID(ctx context.Context, deviceID uuid.UUID) (*Device, error)
	GetByHardwareUID(ctx context.Context, hardwareUID string) (*Device, error)
	Update(ctx context.Context, device *Device) error
	Delete(ctx context.Context, deviceID uuid.UUID) error
	AssignOwner(ctx context.Context, deviceID, shipperID uuid.UUID) error
	UnassignOwner(ctx context.Context, deviceID uuid.UUID) error
	UpdateStatus(ctx context.Context, deviceID uuid.UUID, status DeviceStatus) error
	UpdateBattery(ctx context.Context, deviceID uuid.UUID, batteryLevel int) error
	UpdateLastSeen(ctx context.Context, deviceID uuid.UUID) error
	List(ctx context.Context, filter *Filter) ([]*Device, int64, error)
	GetStatistics(ctx context.Context) (*Statistics, error)
}

// Filter represents filtering options for listing devices
type Filter struct {
	Status         *DeviceStatus
	OwnerShipperID *uuid.UUID
	MinBattery     *int
	MaxBattery     *int
	IsOffline      *bool
	Search         string
	Page           int
	PageSize       int
	SortBy         string
	SortOrder      string
}

// Statistics represents device statistics
type Statistics struct {
	TotalDevices       int
	AvailableDevices   int
	InTransitDevices   int
	MaintenanceDevices int
	RetiredDevices     int
	ByOwner            []OwnerStats
	LowBatteryDevices  int
	OfflineDevices     int
}

// OwnerStats represents statistics by owner
type OwnerStats struct {
	OwnerID     string
	OwnerName   string
	DeviceCount int
}
