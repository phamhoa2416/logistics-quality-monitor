CREATE TYPE device_status AS ENUM (
    'available',
    'in_transit',
    'maintenance',
    'retired'
    );

CREATE TABLE devices
(
    id                 UUID PRIMARY KEY                  DEFAULT gen_random_uuid(),
    hardware_uid       VARCHAR(255) UNIQUE      NOT NULL,
    owner_shipper_id   UUID                     REFERENCES users (id) ON DELETE SET NULL,
    current_shipmen_id UUID                     REFERENCES shipments (id) ON DELETE SET NULL,
    status             device_status            NOT NULL DEFAULT 'available',
    device_name        VARCHAR(100),
    model              VARCHAR(50),
    firmware_version   VARCHAR(50),
    battery_level      INTEGER CHECK (battery_level >= 0 AND battery_level <= 100),
    total_trips        INTEGER                           DEFAULT 0,

    last_seen_at       TIMESTAMP WITH TIME ZONE,
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION validate_owner_is_shipper()
    RETURNS TRIGGER AS
$$
BEGIN
    IF NEW.owner_shipper_id IS NOT NULL THEN
        IF NOT EXISTS (SELECT 1
                       FROM users
                       WHERE id = NEW.owner_shipper_id
                         AND role = 'shipper') THEN
            RAISE EXCEPTION 'owner_shipper_id % is not a shipper', NEW.owner_shipper_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE INDEX idx_devices_owner ON devices (owner_shipper_id);
CREATE INDEX idx_devices_hardware_uid ON devices (hardware_uid);
CREATE INDEX idx_devices_status ON devices (status) WHERE status = 'available';

CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE
    ON devices
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();

CREATE TRIGGER trg_validate_owner_shipper
    BEFORE INSERT OR UPDATE
    ON devices
    FOR EACH ROW
EXECUTE PROCEDURE validate_owner_is_shipper();