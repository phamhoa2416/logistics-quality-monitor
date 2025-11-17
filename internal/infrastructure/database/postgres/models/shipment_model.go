package models

import (
	"time"

	"github.com/google/uuid"
)

// ShipmentModel represents the database model for Shipments
type ShipmentModel struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CustomerID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	ProviderID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	ShipperID           *uuid.UUID `gorm:"type:uuid;index"`
	LinkedDeviceID      *uuid.UUID `gorm:"type:uuid"`
	Status              string     `gorm:"type:shipment_status;not null;default:'demand_created';index"`
	GoodsDescription    string     `gorm:"type:text;not null"`
	GoodsValue          *float64   `gorm:"type:decimal(12,2)"`
	GoodsWeight         *float64   `gorm:"type:decimal(8,2)"`
	PickupAddress       string     `gorm:"type:text;not null"`
	DeliveryAddress     string     `gorm:"type:text;not null"`
	EstimatedPickupAt   *time.Time `gorm:"type:timestamptz"`
	EstimatedDeliveryAt *time.Time `gorm:"type:timestamptz"`
	ActualPickupAt      *time.Time `gorm:"type:timestamptz"`
	ActualDeliveryAt    *time.Time `gorm:"type:timestamptz"`
	CustomerNotes       *string    `gorm:"type:text"`
	CompletionNotes     *string    `gorm:"type:text"`
	CustomerRating      *int       `gorm:"type:integer;check:customer_rating >= 1 AND customer_rating <= 5"`
	CreatedAt           time.Time  `gorm:"not null;index"`
	UpdatedAt           time.Time  `gorm:"not null"`

	// Relations
	Customer *UserModel   `gorm:"foreignKey:CustomerID"`
	Provider *UserModel   `gorm:"foreignKey:ProviderID"`
	Shipper  *UserModel   `gorm:"foreignKey:ShipperID"`
	Device   *DeviceModel `gorm:"foreignKey:LinkedDeviceID"`
}

func (ShipmentModel) TableName() string {
	return "shipments"
}

// ShippingRulesModel represents the database model for ShippingRules
type ShippingRulesModel struct {
	ID                    uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ShipmentID            uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	ReportCycleSec        int        `gorm:"type:integer;not null"`
	TempMin               *float64   `gorm:"type:decimal(5,2)"`
	TempMax               *float64   `gorm:"type:decimal(5,2)"`
	HumidityMin           *float64   `gorm:"type:decimal(5,2)"`
	HumidityMax           *float64   `gorm:"type:decimal(5,2)"`
	LightMax              *float64   `gorm:"type:decimal(10,2)"`
	TiltMaxAngle          *float64   `gorm:"type:decimal(5,2)"`
	ImpactThresholdG      *float64   `gorm:"type:decimal(5,2)"`
	EnablePredictiveAlert bool       `gorm:"default:false;not null"`
	AlertBufferTimeMin    int        `gorm:"type:integer;default:0"`
	SetByProviderID       uuid.UUID  `gorm:"type:uuid;not null"`
	ConfirmedByShipperID  *uuid.UUID `gorm:"type:uuid"`
	SetAt                 time.Time  `gorm:"not null"`
	ConfirmedAt           *time.Time `gorm:"type:timestamptz"`

	Shipment *ShipmentModel `gorm:"foreignKey:ShipmentID"`
}

func (ShippingRulesModel) TableName() string {
	return "shipping_rules"
}
