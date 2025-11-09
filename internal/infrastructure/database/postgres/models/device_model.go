package models

import (
	"time"

	"github.com/google/uuid"
)

// DeviceModel represents the database model for Devices.
type DeviceModel struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	HardwareUID       string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	DeviceName        *string    `gorm:"type:varchar(255)"`
	Model             *string    `gorm:"type:varchar(255)"`
	OwnerShipperID    *uuid.UUID `gorm:"type:uuid;index"`
	CurrentShipmentID *uuid.UUID `gorm:"type:uuid"`
	Status            string     `gorm:"type:varchar(50);not null;default:'available'"`
	FirmwareVersion   *string    `gorm:"type:varchar(100)"`
	BatteryLevel      *int       `gorm:"type:integer"`
	TotalTrips        int        `gorm:"type:integer;default:0"`
	LastSeenAt        *time.Time `gorm:"type:timestamp"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
}

func (DeviceModel) TableName() string {
	return "devices"
}
