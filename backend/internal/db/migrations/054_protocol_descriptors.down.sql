-- +migrate Down
-- PROTO-03: Protocol Descriptors Registry (rollback)

DROP TRIGGER IF EXISTS trg_protocol_descriptors_updated ON protocol_descriptors;
DROP FUNCTION IF EXISTS trg_protocol_descriptors_updated();

DROP TABLE IF EXISTS protocol_descriptors;
