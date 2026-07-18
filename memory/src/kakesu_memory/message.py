"""Strict canonical cross-plane message envelope for the Memory Service."""

import json
import re
from dataclasses import dataclass, field
from datetime import datetime
from hashlib import sha256
from pathlib import Path
from typing import Any

_ENVELOPE_FIELDS = {
    "message_id",
    "message_type",
    "from_component",
    "to_component",
    "correlation_id",
    "causation_message_id",
    "occurred_at",
    "payload_schema",
    "payload",
}
_SCHEMA_FIELDS = {"schema_id", "schema_revision", "schema_digest"}
_DIGEST = re.compile(r"sha256:[0-9a-f]{64}\Z")
_RFC3339_UTC = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z\Z")


@dataclass(frozen=True, slots=True)
class SchemaReference:
    schema_id: str
    schema_revision: int
    schema_digest: str


@dataclass(frozen=True, slots=True)
class Envelope:
    message_id: str
    message_type: str
    from_component: str
    to_component: str
    correlation_id: str
    causation_message_id: str | None
    occurred_at: datetime
    payload_schema: SchemaReference
    payload: dict[str, Any]
    _occurred_at_wire: str | None = field(default=None, repr=False, compare=False)

    @classmethod
    def from_json(cls, data: str | bytes, catalog_root: str | Path) -> "Envelope":
        try:
            value = json.loads(data, parse_constant=_reject_json_constant)
        except (TypeError, json.JSONDecodeError) as exc:
            raise ValueError("invalid envelope JSON") from exc
        if not isinstance(value, dict) or set(value) != _ENVELOPE_FIELDS:
            raise ValueError("envelope fields must exactly match the wire contract")
        schema = value["payload_schema"]
        if not isinstance(schema, dict) or set(schema) != _SCHEMA_FIELDS:
            raise ValueError("payload_schema fields must exactly match the wire contract")
        revision = schema["schema_revision"]
        if not isinstance(revision, int) or isinstance(revision, bool):
            raise ValueError("schema_revision must be an integer")
        occurred_at_wire = value["occurred_at"]
        if (
            not isinstance(occurred_at_wire, str)
            or _RFC3339_UTC.fullmatch(occurred_at_wire) is None
        ):
            raise ValueError("occurred_at must be RFC3339 UTC with a trailing Z")
        try:
            occurred_at = datetime.fromisoformat(occurred_at_wire)
        except ValueError as exc:
            raise ValueError("occurred_at must be valid RFC3339") from exc
        causation = value["causation_message_id"]
        if causation is not None and (not isinstance(causation, str) or not causation):
            raise ValueError("causation_message_id must be a non-empty string or null")
        envelope = cls(
            message_id=_string(value["message_id"], "message_id"),
            message_type=_string(value["message_type"], "message_type"),
            from_component=_string(value["from_component"], "from_component"),
            to_component=_string(value["to_component"], "to_component"),
            correlation_id=_string(value["correlation_id"], "correlation_id"),
            causation_message_id=causation,
            occurred_at=occurred_at,
            payload_schema=SchemaReference(
                schema_id=_string(schema["schema_id"], "schema_id"),
                schema_revision=revision,
                schema_digest=_string(schema["schema_digest"], "schema_digest"),
            ),
            payload=value["payload"],
            _occurred_at_wire=occurred_at_wire,
        )
        envelope.validate(Path(catalog_root))
        return envelope

    def validate(self, catalog_root: Path | None = None) -> None:
        if not self.message_id or not self.message_type:
            raise ValueError("message identity is required")
        if (
            not self.from_component
            or not self.to_component
            or self.from_component == self.to_component
        ):
            raise ValueError("distinct source and target components are required")
        if (
            not self.correlation_id
            or self.occurred_at.tzinfo is None
            or self.occurred_at.utcoffset() is None
            or self.occurred_at.utcoffset().total_seconds() != 0
        ):
            raise ValueError("correlation and UTC timestamp are required")
        if self._occurred_at_wire is not None and not self._occurred_at_wire.endswith("Z"):
            raise ValueError("wire timestamp must end in Z")
        if (
            not self.payload_schema.schema_id
            or self.payload_schema.schema_revision < 1
            or not self.payload_schema.schema_digest
        ):
            raise ValueError("payload schema identity, revision, and digest are required")
        if not isinstance(self.payload, dict):
            raise ValueError("payload must be a JSON object")
        if catalog_root is not None:
            _validate_schema_reference(self.payload_schema, catalog_root)


def _string(value: object, field_name: str) -> str:
    if not isinstance(value, str) or not value:
        raise ValueError(f"{field_name} must be a non-empty string")
    return value


def _reject_json_constant(value: str) -> None:
    raise ValueError(f"non-JSON numeric constant is not allowed: {value}")


def _validate_schema_reference(reference: SchemaReference, catalog_root: Path) -> None:
    if _DIGEST.fullmatch(reference.schema_digest) is None:
        raise ValueError("schema_digest must be sha256 followed by lowercase hex")
    matches: list[bytes] = []
    try:
        paths = catalog_root.rglob("*.json")
        for path in paths:
            raw = path.read_bytes()
            metadata = json.loads(raw)
            if not isinstance(metadata, dict):
                continue
            if (
                metadata.get("$id") == reference.schema_id
                and metadata.get("x-schema-revision") == reference.schema_revision
            ):
                matches.append(raw)
    except (OSError, json.JSONDecodeError) as exc:
        raise ValueError("unable to scan schema catalog") from exc
    if len(matches) != 1:
        raise ValueError(f"schema reference resolved {len(matches)} times")
    digest = f"sha256:{sha256(matches[0]).hexdigest()}"
    if digest != reference.schema_digest:
        raise ValueError("schema digest does not match catalog artifact bytes")
