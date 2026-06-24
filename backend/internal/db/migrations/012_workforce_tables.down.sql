-- Migration 012: Down — revert workforce management tables
-- +migrate Down

DROP TABLE IF EXISTS user_certifications;
DROP TABLE IF EXISTS certifications;
DROP TABLE IF EXISTS user_skills;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS user_shift_assignments;
DROP TABLE IF EXISTS shift_configurations;
DROP TABLE IF EXISTS teams;
