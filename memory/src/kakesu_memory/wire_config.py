"""Reader for the shared local wire configuration."""

import json
from dataclasses import dataclass
from pathlib import Path

_COMPONENTS = {"core-runtime", "memory-service", "governance-service"}


@dataclass(frozen=True, slots=True)
class ComponentConfig:
    uds_endpoint: Path


@dataclass(frozen=True, slots=True)
class WireConfig:
    version: int
    schema_catalog_root: Path
    components: dict[str, ComponentConfig]


def load_wire_config(project_root: str | Path) -> WireConfig:
    root = Path(project_root)
    return parse_wire_config((root / "configs/local/wire.json").read_bytes(), root)


def parse_wire_config(data: str | bytes, project_root: str | Path) -> WireConfig:
    try:
        value = json.loads(data, parse_constant=_reject_json_constant)
    except (TypeError, json.JSONDecodeError) as exc:
        raise ValueError("invalid wire config JSON") from exc
    if not isinstance(value, dict) or set(value) != {
        "version",
        "schema_catalog_root",
        "components",
    }:
        raise ValueError("wire config fields must exactly match the contract")
    if value["version"] != 1 or isinstance(value["version"], bool):
        raise ValueError("wire config version must be 1")
    catalog = _safe_relative(value["schema_catalog_root"], "schema_catalog_root")
    components = value["components"]
    if not isinstance(components, dict) or set(components) != _COMPONENTS:
        raise ValueError("wire config must contain exactly the three known components")
    root = Path(project_root)
    resolved: dict[str, ComponentConfig] = {}
    endpoints: set[Path] = set()
    for name in sorted(_COMPONENTS):
        component = components[name]
        if not isinstance(component, dict) or set(component) != {"uds_endpoint"}:
            raise ValueError(f"invalid component config: {name}")
        endpoint = _safe_relative(component["uds_endpoint"], "uds_endpoint")
        if endpoint in endpoints:
            raise ValueError("UDS endpoints must be unique")
        endpoints.add(endpoint)
        resolved[name] = ComponentConfig(root / endpoint)
    return WireConfig(1, root / catalog, resolved)


def _safe_relative(value: object, field_name: str) -> Path:
    if not isinstance(value, str) or not value:
        raise ValueError(f"{field_name} must be a non-empty relative path")
    path = Path(value)
    if path.is_absolute() or ".." in path.parts or path == Path("."):
        raise ValueError(f"{field_name} must be a safe relative path")
    return path


def _reject_json_constant(value: str) -> None:
    raise ValueError(f"non-JSON numeric constant is not allowed: {value}")
