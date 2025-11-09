package models

import (
	"time"

	"github.com/google/uuid"
)

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

// Shipment represents a shipping order
type Shipment struct {
	ID uuid.UUID `json:"id"`

	// Parties involved
	CustomerID uuid.UUID  `json:"customer_id"`
	ProviderID uuid.UUID  `json:"provider_id"`
	ShipperID  *uuid.UUID `json:"shipper_id"`

	// Device assignment
	LinkedDeviceID *uuid.UUID `json:"linked_device_id"`

	// Status
	Status ShipmentStatus `json:"status"`

	// Goods information
	GoodsDescription string   `json:"goods_description"`
	GoodsValue       *float64 `json:"goods_value"`
	GoodsWeight      *float64 `json:"goods_weight"`

	// Addresses
	PickupAddress   string `json:"pickup_address"`
	DeliveryAddress string `json:"delivery_address"`

	// Timing
	EstimatedPickupAt   *time.Time `json:"estimated_pickup_at"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at"`
	ActualPickupAt      *time.Time `json:"actual_pickup_at"`
	ActualDeliveryAt    *time.Time `json:"actual_delivery_at"`

	// Notes and feedback
	CustomerNotes   *string `json:"customer_notes"`
	CompletionNotes *string `json:"completion_notes"`
	CustomerRating  *int    `json:"customer_rating"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations (for joins)
	Customer *PartyInfo     `json:"customer,omitempty"`
	Provider *PartyInfo     `json:"provider,omitempty"`
	Shipper  *PartyInfo     `json:"shipper,omitempty"`
	Device   *DeviceInfo    `json:"device,omitempty"`
	Rules    *ShippingRules `json:"rules,omitempty"`

	// Computed fields
	DurationMinutes *int `json:"duration_minutes,omitempty"`
	IsDelayed       bool `json:"is_delayed,omitempty"`
	AlertsCount     int  `json:"alerts_count,omitempty"`
}

// PartyInfo represents user information in shipment context
type PartyInfo struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
	Phone    *string   `json:"phone"`
}

// DeviceInfo represents device information in shipment context
type DeviceInfo struct {
	ID           uuid.UUID `json:"id"`
	HardwareUID  string    `json:"hardware_uid"`
	DeviceName   *string   `json:"device_name"`
	BatteryLevel *int      `json:"battery_level"`
	IsOnline     bool      `json:"is_online"`
}

// ShippingRules represents quality control rules for shipment
type ShippingRules struct {
	ID                    uuid.UUID  `json:"id"`
	ShipmentID            uuid.UUID  `json:"shipment_id"`
	ReportCycleSec        int        `json:"report_cycle_sec"`
	TempMin               *float64   `json:"temp_min"`
	TempMax               *float64   `json:"temp_max"`
	HumidityMin           *float64   `json:"humidity_min"`
	HumidityMax           *float64   `json:"humidity_max"`
	LightMax              *float64   `json:"light_max"`
	TiltMaxAngle          *float64   `json:"tilt_max_angle"`
	ImpactThresholdG      *float64   `json:"impact_threshold_g"`
	EnablePredictiveAlert bool       `json:"enable_predictive_alert"`
	AlertBufferTimeMin    int        `json:"alert_buffer_time_min"`
	SetByProviderID       uuid.UUID  `json:"set_by_provider_id"`
	ConfirmedByShipperID  *uuid.UUID `json:"confirmed_by_shipper_id"`
	SetAt                 time.Time  `json:"set_at"`
	ConfirmedAt           *time.Time `json:"confirmed_at"`
}

// ShipmentStatistics for analytics
type ShipmentStatistics struct {
	TotalShipments      int               `json:"total_shipments"`
	ByStatus            map[string]int    `json:"by_status"`
	ActiveShipments     int               `json:"active_shipments"`
	CompletedToday      int               `json:"completed_today"`
	AverageDeliveryTime float64           `json:"average_delivery_time_hours"`
	OnTimeDeliveryRate  float64           `json:"on_time_delivery_rate"`
	IssueRate           float64           `json:"issue_rate"`
	TopShippers         []TopShipperStats `json:"top_shippers"`
	RevenueToday        float64           `json:"revenue_today"`
}

type TopShipperStats struct {
	ShipperID      uuid.UUID `json:"shipper_id"`
	ShipperName    string    `json:"shipper_name"`
	TotalShipments int       `json:"total_shipments"`
	CompletedRate  float64   `json:"completed_rate"`
	AvgRating      float64   `json:"avg_rating"`
}
