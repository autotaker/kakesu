package control

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func testInput() CreateTaskInput {
	return CreateTaskInput{
		TaskID:       "TASK-1000",
		OwnerAgentID: "agent-1",
		WorkspaceRef: "workspace-1",
		Contract: ContractSnapshot{
			SchemaID:       "task-contract",
			SchemaRevision: "1",
			SchemaDigest:   "sha256:fixture",
			JSON:           []byte(`{"goal":"durable"}`),
		},
	}
}

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return store
}

func TestStoreMigrationPragmasAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	store := openTestStore(t, path)

	var version, foreignKeys, busyTimeout int
	var journal string
	if err := store.db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion || journal != "wal" || foreignKeys != 1 || busyTimeout != 5000 {
		t.Fatalf("version/pragmas = %d %q %d %d", version, journal, foreignKeys, busyTimeout)
	}

	want, err := store.CreateTask(context.Background(), testInput())
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := store.ReadCreation(context.Background(), testInput().TaskID)
	if err != nil {
		t.Fatalf("ReadCreation: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reopened model mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
	var migrations int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&migrations); err != nil || migrations != 1 {
		t.Fatalf("migration rows = %d, err = %v", migrations, err)
	}
}

func TestStoreMigratesV1AndEnforcesActiveOwnerUniqueness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(migrationV1 + `
CREATE TABLE schema_version (version INTEGER NOT NULL);
INSERT INTO schema_version(version) VALUES (1);`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store := openTestStore(t, path)
	var version int
	if err := store.db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil || version != schemaVersion {
		t.Fatalf("version = %d, err = %v", version, err)
	}
	first := testInput()
	if _, err := store.CreateTask(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := testInput()
	second.TaskID = "TASK-1001"
	_, err = store.CreateTask(context.Background(), second)
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("second active task = %T %v, want ConflictError", err, err)
	}
}

func TestStoreV2MigrationRejectsDuplicateActiveOwnersAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	fixture := migrationV1 + `
CREATE TABLE schema_version (version INTEGER NOT NULL);
INSERT INTO schema_version(version) VALUES (1);
INSERT INTO tasks(task_id) VALUES ('TASK-old-a'), ('TASK-old-b');
INSERT INTO task_owners(task_id, owner_agent_id) VALUES ('TASK-old-a', 'agent-duplicate'), ('TASK-old-b', 'agent-duplicate');`
	if _, err := db.Exec(fixture); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = OpenStore(path)
	var storageErr *StorageError
	if !errors.As(err, &storageErr) {
		t.Fatalf("OpenStore = %T %v, want StorageError", err, err)
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version, stateColumns, releasedColumns, indexes int
	if err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'state'`).Scan(&stateColumns); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('task_owners') WHERE name = 'released_at'`).Scan(&releasedColumns); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'one_active_task_per_agent'`).Scan(&indexes); err != nil {
		t.Fatal(err)
	}
	if version != 1 || stateColumns != 0 || releasedColumns != 0 || indexes != 0 {
		t.Fatalf("partial migration: version=%d state=%d released=%d indexes=%d", version, stateColumns, releasedColumns, indexes)
	}
}

func TestCreateTaskIsAtomicAndTyped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	store := openTestStore(t, path)
	input := testInput()
	store.beforeCommit = func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO missing_table(value) VALUES (1)`)
		return err
	}

	_, err := store.CreateTask(context.Background(), input)
	var storageErr *StorageError
	if !errors.As(err, &storageErr) {
		t.Fatalf("forced failure = %T %v, want StorageError", err, err)
	}
	assertTableCounts(t, store.db, 0)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	assertTableCounts(t, store.db, 0)

	store.beforeCommit = nil
	model, err := store.CreateTask(context.Background(), input)
	if err != nil {
		t.Fatalf("retry CreateTask: %v", err)
	}
	assertTableCounts(t, store.db, 1)
	if model.TaskID != input.TaskID || model.OwnerAgentID != input.OwnerAgentID || model.WorkspaceRef != input.WorkspaceRef {
		t.Fatalf("identity projection = %#v", model)
	}
	if model.ContractVersion != 1 || model.ProgressVersion != 0 {
		t.Fatalf("versions = contract %d, progress %d", model.ContractVersion, model.ProgressVersion)
	}
	if !reflect.DeepEqual(model.Contract, input.Contract) {
		t.Fatalf("contract = %#v, want %#v", model.Contract, input.Contract)
	}
	wantEvents := []TaskEvent{{Sequence: 1, Type: "TaskCreated"}, {Sequence: 2, Type: "OwnerAssigned"}}
	if !reflect.DeepEqual(model.Events, wantEvents) {
		t.Fatalf("events = %#v, want %#v", model.Events, wantEvents)
	}
	_, err = store.CreateTask(context.Background(), input)
	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("duplicate = %T %v, want ConflictError", err, err)
	}
}

func TestOpenStoreRejectsUnknownVersion(t *testing.T) {
	for name, versions := range map[string][]int{
		"unknown":           {99},
		"mixed":             {1, 2},
		"duplicate current": {1, 1},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "control.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL)`); err != nil {
				t.Fatal(err)
			}
			for _, version := range versions {
				if _, err := db.Exec(`INSERT INTO schema_version VALUES (?)`, version); err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			_, err = OpenStore(path)
			var storageErr *StorageError
			if !errors.As(err, &storageErr) {
				t.Fatalf("OpenStore versions %v = %T %v, want StorageError", versions, err, err)
			}
		})
	}
}

func TestCreateTaskDoesNotReadAfterCommit(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	ctx, cancel := context.WithCancel(context.Background())
	store.afterCommit = cancel
	model, err := store.CreateTask(ctx, testInput())
	if err != nil {
		t.Fatalf("CreateTask after commit cancellation: %v", err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) || model.TaskID != testInput().TaskID {
		t.Fatalf("context/model = %v %#v", ctx.Err(), model)
	}
	assertTableCounts(t, store.db, 1)
}

func TestOpenStoreFailsClosedOnInitializationFailure(t *testing.T) {
	tests := map[string]struct {
		pragmas   []string
		migration string
	}{
		"required pragma": {
			pragmas:   append(append([]string{}, requiredPragmas...), `SELECT * FROM missing_pragma_table`),
			migration: migrationSQL,
		},
		"migration SQL": {
			pragmas:   requiredPragmas,
			migration: migrationSQL + `; CREATE TABLE broken (`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "control.db")
			_, err := openStore(path, test.pragmas, test.migration)
			var storageErr *StorageError
			if !errors.As(err, &storageErr) {
				t.Fatalf("open failure = %T %v, want StorageError", err, err)
			}
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			var tables int
			if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('tasks', 'schema_version')`).Scan(&tables); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if tables != 0 {
				t.Fatalf("partial schema tables = %d", tables)
			}
			store, err := OpenStore(path)
			if err != nil {
				t.Fatalf("clean reopen: %v", err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func assertTableCounts(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	for _, table := range []string{"tasks", "task_owners", "task_workspaces", "task_contracts", "task_progress"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != expected {
			t.Fatalf("%s count = %d, want %d", table, count, expected)
		}
	}
	var events int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != expected*2 {
		t.Fatalf("task_events count = %d, want %d", events, expected*2)
	}
}
