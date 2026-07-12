package message

import (
	"encoding/json"
	"errors"
	"time"
)

// Envelope is the language-neutral metadata shared by all Plane messages.
// Payload validation remains the responsibility of the canonical JSON Schema.
type Envelope struct {
	MessageID          string          `json:"message_id"`
	MessageType        string          `json:"message_type"`
	FromComponent      string          `json:"from_component"`
	ToComponent        string          `json:"to_component"`
	CorrelationID      string          `json:"correlation_id"`
	CausationMessageID *string         `json:"causation_message_id"`
	OccurredAt         time.Time       `json:"occurred_at"`
	PayloadSchema      SchemaReference `json:"payload_schema"`
	Payload            json.RawMessage `json:"payload"`
}

type SchemaReference struct {
	SchemaID       string `json:"schema_id"`
	SchemaRevision int    `json:"schema_revision"`
	SchemaDigest   string `json:"schema_digest"`
}

func (e Envelope) Validate() error {
	if e.MessageID == "" || e.MessageType == "" {
		return errors.New("message identity is required")
	}
	if e.FromComponent == "" || e.ToComponent == "" || e.FromComponent == e.ToComponent {
		return errors.New("distinct source and target components are required")
	}
	if e.CorrelationID == "" || e.OccurredAt.IsZero() {
		return errors.New("correlation and timestamp are required")
	}
	if e.PayloadSchema.SchemaID == "" || e.PayloadSchema.SchemaRevision < 1 || e.PayloadSchema.SchemaDigest == "" {
		return errors.New("payload schema identity, revision, and digest are required")
	}
	if !json.Valid(e.Payload) {
		return errors.New("payload must be valid JSON")
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return errors.New("payload must be a JSON object")
	}
	return nil
}
