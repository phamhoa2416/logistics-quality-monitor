package ingestion

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

// SensorDataMessage represents incoming sensor data from IoT device
type SensorDataMessage struct {
	DeviceID       string    `json:"device_id"`
	Timestamp      time.Time `json:"timestamp"`
	Temperature    *float64  `json:"temperature"`
	Humidity       *float64  `json:"humidity"`
	LightLevel     *float64  `json:"light_level"`
	TiltAngle      *float64  `json:"tilt_angle"`
	ImpactG        *float64  `json:"impact_g"`
	BatteryLevel   *int      `json:"battery_level"`
	SignalStrength *int      `json:"signal_strength"`
}

// LocationDataMessage represents GPS location data
type LocationDataMessage struct {
	DeviceID  string    `json:"device_id"`
	Timestamp time.Time `json:"timestamp"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  *float64  `json:"altitude"`
	Speed     *float64  `json:"speed"`
	Accuracy  *float64  `json:"accuracy"`
}

// StatusMessage represents device status updates
type StatusMessage struct {
	DeviceID        string    `json:"device_id"`
	Timestamp       time.Time `json:"timestamp"`
	Status          string    `json:"status"`
	BatteryLevel    int       `json:"battery_level"`
	FirmwareVersion string    `json:"firmware_version"`
}

// HeartbeatMessage represents device heartbeat
type HeartbeatMessage struct {
	DeviceID  string    `json:"device_id"`
	Timestamp time.Time `json:"timestamp"`
	Online    bool      `json:"online"`
}

type SensorDataRecord struct {
	Time           time.Time
	DeviceID       uuid.UUID
	Latitude       *float64
	Longitude      *float64
	Altitude       *float64
	Speed          *float64
	Temperature    *float64
	Humidity       *float64
	LightLevel     *float64
	TiltAngle      *float64
	ImpactG        *float64
	BatteryLevel   *int
	SignalStrength *int
}

// DetailedMessage combines sensor and location data
type DetailedMessage struct {
	DeviceID  string
	Timestamp time.Time
	// Location
	Latitude  *float64
	Longitude *float64
	Altitude  *float64
	Speed     *float64
	// Sensors
	Temperature    *float64
	Humidity       *float64
	LightLevel     *float64
	TiltAngle      *float64
	ImpactG        *float64
	BatteryLevel   *int
	SignalStrength *int
}

// ParseSensorData parses JSON payload to SensorDataMessage
func ParseSensorData(payload []byte) (*SensorDataMessage, error) {
	var msg SensorDataMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, err
	}
	// Set timestamp if not provided
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	return &msg, nil
}

// ParseLocationData parses JSON payload to LocationDataMessage
func ParseLocationData(payload []byte) (*LocationDataMessage, error) {
	var msg LocationDataMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, err
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	return &msg, nil
}
