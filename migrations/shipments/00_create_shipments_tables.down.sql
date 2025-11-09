-- Drop foreign key constraint from devices table
ALTER TABLE devices DROP CONSTRAINT IF EXISTS fk_devices_current_shipment;

-- Drop trigger
DROP TRIGGER IF EXISTS update_shipments_updated_at ON shipments;

-- Drop indexes
DROP INDEX IF EXISTS idx_shipments_created_at;
DROP INDEX IF EXISTS idx_shipments_status;
DROP INDEX IF EXISTS idx_shipments_shipper;
DROP INDEX IF EXISTS idx_shipments_provider;
DROP INDEX IF EXISTS idx_shipments_customer;

-- Drop table
DROP TABLE IF EXISTS shipments;

-- Drop type
DROP TYPE IF EXISTS shipment_status;

