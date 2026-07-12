"""Canonical cross-plane message envelope used by the Memory Service."""

from dataclasses import dataclass
from datetime import datetime
from typing import Any


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

    def validate(self) -> None:
        if not self.message_id or not self.message_type:
            raise ValueError("message identity is required")
        if (
            not self.from_component
            or not self.to_component
            or self.from_component == self.to_component
        ):
            raise ValueError("distinct source and target components are required")
        if not self.correlation_id or self.occurred_at.tzinfo is None:
            raise ValueError("correlation and timezone-aware timestamp are required")
        if (
            not self.payload_schema.schema_id
            or self.payload_schema.schema_revision < 1
            or not self.payload_schema.schema_digest
        ):
            raise ValueError("payload schema identity, revision, and digest are required")
