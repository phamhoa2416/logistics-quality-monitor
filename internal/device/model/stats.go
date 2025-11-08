package model

type DeviceStatistics struct {
	TotalDevices       int          `json:"total_devices"`
	AvailableDevices   int          `json:"available_devices"`
	InTransitDevices   int          `json:"in_transit_devices"`
	MaintenanceDevices int          `json:"maintenance_devices"`
	RetiredDevices     int          `json:"retired_devices"`
	ByOwner            []OwnerStats `json:"by_owner"`
	LowBatteryDevices  int          `json:"low_battery_devices"`
	OfflineDevices     int          `json:"offline_devices"`
}

type OwnerStats struct {
	OwnerID     string `json:"owner_id"`
	OwnerName   string `json:"owner_name"`
	DeviceCount int    `json:"device_count"`
}
