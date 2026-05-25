import importlib
from abc import ABC, abstractmethod

import yaml

from core import config


class ChannelAdapter(ABC):
    @abstractmethod
    def dry_run(self, event: dict) -> dict:
        """Preview what would be dispatched. No side effects."""

    @abstractmethod
    def dispatch(self, event: dict) -> dict:
        """Execute real external action.

        Returns:
            dict with keys:
                status: 'dispatched' | 'failed' | 'skipped' | 'preview'
                external_ref: target system's reference ID or None
                error: error message or None
                message: human-readable description (optional)
                payload: generated payload for dry_run (optional)
        """


def load_adapter_registry():
    with open(config.ADAPTER_REGISTRY_FILE) as f:
        return yaml.safe_load(f)


def resolve_adapter(channel_name, registry=None, dry_run=False):
    if registry is None:
        registry = load_adapter_registry()

    adapter_config = None
    for name, cfg in registry.get('adapters', {}).items():
        if channel_name in cfg.get('allowed_target_channels', []):
            adapter_config = (name, cfg)
            break

    if adapter_config is None:
        raise ValueError(f"No adapter found for channel: {channel_name}")

    name, cfg = adapter_config
    try:
        module = importlib.import_module(cfg['module'])
        adapter_class = getattr(module, cfg['class'])
        return adapter_class(dry_run=dry_run)
    except ImportError as e:
        raise ImportError(f"Failed to import adapter '{name}' from {cfg['module']}: {e}")
    except AttributeError as e:
        raise AttributeError(f"Adapter '{name}' class {cfg['class']} not found in {cfg['module']}: {e}")
