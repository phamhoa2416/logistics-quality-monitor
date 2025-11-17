package ingestion

import (
	"context"
	"fmt"
	"log"
)

// AlertEngine checks sensor data against shipping rules and generates alerts
type AlertEngine struct {
	repo *Repository
}

func NewAlertEngine(repo *Repository) *AlertEngine {
	return &AlertEngine{repo: repo}
}

// CheckViolations checks sensor data against rules and generates alerts
func (e *AlertEngine) CheckViolations(ctx context.Context, data *SensorDataRecord) ([]*Alert, error) {
	// Get shipment info and rules
	shipmentInfo, err := e.repo.GetDeviceShipment(ctx, data.DeviceID)
	if err != nil {
		// No active shipment or rules - skip
		return nil, nil
	}

	if !shipmentInfo.HasRules {
		return nil, nil
	}

	alerts := []*Alert{}

	// Check temperature violations
	if data.Temperature != nil {
		if shipmentInfo.TempMin != nil && *data.Temperature < *shipmentInfo.TempMin {
			alerts = append(alerts, &Alert{
				Time:           data.Time,
				DeviceID:       data.DeviceID,
				ShipmentID:     shipmentInfo.ShipmentID,
				AlertType:      "immediate",
				Severity:       "high",
				ViolationType:  "temperature",
				TriggerValue:   fmt.Sprintf("%.2f°C", *data.Temperature),
				ThresholdValue: fmt.Sprintf("min: %.2f°C", *shipmentInfo.TempMin),
				Message:        fmt.Sprintf("Temperature %.2f°C is below minimum threshold %.2f°C", *data.Temperature, *shipmentInfo.TempMin),
			})
		}
		if shipmentInfo.TempMax != nil && *data.Temperature > *shipmentInfo.TempMax {
			alerts = append(alerts, &Alert{
				Time:           data.Time,
				DeviceID:       data.DeviceID,
				ShipmentID:     shipmentInfo.ShipmentID,
				AlertType:      "immediate",
				Severity:       "high",
				ViolationType:  "temperature",
				TriggerValue:   fmt.Sprintf("%.2f°C", *data.Temperature),
				ThresholdValue: fmt.Sprintf("max: %.2f°C", *shipmentInfo.TempMax),
				Message:        fmt.Sprintf("Temperature %.2f°C exceeds maximum threshold %.2f°C", *data.Temperature, *shipmentInfo.TempMax),
			})
		}
	}

	// Check humidity violations
	if data.Humidity != nil {
		if shipmentInfo.HumidityMin != nil && *data.Humidity < *shipmentInfo.HumidityMin {
			alerts = append(alerts, &Alert{
				Time:           data.Time,
				DeviceID:       data.DeviceID,
				ShipmentID:     shipmentInfo.ShipmentID,
				AlertType:      "immediate",
				Severity:       "medium",
				ViolationType:  "humidity",
				TriggerValue:   fmt.Sprintf("%.2f%%", *data.Humidity),
				ThresholdValue: fmt.Sprintf("min: %.2f%%", *shipmentInfo.HumidityMin),
				Message:        fmt.Sprintf("Humidity %.2f%% is below minimum threshold %.2f%%", *data.Humidity, *shipmentInfo.HumidityMin),
			})
		}
		if shipmentInfo.HumidityMax != nil && *data.Humidity > *shipmentInfo.HumidityMax {
			alerts = append(alerts, &Alert{
				Time:           data.Time,
				DeviceID:       data.DeviceID,
				ShipmentID:     shipmentInfo.ShipmentID,
				AlertType:      "immediate",
				Severity:       "medium",
				ViolationType:  "humidity",
				TriggerValue:   fmt.Sprintf("%.2f%%", *data.Humidity),
				ThresholdValue: fmt.Sprintf("max: %.2f%%", *shipmentInfo.HumidityMax),
				Message:        fmt.Sprintf("Humidity %.2f%% exceeds maximum threshold %.2f%%", *data.Humidity, *shipmentInfo.HumidityMax),
			})
		}
	}

	// Check light exposure
	if data.LightLevel != nil && shipmentInfo.LightMax != nil {
		if *data.LightLevel > *shipmentInfo.LightMax {
			alerts = append(alerts, &Alert{
				Time:           data.Time,
				DeviceID:       data.DeviceID,
				ShipmentID:     shipmentInfo.ShipmentID,
				AlertType:      "immediate",
				Severity:       "medium",
				ViolationType:  "light",
				TriggerValue:   fmt.Sprintf("%.2f lux", *data.LightLevel),
				ThresholdValue: fmt.Sprintf("max: %.2f lux", *shipmentInfo.LightMax),
				Message:        fmt.Sprintf("Light exposure %.2f lux exceeds threshold %.2f lux", *data.LightLevel, *shipmentInfo.LightMax),
			})
		}
	}

	// Check tilt angle
	if data.TiltAngle != nil && shipmentInfo.TiltMaxAngle != nil {
		if *data.TiltAngle > *shipmentInfo.TiltMaxAngle {
			alerts = append(alerts, &Alert{
				Time:           data.Time,
				DeviceID:       data.DeviceID,
				ShipmentID:     shipmentInfo.ShipmentID,
				AlertType:      "immediate",
				Severity:       "high",
				ViolationType:  "tilt",
				TriggerValue:   fmt.Sprintf("%.2f°", *data.TiltAngle),
				ThresholdValue: fmt.Sprintf("max: %.2f°", *shipmentInfo.TiltMaxAngle),
				Message:        fmt.Sprintf("Package tilted %.2f° exceeding threshold %.2f°", *data.TiltAngle, *shipmentInfo.TiltMaxAngle),
			})
		}
	}

	// Check impact (critical)
	if data.ImpactG != nil && shipmentInfo.ImpactThresholdG != nil {
		if *data.ImpactG > *shipmentInfo.ImpactThresholdG {
			alerts = append(alerts, &Alert{
				Time:           data.Time,
				DeviceID:       data.DeviceID,
				ShipmentID:     shipmentInfo.ShipmentID,
				AlertType:      "immediate",
				Severity:       "critical",
				ViolationType:  "impact",
				TriggerValue:   fmt.Sprintf("%.2fG", *data.ImpactG),
				ThresholdValue: fmt.Sprintf("max: %.2fG", *shipmentInfo.ImpactThresholdG),
				Message:        fmt.Sprintf("Impact detected: %.2fG exceeds threshold %.2fG - potential damage!", *data.ImpactG, *shipmentInfo.ImpactThresholdG),
			})
		}
	}

	// Check battery level (warning, not violation)
	if data.BatteryLevel != nil && *data.BatteryLevel < 20 {
		severity := "low"
		if *data.BatteryLevel < 10 {
			severity = "medium"
		}
		alerts = append(alerts, &Alert{
			Time:           data.Time,
			DeviceID:       data.DeviceID,
			ShipmentID:     shipmentInfo.ShipmentID,
			AlertType:      "immediate",
			Severity:       severity,
			ViolationType:  "battery",
			TriggerValue:   fmt.Sprintf("%d%%", *data.BatteryLevel),
			ThresholdValue: "20%",
			Message:        fmt.Sprintf("Low battery: %d%%", *data.BatteryLevel),
		})
	}

	return alerts, nil
}

// SaveAlerts saves alerts to database
func (e *AlertEngine) SaveAlerts(ctx context.Context, alerts []*Alert) error {
	for _, alert := range alerts {
		if err := e.repo.InsertAlert(ctx, alert); err != nil {
			log.Printf("Failed to save alert: %v", err)
			continue
		}
		log.Printf("🚨 ALERT [%s/%s]: %s", alert.Severity, alert.ViolationType, alert.Message)
	}
	return nil
}
