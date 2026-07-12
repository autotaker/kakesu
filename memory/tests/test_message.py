from datetime import UTC, datetime

from kakesu_memory import Envelope, SchemaReference


def test_canonical_envelope_validation() -> None:
    envelope = Envelope(
        message_id="msg-1",
        message_type="EpisodeAgentInput",
        from_component="run-coordinator",
        to_component="episode-agent-runner",
        correlation_id="task-1",
        causation_message_id=None,
        occurred_at=datetime.now(UTC),
        payload_schema=SchemaReference(
            schema_id="urn:kakesu:memory-plane:episode-agent-input:draft-v0:r1",
            schema_revision=1,
            schema_digest="sha256:fixture",
        ),
        payload={"task_id": "task-1"},
    )

    envelope.validate()
