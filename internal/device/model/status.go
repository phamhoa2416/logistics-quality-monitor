package model

type DeviceStatus string

const (
	StatusAvailable   DeviceStatus = "available"
	StatusInTransit   DeviceStatus = "in_transit"
	StatusMaintenance DeviceStatus = "maintenance"
	StatusRetired     DeviceStatus = "retired"
)
