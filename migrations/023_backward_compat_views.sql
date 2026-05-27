-- +goose Up
-- +goose StatementBegin

-- Migration 023: Backward-compatibility views
-- Merges normalized ontology/marking/lineage tables into unified views
-- so existing queries can read a single flattened result.

-- ============================================================
-- 1. v_object_types: object_type_registry + object_property
--    One row per (object_type, property) with type metadata.
-- ============================================================
CREATE OR REPLACE VIEW gov.v_object_types AS
SELECT
    r.object_type_id,
    r.display_name          AS type_display_name,
    r.source_tables,
    r.grain,
    r.owner_role,
    r.sensitivity           AS type_sensitivity,
    r.is_active             AS type_is_active,
    p.property_id,
    p.property_name,
    p.property_type,
    p.is_pk,
    p.source_column,
    p.aggregation,
    p.sensitivity           AS property_sensitivity,
    r.created_at            AS type_created_at,
    r.updated_at            AS type_updated_at
FROM gov.object_type_registry r
LEFT JOIN gov.object_property p
    ON p.object_type_id = r.object_type_id;

COMMENT ON VIEW gov.v_object_types IS
    'Unified view: object_type_registry merged with object_property (one row per property per type)';

-- ============================================================
-- 2. v_marking_assignments: marking_definition + marking_assignment
--    Shows each marking definition alongside its resource assignments.
-- ============================================================
CREATE OR REPLACE VIEW gov.v_marking_assignments AS
SELECT
    d.marking_id,
    d.display_name          AS marking_display_name,
    d.description           AS marking_description,
    d.mandatory_control,
    d.access_type,
    d.conjunctive,
    d.inheritance_rules,
    d.policy,
    d.expand_access_permission,
    d.is_active             AS marking_is_active,
    a.assignment_id,
    a.resource_type,
    a.resource_path,
    a.is_active             AS assignment_is_active,
    a.created_at            AS assignment_created_at
FROM gov.marking_definition d
LEFT JOIN gov.marking_assignment a
    ON a.marking_id = d.marking_id;

COMMENT ON VIEW gov.v_marking_assignments IS
    'Unified view: marking_definition merged with marking_assignment (one row per assignment per marking)';

-- ============================================================
-- 3. v_lineage_graph: lineage_node + lineage_edge
--    Shows each edge with source and target node details.
-- ============================================================
CREATE OR REPLACE VIEW gov.v_lineage_graph AS
SELECT
    e.edge_id,
    e.source_node_id,
    sn.node_type           AS source_node_type,
    sn.label               AS source_label,
    sn.status              AS source_status,
    sn.linked_to           AS source_linked_to,
    e.target_node_id,
    tn.node_type           AS target_node_type,
    tn.label               AS target_label,
    tn.status              AS target_status,
    tn.linked_to           AS target_linked_to,
    e.description          AS edge_description,
    e.transform_type,
    e.created_at           AS edge_created_at
FROM gov.lineage_edge e
JOIN gov.lineage_node sn ON sn.node_id = e.source_node_id
JOIN gov.lineage_node tn ON tn.node_id = e.target_node_id;

COMMENT ON VIEW gov.v_lineage_graph IS
    'Unified view: lineage_edge enriched with source and target node metadata';

-- ============================================================
-- 4. v_governance_summary: data_classification + marking_definition
--    Cross-references classification levels with available markings
--    to give a single governance-overview row.
-- ============================================================
CREATE OR REPLACE VIEW gov.v_governance_summary AS
SELECT
    c.classification_id,
    c.field_path,
    c.classification_level,
    c.sensitivity_score,
    c.description          AS classification_description,
    c.created_at           AS classification_created_at,
    m.marking_id,
    m.display_name         AS marking_display_name,
    m.description          AS marking_description,
    m.mandatory_control,
    m.access_type,
    m.policy
FROM gov.data_classification c
LEFT JOIN gov.marking_definition m
    ON m.marking_id = c.classification_level
    OR m.display_name = c.classification_level;

COMMENT ON VIEW gov.v_governance_summary IS
    'Unified view: data_classification merged with marking_definition for governance overview';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop views in reverse dependency order
DROP VIEW IF EXISTS gov.v_governance_summary;
DROP VIEW IF EXISTS gov.v_lineage_graph;
DROP VIEW IF EXISTS gov.v_marking_assignments;
DROP VIEW IF EXISTS gov.v_object_types;

-- +goose StatementEnd
