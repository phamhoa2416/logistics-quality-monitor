package repository

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/internal/database"
	"logistics-quality-monitor/internal/device/model"
	appErrors "logistics-quality-monitor/pkg/errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *database.Database
}

func NewRepository(db *database.Database) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) CreateDevice(ctx context.Context, device *model.Device) error {
	device.ID = uuid.New()
	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()
	device.Status = model.StatusAvailable
	device.TotalTrips = 0

	if err := r.db.DB.WithContext(ctx).Create(device).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return appErrors.NewAppError("DEVICE_ALREADY_EXISTS", "Device with this hardware UID already exists", err)
		}
		return fmt.Errorf("failed to create device: %w", err)
	}

	return nil
}

func (r *DeviceRepository) GetDeviceByID(ctx context.Context, deviceID uuid.UUID) (*model.Device, error) {
	var device model.Device
	err := r.db.DB.WithContext(ctx).
		Preload("OwnerShipper").
		Where("id = ?", deviceID).
		First(&device).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.NewAppError("DEVICE_NOT_FOUND", "Device not found", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return &device, nil
}

func (r *DeviceRepository) GetDeviceByHardwareUID(ctx context.Context, hardwareUID string) (*model.Device, error) {
	var device model.Device
	err := r.db.DB.WithContext(ctx).
		Where("hardware_uid = ?", hardwareUID).
		First(&device).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.NewAppError("DEVICE_NOT_FOUND", "Device not found", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return &device, nil
}

func (r *DeviceRepository) UpdateDevice(ctx context.Context, device *model.Device) error {
	device.UpdatedAt = time.Now()
	result := r.db.DB.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ?", device.ID).
		Updates(map[string]interface{}{
			"device_name":      device.DeviceName,
			"model":            device.Model,
			"status":           device.Status,
			"firmware_version": device.FirmwareVersion,
			"updated_at":       device.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update device: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return appErrors.NewAppError("DEVICE_NOT_FOUND", "Device not found", nil)
	}
	return nil
}

func (r *DeviceRepository) AssignOwner(ctx context.Context, deviceID, shipperID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ? AND (owner_shipper_id IS NULL OR owner_shipper_id != ?)", deviceID, shipperID).
		Updates(map[string]interface{}{
			"owner_shipper_id": shipperID,
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to assign owner: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.NewAppError("ASSIGNMENT_FAILED", "Device already assigned to this shipper or not found", nil)
	}

	return nil
}

func (r *DeviceRepository) UnassignOwner(ctx context.Context, deviceID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ? AND owner_shipper_id IS NOT NULL", deviceID).
		Updates(map[string]interface{}{
			"owner_shipper_id": nil,
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to unassign owner: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.NewAppError("UNASSIGNMENT_FAILED", "Device has no owner or not found", nil)
	}

	return nil
}

func (r *DeviceRepository) UpdateStatus(ctx context.Context, deviceID uuid.UUID, status model.DeviceStatus) error {
	result := r.db.DB.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.NewAppError("DEVICE_NOT_FOUND", "Device not found", nil)
	}

	return nil
}

func (r *DeviceRepository) UpdateBattery(ctx context.Context, deviceID uuid.UUID, batteryLevel int) error {
	result := r.db.DB.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"battery_level": batteryLevel,
			"updated_at":    time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update battery: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.NewAppError("DEVICE_NOT_FOUND", "Device not found", nil)
	}

	return nil
}

func (r *DeviceRepository) UpdateLastSeen(ctx context.Context, deviceID uuid.UUID) error {
	now := time.Now()
	return r.db.DB.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"last_seen_at": now,
			"updated_at":   now,
		}).Error
}

func (r *DeviceRepository) DeleteDevice(ctx context.Context, deviceID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ? AND current_shipment_id IS NULL", deviceID).
		Updates(map[string]interface{}{
			"status":     model.StatusRetired,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to delete device: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.NewAppError("DELETE_FAILED", "Cannot delete device in use or not found", nil)
	}

	return nil
}

func (r *DeviceRepository) GetStatistics(ctx context.Context) (*model.DeviceStatistics, error) {
	stats := &model.DeviceStatistics{}
	err := r.db.DB.WithContext(ctx).Raw(`
        SELECT 
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE status = 'available') as available,
            COUNT(*) FILTER (WHERE status = 'in_transit') as in_transit,
            COUNT(*) FILTER (WHERE status = 'maintenance') as maintenance,
            COUNT(*) FILTER (WHERE status = 'retired') as retired,
            COUNT(*) FILTER (WHERE battery_level < 20) as low_battery,
            COUNT(*) FILTER (WHERE last_seen_at IS NULL OR last_seen_at < NOW() - INTERVAL '5 minutes') as offline
        FROM devices
    `).Scan(stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	var ownerStats []model.OwnerStats
	err = r.db.DB.WithContext(ctx).Raw(`
        SELECT 
            u.id, u.full_name, COUNT(d.id) as device_count
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

func (r *DeviceRepository) ListDevices(ctx context.Context, filter *model.DeviceFilterRequest) ([]model.Device, int64, error) {
	var devices []model.Device
	var total int64

	db := r.db.DB.WithContext(ctx).Model(&model.Device{}).Preload("OwnerShipper").Joins("LEFT JOIN users u ON devices.owner_shipper_id = u.id")

	if filter.Status != nil {
		db = db.Where("devices.status = ?", *filter.Status)
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

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count devices: %w", err)
	}

	sortBy := "devices.created_at"
	if filter.SortBy != "" {
		sortBy = "devices." + filter.SortBy
	}
	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	err := db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&devices).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list devices: %w", err)
	}

	return devices, total, nil
}
