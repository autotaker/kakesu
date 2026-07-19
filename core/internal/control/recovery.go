package control

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

type RecoveredTask struct {
	TaskID          string
	State           TaskState
	OwnerAgentID    string
	OwnerActive     bool
	WorkspaceRef    string
	CurrentContract ContractVersion
	CurrentProgress ProgressVersion
	ContractHistory []ContractVersion
	ProgressHistory []ProgressVersion
	Events          []TaskEvent
}

type CorruptionError struct {
	TaskID string
	Err    error
}

func (e *CorruptionError) Error() string {
	return fmt.Sprintf("control task %q is corrupt: %v", e.TaskID, e.Err)
}
func (e *CorruptionError) Unwrap() error { return e.Err }

func (s *Store) ReadRecoveredTask(ctx context.Context, taskID string) (RecoveredTask, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return RecoveredTask{}, &StorageError{Operation: "begin recovery read", Err: err}
	}
	defer tx.Rollback()
	model, err := readRecoveredTask(ctx, tx, taskID)
	if err != nil {
		return RecoveredTask{}, err
	}
	if err := tx.Commit(); err != nil {
		return RecoveredTask{}, &StorageError{Operation: "commit recovery read", Err: err}
	}
	return model, nil
}

func readRecoveredTask(ctx context.Context, tx *sql.Tx, taskID string) (RecoveredTask, error) {
	lifecycle, err := readLifecycle(ctx, tx, taskID)
	if err != nil {
		return RecoveredTask{}, err
	}
	model := RecoveredTask{TaskID: taskID, State: lifecycle.State, OwnerAgentID: lifecycle.OwnerAgentID, OwnerActive: lifecycle.OwnerActive, Events: lifecycle.Events}
	err = tx.QueryRowContext(ctx, `SELECT w.workspace_ref,
		c.version, c.schema_id, c.schema_revision, c.schema_digest, c.payload,
		p.version, p.schema_id, p.schema_revision, p.schema_digest, p.payload, p.through_task_event_sequence, p.through_agent_run_event_sequence
		FROM task_workspaces w JOIN task_contracts c USING (task_id) JOIN task_progress p USING (task_id) WHERE w.task_id = ?`, taskID).Scan(
		&model.WorkspaceRef, &model.CurrentContract.Version, &model.CurrentContract.Schema.ID, &model.CurrentContract.Schema.Revision, &model.CurrentContract.Schema.Digest, &model.CurrentContract.Payload,
		&model.CurrentProgress.Version, &model.CurrentProgress.Schema.ID, &model.CurrentProgress.Schema.Revision, &model.CurrentProgress.Schema.Digest, &model.CurrentProgress.Payload, &model.CurrentProgress.ThroughTaskEventSequence, &model.CurrentProgress.ThroughAgentRunEventSequence)
	if err != nil {
		return RecoveredTask{}, &StorageError{Operation: "read recovery current", Err: err}
	}
	contractRows, err := tx.QueryContext(ctx, `SELECT version, schema_id, schema_revision, schema_digest, payload FROM task_contract_history WHERE task_id = ? ORDER BY version`, taskID)
	if err != nil {
		return RecoveredTask{}, &StorageError{Operation: "read contract history", Err: err}
	}
	for contractRows.Next() {
		var item ContractVersion
		if err := contractRows.Scan(&item.Version, &item.Schema.ID, &item.Schema.Revision, &item.Schema.Digest, &item.Payload); err != nil {
			contractRows.Close()
			return RecoveredTask{}, &StorageError{Operation: "scan contract history", Err: err}
		}
		model.ContractHistory = append(model.ContractHistory, item)
	}
	if err := contractRows.Err(); err != nil {
		contractRows.Close()
		return RecoveredTask{}, &StorageError{Operation: "iterate contract history", Err: err}
	}
	if err := contractRows.Close(); err != nil {
		return RecoveredTask{}, &StorageError{Operation: "close contract history", Err: err}
	}
	progressRows, err := tx.QueryContext(ctx, `SELECT version, schema_id, schema_revision, schema_digest, payload, through_task_event_sequence, through_agent_run_event_sequence FROM task_progress_history WHERE task_id = ? ORDER BY version`, taskID)
	if err != nil {
		return RecoveredTask{}, &StorageError{Operation: "read progress history", Err: err}
	}
	for progressRows.Next() {
		var item ProgressVersion
		if err := progressRows.Scan(&item.Version, &item.Schema.ID, &item.Schema.Revision, &item.Schema.Digest, &item.Payload, &item.ThroughTaskEventSequence, &item.ThroughAgentRunEventSequence); err != nil {
			progressRows.Close()
			return RecoveredTask{}, &StorageError{Operation: "scan progress history", Err: err}
		}
		model.ProgressHistory = append(model.ProgressHistory, item)
	}
	if err := progressRows.Err(); err != nil {
		progressRows.Close()
		return RecoveredTask{}, &StorageError{Operation: "iterate progress history", Err: err}
	}
	if err := progressRows.Close(); err != nil {
		return RecoveredTask{}, &StorageError{Operation: "close progress history", Err: err}
	}
	if err := validateRecoveredTask(model); err != nil {
		return RecoveredTask{}, &CorruptionError{TaskID: taskID, Err: err}
	}
	return model, nil
}

func validateRecoveredTask(model RecoveredTask) error {
	for index, item := range model.ContractHistory {
		if item.Version != index+1 || !schemaPresent(item.Schema) {
			return fmt.Errorf("contract history is missing, non-monotonic, or has an incomplete schema reference")
		}
	}
	for index, item := range model.ProgressHistory {
		if item.Version != index || item.ThroughTaskEventSequence < 0 || item.ThroughAgentRunEventSequence < 0 || !schemaPresent(item.Schema) {
			return fmt.Errorf("progress history is missing, non-monotonic, or has an incomplete schema reference")
		}
	}
	if len(model.ContractHistory) == 0 || !reflect.DeepEqual(model.CurrentContract, model.ContractHistory[len(model.ContractHistory)-1]) {
		return fmt.Errorf("current contract does not match immutable history")
	}
	if len(model.ProgressHistory) == 0 || !reflect.DeepEqual(model.CurrentProgress, model.ProgressHistory[len(model.ProgressHistory)-1]) {
		return fmt.Errorf("current progress does not match immutable history")
	}
	for index, event := range model.Events {
		if event.Sequence != index+1 || !schemaPresent(event.Schema) {
			return fmt.Errorf("event sequence or schema reference is invalid")
		}
	}
	return nil
}

func schemaPresent(schema SchemaReference) bool {
	return schema.ID != "" && schema.Revision != "" && schemaDigestPattern.MatchString(schema.Digest)
}
