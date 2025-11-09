-- Drop triggers
DROP TRIGGER IF EXISTS trg_validate_owner_shipper ON devices;
DROP TRIGGER IF EXISTS update_devices_updated_at ON devices;

-- Drop indexes
DROP INDEX IF EXISTS idx_devices_status;
DROP INDEX IF EXISTS idx_devices_hardware_uid;
DROP INDEX IF EXISTS idx_devices_owner;

-- Drop table
DROP TABLE IF EXISTS devices;

-- Drop functions
DROP FUNCTION IF EXISTS validate_owner_is_shipper();

-- Drop type
DROP TYPE IF EXISTS device_status;

