package models

import (
	"fmt"
	appErrors "logistics-quality-monitor/pkg/errors"
	"time"

	"gorm.io/gorm"
)

// GetStatistics retrieves shipment analytics
func (r *Repository) GetStatistics(ctx context.Context) (*ShipmentStatistics, error) {

	// Active shipments
	var activeCount int64
	r.db.DB.WithContext(ctx).
		Model(&Shipment{}).
		Where("status IN ?", []string{"shipping_assigned", "in_transit", "issue_reported"}).
		Count(&activeCount)
	stats.ActiveShipments = int(activeCount)

	// Completed today
	var completedToday int64
	r.db.DB.WithContext(ctx).
		Model(&Shipment{}).
		Where("status = ? AND DATE(actual_delivery_at) = CURRENT_DATE", "completed").
		Count(&completedToday)
	stats.CompletedToday = int(completedToday)

	// Revenue today
	type RevenueResult struct {
		Total float64
	}
	var revenue RevenueResult
	r.db.DB.WithContext(ctx).
		Model(&Shipment{}).
		Select("COALESCE(SUM(goods_value), 0) as total").
		Where("status = ? AND DATE(actual_delivery_at) = CURRENT_DATE", "completed").
		Scan(&revenue)
	stats.RevenueToday = revenue.Total

	// Counts by status
	type StatusCount struct {
		Status string
		Count  int
	}
	var statusCounts []StatusCount
	r.db.DB.WithContext(ctx).
		Model(&Shipment{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts)

	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	// Calculate metrics
	if stats.TotalShipments > 0 {
		completedCount := stats.ByStatus["completed"]
		issueCount := stats.ByStatus["issue_reported"]

		// On-time delivery rate
		var onTimeCount int64
		r.db.DB.WithContext(ctx).
			Model(&Shipment{}).
			Where("status = ? AND actual_delivery_at <= estimated_delivery_at", "completed").
			Count(&onTimeCount)

		if completedCount > 0 {
			stats.OnTimeDeliveryRate = float64(onTimeCount) / float64(completedCount) * 100
		}

		// Issue rate
		stats.IssueRate = float64(issueCount) / float64(stats.TotalShipments) * 100

		// Average delivery time in hours
		type AvgTime struct {
			AvgHours float64
		}
		var avgTime AvgTime
		r.db.DB.WithContext(ctx).
			Model(&Shipment{}).
			Select("AVG(EXTRACT(EPOCH FROM (actual_delivery_at - actual_pickup_at))/3600) as avg_hours").
			Where("status = ? AND actual_pickup_at IS NOT NULL AND actual_delivery_at IS NOT NULL", "completed").
			Scan(&avgTime)
		stats.AverageDeliveryTime = avgTime.AvgHours
	}

	// Top shippers
	var topShippers []TopShipperStats
	r.db.DB.WithContext(ctx).
		Model(&User{}).
		Select(`users.id as shipper_id, 
				users.full_name as shipper_name,
				COUNT(shipments.id) as total_shipments,
				COUNT(*) FILTER (WHERE shipments.status = 'completed') * 100.0 / NULLIF(COUNT(shipments.id), 0) as completed_rate,
				AVG(shipments.customer_rating) as avg_rating`).
		Joins("INNER JOIN shipments ON users.id = shipments.shipper_id").
		Where("users.role = ?", "shipper").
		Group("users.id, users.full_name").
		Order("total_shipments DESC").
		Limit(10).
		Scan(&topShippers)

	stats.TopShippers = topShippers

	return stats, nil
}
