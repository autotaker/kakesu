package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type TaskState string

const (
	StateReady               TaskState = "ready"
	StateRunning             TaskState = "running"
	StateWaiting             TaskState = "waiting"
	StateSuspended           TaskState = "suspended"
	StateReviewingCompletion TaskState = "reviewing_completion"
	StateCompleted           TaskState = "completed"
	StateCancelled           TaskState = "cancelled"
)

var allowedTransitions = map[TaskState]map[TaskState]string{
	StateReady:               {StateRunning: "TaskStarted", StateCancelled: "TaskCancelled"},
	StateRunning:             {StateWaiting: "TaskWaiting", StateSuspended: "TaskSuspended", StateReviewingCompletion: "TaskReviewingCompletion", StateCancelled: "TaskCancelled"},
	StateWaiting:             {StateRunning: "TaskResumed", StateCancelled: "TaskCancelled"},
	StateSuspended:           {StateRunning: "TaskResumed", StateCancelled: "TaskCancelled"},
	StateReviewingCompletion: {StateRunning: "TaskResumed", StateCompleted: "TaskCompleted", StateCancelled: "TaskCancelled"},
}

type TransitionError struct {
	Current TaskState
	Target  TaskState
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("control transition %q -> %q is not allowed", e.Current, e.Target)
}

type TransitionInput struct {
	TaskID   string
	Expected TaskState
	Target   TaskState
	Payload  []byte
}

type EventPayloadError struct {
	EventType string
	Err       error
}

func (e *EventPayloadError) Error() string {
	return fmt.Sprintf("control event %q payload: %v", e.EventType, e.Err)
}
func (e *EventPayloadError) Unwrap() error { return e.Err }

type LifecycleReadModel struct {
	TaskID          string
	State           TaskState
	OwnerAgentID    string
	OwnerActive     bool
	ProgressVersion int
	Events          []TaskEvent
}

func (s *Store) TransitionTask(ctx context.Context, input TransitionInput) (LifecycleReadModel, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return LifecycleReadModel{}, &StorageError{Operation: "begin transition", Err: err}
	}
	defer tx.Rollback()
	var current TaskState
	if err := tx.QueryRowContext(ctx, `SELECT state FROM tasks WHERE task_id = ?`, input.TaskID).Scan(&current); err != nil {
		return LifecycleReadModel{}, &StorageError{Operation: "read task state", Err: err}
	}
	if current != input.Expected {
		return LifecycleReadModel{}, &ConflictError{TaskID: input.TaskID, Err: fmt.Errorf("expected state %q, found %q", input.Expected, current)}
	}
	eventType, ok := allowedTransitions[input.Expected][input.Target]
	if !ok {
		return LifecycleReadModel{}, &TransitionError{Current: input.Expected, Target: input.Target}
	}
	if err := validateEventPayload(eventType, input.Payload); err != nil {
		return LifecycleReadModel{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE tasks SET state = ? WHERE task_id = ? AND state = ?`, input.Target, input.TaskID, input.Expected)
	if err != nil {
		return LifecycleReadModel{}, classifyTransitionError(input.TaskID, "update task state", err)
	}
	if changed, err := result.RowsAffected(); err != nil || changed != 1 {
		if err == nil {
			err = fmt.Errorf("expected one updated row, got %d", changed)
		}
		return LifecycleReadModel{}, &ConflictError{TaskID: input.TaskID, Err: err}
	}
	if err := s.runLifecycleHook(tx, "state"); err != nil {
		return LifecycleReadModel{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_events(task_id, sequence, event_type, payload)
		SELECT ?, COALESCE(MAX(sequence), 0) + 1, ?, ? FROM task_events WHERE task_id = ?`, input.TaskID, eventType, input.Payload, input.TaskID); err != nil {
		return LifecycleReadModel{}, classifyTransitionError(input.TaskID, "append task event", err)
	}
	if err := s.runLifecycleHook(tx, "event"); err != nil {
		return LifecycleReadModel{}, err
	}
	if input.Target == StateCompleted || input.Target == StateCancelled {
		result, err := tx.ExecContext(ctx, `UPDATE task_owners SET released_at = CURRENT_TIMESTAMP WHERE task_id = ? AND released_at IS NULL`, input.TaskID)
		if err != nil {
			return LifecycleReadModel{}, classifyTransitionError(input.TaskID, "release owner", err)
		}
		if changed, err := result.RowsAffected(); err != nil || changed != 1 {
			if err == nil {
				err = fmt.Errorf("expected one released owner, got %d", changed)
			}
			return LifecycleReadModel{}, &ConflictError{TaskID: input.TaskID, Err: err}
		}
		if err := s.runLifecycleHook(tx, "owner"); err != nil {
			return LifecycleReadModel{}, err
		}
	}
	model, err := readLifecycle(ctx, tx, input.TaskID)
	if err != nil {
		return LifecycleReadModel{}, err
	}
	if err := tx.Commit(); err != nil {
		return LifecycleReadModel{}, classifyTransitionError(input.TaskID, "commit transition", err)
	}
	return model, nil
}

