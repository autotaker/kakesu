package control

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
)

const (
	schemaVersion = 2
	migrationV1   = `
CREATE TABLE tasks (task_id TEXT PRIMARY KEY, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE task_owners (task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE CASCADE, owner_agent_id TEXT NOT NULL);
CREATE TABLE task_workspaces (task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE CASCADE, workspace_ref TEXT NOT NULL);
CREATE TABLE task_contracts (task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE CASCADE, version INTEGER NOT NULL CHECK (version = 1), schema_id TEXT NOT NULL, schema_revision TEXT NOT NULL, schema_digest TEXT NOT NULL, payload BLOB NOT NULL);
CREATE TABLE task_progress (task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE CASCADE, version INTEGER NOT NULL CHECK (version = 0));
CREATE TABLE task_events (task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE, sequence INTEGER NOT NULL, event_type TEXT NOT NULL, PRIMARY KEY (task_id, sequence));`
	migrationV2 = `
ALTER TABLE tasks ADD COLUMN state TEXT NOT NULL DEFAULT 'ready' CHECK (state IN ('ready','running','waiting','suspended','reviewing_completion','completed','cancelled'));
ALTER TABLE task_owners ADD COLUMN released_at TEXT;
ALTER TABLE task_events ADD COLUMN payload BLOB NOT NULL DEFAULT '{}';
CREATE UNIQUE INDEX one_active_task_per_agent ON task_owners(owner_agent_id) WHERE released_at IS NULL;`
	migrationSQL = migrationV1 + migrationV2
)

var requiredPragmas = []string{
	"PRAGMA journal_mode = WAL",
	"PRAGMA foreign_keys = ON",
	"PRAGMA busy_timeout = 5000",
}

type StorageError struct {
	Operation string
	Err       error
}

func (e *StorageError) Error() string { return fmt.Sprintf("control store %s: %v", e.Operation, e.Err) }
func (e *StorageError) Unwrap() error { return e.Err }

type ConflictError struct {
	TaskID string
	Err    error
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("control task %q conflicts: %v", e.TaskID, e.Err)
}
func (e *ConflictError) Unwrap() error { return e.Err }

type ContractSnapshot struct {
	SchemaID       string
	SchemaRevision string
	SchemaDigest   string
	JSON           []byte
}

type CreateTaskInput struct {
	TaskID       string
	OwnerAgentID string
	WorkspaceRef string
	Contract     ContractSnapshot
}

type TaskEvent struct {
	Sequence int
	Type     string
	Payload  []byte
}

type CreationReadModel struct {
	TaskID          string
	CreatedAt       string
	OwnerAgentID    string
	WorkspaceRef    string
	Contract        ContractSnapshot
	ContractVersion int
	ProgressVersion int
	Events          []TaskEvent
}

type Store struct {
	db            *sql.DB
	beforeCommit  func(*sql.Tx) error
	afterCommit   func()
	lifecycleHook func(*sql.Tx, string) error
}

func OpenStore(path string) (*Store, error) {
	return openStore(path, requiredPragmas, migrationSQL)
}

func openStore(path string, pragmas []string, migration string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, &StorageError{Operation: "open", Err: err}
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.initialize(pragmas, migration); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return &StorageError{Operation: "close", Err: err}
	}
	return nil
}

func (s *Store) initialize(pragmas []string, migration string) error {
	for _, statement := range pragmas {
		if _, err := s.db.Exec(statement); err != nil {
			return &StorageError{Operation: "pragma", Err: err}
		}
	}
	return s.migrate(migration)
}

func (s *Store) migrate(migration string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return &StorageError{Operation: "begin migration", Err: err}
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return &StorageError{Operation: "create schema version", Err: err}
	}
	var count, minimum, maximum int
	err = tx.QueryRow(`SELECT COUNT(*), COALESCE(MIN(version), 0), COALESCE(MAX(version), 0) FROM schema_version`).Scan(&count, &minimum, &maximum)
	switch {
	case err != nil:
		return &StorageError{Operation: "read schema version", Err: err}
	case count == 0:
		if _, err := tx.Exec(migration); err != nil {
			return &StorageError{Operation: "migration", Err: err}
		}
		if _, err := tx.Exec(`INSERT INTO schema_version(version) VALUES (?)`, schemaVersion); err != nil {
			return &StorageError{Operation: "record migration", Err: err}
		}
	case count == 1 && minimum == 1 && maximum == 1:
		if _, err := tx.Exec(migrationV2); err != nil {
			return &StorageError{Operation: "migration v2", Err: err}
		}
		if _, err := tx.Exec(`UPDATE schema_version SET version = ?`, schemaVersion); err != nil {
			return &StorageError{Operation: "record migration v2", Err: err}
		}
	case count != 1 || minimum != schemaVersion || maximum != schemaVersion:
		return &StorageError{Operation: "schema version", Err: fmt.Errorf("unsupported version set count=%d min=%d max=%d", count, minimum, maximum)}
	}
	if err := tx.Commit(); err != nil {
		return &StorageError{Operation: "commit migration", Err: err}
	}
	return nil
}

