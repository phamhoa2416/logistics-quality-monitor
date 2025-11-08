CREATE TYPE shipment_status AS ENUM (
    'demand_created',
    'order_posted',
    'shipping_assigned',
    'in_transit',
    'completed',
    'issue_reported',
    'cancelled'
    );

CREATE TABLE shipments
(
    id                    UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    customer_id           UUID            NOT NULL REFERENCES users (id),
    provider_id           UUID            NOT NULL REFERENCES users (id),
    shipper_id            UUID REFERENCES users (id),
    linked_device_id      UUID REFERENCES devices (id),

    status                shipment_status NOT NULL DEFAULT 'demand_created',
    goods_description     TEXT            NOT NULL,
    goods_value           DECIMAL(12, 2),
    goods_weight          DECIMAL(8, 2),
    pickup_address        TEXT            NOT NULL,
    delivery_address      TEXT            NOT NULL,
    estimated_pickup_at   TIMESTAMPTZ,
    estimated_delivery_at TIMESTAMPTZ,
    actual_pickup_at      TIMESTAMPTZ,
    actual_delivery_at    TIMESTAMPTZ,
    customer_notes        TEXT,
    completion_notes      TEXT,
    customer_rating       INTEGER CHECK (customer_rating >= 1 AND customer_rating <= 5),
    created_at            TIMESTAMPTZ              DEFAULT now(),
    updated_at            TIMESTAMPTZ              DEFAULT now(),

    CONSTRAINT check_different_parties CHECK (
        customer_id != provider_id AND
        (shipper_id IS NULL OR (shipper_id != customer_id AND shipper_id != provider_id))
        ),
    CONSTRAINT check_delivery_after_pickup CHECK (
        estimated_delivery_at IS NULL OR
        estimated_pickup_at IS NULL OR
        estimated_delivery_at > estimated_pickup_at
        )
);

CREATE INDEX idx_shipments_customer ON shipments (customer_id);
CREATE INDEX idx_shipments_provider ON shipments (provider_id);
CREATE INDEX idx_shipments_shipper ON shipments (shipper_id);
CREATE INDEX idx_shipments_status ON shipments (status);
CREATE INDEX idx_shipments_created_at ON shipments (created_at DESC);

CREATE TRIGGER update_shipments_updated_at
    BEFORE UPDATE
    ON shipments
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();