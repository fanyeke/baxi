import pytest

from api.errors import APIError
from api.routers.governance import _load_yaml


def test_load_yaml_rejects_dotdot():
    with pytest.raises(APIError) as exc_info:
        _load_yaml("../../../etc/passwd")
    assert exc_info.value.error_code == "INVALID_FILENAME"


def test_load_yaml_rejects_null_byte():
    with pytest.raises(APIError) as exc_info:
        _load_yaml("file\x00.txt")
    assert exc_info.value.error_code == "INVALID_FILENAME"


def test_load_yaml_rejects_empty():
    with pytest.raises(APIError) as exc_info:
        _load_yaml("")
    assert exc_info.value.error_code == "INVALID_FILENAME"


def test_load_yaml_allows_valid():
    result = _load_yaml("data_catalog.yml")
    assert isinstance(result, dict)
