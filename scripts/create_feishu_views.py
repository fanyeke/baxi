#!/usr/bin/env python3
"""create_feishu_views.py - YAML-driven Feishu view management.
v0.3.1: Creates and validates 6 dimensional alert views.
Dry-run validates without API calls.
"""
import os
import sys
import yaml
import argparse

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config

# 6 views for v0.3.1 dimensional alerts
VIEWS = [
    {
        'view_name': 'high_impact_alerts',
        'table_id': 'alert_events',
        'filter': {'status': ['new', 'investigating']},
        'sort': [{'field': 'impact_score', 'order': 'desc'}],
    },
    {
        'view_name': 'seller_alerts',
        'table_id': 'alert_events',
        'filter': {'object_type': ['seller']},
        'sort': [{'field': 'impact_score', 'order': 'desc'}],
    },
    {
        'view_name': 'category_alerts',
        'table_id': 'alert_events',
        'filter': {'object_type': ['category']},
        'sort': [{'field': 'impact_score', 'order': 'desc'}],
    },
    {
        'view_name': 'region_alerts',
        'table_id': 'alert_events',
        'filter': {'object_type': ['region']},
        'sort': [{'field': 'impact_score', 'order': 'desc'}],
    },
    {
        'view_name': 'high_confidence_strategies',
        'table_id': 'recommendations',
        'filter': {'confidence': ['high', 'medium']},
        'sort': [
            {'field': 'risk_level', 'order': 'desc'},
            {'field': 'created_at', 'order': 'desc'},
        ],
    },
    {
        'view_name': 'object_tasks',
        'table_id': 'action_tasks',
        'filter': {'status': ['todo', 'in_progress']},
        'sort': [
            {'field': 'priority', 'order': 'desc'},
            {'field': 'deadline', 'order': 'asc'},
        ],
    },
]


def load_schema():
    """Load the Feishu base schema YAML."""
    with open(config.FEISHU_BASE_SCHEMA_FILE, 'r', encoding='utf-8') as f:
        return yaml.safe_load(f)


def build_field_lookup(schema):
    """Build table_id → set(field_ids) lookup from schema."""
    lookup = {}
    for table in schema['tables']:
        table_id = table['table_id']
        field_ids = {field['field_id'] for field in table['fields']}
        lookup[table_id] = field_ids
    return lookup


def validate_view(view, field_lookup):
    """Validate a single view's field references against the schema.
    Returns a list of error strings (empty = valid).
    """
    errors = []
    table_id = view['table_id']
    view_name = view['view_name']

    # Validate table exists
    if table_id not in field_lookup:
        errors.append(
            f"[{view_name}] table_id '{table_id}' not found in schema. "
            f"Available: {', '.join(sorted(field_lookup.keys()))}"
        )
        return errors

    valid_fields = field_lookup[table_id]

    # Validate filter fields
    for field_name in view.get('filter', {}).keys():
        if field_name not in valid_fields:
            errors.append(
                f"[{view_name}] filter field '{field_name}' not found "
                f"in table '{table_id}'"
            )

    # Validate sort fields
    for sort_entry in view.get('sort', []):
        sort_field = sort_entry.get('field', '')
        if sort_field not in valid_fields:
            errors.append(
                f"[{view_name}] sort field '{sort_field}' not found "
                f"in table '{table_id}'"
            )

    return errors


def print_view_spec(view):
    """Pretty-print a view specification."""
    table_id = view['table_id']
    view_name = view['view_name']
    filter_spec = view.get('filter', {})
    sort_spec = view.get('sort', [])

    print(f"\n{'=' * 60}")
    print(f"  View: {view_name}")
    print(f"  Table: {table_id}")
    print(f"{'=' * 60}")

    if filter_spec:
        print(f"\n  Filters:")
        for field, values in filter_spec.items():
            print(f"    - {field} IN ({', '.join(str(v) for v in values)})")

    if sort_spec:
        print(f"\n  Sorting:")
        for entry in sort_spec:
            print(f"    - {entry['field']} ({entry['order']})")

    print()


def main():
    parser = argparse.ArgumentParser(
        description='Create and validate Feishu Bitable views from YAML config'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        default=True,
        help='Validate views without API calls (default)',
    )
    parser.add_argument(
        '--apply',
        action='store_true',
        default=False,
        help='Execute real view creation via Feishu API',
    )

    args = parser.parse_args()
    dry_run = True
    if args.apply:
        dry_run = False
    elif args.dry_run:
        dry_run = True

    # Load schema
    schema = load_schema()
    field_lookup = build_field_lookup(schema)

    print(f"Feishu View Manager v0.3.1")
    print(f"Schema: {config.FEISHU_BASE_SCHEMA_FILE}")
    print(f"Mode: {'DRY RUN (validation only)' if dry_run else 'APPLY'}")
    print(f"Views to process: {len(VIEWS)}")

    # Validate all views
    all_errors = []
    for view in VIEWS:
        errors = validate_view(view, field_lookup)
        all_errors.extend(errors)

    # Report validation results
    print(f"\n{'─' * 60}")
    print(f"  Validation Results")
    print(f"{'─' * 60}")

    if all_errors:
        print(f"\n  ERRORS ({len(all_errors)}):")
        for error in all_errors:
            print(f"    ❌ {error}")
        print(f"\n  ❌ Validation FAILED with {len(all_errors)} error(s)")
        sys.exit(1)

    print(f"\n  ✅ All views validated successfully")

    # Print view specs
    for view in VIEWS:
        print_view_spec(view)

    if not dry_run:
        # --apply mode: stub for real API calls
        print(f"\n{'─' * 60}")
        print(f"  API Calls (stub)")
        print(f"{'─' * 60}")
        for view in VIEWS:
            print(f"  → API call would be made to create view: {view['view_name']} on {view['table_id']}")
        print(f"\n  Note: --apply mode is a stub. No actual API calls were made.")

    sys.exit(0)


if __name__ == '__main__':
    main()
