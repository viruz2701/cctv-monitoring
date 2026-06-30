-- +migrate Down

DROP TABLE IF EXISTS custom_field_value_audit;
DROP TABLE IF EXISTS custom_field_values;
DROP TABLE IF EXISTS custom_field_definitions;
DROP TABLE IF EXISTS custom_field_groups;
