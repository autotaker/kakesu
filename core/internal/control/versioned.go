package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
)

type SchemaReference struct {
	ID       string `json:"schema_id"`
	Revision string `json:"schema_revision"`
	Digest   string `json:"schema_digest"`
}

type ContractVersion struct {
	Version int
	Schema  SchemaReference
	Payload []byte
}

type ProgressVersion struct {
	Version                      int
	Schema                       SchemaReference
	Payload                      []byte
	ThroughTaskEventSequence     int
	ThroughAgentRunEventSequence int
}

type ContractUpdateInput struct {
	TaskID          string
	ExpectedVersion int
	New             ContractVersion
}

type ProgressUpdateInput struct {
	TaskID          string
	ExpectedVersion int
	New             ProgressVersion
}

type VersionedUpdateResult struct {
	Event TaskEvent
}

type VersionedInputError struct {
	Kind string
	Err  error
}

func (e *VersionedInputError) Error() string {
	return fmt.Sprintf("control %s input: %v", e.Kind, e.Err)
}
func (e *VersionedInputError) Unwrap() error { return e.Err }

var schemaDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func (s *Store) UpdateContract(ctx context.Context, input ContractUpdateInput) (VersionedUpdateResult, error) {
	if err := validateVersionedInput("contract", input.ExpectedVersion, input.New.Version, input.New.Schema, input.New.Payload); err != nil {
		return VersionedUpdateResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return VersionedUpdateResult{}, &StorageError{Operation: "begin contract update", Err: err}
	}
	defer tx.Rollback()
	if err := requireCurrentVersion(ctx, tx, input.TaskID, "task_contracts", input.ExpectedVersion); err != nil {
		return VersionedUpdateResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_contract_history(task_id, version, schema_id, schema_revision, schema_digest, payload) VALUES (?, ?, ?, ?, ?, ?)`,
		input.TaskID, input.New.Version, input.New.Schema.ID, input.New.Schema.Revision, input.New.Schema.Digest, input.New.Payload); err != nil {
		return VersionedUpdateResult{}, classifyVersionedError(input.TaskID, "append contract history", err)
	}
	if err := s.runVersionedHook(tx, "contract", "history"); err != nil {
		return VersionedUpdateResult{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE task_contracts SET version = ?, schema_id = ?, schema_revision = ?, schema_digest = ?, payload = ? WHERE task_id = ? AND version = ?`,
		input.New.Version, input.New.Schema.ID, input.New.Schema.Revision, input.New.Schema.Digest, input.New.Payload, input.TaskID, input.ExpectedVersion)
	if err != nil {
		return VersionedUpdateResult{}, classifyVersionedError(input.TaskID, "update current contract", err)
	}
	if err := requireOneCAS(input.TaskID, result); err != nil {
		return VersionedUpdateResult{}, err
	}
	if err := s.runVersionedHook(tx, "contract", "current"); err != nil {
		return VersionedUpdateResult{}, err
	}
	event, err := appendVersionEvent(ctx, tx, input.TaskID, "ContractChanged", input.New, ProgressVersion{})
	if err != nil {
		return VersionedUpdateResult{}, err
	}
	if err := s.runVersionedHook(tx, "contract", "event"); err != nil {
		return VersionedUpdateResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return VersionedUpdateResult{}, classifyVersionedError(input.TaskID, "commit contract update", err)
	}
	return VersionedUpdateResult{Event: event}, nil
}

