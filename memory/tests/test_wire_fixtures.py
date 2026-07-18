import json
from pathlib import Path

import pytest

from kakesu_memory.message import Envelope

ROOT = Path(__file__).parents[2]
FIXTURES = ROOT / "fixtures/wire/envelope"
CATALOG = ROOT / "schemas/draft-v0"


@pytest.mark.parametrize(
    ("fixture", "valid"),
    [
        ("valid.json", True),
        ("invalid-same-route.json", False),
        ("invalid-non-utc-timestamp.json", False),
        ("invalid-schema-reference-digest.json", False),
        ("invalid-non-object-payload.json", False),
    ],
)
def test_shared_envelope_fixture_matrix(fixture: str, valid: bool) -> None:
    if valid:
        Envelope.from_json((FIXTURES / fixture).read_bytes(), CATALOG)
    else:
        with pytest.raises(ValueError):
            Envelope.from_json((FIXTURES / fixture).read_bytes(), CATALOG)


@pytest.mark.parametrize("change", ["missing", "unknown"])
def test_envelope_requires_exact_fields_and_explicit_null(change: str) -> None:
    value = json.loads((FIXTURES / "valid.json").read_bytes())
    if change == "missing":
        del value["causation_message_id"]
    else:
        value["unknown"] = True
    with pytest.raises(ValueError):
        Envelope.from_json(json.dumps(value), CATALOG)


def test_envelope_rejects_raw_schema_mutation(tmp_path: Path) -> None:
    schema = (CATALOG / "common/error.schema.json").read_bytes()
    (tmp_path / "error.schema.json").write_bytes(schema + b" ")
    with pytest.raises(ValueError, match="digest"):
        Envelope.from_json((FIXTURES / "valid.json").read_bytes(), tmp_path)
