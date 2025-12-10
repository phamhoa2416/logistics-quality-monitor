package shipment

import (
	"time"

	domainShipment "cargo-tracker/internal/domain/shipment"

	"github.com/google/uuid"
)

// Request DTOs
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
	ProofOfDelivery  *string    `json:"proof_of_delivery" validate:"omitempty"`
}

type RateDeliveryRequest struct {
	Rating   int     `json:"rating" validate:"required,min=1,max=5"`
	Feedback *string `json:"feedback" validate:"omitempty,max=1000"`
}

type ReportIssueRequest struct {
	IssueType   string  `json:"issue_type" validate:"required,oneof=quality_violation accident theft delay other"`
	Description string  `json:"description" validate:"required,min=10,max=1000"`
	Severity    string  `json:"severity" validate:"required,oneof=low medium high critical"`
	Evidence    *string `json:"evidence" validate:"omitempty"`
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
	Status     *domainShipment.ShipmentStatus `form:"status"`
	CustomerID *uuid.UUID                     `form:"customer_id"`
	ProviderID *uuid.UUID                     `form:"provider_id"`
	ShipperID  *uuid.UUID                     `form:"shipper_id"`
	DeviceID   *uuid.UUID                     `form:"device_id"`

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

// Response DTOs
type ShipmentResponse struct {
	ID     uuid.UUID                     `json:"id"`
	Status domainShipment.ShipmentStatus `json:"status"`

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
	Rules         *ShippingRulesResponse `json:"rules,omitempty"`
	StatusHistory []StatusHistory        `json:"status_history"`
	RecentAlerts  []AlertSummary         `json:"recent_alerts"`
}

type StatusHistory struct {
	FromStatus *domainShipment.ShipmentStatus `json:"from_status"`
	ToStatus   domainShipment.ShipmentStatus  `json:"to_status"`
	ChangedBy  *uuid.UUID                     `json:"changed_by"`
	ChangedAt  time.Time                      `json:"changed_at"`
	Notes      *string                        `json:"notes"`
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
	Distance            *float64   `json:"distance,omitempty"`
}

type PartyInfo struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
	Phone    *string   `json:"phone"`
}

type DeviceInfo struct {
	ID           uuid.UUID `json:"id"`
	HardwareUID  string    `json:"hardware_uid"`
	DeviceName   *string   `json:"device_name"`
	BatteryLevel *int      `json:"battery_level"`
	IsOnline     bool      `json:"is_online"`
}

type ShippingRulesResponse struct {
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

type ShipmentStatisticsResponse struct {
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

// Conversion functions
func ToShipmentResponse(s *domainShipment.Shipment, rules *domainShipment.ShippingRules) *ShipmentResponse {
	if s == nil {
		return nil
	}

	resp := &ShipmentResponse{
		ID:                  s.ID,
		Status:              s.Status,
		GoodsDescription:    s.GoodsDescription,
		GoodsValue:          s.GoodsValue,
		GoodsWeight:         s.GoodsWeight,
		PickupAddress:       s.PickupAddress,
		DeliveryAddress:     s.DeliveryAddress,
		EstimatedPickupAt:   s.EstimatedPickupAt,
		EstimatedDeliveryAt: s.EstimatedDeliveryAt,
		ActualPickupAt:      s.ActualPickupAt,
		ActualDeliveryAt:    s.ActualDeliveryAt,
		CustomerNotes:       s.CustomerNotes,
		CompletionNotes:     s.CompletionNotes,
		CustomerRating:      s.CustomerRating,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
		HasRules:            rules != nil,
		RulesConfirmed:      rules != nil && rules.ConfirmedByShipperID != nil,
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
		} else if s.Status == domainShipment.StatusInTransit {
			now := time.Now()
			resp.IsDelayed = now.After(*s.EstimatedDeliveryAt)
		}
	}

	return resp
}

func ToDomainFilter(req *ShipmentFilterRequest) *domainShipment.Filter {
	if req == nil {
		return &domainShipment.Filter{}
	}
	return &domainShipment.Filter{
		Status:         req.Status,
		CustomerID:     req.CustomerID,
		ProviderID:     req.ProviderID,
		ShipperID:      req.ShipperID,
		DeviceID:       req.DeviceID,
		CreatedAfter:   req.CreatedAfter,
		CreatedBefore:  req.CreatedBefore,
		DeliveryAfter:  req.DeliveryAfter,
		DeliveryBefore: req.DeliveryBefore,
		HasIssues:      req.HasIssues,
		IsDelayed:      req.IsDelayed,
		HasDevice:      req.HasDevice,
		Search:         req.Search,
		Page:           req.Page,
		PageSize:       req.PageSize,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
	}
}

func ToStatisticsResponse(s *domainShipment.Statistics) *ShipmentStatisticsResponse {
	if s == nil {
		return nil
	}
	topShippers := make([]TopShipperStats, len(s.TopShippers))
	for i, stat := range s.TopShippers {
		topShippers[i] = TopShipperStats{
			ShipperID:      stat.ShipperID,
			ShipperName:    stat.ShipperName,
			TotalShipments: stat.TotalShipments,
			CompletedRate:  stat.CompletedRate,
			AvgRating:      stat.AvgRating,
		}
	}
	return &ShipmentStatisticsResponse{
		TotalShipments:      s.TotalShipments,
		ByStatus:            s.ByStatus,
		ActiveShipments:     s.ActiveShipments,
		CompletedToday:      s.CompletedToday,
		AverageDeliveryTime: s.AverageDeliveryTime,
		OnTimeDeliveryRate:  s.OnTimeDeliveryRate,
		IssueRate:           s.IssueRate,
		TopShippers:         topShippers,
		RevenueToday:        s.RevenueToday,
	}
}
