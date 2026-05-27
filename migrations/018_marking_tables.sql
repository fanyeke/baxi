-- +goose Up
-- +goose StatementBegin

-- v0.6.0 Migration 018: Marking definition, assignment, and pipeline stage marking tables

-- 1. Marking definitions (from data_markings.yml)
CREATE TABLE IF NOT EXISTS gov.marking_definition (
    marking_id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    mandatory_control BOOLEAN NOT NULL DEFAULT true,
    access_type TEXT NOT NULL DEFAULT 'binary',
    conjunctive BOOLEAN NOT NULL DEFAULT true,
    inheritance_rules TEXT[] NOT NULL DEFAULT '{}',
    policy TEXT,
    expand_access_permission TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE gov.marking_definition IS 'Data marking definitions governing access control per resource';
COMMENT ON COLUMN gov.marking_definition.mandatory_control IS 'If true, marking is mandatory (binary all-or-nothing)';
COMMENT ON COLUMN gov.marking_definition.conjunctive IS 'If true, user must satisfy ALL markings on a resource (AND logic)';
COMMENT ON COLUMN gov.marking_definition.inheritance_rules IS 'List of inheritance modes: file_hierarchy, data_dependency';

-- 2. Marking assignments (maps markings to resources)
CREATE TABLE IF NOT EXISTS gov.marking_assignment (
    assignment_id BIGSERIAL PRIMARY KEY,
    marking_id TEXT NOT NULL REFERENCES gov.marking_definition(marking_id),
    resource_type TEXT NOT NULL,
    resource_path TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_marking_assignment_resource UNIQUE (marking_id, resource_type, resource_path)
);

-- 3. Pipeline stage markings (marks pipeline stages with their primary marking)
CREATE TABLE IF NOT EXISTS gov.pipeline_stage_marking (
    stage_name TEXT PRIMARY KEY,
    marking_id TEXT NOT NULL REFERENCES gov.marking_definition(marking_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for marking_assignment
CREATE INDEX IF NOT EXISTS idx_gov_marking_assignment_marking_id ON gov.marking_assignment(marking_id);
CREATE INDEX IF NOT EXISTS idx_gov_marking_assignment_resource_type ON gov.marking_assignment(resource_type);

-- Indexes for pipeline_stage_marking
CREATE INDEX IF NOT EXISTS idx_gov_pipeline_stage_marking_marking_id ON gov.pipeline_stage_marking(marking_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS gov.pipeline_stage_marking;
DROP TABLE IF EXISTS gov.marking_assignment;
DROP TABLE IF EXISTS gov.marking_definition;

-- +goose StatementEnd
