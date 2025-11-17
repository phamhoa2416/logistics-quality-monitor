package ingestion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"logistics-quality-monitor/internal/infrastructure/database/postgres"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *postgres.DB) *Repository {
	return &Repository{db: db.DB}
}

// BatchInsertSensorData inserts multiple sensor data records
func (r *Repository) BatchInsertSensorData(ctx context.Context, records []SensorDataRecord) error {
	if len(records) == 0 {
		return nil
	}

	models := make([]sensorDataModel, len(records))
	for i, record := range records {
		models[i] = sensorDataModel{
			Time:           record.Time,
			DeviceID:       record.DeviceID,
			Latitude:       record.Latitude,
			Longitude:      record.Longitude,
			Altitude:       record.Altitude,
			Speed:          record.Speed,
			Temperature:    record.Temperature,
			Humidity:       record.Humidity,
			LightLevel:     record.LightLevel,
			TiltAngle:      record.TiltAngle,
			ImpactG:        record.ImpactG,
			BatteryLevel:   record.BatteryLevel,
			SignalStrength: record.SignalStrength,
		}
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(models, 500).Error; err != nil {
			return fmt.Errorf("failed to insert sensor batch: %w", err)
		}
		return nil
	})
}

// UpdateDeviceLastSeen updates device last seen timestamp and battery
func (r *Repository) UpdateDeviceLastSeen(ctx context.Context, deviceID uuid.UUID, batteryLevel *int) error {
	now := time.Now()
	updates := map[string]interface{}{
		"last_seen_at": now,
		"updated_at":   now,
	}

	if batteryLevel != nil {
		updates["battery_level"] = *batteryLevel
	}

	return r.db.WithContext(ctx).
		Table("devices").
		Where("id = ?", deviceID).
		Updates(updates).Error
}

// GetDeviceShipment retrieves active shipment for device
func (r *Repository) GetDeviceShipment(ctx context.Context, deviceID uuid.UUID) (*DeviceShipmentInfo, error) {
	query := `
        SELECT 
            s.id,
            s.status,
            s.customer_id,
            s.provider_id,
            s.shipper_id,
            sr.id AS rules_id,
            sr.report_cycle_sec,
            sr.temp_min,
            sr.temp_max,
            sr.humidity_min,
            sr.humidity_max,
            sr.light_max,
            sr.tilt_max_angle,
            sr.impact_threshold_g
        FROM devices d
        INNER JOIN shipments s ON d.current_shipment_id = s.id
        LEFT JOIN shipping_rules sr ON s.id = sr.shipment_id
        WHERE d.id = ? AND s.status = 'in_transit'
    `

	var row deviceShipmentRow
	result := r.db.WithContext(ctx).Raw(query, deviceID).Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch shipment info: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	info := &DeviceShipmentInfo{
		ShipmentID:       row.ShipmentID,
		ShipmentStatus:   row.ShipmentStatus,
		CustomerID:       row.CustomerID,
		ProviderID:       row.ProviderID,
		ShipperID:        row.ShipperID,
		ReportCycleSec:   row.ReportCycleSec,
		TempMin:          row.TempMin,
		TempMax:          row.TempMax,
		HumidityMin:      row.HumidityMin,
		HumidityMax:      row.HumidityMax,
		LightMax:         row.LightMax,
		TiltMaxAngle:     row.TiltMaxAngle,
		ImpactThresholdG: row.ImpactThresholdG,
		HasRules:         row.RulesID != nil,
	}

	return info, nil
}

// InsertAlert inserts a new alert
func (r *Repository) InsertAlert(ctx context.Context, alert *Alert) error {
	if alert == nil {
		return errors.New("alert payload is nil")
	}

	model := alertModel{
		Time:           alert.Time,
		DeviceID:       alert.DeviceID,
		ShipmentID:     alert.ShipmentID,
		AlertType:      alert.AlertType,
		Severity:       alert.Severity,
		ViolationType:  alert.ViolationType,
		TriggerValue:   alert.TriggerValue,
		ThresholdValue: alert.ThresholdValue,
		Message:        alert.Message,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	return nil
}

// DeviceShipmentInfo contains device's active shipment and rules
type DeviceShipmentInfo struct {
	ShipmentID       uuid.UUID
	ShipmentStatus   string
	CustomerID       uuid.UUID
	ProviderID       uuid.UUID
	ShipperID        *uuid.UUID
	HasRules         bool
	ReportCycleSec   *int
	TempMin          *float64
	TempMax          *float64
	HumidityMin      *float64
	HumidityMax      *float64
	LightMax         *float64
	TiltMaxAngle     *float64
	ImpactThresholdG *float64
}

// Alert represents an alert record
type Alert struct {
	Time           time.Time
	DeviceID       uuid.UUID
	ShipmentID     uuid.UUID
	AlertType      string
	Severity       string
	ViolationType  string
	TriggerValue   string
	ThresholdValue string
	Message        string
}

type sensorDataModel struct {
	Time           time.Time `gorm:"column:time"`
	DeviceID       uuid.UUID `gorm:"column:device_id"`
	Latitude       *float64  `gorm:"column:latitude"`
	Longitude      *float64  `gorm:"column:longitude"`
	Altitude       *float64  `gorm:"column:altitude"`
	Speed          *float64  `gorm:"column:speed"`
	Temperature    *float64  `gorm:"column:temperature"`
	Humidity       *float64  `gorm:"column:humidity"`
	LightLevel     *float64  `gorm:"column:light_level"`
	TiltAngle      *float64  `gorm:"column:tilt_angle"`
	ImpactG        *float64  `gorm:"column:impact_g"`
	BatteryLevel   *int      `gorm:"column:battery_level"`
	SignalStrength *int      `gorm:"column:signal_strength"`
}

func (sensorDataModel) TableName() string {
	return "sensor_data"
}

type alertModel struct {
	Time           time.Time `gorm:"column:time"`
	DeviceID       uuid.UUID `gorm:"column:device_id"`
	ShipmentID     uuid.UUID `gorm:"column:shipment_id"`
	AlertType      string    `gorm:"column:alert_type"`
	Severity       string    `gorm:"column:severity"`
	ViolationType  string    `gorm:"column:violation_type"`
	TriggerValue   string    `gorm:"column:trigger_value"`
	ThresholdValue string    `gorm:"column:threshold_value"`
	Message        string    `gorm:"column:message"`
}

func (alertModel) TableName() string {
	return "alerts"
}

type deviceShipmentRow struct {
	ShipmentID       uuid.UUID
	ShipmentStatus   string
	CustomerID       uuid.UUID
	ProviderID       uuid.UUID
	ShipperID        *uuid.UUID
	RulesID          *uuid.UUID
	ReportCycleSec   *int
	TempMin          *float64
	TempMax          *float64
	HumidityMin      *float64
	HumidityMax      *float64
	LightMax         *float64
	TiltMaxAngle     *float64
	ImpactThresholdG *float64
}