func (s *Store) UpdateProgress(ctx context.Context, input ProgressUpdateInput) (VersionedUpdateResult, error) {
	if err := validateVersionedInput("progress", input.ExpectedVersion, input.New.Version, input.New.Schema, input.New.Payload); err != nil {
		return VersionedUpdateResult{}, err
	}
	if input.New.ThroughTaskEventSequence < 0 || input.New.ThroughAgentRunEventSequence < 0 {
		return VersionedUpdateResult{}, &VersionedInputError{Kind: "progress", Err: fmt.Errorf("event watermark sequences must be non-negative")}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return VersionedUpdateResult{}, &StorageError{Operation: "begin progress update", Err: err}
	}
	defer tx.Rollback()
	if err := requireCurrentVersion(ctx, tx, input.TaskID, "task_progress", input.ExpectedVersion); err != nil {
		return VersionedUpdateResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_progress_history(task_id, version, schema_id, schema_revision, schema_digest, payload, through_task_event_sequence, through_agent_run_event_sequence) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		input.TaskID, input.New.Version, input.New.Schema.ID, input.New.Schema.Revision, input.New.Schema.Digest, input.New.Payload, input.New.ThroughTaskEventSequence, input.New.ThroughAgentRunEventSequence); err != nil {
		return VersionedUpdateResult{}, classifyVersionedError(input.TaskID, "append progress history", err)
	}
	if err := s.runVersionedHook(tx, "progress", "history"); err != nil {
		return VersionedUpdateResult{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE task_progress SET version = ?, schema_id = ?, schema_revision = ?, schema_digest = ?, payload = ?, through_task_event_sequence = ?, through_agent_run_event_sequence = ? WHERE task_id = ? AND version = ?`,
		input.New.Version, input.New.Schema.ID, input.New.Schema.Revision, input.New.Schema.Digest, input.New.Payload, input.New.ThroughTaskEventSequence, input.New.ThroughAgentRunEventSequence, input.TaskID, input.ExpectedVersion)
	if err != nil {
		return VersionedUpdateResult{}, classifyVersionedError(input.TaskID, "update current progress", err)
	}
	if err := requireOneCAS(input.TaskID, result); err != nil {
		return VersionedUpdateResult{}, err
	}
	if err := s.runVersionedHook(tx, "progress", "current"); err != nil {
		return VersionedUpdateResult{}, err
	}
	event, err := appendVersionEvent(ctx, tx, input.TaskID, "ProgressRefreshed", ContractVersion{}, input.New)
	if err != nil {
		return VersionedUpdateResult{}, err
	}
	if err := s.runVersionedHook(tx, "progress", "event"); err != nil {
		return VersionedUpdateResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return VersionedUpdateResult{}, classifyVersionedError(input.TaskID, "commit progress update", err)
	}
	return VersionedUpdateResult{Event: event}, nil
}

func validateVersionedInput(kind string, expected, next int, schema SchemaReference, payload []byte) error {
	if expected < 0 || next != expected+1 {
		return &VersionedInputError{Kind: kind, Err: fmt.Errorf("new version must equal expected version plus one")}
	}
	if schema.ID == "" || schema.Revision == "" || !schemaDigestPattern.MatchString(schema.Digest) {
		return &VersionedInputError{Kind: kind, Err: fmt.Errorf("valid schema ID, revision, and sha256 digest are required")}
	}
	var object map[string]json.RawMessage
	if len(payload) == 0 || json.Unmarshal(payload, &object) != nil || object == nil {
		return &VersionedInputError{Kind: kind, Err: fmt.Errorf("payload must be a JSON object")}
	}
	return nil
}

func requireCurrentVersion(ctx context.Context, tx *sql.Tx, taskID, table string, expected int) error {
	var current int
	if err := tx.QueryRowContext(ctx, `SELECT version FROM `+table+` WHERE task_id = ?`, taskID).Scan(&current); err != nil {
		return &StorageError{Operation: "read current " + table, Err: err}
	}
	if current != expected {
		return &ConflictError{TaskID: taskID, Err: fmt.Errorf("expected version %d, found %d", expected, current)}
	}
	return nil
}

func requireOneCAS(taskID string, result sql.Result) error {
	changed, err := result.RowsAffected()
	if err == nil && changed == 1 {
		return nil
	}
	if err == nil {
		err = fmt.Errorf("expected one updated row, got %d", changed)
	}
	return &ConflictError{TaskID: taskID, Err: err}
}

func appendVersionEvent(ctx context.Context, tx *sql.Tx, taskID, eventType string, contract ContractVersion, progress ProgressVersion) (TaskEvent, error) {
	schema := contract.Schema
	payloadFields := map[string]any{"version": contract.Version, "schema": schema}
	if eventType == "ProgressRefreshed" {
		schema = progress.Schema
		payloadFields = map[string]any{"progress_version": progress.Version, "through_task_event_sequence": progress.ThroughTaskEventSequence, "through_agent_run_event_sequence": progress.ThroughAgentRunEventSequence, "schema": schema}
	}
	payload, _ := json.Marshal(payloadFields)
	var sequence int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM task_events WHERE task_id = ?`, taskID).Scan(&sequence); err != nil {
		return TaskEvent{}, &StorageError{Operation: "read event sequence", Err: err}
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO task_events(task_id, sequence, event_type, payload, schema_id, schema_revision, schema_digest) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		taskID, sequence, eventType, payload, schema.ID, schema.Revision, schema.Digest)
	if err != nil {
		return TaskEvent{}, classifyVersionedError(taskID, "append version event", err)
	}
	return TaskEvent{Sequence: sequence, Type: eventType, Payload: payload, Schema: schema}, nil
}

func (s *Store) runVersionedHook(tx *sql.Tx, kind, stage string) error {
	if s.versionedHook == nil {
		return nil
	}
	if err := s.versionedHook(tx, kind, stage); err != nil {
		return &StorageError{Operation: kind + " update hook " + stage, Err: err}
	}
	return nil
}

func classifyVersionedError(taskID, operation string, err error) error {
	if conflict := classifyCreateError(taskID, err); conflict != nil {
		if _, ok := conflict.(*ConflictError); ok {
			return conflict
		}
	}
	return &StorageError{Operation: operation, Err: err}
}
