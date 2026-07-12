package message

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnvelopeValidation(t *testing.T) {
	t.Parallel()
	envelope := Envelope{
		MessageID:     "msg-1",
		MessageType:   "MemoryContextRequest",
		FromComponent: "run-coordinator",
		ToComponent:   "memory-context-service",
		CorrelationID: "task-1",
		OccurredAt:    time.Now().UTC(),
		PayloadSchema: SchemaReference{
			SchemaID:       "urn:kakesu:memory-plane:memory-context-request:draft-v0:r1",
			SchemaRevision: 1,
			SchemaDigest:   "sha256:fixture",
		},
		Payload: json.RawMessage(`{"request_id":"request-1"}`),
	}
	if err := envelope.Validate(); err != nil {
		t.Fatalf("expected valid envelope: %v", err)
	}
}

func TestEnvelopeRejectsSameComponent(t *testing.T) {
	t.Parallel()
	envelope := Envelope{
		MessageID:     "msg-1",
		MessageType:   "event",
		FromComponent: "run-coordinator",
		ToComponent:   "run-coordinator",
		CorrelationID: "task-1",
		OccurredAt:    time.Now().UTC(),
		PayloadSchema: SchemaReference{
			SchemaID:       "urn:fixture",
			SchemaRevision: 1,
			SchemaDigest:   "sha256:fixture",
		},
		Payload: json.RawMessage(`{}`),
	}
	if err := envelope.Validate(); err == nil {
		t.Fatal("expected same-component envelope to be rejected")
	}
}
