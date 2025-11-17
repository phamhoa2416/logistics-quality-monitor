package shipment

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for shipment repository operations
type Repository interface {
	Create(ctx context.Context, shipment *Shipment) error
	GetByID(ctx context.Context, shipmentID uuid.UUID) (*Shipment, error)
	Update(ctx context.Context, shipment *Shipment) error
	Delete(ctx context.Context, shipmentID uuid.UUID) error
	UpdateStatus(ctx context.Context, shipmentID uuid.UUID, status ShipmentStatus) error
	List(ctx context.Context, filter *Filter) ([]*Shipment, int64, error)
	GetStatistics(ctx context.Context) (*Statistics, error)

	SetActualPickup(ctx context.Context, shipmentID uuid.UUID, pickupTime time.Time) error
	SetActualDelivery(ctx context.Context, shipmentID uuid.UUID, deliveryTime time.Time, notes *string) error
	SetCustomerRating(ctx context.Context, shipmentID uuid.UUID, rating int, feedback *string) error
	GetMarketplaceListings(ctx context.Context, page, pageSize int) ([]*Shipment, int64, error)
	AssignShipper(ctx context.Context, shipmentID, shipperID uuid.UUID) error
	AssignDevice(ctx context.Context, shipmentID, deviceID uuid.UUID) error

	CreateRules(ctx context.Context, rules *ShippingRules) error
	GetRulesByShipmentID(ctx context.Context, shipmentID uuid.UUID) (*ShippingRules, error)
	UpdateRules(ctx context.Context, rules *ShippingRules) error
	ConfirmRules(ctx context.Context, shipmentID, shipperID uuid.UUID) error
}

// Filter represents filtering options for listing shipments
type Filter struct {
	Status     *ShipmentStatus
	CustomerID *uuid.UUID
	ProviderID *uuid.UUID
	ShipperID  *uuid.UUID
	DeviceID   *uuid.UUID

	// Date range filters
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	DeliveryAfter  *time.Time
	DeliveryBefore *time.Time

	// Boolean filters
	HasIssues *bool
	IsDelayed *bool
	HasDevice *bool

	// Search
	Search string

	// Pagination
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
}
