"""Shared query-building utilities for services.

Provides a single _build_conditions() function that extracts the common
WHERE-clause construction pattern used across multiple service modules.
"""


def _build_conditions(columns):
    """Build a WHERE clause and params list from a dict of column->value mappings.

    Only columns with non-None values are included in the WHERE clause.
    All conditions use equality ("=") and parameterised placeholders ("?").

    Args:
        columns: A dict mapping column names to filter values.
                 Columns whose value is None are skipped.

    Returns:
        A tuple (where_clause, params_list).
        - where_clause: A string like ``WHERE col1 = ? AND col2 = ?``,
          or an empty string when no columns are provided.
        - params_list: A list of values in the same order as the conditions.
    """
    conditions = []
    params = []
    for col, val in columns.items():
        if val is not None:
            if isinstance(val, (list, tuple)):
                placeholders = ", ".join(["?" for _ in val])
                conditions.append(f"{col} IN ({placeholders})")
                params.extend(val)
            else:
                conditions.append(f"{col} = ?")
                params.append(val)
    where = "WHERE " + " AND ".join(conditions) if conditions else ""
    return where, params
