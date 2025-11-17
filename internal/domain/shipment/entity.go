package shipment

import (
	"time"

	"github.com/google/uuid"
)

// ShipmentStatus represents the status of a shipment
type ShipmentStatus string

const (
	StatusDemandCreated    ShipmentStatus = "demand_created"    // Customer creates demand
	StatusOrderPosted      ShipmentStatus = "order_posted"      // Provider posts to marketplace
	StatusShippingAssigned ShipmentStatus = "shipping_assigned" // Shipper accepts order
	StatusInTransit        ShipmentStatus = "in_transit"        // Actively shipping
	StatusCompleted        ShipmentStatus = "completed"         // Successfully delivered
	StatusIssueReported    ShipmentStatus = "issue_reported"    // Problem during shipping
	StatusCancelled        ShipmentStatus = "cancelled"         // Cancelled before completion
)

// Shipment represents a shipping order entity in the domain
type Shipment struct {
	ID uuid.UUID

	// Parties involved
	CustomerID uuid.UUID
	ProviderID uuid.UUID
	ShipperID  *uuid.UUID

	// Device assignment
	LinkedDeviceID *uuid.UUID

	// Status
	Status ShipmentStatus

	// Goods information
	GoodsDescription string
	GoodsValue       *float64
	GoodsWeight      *float64

	// Addresses
	PickupAddress   string
	DeliveryAddress string

	// Timing
	EstimatedPickupAt   *time.Time
	EstimatedDeliveryAt *time.Time
	ActualPickupAt      *time.Time
	ActualDeliveryAt    *time.Time

	// Notes and feedback
	CustomerNotes   *string
	CompletionNotes *string
	CustomerRating  *int

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ShippingRules represents quality control rules for shipment
type ShippingRules struct {
	ID                    uuid.UUID
	ShipmentID            uuid.UUID
	ReportCycleSec        int
	TempMin               *float64
	TempMax               *float64
	HumidityMin           *float64
	HumidityMax           *float64
	LightMax              *float64
	TiltMaxAngle          *float64
	ImpactThresholdG      *float64
	EnablePredictiveAlert bool
	AlertBufferTimeMin    int
	SetByProviderID       uuid.UUID
	ConfirmedByShipperID  *uuid.UUID
	SetAt                 time.Time
	ConfirmedAt           *time.Time
}

// Statistics represents shipment statistics
type Statistics struct {
	TotalShipments      int
	ByStatus            map[string]int
	ActiveShipments     int
	CompletedToday      int
	AverageDeliveryTime float64
	OnTimeDeliveryRate  float64
	IssueRate           float64
	TopShippers         []TopShipperStats
	RevenueToday        float64
}

// TopShipperStats represents statistics by shipper
type TopShipperStats struct {
	ShipperID      uuid.UUID
	ShipperName    string
	TotalShipments int
	CompletedRate  float64
	AvgRating      float64
}
