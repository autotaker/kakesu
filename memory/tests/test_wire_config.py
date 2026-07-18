import copy
import json
from pathlib import Path

import pytest

from kakesu_memory.wire_config import load_wire_config, parse_wire_config

ROOT = Path(__file__).parents[2]


def test_load_wire_config_resolves_shared_paths() -> None:
    config = load_wire_config(ROOT)
    assert config.version == 1
    assert config.schema_catalog_root == ROOT / "schemas/draft-v0"
    assert set(config.components) == {
        "core-runtime",
        "memory-service",
        "governance-service",
    }
    assert len({component.uds_endpoint for component in config.components.values()}) == 3


@pytest.mark.parametrize(
    "mutation",
    [
        "version",
        "missing",
        "extra",
        "absolute_catalog",
        "absolute_endpoint",
        "parent_endpoint",
        "empty_endpoint",
        "duplicate_endpoint",
    ],
)
def test_wire_config_rejects_invalid_boundaries(mutation: str) -> None:
    original = json.loads((ROOT / "configs/local/wire.json").read_bytes())
    value = copy.deepcopy(original)
    if mutation == "version":
        value["version"] = 2
    elif mutation == "missing":
        del value["components"]["core-runtime"]
    elif mutation == "extra":
        value["components"]["extra"] = {"uds_endpoint": "run/extra.sock"}
    elif mutation == "absolute_catalog":
        value["schema_catalog_root"] = "/schemas"
    elif mutation == "absolute_endpoint":
        value["components"]["core-runtime"]["uds_endpoint"] = "/run/core.sock"
    elif mutation == "parent_endpoint":
        value["components"]["core-runtime"]["uds_endpoint"] = "run/../core.sock"
    elif mutation == "empty_endpoint":
        value["components"]["core-runtime"]["uds_endpoint"] = ""
    else:
        value["components"]["core-runtime"]["uds_endpoint"] = value["components"]["memory-service"][
            "uds_endpoint"
        ]
    with pytest.raises(ValueError):
        parse_wire_config(json.dumps(value), ROOT)
