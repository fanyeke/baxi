-- +goose Up
-- +goose StatementBegin

-- Migration 017: Ontology tables in gov schema
-- Stores object type registry, properties, and relationships
-- for the AIP semantic object layer.

-- 1. Object type registry - central catalog of semantic objects
CREATE TABLE IF NOT EXISTS gov.object_type_registry (
    object_type_id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    source_tables TEXT[] NOT NULL DEFAULT '{}',
    grain TEXT NOT NULL,
    owner_role TEXT,
    sensitivity TEXT NOT NULL DEFAULT 'L0',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Object properties - typed fields for each object type
CREATE TABLE IF NOT EXISTS gov.object_property (
    property_id BIGSERIAL PRIMARY KEY,
    object_type_id TEXT NOT NULL REFERENCES gov.object_type_registry(object_type_id) ON DELETE CASCADE,
    property_name TEXT NOT NULL,
    property_type TEXT NOT NULL,
    is_pk BOOLEAN NOT NULL DEFAULT FALSE,
    source_column TEXT,
    aggregation TEXT,
    sensitivity TEXT NOT NULL DEFAULT 'L0',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_object_property_name UNIQUE (object_type_id, property_name)
);

-- 3. Object relationships - links between object types
CREATE TABLE IF NOT EXISTS gov.object_relationship (
    relationship_id BIGSERIAL PRIMARY KEY,
    source_object_type TEXT NOT NULL REFERENCES gov.object_type_registry(object_type_id) ON DELETE CASCADE,
    target_object_type TEXT NOT NULL REFERENCES gov.object_type_registry(object_type_id) ON DELETE CASCADE,
    relationship_name TEXT NOT NULL,
    join_key TEXT NOT NULL,
    cardinality TEXT NOT NULL DEFAULT 'many_to_one',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_object_relationship_name UNIQUE (source_object_type, target_object_type, relationship_name)
);

-- Indexes for object_type_registry
CREATE INDEX IF NOT EXISTS idx_gov_otr_is_active ON gov.object_type_registry(is_active);
CREATE INDEX IF NOT EXISTS idx_gov_otr_sensitivity ON gov.object_type_registry(sensitivity);

-- Indexes for object_property
CREATE INDEX IF NOT EXISTS idx_gov_op_object_type ON gov.object_property(object_type_id);
CREATE INDEX IF NOT EXISTS idx_gov_op_is_pk ON gov.object_property(is_pk) WHERE is_pk = TRUE;
CREATE INDEX IF NOT EXISTS idx_gov_op_type ON gov.object_property(property_type);

-- Indexes for object_relationship
CREATE INDEX IF NOT EXISTS idx_gov_orel_source ON gov.object_relationship(source_object_type);
CREATE INDEX IF NOT EXISTS idx_gov_orel_target ON gov.object_relationship(target_object_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse order: drop in dependency order (children before parents)
DROP TABLE IF EXISTS gov.object_relationship;
DROP TABLE IF EXISTS gov.object_property;
DROP TABLE IF EXISTS gov.object_type_registry;

-- +goose StatementEnd
