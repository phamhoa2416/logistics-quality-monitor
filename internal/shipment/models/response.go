package models

import (
	"github.com/google/uuid"
	"time"
)

type ShipmentResponse struct {
	ID     uuid.UUID      `json:"id"`
	Status ShipmentStatus `json:"status"`

	// Parties
	Customer *PartyInfo `json:"customer"`
	Provider *PartyInfo `json:"provider"`
	Shipper  *PartyInfo `json:"shipper,omitempty"`

	// Device
	Device *DeviceInfo `json:"device,omitempty"`

	// Goods
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
	DurationMinutes     *int       `json:"duration_minutes"`

	// Status flags
	IsDelayed      bool `json:"is_delayed"`
	HasRules       bool `json:"has_rules"`
	RulesConfirmed bool `json:"rules_confirmed"`
	AlertsCount    int  `json:"alerts_count"`

	// Notes
	CustomerNotes   *string `json:"customer_notes"`
	CompletionNotes *string `json:"completion_notes"`
	CustomerRating  *int    `json:"customer_rating"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ShipmentDetailResponse struct {
	*ShipmentResponse
	Rules         *ShippingRules  `json:"rules,omitempty"`
	StatusHistory []StatusHistory `json:"status_history"`
	RecentAlerts  []AlertSummary  `json:"recent_alerts"`
}

type StatusHistory struct {
	FromStatus *ShipmentStatus `json:"from_status"`
	ToStatus   ShipmentStatus  `json:"to_status"`
	ChangedBy  *uuid.UUID      `json:"changed_by"`
	ChangedAt  time.Time       `json:"changed_at"`
	Notes      *string         `json:"notes"`
}

type AlertSummary struct {
	Time          time.Time `json:"time"`
	AlertType     string    `json:"alert_type"`
	Severity      string    `json:"severity"`
	ViolationType string    `json:"violation_type"`
	Message       string    `json:"message"`
}

type ShipmentListResponse struct {
	Shipments  []ShipmentResponse `json:"shipments"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

type MarketplaceListingResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Provider            *PartyInfo `json:"provider"`
	GoodsDescription    string     `json:"goods_description"`
	GoodsValue          *float64   `json:"goods_value"`
	GoodsWeight         *float64   `json:"goods_weight"`
	PickupAddress       string     `json:"pickup_address"`
	DeliveryAddress     string     `json:"delivery_address"`
	EstimatedPickupAt   *time.Time `json:"estimated_pickup_at"`
	EstimatedDeliveryAt *time.Time `json:"estimated_delivery_at"`
	HasQualityRules     bool       `json:"has_quality_rules"`
	PostedAt            time.Time  `json:"posted_at"`
	Distance            *float64   `json:"distance,omitempty"` // Calculated from shipper location
}

func (s *Shipment) ToResponse() *ShipmentResponse {
	resp := &ShipmentResponse{
		ID:                  s.ID,
		Status:              s.Status,
		Customer:            s.Customer,
		Provider:            s.Provider,
		Shipper:             s.Shipper,
		Device:              s.Device,
		GoodsDescription:    s.GoodsDescription,
		GoodsValue:          s.GoodsValue,
		GoodsWeight:         s.GoodsWeight,
		PickupAddress:       s.PickupAddress,
		DeliveryAddress:     s.DeliveryAddress,
		EstimatedPickupAt:   s.EstimatedPickupAt,
		EstimatedDeliveryAt: s.EstimatedDeliveryAt,
		ActualPickupAt:      s.ActualPickupAt,
		ActualDeliveryAt:    s.ActualDeliveryAt,
		DurationMinutes:     s.DurationMinutes,
		IsDelayed:           s.IsDelayed,
		HasRules:            s.Rules != nil,
		RulesConfirmed:      s.Rules != nil && s.Rules.ConfirmedByShipperID != nil,
		AlertsCount:         s.AlertsCount,
		CustomerNotes:       s.CustomerNotes,
		CompletionNotes:     s.CompletionNotes,
		CustomerRating:      s.CustomerRating,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}

	// Calculate duration if both pickup and delivery are set
	if s.ActualPickupAt != nil && s.ActualDeliveryAt != nil {
		duration := int(s.ActualDeliveryAt.Sub(*s.ActualPickupAt).Minutes())
		resp.DurationMinutes = &duration
	}

	// Check if delayed
	if s.EstimatedDeliveryAt != nil {
		if s.ActualDeliveryAt != nil {
			resp.IsDelayed = s.ActualDeliveryAt.After(*s.EstimatedDeliveryAt)
		} else if s.Status == StatusInTransit {
			resp.IsDelayed = time.Now().After(*s.EstimatedDeliveryAt)
		}
	}

	return resp
}
