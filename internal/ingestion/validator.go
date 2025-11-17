package ingestion

import (
	"fmt"
	"github.com/google/uuid"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error [%s]: %s", e.Field, e.Message)
}

// ValidateSensorData validates sensor data message
func ValidateSensorData(msg *SensorDataMessage) error {
	// Validate device ID
	if msg.DeviceID == "" {
		return &ValidationError{Field: "device_id", Message: "device_id is required"}
	}
	if _, err := uuid.Parse(msg.DeviceID); err != nil {
		return &ValidationError{Field: "device_id", Message: "device_id must be valid UUID"}
	}

	// Validate timestamp
	if msg.Timestamp.IsZero() {
		return &ValidationError{Field: "timestamp", Message: "timestamp is required"}
	}

	// Validate temperature range
	if msg.Temperature != nil {
		if *msg.Temperature < -100 || *msg.Temperature > 100 {
			return &ValidationError{Field: "temperature", Message: "temperature must be between -100 and 100"}
		}
	}

	// Validate humidity range
	if msg.Humidity != nil {
		if *msg.Humidity < 0 || *msg.Humidity > 100 {
			return &ValidationError{Field: "humidity", Message: "humidity must be between 0 and 100"}
		}
	}

	// Validate light level
	if msg.LightLevel != nil {
		if *msg.LightLevel < 0 {
			return &ValidationError{Field: "light_level", Message: "light_level must be non-negative"}
		}
	}

	// Validate tilt angle
	if msg.TiltAngle != nil {
		if *msg.TiltAngle < 0 || *msg.TiltAngle > 180 {
			return &ValidationError{Field: "tilt_angle", Message: "tilt_angle must be between 0 and 180"}
		}
	}

	// Validate impact
	if msg.ImpactG != nil {
		if *msg.ImpactG < 0 || *msg.ImpactG > 50 {
			return &ValidationError{Field: "impact_g", Message: "impact_g must be between 0 and 50"}
		}
	}

	// Validate battery level
	if msg.BatteryLevel != nil {
		if *msg.BatteryLevel < 0 || *msg.BatteryLevel > 100 {
			return &ValidationError{Field: "battery_level", Message: "battery_level must be between 0 and 100"}
		}
	}

	// Validate signal strength (dBm)
	if msg.SignalStrength != nil {
		if *msg.SignalStrength < -120 || *msg.SignalStrength > 0 {
			return &ValidationError{Field: "signal_strength", Message: "signal_strength must be between -120 and 0"}
		}
	}

	return nil
}

// ValidateLocationData validates location data message
func ValidateLocationData(msg *LocationDataMessage) error {
	// Validate device ID
	if msg.DeviceID == "" {
		return &ValidationError{Field: "device_id", Message: "device_id is required"}
	}
	if _, err := uuid.Parse(msg.DeviceID); err != nil {
		return &ValidationError{Field: "device_id", Message: "device_id must be valid UUID"}
	}

	// Validate timestamp
	if msg.Timestamp.IsZero() {
		return &ValidationError{Field: "timestamp", Message: "timestamp is required"}
	}

	// Validate latitude
	if msg.Latitude < -90 || msg.Latitude > 90 {
		return &ValidationError{Field: "latitude", Message: "latitude must be between -90 and 90"}
	}

	// Validate longitude
	if msg.Longitude < -180 || msg.Longitude > 180 {
		return &ValidationError{Field: "longitude", Message: "longitude must be between -180 and 180"}
	}

	// Validate speed
	if msg.Speed != nil {
		if *msg.Speed < 0 {
			return &ValidationError{Field: "speed", Message: "speed must be non-negative"}
		}
	}

	return nil
}
