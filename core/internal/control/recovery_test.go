package control

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecoveredTaskIsByteEquivalentAfterCloseAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	store := openTestStore(t, path)
	createVersionedTask(t, store, "TASK-reopen", "agent-reopen")
	if _, err := store.TransitionTask(context.Background(), transitionInput("TASK-reopen", StateReady, StateRunning)); err != nil {
		t.Fatal(err)
	}
	for version := 1; version <= 2; version++ {
		if _, err := store.UpdateContract(context.Background(), contractUpdate("TASK-reopen", version, "contract")); err != nil {
			t.Fatal(err)
		}
	}
	for version := 0; version <= 1; version++ {
		if _, err := store.UpdateProgress(context.Background(), progressUpdate("TASK-reopen", version, 3+version, "progress")); err != nil {
			t.Fatal(err)
		}
	}
	before, err := store.ReadRecoveredTask(context.Background(), "TASK-reopen")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.ReadRecoveredTask(context.Background(), "TASK-reopen")
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("recovered model differs: err=%v before=%#v after=%#v", err, before, after)
	}
}

func TestRecoveryFailsFastOnCurrentHistoryAndEventCorruption(t *testing.T) {
	tests := map[string]func(*testing.T, *Store){
		"contract current mismatch": func(t *testing.T, store *Store) {
			_, err := store.db.Exec(`UPDATE task_contracts SET schema_digest = ? WHERE task_id = 'TASK-corrupt'`, versionedSchema.Digest)
			if err != nil {
				t.Fatal(err)
			}
		},
		"missing contract history": func(t *testing.T, store *Store) {
			if _, err := store.db.Exec(`DELETE FROM task_contract_history WHERE task_id = 'TASK-corrupt' AND version = 1`); err != nil {
				t.Fatal(err)
			}
		},
		"invalid progress digest": func(t *testing.T, store *Store) {
			if _, err := store.db.Exec(`UPDATE task_progress SET schema_digest = 'invalid'; UPDATE task_progress_history SET schema_digest = 'invalid' WHERE task_id = 'TASK-corrupt'`); err != nil {
				t.Fatal(err)
			}
		},
		"event sequence gap": func(t *testing.T, store *Store) {
			if _, err := store.db.Exec(`DELETE FROM task_events WHERE task_id = 'TASK-corrupt' AND sequence = 1`); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
			createVersionedTask(t, store, "TASK-corrupt", "agent-corrupt")
			corrupt(t, store)
			model, err := store.ReadRecoveredTask(context.Background(), "TASK-corrupt")
			var corruption *CorruptionError
			if !errors.As(err, &corruption) || !reflect.DeepEqual(model, RecoveredTask{}) {
				t.Fatalf("read = %#v, error = %T %v, want empty/CorruptionError", model, err, err)
			}
		})
	}
}

func TestVersionHistoryPrimaryKeysRejectDuplicates(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	createVersionedTask(t, store, "TASK-duplicate", "agent-duplicate-history")
	for _, statement := range []string{
		`INSERT INTO task_contract_history SELECT * FROM task_contract_history WHERE task_id = 'TASK-duplicate' AND version = 1`,
		`INSERT INTO task_progress_history SELECT * FROM task_progress_history WHERE task_id = 'TASK-duplicate' AND version = 0`,
	} {
		if _, err := store.db.Exec(statement); err == nil {
			t.Fatalf("duplicate history succeeded: %s", statement)
		}
	}
}

func seedV2Store(t *testing.T, path string, extra string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	fixture := migrationV1 + migrationV2 + `
CREATE TABLE schema_version (version INTEGER NOT NULL);
INSERT INTO schema_version VALUES (2);
INSERT INTO tasks(task_id) VALUES ('TASK-v2');
INSERT INTO task_owners(task_id, owner_agent_id) VALUES ('TASK-v2', 'agent-v2');
INSERT INTO task_workspaces(task_id, workspace_ref) VALUES ('TASK-v2', 'workspace-v2');
INSERT INTO task_contracts(task_id, version, schema_id, schema_revision, schema_digest, payload) VALUES ('TASK-v2', 1, 'contract-v2', '1', 'sha256:1111111111111111111111111111111111111111111111111111111111111111', '{"v":1}');
INSERT INTO task_progress(task_id, version) VALUES ('TASK-v2', 0);
INSERT INTO task_events(task_id, sequence, event_type, payload) VALUES ('TASK-v2', 1, 'TaskCreated', '{}');` + extra
	if _, err := db.Exec(fixture); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestV3MigrationPreservesV2DataAndForeignKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	seedV2Store(t, path, "")
	store := openTestStore(t, path)
	model, err := store.ReadRecoveredTask(context.Background(), "TASK-v2")
	if err != nil {
		t.Fatal(err)
	}
	if model.State != StateReady || model.OwnerAgentID != "agent-v2" || model.WorkspaceRef != "workspace-v2" || len(model.ContractHistory) != 1 || len(model.ProgressHistory) != 1 {
		t.Fatalf("migrated model = %#v", model)
	}
	if _, err := store.UpdateContract(context.Background(), contractUpdate("TASK-v2", 1, "migrated")); err != nil {
		t.Fatalf("migrated contract is not updateable: %v", err)
	}
	if _, err := store.db.Exec(`DELETE FROM tasks WHERE task_id = 'TASK-v2'`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"task_contracts", "task_contract_history", "task_progress", "task_progress_history", "task_events"} {
		var count int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE task_id = 'TASK-v2'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("cascade %s count=%d err=%v", table, count, err)
		}
	}
}

func TestV3MigrationFailureRollsBackTableRebuild(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	seedV2Store(t, path, `CREATE TABLE task_contract_history(blocker INTEGER);`)
	_, err := OpenStore(path)
	var storageErr *StorageError
	if !errors.As(err, &storageErr) {
		t.Fatalf("OpenStore = %T %v, want StorageError", err, err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version, taskRows, renamedTables int
	if err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_contracts WHERE task_id = 'TASK-v2'`).Scan(&taskRows); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('task_contracts_v2', 'task_progress_v2')`).Scan(&renamedTables); err != nil {
		t.Fatal(err)
	}
	if version != 2 || taskRows != 1 || renamedTables != 0 {
		t.Fatalf("partial migration: version=%d taskRows=%d renamed=%d", version, taskRows, renamedTables)
	}
}