func (s *Store) runLifecycleHook(tx *sql.Tx, stage string) error {
	if s.lifecycleHook == nil {
		return nil
	}
	if err := s.lifecycleHook(tx, stage); err != nil {
		return &StorageError{Operation: "transition hook " + stage, Err: err}
	}
	return nil
}

func classifyTransitionError(taskID, operation string, err error) error {
	if conflict := classifyCreateError(taskID, err); conflict != nil {
		if _, ok := conflict.(*ConflictError); ok {
			return conflict
		}
	}
	return &StorageError{Operation: operation, Err: err}
}

func (s *Store) ReadLifecycle(ctx context.Context, taskID string) (LifecycleReadModel, error) {
	return readLifecycle(ctx, s.db, taskID)
}

func readLifecycle(ctx context.Context, reader creationReader, taskID string) (LifecycleReadModel, error) {
	model := LifecycleReadModel{TaskID: taskID}
	var releasedAt sql.NullString
	err := reader.QueryRowContext(ctx, `SELECT t.state, o.owner_agent_id, o.released_at, p.version
		FROM tasks t JOIN task_owners o USING (task_id) JOIN task_progress p USING (task_id)
		WHERE t.task_id = ?`, taskID).Scan(&model.State, &model.OwnerAgentID, &releasedAt, &model.ProgressVersion)
	if err != nil {
		return LifecycleReadModel{}, &StorageError{Operation: "read lifecycle", Err: err}
	}
	model.OwnerActive = !releasedAt.Valid
	rows, err := reader.QueryContext(ctx, `SELECT sequence, event_type, payload FROM task_events WHERE task_id = ? ORDER BY sequence`, taskID)
	if err != nil {
		return LifecycleReadModel{}, &StorageError{Operation: "read lifecycle events", Err: err}
	}
	defer rows.Close()
	for rows.Next() {
		var event TaskEvent
		if err := rows.Scan(&event.Sequence, &event.Type, &event.Payload); err != nil {
			return LifecycleReadModel{}, &StorageError{Operation: "scan lifecycle events", Err: err}
		}
		model.Events = append(model.Events, event)
	}
	if err := rows.Err(); err != nil {
		return LifecycleReadModel{}, &StorageError{Operation: "read lifecycle events", Err: err}
	}
	return model, nil
}

func validateEventPayload(eventType string, payload []byte) error {
	var fields map[string]json.RawMessage
	if len(payload) == 0 || json.Unmarshal(payload, &fields) != nil || fields == nil {
		return &EventPayloadError{EventType: eventType, Err: fmt.Errorf("must be a JSON object")}
	}
	required := map[string]string{
		"TaskStarted": "run_id", "TaskResumed": "run_id",
		"TaskCompleted": "outcome_ref", "TaskCancelled": "reason",
	}[eventType]
	if required == "" {
		return nil
	}
	var value string
	if json.Unmarshal(fields[required], &value) != nil || value == "" {
		return &EventPayloadError{EventType: eventType, Err: fmt.Errorf("%s must be a non-empty string", required)}
	}
	return nil
}
