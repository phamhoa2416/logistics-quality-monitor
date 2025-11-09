package models

import (
	"github.com/google/uuid"
	"time"
)

type CreateDemandRequest struct {
	ProviderID          uuid.UUID  `json:"provider_id" validate:"required,uuid"`
	GoodsDescription    string     `json:"goods_description" validate:"required,min=10,max=1000"`
	GoodsValue          *float64   `json:"goods_value" validate:"omitempty,min=0"`
	GoodsWeight         *float64   `json:"goods_weight" validate:"omitempty,min=0"`
	PickupAddress       string     `json:"pickup_address" validate:"required,min=10"`
	DeliveryAddress     string     `json:"delivery_address" validate:"required,min=10"`
	EstimatedPickupAt   *time.Time `json:"estimated_pickup_at" validate:"omitempty"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at" validate:"omitempty"`
	CustomerNotes       *string    `json:"customer_notes" validate:"omitempty,max=500"`
}

type PostOrderRequest struct {
	// Shipping rules (digital seal)
	ReportCycleSec        int      `json:"report_cycle_sec" validate:"required,min=10,max=300"`
	TempMin               *float64 `json:"temp_min" validate:"omitempty,min=-50,max=100"`
	TempMax               *float64 `json:"temp_max" validate:"omitempty,min=-50,max=100"`
	HumidityMin           *float64 `json:"humidity_min" validate:"omitempty,min=0,max=100"`
	HumidityMax           *float64 `json:"humidity_max" validate:"omitempty,min=0,max=100"`
	LightMax              *float64 `json:"light_max" validate:"omitempty,min=0"`
	TiltMaxAngle          *float64 `json:"tilt_max_angle" validate:"omitempty,min=0,max=90"`
	ImpactThresholdG      *float64 `json:"impact_threshold_g" validate:"omitempty,min=0,max=20"`
	EnablePredictiveAlert bool     `json:"enable_predictive_alert"`
	AlertBufferTimeMin    int      `json:"alert_buffer_time_min" validate:"omitempty,min=5,max=120"`
}

type AcceptOrderRequest struct {
	DeviceID uuid.UUID `json:"device_id" validate:"required,uuid"`
}

type StartShippingRequest struct {
	ActualPickupAt *time.Time `json:"actual_pickup_at" validate:"omitempty"`
	Notes          *string    `json:"notes" validate:"omitempty,max=500"`
}

type CompleteDeliveryRequest struct {
	ActualDeliveryAt *time.Time `json:"actual_delivery_at" validate:"omitempty"`
	CompletionNotes  *string    `json:"completion_notes" validate:"omitempty,max=500"`
	ProofOfDelivery  *string    `json:"proof_of_delivery" validate:"omitempty"` // Photo URL, signature, etc.
}

type RateDeliveryRequest struct {
	Rating   int     `json:"rating" validate:"required,min=1,max=5"`
	Feedback *string `json:"feedback" validate:"omitempty,max=1000"`
}

type ReportIssueRequest struct {
	IssueType   string  `json:"issue_type" validate:"required,oneof=quality_violation accident theft delay other"`
	Description string  `json:"description" validate:"required,min=10,max=1000"`
	Severity    string  `json:"severity" validate:"required,oneof=low medium high critical"`
	Evidence    *string `json:"evidence" validate:"omitempty"` // Photo URLs, etc.
}

type UpdateShipmentRequest struct {
	GoodsDescription    *string    `json:"goods_description" validate:"omitempty,min=10,max=1000"`
	PickupAddress       *string    `json:"pickup_address" validate:"omitempty,min=10"`
	DeliveryAddress     *string    `json:"delivery_address" validate:"omitempty,min=10"`
	EstimatedPickupAt   *time.Time `json:"estimated_pickup_at" validate:"omitempty"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at" validate:"omitempty"`
	CustomerNotes       *string    `json:"customer_notes" validate:"omitempty,max=500"`
}

type CancelShipmentRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

type ShipmentFilterRequest struct {
	Status     *ShipmentStatus `form:"status"`
	CustomerID *uuid.UUID      `form:"customer_id"`
	ProviderID *uuid.UUID      `form:"provider_id"`
	ShipperID  *uuid.UUID      `form:"shipper_id"`
	DeviceID   *uuid.UUID      `form:"device_id"`

	// Date range filters
	CreatedAfter   *time.Time `form:"created_after"`
	CreatedBefore  *time.Time `form:"created_before"`
	DeliveryAfter  *time.Time `form:"delivery_after"`
	DeliveryBefore *time.Time `form:"delivery_before"`

	// Boolean filters
	HasIssues *bool `form:"has_issues"`
	IsDelayed *bool `form:"is_delayed"`
	HasDevice *bool `form:"has_device"`

	// Search
	Search string `form:"search"`

	// Pagination
	Page      int    `form:"page" validate:"omitempty,min=1"`
	PageSize  int    `form:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy    string `form:"sort_by" validate:"omitempty,oneof=created_at updated_at estimated_delivery_at actual_delivery_at goods_value"`
	SortOrder string `form:"sort_order" validate:"omitempty,oneof=asc desc"`
}
