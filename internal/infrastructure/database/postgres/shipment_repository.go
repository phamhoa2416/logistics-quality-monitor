package postgres

import (
	"cargo-tracker/internal/domain/shipment"
	"cargo-tracker/internal/infrastructure/database/postgres/models"
	appErrors "cargo-tracker/pkg/errors"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShipmentRepository struct {
	db *DB
}

func NewShipmentRepository(db *DB) *ShipmentRepository {
	return &ShipmentRepository{db: db}
}

func (r *ShipmentRepository) Create(ctx context.Context, s *shipment.Shipment) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	if s.Status == "" {
		s.Status = shipment.StatusDemandCreated
	}

	dbModel := toShipmentModel(s)
	if err := r.db.DB.WithContext(ctx).Create(dbModel).Error; err != nil {
		return fmt.Errorf("failed to create shipment: %w", err)
	}

	s.ID = dbModel.ID
	s.CreatedAt = dbModel.CreatedAt
	s.UpdatedAt = dbModel.UpdatedAt

	return nil
}

func (r *ShipmentRepository) GetByID(ctx context.Context, shipmentID uuid.UUID) (*shipment.Shipment, error) {
	var dbModel models.ShipmentModel
	err := r.db.DB.WithContext(ctx).
		Preload("Customer").
		Preload("Provider").
		Preload("Shipper").
		Preload("Device").
		Where("id = ?", shipmentID).
		First(&dbModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, shipment.ErrShipmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	return toShipmentEntity(&dbModel), nil
}

func (r *ShipmentRepository) Update(ctx context.Context, s *shipment.Shipment) error {
	s.UpdatedAt = time.Now()

	result := r.db.DB.WithContext(ctx).
		Model(&models.ShipmentModel{}).
		Where("id = ?", s.ID).
		Updates(map[string]interface{}{
			"shipper_id":            s.ShipperID,
			"linked_device_id":      s.LinkedDeviceID,
			"status":                string(s.Status),
			"goods_description":     s.GoodsDescription,
			"goods_value":           s.GoodsValue,
			"goods_weight":          s.GoodsWeight,
			"pickup_address":        s.PickupAddress,
			"delivery_address":      s.DeliveryAddress,
			"estimated_pickup_at":   s.EstimatedPickupAt,
			"estimated_delivery_at": s.EstimatedDeliveryAt,
			"actual_pickup_at":      s.ActualPickupAt,
			"actual_delivery_at":    s.ActualDeliveryAt,
			"customer_notes":        s.CustomerNotes,
			"completion_notes":      s.CompletionNotes,
			"customer_rating":       s.CustomerRating,
			"updated_at":            s.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update shipment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return shipment.ErrShipmentNotFound
	}

	return nil
}

func (r *ShipmentRepository) Delete(ctx context.Context, shipmentID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Where("id = ?", shipmentID).
		Delete(&models.ShipmentModel{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete shipment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return shipment.ErrShipmentNotFound
	}

	return nil
}

func (r *ShipmentRepository) UpdateStatus(ctx context.Context, shipmentID uuid.UUID, status shipment.ShipmentStatus) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.ShipmentModel{}).
		Where("id = ?", shipmentID).
		Updates(map[string]interface{}{
			"status":     string(status),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update shipment status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return shipment.ErrShipmentNotFound
	}

	return nil
}

func (r *ShipmentRepository) List(ctx context.Context, filter *shipment.Filter) ([]*shipment.Shipment, int64, error) {
	var dbModels []models.ShipmentModel
	var total int64

	db := r.db.DB.WithContext(ctx).Model(&models.ShipmentModel{}).
		Preload("Customer").
		Preload("Provider").
		Preload("Shipper").
		Preload("Device")

	// Apply filters
	if filter.Status != nil {
		db = db.Where("status = ?", string(*filter.Status))
	}
	if filter.CustomerID != nil {
		db = db.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.ProviderID != nil {
		db = db.Where("provider_id = ?", *filter.ProviderID)
	}
	if filter.ShipperID != nil {
		db = db.Where("shipper_id = ?", *filter.ShipperID)
	}
	if filter.DeviceID != nil {
		db = db.Where("linked_device_id = ?", *filter.DeviceID)
	}
	if filter.CreatedAfter != nil {
		db = db.Where("created_at >= ?", filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		db = db.Where("created_at <= ?", filter.CreatedBefore)
	}
	if filter.DeliveryAfter != nil {
		db = db.Where("estimated_delivery_at >= ?", filter.DeliveryAfter)
	}
	if filter.DeliveryBefore != nil {
		db = db.Where("estimated_delivery_at <= ?", filter.DeliveryBefore)
	}
	if filter.HasIssues != nil && *filter.HasIssues {
		db = db.Where("status = ?", string(shipment.StatusIssueReported))
	}
	if filter.IsDelayed != nil && *filter.IsDelayed {
		now := time.Now()
		db = db.Where("status = ? AND estimated_delivery_at < ?", string(shipment.StatusInTransit), now)
	}
	if filter.HasDevice != nil {
		if *filter.HasDevice {
			db = db.Where("linked_device_id IS NOT NULL")
		} else {
			db = db.Where("linked_device_id IS NULL")
		}
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		db = db.Where("goods_description ILIKE ? OR pickup_address ILIKE ? OR delivery_address ILIKE ?",
			search, search, search)
	}

	// Count total
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count shipments: %w", err)
	}

	// Apply sorting
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	// Apply pagination
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Execute query
	err := db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&dbModels).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shipments: %w", err)
	}

	// Convert to domain entities
	shipments := make([]*shipment.Shipment, len(dbModels))
	for i, dbModel := range dbModels {
		shipments[i] = toShipmentEntity(&dbModel)
	}

	return shipments, total, nil
}

func (r *ShipmentRepository) GetStatistics(ctx context.Context) (*shipment.Statistics, error) {
	stats := &shipment.Statistics{
		ByStatus: make(map[string]int),
	}

	// Get total and basic counts
	var totalShipments int64
	r.db.DB.WithContext(ctx).Model(&models.ShipmentModel{}).Count(&totalShipments)
	stats.TotalShipments = int(totalShipments)

	// Get total and by status
	var statusCounts []struct {
		Status string
		Count  int
	}
	err := r.db.DB.WithContext(ctx).Raw(`
		SELECT status, COUNT(*) as count
		FROM shipments
		GROUP BY status
	`).Scan(&statusCounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	for _, sc := range statusCounts {
		stats.TotalShipments += sc.Count
		stats.ByStatus[sc.Status] = sc.Count
	}

	// Get active shipments (in_transit, shipping_assigned)
	err = r.db.DB.WithContext(ctx).Raw(`
		SELECT COUNT(*) as count
		FROM shipments
		WHERE status IN ('in_transit', 'shipping_assigned')
	`).Scan(&stats.ActiveShipments).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active shipments: %w", err)
	}

	// Get completed today
	today := time.Now().Truncate(24 * time.Hour)
	err = r.db.DB.WithContext(ctx).Raw(`
		SELECT COUNT(*) as count
		FROM shipments
		WHERE status = 'completed' AND DATE(actual_delivery_at) = DATE(?)
	`, today).Scan(&stats.CompletedToday).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get completed today: %w", err)
	}

	// Get revenue today
	err = r.db.DB.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(goods_value), 0) as total
		FROM shipments
		WHERE status = 'completed' AND DATE(actual_delivery_at) = DATE(?)
	`, today).Scan(&stats.RevenueToday).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue today: %w", err)
	}

	// Calculate metrics
	if stats.TotalShipments > 0 {
		completedCount := stats.ByStatus["completed"]
		issueCount := stats.ByStatus["issue_reported"]

		// On-time delivery rate
		var onTimeCount int
		err = r.db.DB.WithContext(ctx).Raw(`
			SELECT COUNT(*) as count
			FROM shipments
			WHERE status = 'completed' AND actual_delivery_at <= estimated_delivery_at
		`).Scan(&onTimeCount).Error
		if err != nil {
			return nil, fmt.Errorf("failed to get on-time delivery count: %w", err)
		}

		if completedCount > 0 {
			stats.OnTimeDeliveryRate = float64(onTimeCount) / float64(completedCount) * 100
		}

		stats.IssueRate = float64(issueCount) / float64(stats.TotalShipments) * 100

		// Get average delivery time
		err = r.db.DB.WithContext(ctx).Raw(`
		SELECT AVG(EXTRACT(EPOCH FROM (actual_delivery_at - actual_pickup_at)) / 3600.0) as avg_hours
		FROM shipments
		WHERE status = 'completed' AND actual_pickup_at IS NOT NULL AND actual_delivery_at IS NOT NULL
		`).Scan(&stats.AverageDeliveryTime).Error
		if err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func (r *ShipmentRepository) SetActualPickup(ctx context.Context, shipmentID uuid.UUID, pickupTime time.Time) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.ShipmentModel{}).
		Where("id = ?", shipmentID).
		Updates(map[string]interface{}{
			"actual_pickup_at": pickupTime,
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to set actual pickup time: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return shipment.ErrShipmentNotFound
	}

	return nil
}

func (r *ShipmentRepository) SetActualDelivery(ctx context.Context, shipmentID uuid.UUID, deliveryTime time.Time, notes *string) error {
	updates := map[string]interface{}{
		"actual_delivery_at": deliveryTime,
		"updated_at":         time.Now(),
	}

	if notes != nil {
		updates["completion_notes"] = *notes
	}

	result := r.db.DB.WithContext(ctx).
		Model(&models.ShipmentModel{}).
		Where("id = ?", shipmentID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to set actual delivery time: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return shipment.ErrShipmentNotFound
	}

	return nil
}

func (r *ShipmentRepository) SetCustomerRating(ctx context.Context, shipmentID uuid.UUID, rating int, feedback *string) error {
	updates := map[string]interface{}{
		"customer_rating": rating,
		"updated_at":      time.Now(),
	}

	if feedback != nil {
		result := r.db.DB.WithContext(ctx).
			Model(&models.ShipmentModel{}).
			Where("id = ? AND status = ?", shipmentID, "completed").
			Update("completion_notes", gorm.Expr("COALESCE(completion_notes, '') || ?", "\nCustomer Feedback: "+*feedback)).
			Update("customer_rating", rating).
			Update("updated_at", time.Now())

		if result.Error != nil {
			return fmt.Errorf("failed to set customer rating and feedback: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return appErrors.NewAppError("RATING_FAILED", "Shipment not completed or not found", nil)
		}

		return nil
	}

	result := r.db.DB.WithContext(ctx).
		Model(&models.ShipmentModel{}).
		Where("id = ? AND status = ?", shipmentID, "completed").
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to set customer rating: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.NewAppError("RATING_FAILED", "Shipment not completed or not found", nil)
	}

	return nil
}

func (r *ShipmentRepository) GetMarketplaceListings(ctx context.Context, page, pageSize int) ([]*shipment.Shipment, int64, error) {
	status := shipment.StatusOrderPosted
	filter := &shipment.Filter{
		Status:    &status,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	return r.List(ctx, filter)
}

func (r *ShipmentRepository) AssignShipper(ctx context.Context, shipmentID, shipperID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.ShipmentModel{}).
		Where("id = ? AND shipper_id IS NULL", shipmentID).
		Updates(map[string]interface{}{
			"shipper_id": shipperID,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to assign shipper: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return appErrors.NewAppError("ASSIGNMENT_FAILED", "Shipment already has a shipper or not found", nil)
	}

	return nil
}

func (r *ShipmentRepository) AssignDevice(ctx context.Context, shipmentID, deviceID uuid.UUID) error {
	return r.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ShipmentModel{}).
			Where("id = ? AND linked_device_id IS NULL", shipmentID).
			Updates(map[string]interface{}{
				"linked_device_id": deviceID,
				"updated_at":       time.Now(),
			})

		if result.Error != nil {
			return fmt.Errorf("failed to assign device to shipment: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return appErrors.NewAppError("ASSIGNMENT_FAILED", "Shipment already has a device or not found", nil)
		}

		if err := tx.Model(&models.DeviceModel{}).
			Where("id = ? AND current_shipment_id IS NULL", deviceID).
			Updates(map[string]interface{}{
				"current_shipment_id": shipmentID,
				"status":              "in_transit",
				"updated_at":          time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to update device: %w", err)
		}

		return nil
	})
}

func (r *ShipmentRepository) CreateRules(ctx context.Context, rules *shipment.ShippingRules) error {
	rules.ID = uuid.New()
	rules.SetAt = time.Now()

	dbModel := toShippingRulesModel(rules)
	if err := r.db.DB.WithContext(ctx).Create(dbModel).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("rules already exist for this shipment")
		}
		return fmt.Errorf("failed to create shipping rules: %w", err)
	}

	rules.ID = dbModel.ID
	rules.SetAt = dbModel.SetAt

	return nil
}

func (r *ShipmentRepository) ConfirmRules(ctx context.Context, shipmentID, shipperID uuid.UUID) error {
	now := time.Now()
	result := r.db.DB.WithContext(ctx).
		Model(&models.ShippingRulesModel{}).
		Where("shipment_id = ? AND confirmed_by_shipper_id IS NULL", shipmentID).
		Updates(map[string]interface{}{
			"confirmed_by_shipper_id": shipperID,
			"confirmed_at":            now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to confirm shipping rules: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("shipping rules not found")
	}

	return nil
}

func (r *ShipmentRepository) UpdateRules(ctx context.Context, rules *shipment.ShippingRules) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.ShippingRulesModel{}).
		Where("id = ?", rules.ID).
		Updates(map[string]interface{}{
			"report_cycle_sec":        rules.ReportCycleSec,
			"temp_min":                rules.TempMin,
			"temp_max":                rules.TempMax,
			"humidity_min":            rules.HumidityMin,
			"humidity_max":            rules.HumidityMax,
			"light_max":               rules.LightMax,
			"tilt_max_angle":          rules.TiltMaxAngle,
			"impact_threshold_g":      rules.ImpactThresholdG,
			"enable_predictive_alert": rules.EnablePredictiveAlert,
			"alert_buffer_time_min":   rules.AlertBufferTimeMin,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update shipping rules: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("shipping rules not found")
	}

	return nil
}

func (r *ShipmentRepository) GetRulesByShipmentID(ctx context.Context, shipmentID uuid.UUID) (*shipment.ShippingRules, error) {
	var dbModel models.ShippingRulesModel
	err := r.db.DB.WithContext(ctx).
		Where("shipment_id = ?", shipmentID).
		First(&dbModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // Rules are optional
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get shipping rules: %w", err)
	}

	return toShippingRulesEntity(&dbModel), nil
}

// Helper functions to convert between domain entities and database models
func toShipmentModel(s *shipment.Shipment) *models.ShipmentModel {
	return &models.ShipmentModel{
		ID:                  s.ID,
		CustomerID:          s.CustomerID,
		ProviderID:          s.ProviderID,
		ShipperID:           s.ShipperID,
		LinkedDeviceID:      s.LinkedDeviceID,
		Status:              string(s.Status),
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
	}
}

func toShipmentEntity(m *models.ShipmentModel) *shipment.Shipment {
	status := shipment.ShipmentStatus(m.Status)
	return &shipment.Shipment{
		ID:                  m.ID,
		CustomerID:          m.CustomerID,
		ProviderID:          m.ProviderID,
		ShipperID:           m.ShipperID,
		LinkedDeviceID:      m.LinkedDeviceID,
		Status:              status,
		GoodsDescription:    m.GoodsDescription,
		GoodsValue:          m.GoodsValue,
		GoodsWeight:         m.GoodsWeight,
		PickupAddress:       m.PickupAddress,
		DeliveryAddress:     m.DeliveryAddress,
		EstimatedPickupAt:   m.EstimatedPickupAt,
		EstimatedDeliveryAt: m.EstimatedDeliveryAt,
		ActualPickupAt:      m.ActualPickupAt,
		ActualDeliveryAt:    m.ActualDeliveryAt,
		CustomerNotes:       m.CustomerNotes,
		CompletionNotes:     m.CompletionNotes,
		CustomerRating:      m.CustomerRating,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

func toShippingRulesModel(r *shipment.ShippingRules) *models.ShippingRulesModel {
	return &models.ShippingRulesModel{
		ID:                    r.ID,
		ShipmentID:            r.ShipmentID,
		ReportCycleSec:        r.ReportCycleSec,
		TempMin:               r.TempMin,
		TempMax:               r.TempMax,
		HumidityMin:           r.HumidityMin,
		HumidityMax:           r.HumidityMax,
		LightMax:              r.LightMax,
		TiltMaxAngle:          r.TiltMaxAngle,
		ImpactThresholdG:      r.ImpactThresholdG,
		EnablePredictiveAlert: r.EnablePredictiveAlert,
		AlertBufferTimeMin:    r.AlertBufferTimeMin,
		SetByProviderID:       r.SetByProviderID,
		ConfirmedByShipperID:  r.ConfirmedByShipperID,
		SetAt:                 r.SetAt,
		ConfirmedAt:           r.ConfirmedAt,
	}
}

func toShippingRulesEntity(m *models.ShippingRulesModel) *shipment.ShippingRules {
	return &shipment.ShippingRules{
		ID:                    m.ID,
		ShipmentID:            m.ShipmentID,
		ReportCycleSec:        m.ReportCycleSec,
		TempMin:               m.TempMin,
		TempMax:               m.TempMax,
		HumidityMin:           m.HumidityMin,
		HumidityMax:           m.HumidityMax,
		LightMax:              m.LightMax,
		TiltMaxAngle:          m.TiltMaxAngle,
		ImpactThresholdG:      m.ImpactThresholdG,
		EnablePredictiveAlert: m.EnablePredictiveAlert,
		AlertBufferTimeMin:    m.AlertBufferTimeMin,
		SetByProviderID:       m.SetByProviderID,
		ConfirmedByShipperID:  m.ConfirmedByShipperID,
		SetAt:                 m.SetAt,
		ConfirmedAt:           m.ConfirmedAt,
	}
}
