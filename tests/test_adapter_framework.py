import pytest, sys, json, os
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'scripts'))
from adapters.base import ChannelAdapter, load_adapter_registry, resolve_adapter
from adapters.feishu_adapter import FeishuAdapter
from adapters.github_issue_adapter import GitHubIssueAdapter
from adapters.local_cli_adapter import LocalCLIAdapter
from adapters.manual_adapter import ManualAdapter

PAYLOAD = json.dumps({"rule_id":"gmv_drop","metric_name":"gmv","current_value":1000,"baseline_value":1500,"severity":"high","owner_role":"business_ops"})
EVENT = {"outbox_id": "test-001", "payload_json": PAYLOAD}


class TestChannelAdapter:
    def test_abc_is_abstract(self):
        import inspect
        assert inspect.isabstract(ChannelAdapter)

    def test_abc_has_dry_run(self):
        assert hasattr(ChannelAdapter, "dry_run")

    def test_abc_has_dispatch(self):
        assert hasattr(ChannelAdapter, "dispatch")

    def test_registry_loads(self):
        reg = load_adapter_registry()
        assert "adapters" in reg
        assert len(reg["adapters"]) == 4
        for a in ["feishu", "github_issue", "local_cli", "manual"]:
            assert a in reg["adapters"]
            assert "module" in reg["adapters"][a]
            assert "class" in reg["adapters"][a]

    def test_resolve_feishu_adapter(self):
        adapter = resolve_adapter("feishu_cli", dry_run=True)
        assert isinstance(adapter, FeishuAdapter)

    def test_resolve_unknown_channel(self):
        with pytest.raises(ValueError):
            resolve_adapter("nonexistent")


class TestFeishuAdapter:
    def test_extends_channel_adapter(self):
        a = FeishuAdapter(dry_run=True)
        assert isinstance(a, ChannelAdapter)

    def test_dry_run_format(self):
        a = FeishuAdapter(dry_run=True)
        r = a.dry_run(EVENT)
        assert r["status"] == "preview"
        assert "gmv_drop" in r["message"]
        assert "gmv" in r["message"]
        assert r["external_ref"] is None

    def test_null_payload(self):
        a = FeishuAdapter(dry_run=True)
        r = a.dispatch({"payload_json": None})
        assert r["status"] == "failed"
        assert r["error"] is not None

    def test_invalid_json(self):
        a = FeishuAdapter(dry_run=True)
        r = a.dispatch({"payload_json": "not valid json"})
        assert r["status"] == "failed"

    def test_feishu_adapter_constructor(self):
        a = FeishuAdapter(dry_run=True)
        assert isinstance(a, ChannelAdapter)
        assert hasattr(a, "dispatch") and callable(a.dispatch)


class TestGitHubIssueAdapter:
    def test_extends_channel_adapter(self):
        a = GitHubIssueAdapter()
        assert isinstance(a, ChannelAdapter)

    def test_dry_run_payload(self):
        a = GitHubIssueAdapter()
        r = a.dry_run(EVENT)
        assert r["status"] == "preview"
        p = r["payload"]
        assert "title" in p
        assert "body" in p
        assert "gmv_drop" in p["title"]
        assert "labels" in p

    def test_dispatch_is_blocked(self):
        a = GitHubIssueAdapter()
        r = a.dispatch(EVENT)
        assert r["status"] in ("skipped", "failed")

    def test_null_payload_dry_run(self):
        a = GitHubIssueAdapter()
        r = a.dry_run({"payload_json": None})
        assert r["status"] == "preview"


class TestLocalCLIAdapter:
    def test_extends_channel_adapter(self):
        a = LocalCLIAdapter()
        assert isinstance(a, ChannelAdapter)

    def test_dry_run_format(self):
        a = LocalCLIAdapter()
        r = a.dry_run(EVENT)
        assert r["status"] == "preview"
        assert "run_alert_detection" in r["message"]
        assert "--rule gmv_drop" in r["message"]

    def test_dispatch_writes_csv(self):
        import tempfile
        a = LocalCLIAdapter()
        r = a.dispatch(EVENT)
        assert r["status"] == "dispatched"
        assert r["external_ref"] is not None

    def test_null_payload_dry_run(self):
        a = LocalCLIAdapter()
        r = a.dry_run({"payload_json": None})
        assert r["status"] == "preview"


class TestManualAdapter:
    def test_extends_channel_adapter(self):
        a = ManualAdapter()
        assert isinstance(a, ChannelAdapter)

    def test_dispatch_skipped(self):
        a = ManualAdapter()
        r = a.dispatch(EVENT)
        assert r["status"] == "skipped"
        assert "manual" in r.get("message", "").lower()

    def test_dry_run_preview(self):
        a = ManualAdapter()
        r = a.dry_run(EVENT)
        assert r["status"] == "preview"
        assert "manual" in r.get("message", "").lower()
