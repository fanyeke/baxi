"""Tests for scripts/create_feishu_views.py — validation logic and dry-run behavior."""
import subprocess
import sys
import os
import importlib
import pytest


SCRIPT_PATH = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    'scripts', 'create_feishu_views.py',
)


def _run_script(*args):
    """Run create_feishu_views.py with extra args, return CompletedProcess."""
    result = subprocess.run(
        [sys.executable, SCRIPT_PATH, *args],
        capture_output=True,
        text=True,
        cwd=os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    )
    return result


class TestDryRunExitsZero:
    """test_dry_run_exits_zero — subprocess.run with --dry-run exits 0."""

    def test_dry_run_exits_zero(self):
        result = _run_script('--dry-run')
        assert result.returncode == 0, f"stdout: {result.stdout}\nstderr: {result.stderr}"

    def test_no_args_exits_zero(self):
        """Default mode is dry-run, should also exit 0."""
        result = _run_script()
        assert result.returncode == 0


class TestDryRunPrintsAllSixViews:
    """test_dry_run_prints_all_six_views — output contains all 6 view names."""

    VIEW_NAMES = [
        'high_impact_alerts',
        'seller_alerts',
        'category_alerts',
        'region_alerts',
        'high_confidence_strategies',
        'object_tasks',
    ]

    def test_dry_run_prints_all_six_views(self):
        result = _run_script('--dry-run')
        assert result.returncode == 0
        for name in self.VIEW_NAMES:
            assert name in result.stdout, f"View '{name}' not found in output"

    def test_dry_run_reports_six_views_to_process(self):
        result = _run_script('--dry-run')
        assert 'Views to process: 6' in result.stdout


class TestViewFieldsExistInSchema:
    """test_view_fields_exist_in_schema — all filter/sort fields exist in schema."""

    def _import_views_and_schema(self):
        """Import VIEWS and field_lookup from the script."""
        spec = importlib.util.spec_from_file_location(
            'create_feishu_views', SCRIPT_PATH,
        )
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        schema = mod.load_schema()
        field_lookup = mod.build_field_lookup(schema)
        return mod.VIEWS, field_lookup

    def test_all_filter_fields_exist(self):
        views, field_lookup = self._import_views_and_schema()
        errors = []
        for view in views:
            table_id = view['table_id']
            valid = field_lookup.get(table_id, set())
            for field_name in view.get('filter', {}).keys():
                if field_name not in valid:
                    errors.append(
                        f"[{view['view_name']}] filter field '{field_name}' "
                        f"not in table '{table_id}'"
                    )
        assert not errors, f"Invalid filter fields: {errors}"

    def test_all_sort_fields_exist(self):
        views, field_lookup = self._import_views_and_schema()
        errors = []
        for view in views:
            table_id = view['table_id']
            valid = field_lookup.get(table_id, set())
            for sort_entry in view.get('sort', []):
                if sort_entry['field'] not in valid:
                    errors.append(
                        f"[{view['view_name']}] sort field '{sort_entry['field']}' "
                        f"not in table '{table_id}'"
                    )
        assert not errors, f"Invalid sort fields: {errors}"

    def test_validate_view_function_returns_no_errors(self):
        """Use the script's own validate_view to confirm all views pass."""
        spec = importlib.util.spec_from_file_location(
            'create_feishu_views', SCRIPT_PATH,
        )
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        schema = mod.load_schema()
        field_lookup = mod.build_field_lookup(schema)

        all_errors = []
        for view in mod.VIEWS:
            all_errors.extend(mod.validate_view(view, field_lookup))
        assert not all_errors, f"validate_view found errors: {all_errors}"


class TestInvalidFieldExitsNonzero:
    """test_invalid_field_exits_nonzero — non-existent field causes exit 1."""

    def test_validate_view_catches_bad_filter_field(self):
        spec = importlib.util.spec_from_file_location(
            'create_feishu_views', SCRIPT_PATH,
        )
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        schema = mod.load_schema()
        field_lookup = mod.build_field_lookup(schema)

        bad_view = {
            'view_name': 'test_bad',
            'table_id': 'alert_events',
            'filter': {'nonexistent_field': ['a']},
            'sort': [],
        }
        errors = mod.validate_view(bad_view, field_lookup)
        assert len(errors) == 1
        assert 'nonexistent_field' in errors[0]

    def test_validate_view_catches_bad_sort_field(self):
        spec = importlib.util.spec_from_file_location(
            'create_feishu_views', SCRIPT_PATH,
        )
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        schema = mod.load_schema()
        field_lookup = mod.build_field_lookup(schema)

        bad_view = {
            'view_name': 'test_bad_sort',
            'table_id': 'alert_events',
            'filter': {},
            'sort': [{'field': 'ghost_field', 'order': 'asc'}],
        }
        errors = mod.validate_view(bad_view, field_lookup)
        assert len(errors) == 1
        assert 'ghost_field' in errors[0]

    def test_validate_view_catches_bad_table(self):
        spec = importlib.util.spec_from_file_location(
            'create_feishu_views', SCRIPT_PATH,
        )
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        schema = mod.load_schema()
        field_lookup = mod.build_field_lookup(schema)

        bad_view = {
            'view_name': 'test_bad_table',
            'table_id': 'nonexistent_table',
            'filter': {},
            'sort': [],
        }
        errors = mod.validate_view(bad_view, field_lookup)
        assert len(errors) == 1
        assert 'nonexistent_table' in errors[0]


class TestViewFilterUsesValidOperators:
    """test_view_filter_uses_valid_operators — filters use supported operations (IN, =)."""

    def test_all_filters_are_in_style(self):
        """All filter values should be lists (IN semantics) or scalars (= semantics)."""
        spec = importlib.util.spec_from_file_location(
            'create_feishu_views', SCRIPT_PATH,
        )
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        for view in mod.VIEWS:
            filter_spec = view.get('filter', {})
            for field, values in filter_spec.items():
                assert isinstance(values, list), (
                    f"[{view['view_name']}] filter '{field}' should be list "
                    f"(IN operation), got {type(values).__name__}"
                )
                assert len(values) > 0, (
                    f"[{view['view_name']}] filter '{field}' has empty values list"
                )

    def test_sort_orders_are_valid(self):
        """Sort orders should be 'asc' or 'desc'."""
        spec = importlib.util.spec_from_file_location(
            'create_feishu_views', SCRIPT_PATH,
        )
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        valid_orders = {'asc', 'desc'}
        for view in mod.VIEWS:
            for sort_entry in view.get('sort', []):
                assert sort_entry['order'] in valid_orders, (
                    f"[{view['view_name']}] sort '{sort_entry['field']}' "
                    f"has invalid order '{sort_entry['order']}'"
                )


class TestApplyModeShowsApiStub:
    """test_apply_mode_shows_api_stub — --apply output contains expected stub markers."""

    def test_apply_mode_has_apply_in_output(self):
        result = _run_script('--apply')
        assert result.returncode == 0
        assert 'APPLY' in result.stdout, "Expected 'APPLY' in --apply mode output"

    def test_apply_mode_shows_api_call_stub(self):
        result = _run_script('--apply')
        assert 'API call' in result.stdout, "Expected 'API call' stub in --apply output"

    def test_apply_mode_notes_stub_status(self):
        result = _run_script('--apply')
        assert 'stub' in result.stdout.lower(), (
            "Expected stub disclaimer in --apply output"
        )
