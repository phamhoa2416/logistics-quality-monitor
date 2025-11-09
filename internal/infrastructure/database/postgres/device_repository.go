package postgres

import (
	"context"
	"errors"
	"fmt"
	domainDevice "logistics-quality-monitor/internal/domain/device"
	"logistics-quality-monitor/internal/infrastructure/database/postgres/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceRepository implements domain.Device.Repository interface
type DeviceRepository struct {
	db *DB
}

// NewDeviceRepository creates a new device repository
func NewDeviceRepository(db *DB) domainDevice.Repository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(ctx context.Context, d *domainDevice.Device) error {
	d.ID = uuid.New()
	d.CreatedAt = time.Now()
	d.UpdatedAt = time.Now()
	d.Status = domainDevice.StatusAvailable
	d.TotalTrips = 0

	dbModel := toDeviceModel(d)
	if err := r.db.DB.WithContext(ctx).Create(dbModel).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return domainDevice.ErrDeviceAlreadyExists
		}
		return fmt.Errorf("failed to create device: %w", err)
	}

	// Update domain entity with generated ID
	d.ID = dbModel.ID
	d.CreatedAt = dbModel.CreatedAt
	d.UpdatedAt = dbModel.UpdatedAt

	return nil
}

func (r *DeviceRepository) GetByID(ctx context.Context, deviceID uuid.UUID) (*domainDevice.Device, error) {
	var dbModel models.DeviceModel
	err := r.db.DB.WithContext(ctx).
		Preload("OwnerShipper").
		Where("id = ?", deviceID).
		First(&dbModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainDevice.ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return toDeviceEntity(&dbModel), nil
}

func (r *DeviceRepository) GetByHardwareUID(ctx context.Context, hardwareUID string) (*domainDevice.Device, error) {
	var dbModel models.DeviceModel
	err := r.db.DB.WithContext(ctx).
		Preload("OwnerShipper").
		Where("hardware_uid = ?", hardwareUID).
		First(&dbModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainDevice.ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return toDeviceEntity(&dbModel), nil
}

func (r *DeviceRepository) Update(ctx context.Context, d *domainDevice.Device) error {
	d.UpdatedAt = time.Now()

	result := r.db.DB.WithContext(ctx).
		Model(&models.DeviceModel{}).
		Where("id = ?", d.ID).
		Updates(map[string]interface{}{
			"device_name":      d.DeviceName,
			"model":            d.Model,
			"status":           string(d.Status),
			"firmware_version": d.FirmwareVersion,
			"updated_at":       d.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update device: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainDevice.ErrDeviceNotFound
	}

	return nil
}

func (r *DeviceRepository) AssignOwner(ctx context.Context, deviceID, shipperID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.DeviceModel{}).
		Where("id = ? AND (owner_shipper_id IS NULL OR owner_shipper_id != ?)", deviceID, shipperID).
		Updates(map[string]interface{}{
			"owner_shipper_id": shipperID,
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to assign owner: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainDevice.ErrAssignmentFailed
	}

	return nil
}

func (r *DeviceRepository) UnassignOwner(ctx context.Context, deviceID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.DeviceModel{}).
		Where("id = ? AND owner_shipper_id IS NOT NULL", deviceID).
		Updates(map[string]interface{}{
			"owner_shipper_id": nil,
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to unassign owner: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainDevice.ErrUnassignmentFailed
	}

	return nil
}

func (r *DeviceRepository) UpdateStatus(ctx context.Context, deviceID uuid.UUID, status domainDevice.DeviceStatus) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.DeviceModel{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"status":     string(status),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainDevice.ErrDeviceNotFound
	}

	return nil
}

func (r *DeviceRepository) UpdateBattery(ctx context.Context, deviceID uuid.UUID, batteryLevel int) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.DeviceModel{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"battery_level": batteryLevel,
			"updated_at":    time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update battery: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainDevice.ErrDeviceNotFound
	}

	return nil
}

func (r *DeviceRepository) UpdateLastSeen(ctx context.Context, deviceID uuid.UUID) error {
	now := time.Now()
	return r.db.DB.WithContext(ctx).
		Model(&models.DeviceModel{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"last_seen_at": now,
			"updated_at":   now,
		}).Error
}

func (r *DeviceRepository) Delete(ctx context.Context, deviceID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.DeviceModel{}).
		Where("id = ? AND current_shipment_id IS NULL", deviceID).
		Updates(map[string]interface{}{
			"status":     string(domainDevice.StatusRetired),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to delete device: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainDevice.ErrDeviceInUse
	}

	return nil
}

func (r *DeviceRepository) GetStatistics(ctx context.Context) (*domainDevice.Statistics, error) {
	stats := &domainDevice.Statistics{}
	err := r.db.DB.WithContext(ctx).Raw(`
        SELECT 
            COUNT(*) as total_devices,
            COUNT(*) FILTER (WHERE status = 'available') as available_devices,
            COUNT(*) FILTER (WHERE status = 'in_transit') as in_transit_devices,
            COUNT(*) FILTER (WHERE status = 'maintenance') as maintenance_devices,
            COUNT(*) FILTER (WHERE status = 'retired') as retired_devices,
            COUNT(*) FILTER (WHERE battery_level < 20) as low_battery_devices,
            COUNT(*) FILTER (WHERE last_seen_at IS NULL OR last_seen_at < NOW() - INTERVAL '5 minutes') as offline_devices
        FROM devices
    `).Scan(stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	var ownerStats []domainDevice.OwnerStats
	err = r.db.DB.WithContext(ctx).Raw(`
        SELECT 
            u.id::text as owner_id, u.full_name as owner_name, COUNT(d.id) as device_count
        FROM users u
        LEFT JOIN devices d ON u.id = d.owner_shipper_id
        WHERE u.role = 'shipper'
        GROUP BY u.id, u.full_name
        HAVING COUNT(d.id) > 0
        ORDER BY device_count DESC
    `).Scan(&ownerStats).Error

	if err == nil {
		stats.ByOwner = ownerStats
	}

	return stats, nil
}

func (r *DeviceRepository) List(ctx context.Context, filter *domainDevice.Filter) ([]*domainDevice.Device, int64, error) {
	var dbModels []models.DeviceModel
	var total int64

	db := r.db.DB.WithContext(ctx).Model(&models.DeviceModel{}).
		Preload("OwnerShipper").
		Joins("LEFT JOIN users u ON devices.owner_shipper_id = u.id")

	// Apply filters
	if filter.Status != nil {
		db = db.Where("devices.status = ?", string(*filter.Status))
	}
	if filter.OwnerShipperID != nil {
		db = db.Where("devices.owner_shipper_id = ?", *filter.OwnerShipperID)
	}
	if filter.MinBattery != nil {
		db = db.Where("devices.battery_level >= ?", *filter.MinBattery)
	}
	if filter.MaxBattery != nil {
		db = db.Where("devices.battery_level <= ?", *filter.MaxBattery)
	}
	if filter.IsOffline != nil && *filter.IsOffline {
		db = db.Where("(devices.last_seen_at IS NULL OR devices.last_seen_at < NOW() - INTERVAL '5 minutes')")
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		db = db.Where("devices.hardware_uid ILIKE ? OR devices.device_name ILIKE ?", search, search)
	}

	// Count total
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count devices: %w", err)
	}

	// Apply sorting
	sortBy := "devices.created_at"
	if filter.SortBy != "" {
		sortBy = "devices." + filter.SortBy
	}
	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	// Apply pagination
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Execute query
	err := db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&dbModels).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list devices: %w", err)
	}

	// Convert to domain entities
	devices := make([]*domainDevice.Device, len(dbModels))
	for i, dbModel := range dbModels {
		devices[i] = toDeviceEntity(&dbModel)
	}

	return devices, total, nil
}