func (s *Store) CreateTask(ctx context.Context, input CreateTaskInput) (CreationReadModel, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CreationReadModel{}, &StorageError{Operation: "begin create", Err: err}
	}
	defer tx.Rollback()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO tasks(task_id) VALUES (?)`, []any{input.TaskID}},
		{`INSERT INTO task_owners(task_id, owner_agent_id) VALUES (?, ?)`, []any{input.TaskID, input.OwnerAgentID}},
		{`INSERT INTO task_workspaces(task_id, workspace_ref) VALUES (?, ?)`, []any{input.TaskID, input.WorkspaceRef}},
		{`INSERT INTO task_contracts(task_id, version, schema_id, schema_revision, schema_digest, payload) VALUES (?, 1, ?, ?, ?, ?)`, []any{input.TaskID, input.Contract.SchemaID, input.Contract.SchemaRevision, input.Contract.SchemaDigest, input.Contract.JSON}},
		{`INSERT INTO task_progress(task_id, version) VALUES (?, 0)`, []any{input.TaskID}},
		{`INSERT INTO task_events(task_id, sequence, event_type) VALUES (?, 1, 'TaskCreated'), (?, 2, 'OwnerAssigned')`, []any{input.TaskID, input.TaskID}},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return CreationReadModel{}, classifyCreateError(input.TaskID, err)
		}
	}
	if s.beforeCommit != nil {
		if err := s.beforeCommit(tx); err != nil {
			return CreationReadModel{}, &StorageError{Operation: "create hook", Err: err}
		}
	}
	model, err := readCreation(ctx, tx, input.TaskID)
	if err != nil {
		return CreationReadModel{}, err
	}
	if err := tx.Commit(); err != nil {
		return CreationReadModel{}, classifyCreateError(input.TaskID, err)
	}
	if s.afterCommit != nil {
		s.afterCommit()
	}
	return model, nil
}

func classifyCreateError(taskID string, err error) error {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case 5, 6, 19: // SQLITE_BUSY, SQLITE_LOCKED, SQLITE_CONSTRAINT
			return &ConflictError{TaskID: taskID, Err: err}
		}
	}
	return &StorageError{Operation: "create", Err: err}
}

func (s *Store) ReadCreation(ctx context.Context, taskID string) (CreationReadModel, error) {
	return readCreation(ctx, s.db, taskID)
}

type creationReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readCreation(ctx context.Context, reader creationReader, taskID string) (CreationReadModel, error) {
	var model CreationReadModel
	model.TaskID = taskID
	err := reader.QueryRowContext(ctx, `SELECT t.created_at, o.owner_agent_id, w.workspace_ref,
		c.version, c.schema_id, c.schema_revision, c.schema_digest, c.payload, p.version
		FROM tasks t JOIN task_owners o USING (task_id) JOIN task_workspaces w USING (task_id)
		JOIN task_contracts c USING (task_id) JOIN task_progress p USING (task_id) WHERE t.task_id = ?`, taskID).Scan(
		&model.CreatedAt, &model.OwnerAgentID, &model.WorkspaceRef,
		&model.ContractVersion, &model.Contract.SchemaID, &model.Contract.SchemaRevision,
		&model.Contract.SchemaDigest, &model.Contract.JSON, &model.ProgressVersion,
	)
	if err != nil {
		return CreationReadModel{}, &StorageError{Operation: "read creation", Err: err}
	}
	rows, err := reader.QueryContext(ctx, `SELECT sequence, event_type, payload FROM task_events WHERE task_id = ? ORDER BY sequence`, taskID)
	if err != nil {
		return CreationReadModel{}, &StorageError{Operation: "read events", Err: err}
	}
	defer rows.Close()
	for rows.Next() {
		var event TaskEvent
		if err := rows.Scan(&event.Sequence, &event.Type, &event.Payload); err != nil {
			return CreationReadModel{}, &StorageError{Operation: "scan events", Err: err}
		}
		model.Events = append(model.Events, event)
	}
	if err := rows.Err(); err != nil {
		return CreationReadModel{}, &StorageError{Operation: "read events", Err: err}
	}
	return model, nil
}
