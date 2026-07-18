package message

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

var schemaDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// DecodeAndValidate strictly decodes a wire envelope and binds its schema
// reference to one exact artifact in catalogRoot.
func DecodeAndValidate(data []byte, catalogRoot string) (*Envelope, error) {
	var wire struct {
		MessageID          string          `json:"message_id"`
		MessageType        string          `json:"message_type"`
		FromComponent      string          `json:"from_component"`
		ToComponent        string          `json:"to_component"`
		CorrelationID      string          `json:"correlation_id"`
		CausationMessageID json.RawMessage `json:"causation_message_id"`
		OccurredAt         string          `json:"occurred_at"`
		PayloadSchema      SchemaReference `json:"payload_schema"`
		Payload            json.RawMessage `json:"payload"`
	}
	if err := decodeStrict(data, &wire); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	if wire.CausationMessageID == nil {
		return nil, errors.New("causation_message_id is required, including when null")
	}
	var causation *string
	if err := json.Unmarshal(wire.CausationMessageID, &causation); err != nil {
		return nil, errors.New("causation_message_id must be a string or null")
	}
	if causation != nil && *causation == "" {
		return nil, errors.New("causation_message_id must not be empty")
	}
	if !strings.HasSuffix(wire.OccurredAt, "Z") {
		return nil, errors.New("occurred_at must be RFC3339 UTC with a trailing Z")
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, wire.OccurredAt)
	if err != nil || occurredAt.Location() != time.UTC {
		return nil, errors.New("occurred_at must be RFC3339 UTC with a trailing Z")
	}
	envelope := &Envelope{
		MessageID: wire.MessageID, MessageType: wire.MessageType,
		FromComponent: wire.FromComponent, ToComponent: wire.ToComponent,
		CorrelationID: wire.CorrelationID, CausationMessageID: causation,
		OccurredAt: occurredAt, PayloadSchema: wire.PayloadSchema, Payload: wire.Payload,
	}
	if err := envelope.Validate(); err != nil {
		return nil, err
	}
	if err := validateSchemaReference(envelope.PayloadSchema, catalogRoot); err != nil {
		return nil, err
	}
	return envelope, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
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
	if err := json.Unmarshal(e.Payload, &payload); err != nil || payload == nil {
		return errors.New("payload must be a JSON object")
	}
	return nil
}

func validateSchemaReference(reference SchemaReference, catalogRoot string) error {
	if !schemaDigestPattern.MatchString(reference.SchemaDigest) {
		return errors.New("schema_digest must be sha256 followed by lowercase hex")
	}
	type metadata struct {
		SchemaID string `json:"$id"`
		Revision int    `json:"x-schema-revision"`
	}
	var matches [][]byte
	err := filepath.WalkDir(catalogRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var item metadata
		var document any
		if err := json.Unmarshal(raw, &document); err != nil {
			return fmt.Errorf("parse schema %s: %w", path, err)
		}
		if _, ok := document.(map[string]any); !ok {
			return nil
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return fmt.Errorf("read schema metadata %s: %w", path, err)
		}
		if item.SchemaID == reference.SchemaID && item.Revision == reference.SchemaRevision {
			matches = append(matches, raw)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan schema catalog: %w", err)
	}
	if len(matches) != 1 {
		return fmt.Errorf("schema reference must resolve exactly once, resolved %d times", len(matches))
	}
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(matches[0]))
	if digest != reference.SchemaDigest {
		return errors.New("schema digest does not match catalog artifact bytes")
	}
	return nil
}