// Helper functions to convert between domain entities and database models

func toDeviceModel(d *domainDevice.Device) *models.DeviceModel {
	return &models.DeviceModel{
		ID:                d.ID,
		HardwareUID:       d.HardwareUID,
		DeviceName:        d.DeviceName,
		Model:             d.Model,
		OwnerShipperID:    d.OwnerShipperID,
		CurrentShipmentID: d.CurrentShipmentID,
		Status:            string(d.Status),
		FirmwareVersion:   d.FirmwareVersion,
		BatteryLevel:      d.BatteryLevel,
		TotalTrips:        d.TotalTrips,
		LastSeenAt:        d.LastSeenAt,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

func toDeviceEntity(m *models.DeviceModel) *domainDevice.Device {
	status := domainDevice.DeviceStatus(m.Status)
	return &domainDevice.Device{
		ID:                m.ID,
		HardwareUID:       m.HardwareUID,
		DeviceName:        m.DeviceName,
		Model:             m.Model,
		OwnerShipperID:    m.OwnerShipperID,
		CurrentShipmentID: m.CurrentShipmentID,
		Status:            status,
		FirmwareVersion:   m.FirmwareVersion,
		BatteryLevel:      m.BatteryLevel,
		TotalTrips:        m.TotalTrips,
		LastSeenAt:        m.LastSeenAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}
