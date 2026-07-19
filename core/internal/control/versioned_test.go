package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

var versionedSchema = SchemaReference{
	ID: "urn:kakesu:test:versioned", Revision: "1",
	Digest: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
}

func createVersionedTask(t *testing.T, store *Store, taskID, owner string) {
	t.Helper()
	if _, err := store.CreateTask(context.Background(), lifecycleInput(taskID, owner)); err != nil {
		t.Fatal(err)
	}
}

func contractUpdate(taskID string, expected int, value string) ContractUpdateInput {
	return ContractUpdateInput{TaskID: taskID, ExpectedVersion: expected, New: ContractVersion{
		Version: expected + 1, Schema: versionedSchema, Payload: []byte(fmt.Sprintf(`{"contract":%q}`, value)),
	}}
}

func progressUpdate(taskID string, expected, through int, value string) ProgressUpdateInput {
	return ProgressUpdateInput{TaskID: taskID, ExpectedVersion: expected, New: ProgressVersion{
		Version: expected + 1, Schema: versionedSchema, Payload: []byte(fmt.Sprintf(`{"progress":%q}`, value)),
		ThroughTaskEventSequence: through, ThroughAgentRunEventSequence: through + 100,
	}}
}

func TestVersionedUpdatesAppendImmutableHistoryAndEvents(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	createVersionedTask(t, store, "TASK-versioned", "agent-versioned")
	initial, err := store.ReadRecoveredTask(context.Background(), "TASK-versioned")
	if err != nil {
		t.Fatal(err)
	}
	contractResult, err := store.UpdateContract(context.Background(), contractUpdate("TASK-versioned", 1, "v2"))
	if err != nil {
		t.Fatal(err)
	}
	progressResult, err := store.UpdateProgress(context.Background(), progressUpdate("TASK-versioned", 0, contractResult.Event.Sequence, "v1"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := store.ReadRecoveredTask(context.Background(), "TASK-versioned")
	if err != nil {
		t.Fatal(err)
	}
	if model.CurrentContract.Version != 2 || model.CurrentProgress.Version != 1 || len(model.ContractHistory) != 2 || len(model.ProgressHistory) != 2 {
		t.Fatalf("versioned model = %#v", model)
	}
	if !reflect.DeepEqual(model.ContractHistory[0], initial.CurrentContract) || !reflect.DeepEqual(model.ProgressHistory[0], initial.CurrentProgress) {
		t.Fatalf("initial history changed: before=%#v after=%#v", initial, model)
	}
	if contractResult.Event.Type != "ContractChanged" || progressResult.Event.Type != "ProgressRefreshed" || progressResult.Event.Sequence != contractResult.Event.Sequence+1 {
		t.Fatalf("events = %#v %#v", contractResult.Event, progressResult.Event)
	}
	if !reflect.DeepEqual(contractResult.Event.Schema, versionedSchema) || !reflect.DeepEqual(progressResult.Event.Schema, versionedSchema) {
		t.Fatalf("event schema refs = %#v %#v", contractResult.Event.Schema, progressResult.Event.Schema)
	}
	var contractPayload, progressPayload map[string]any
	if json.Unmarshal(contractResult.Event.Payload, &contractPayload) != nil || contractPayload["version"] != float64(2) {
		t.Fatalf("contract event payload = %s", contractResult.Event.Payload)
	}
	if json.Unmarshal(progressResult.Event.Payload, &progressPayload) != nil || progressPayload["progress_version"] != float64(1) || progressPayload["through_task_event_sequence"] != float64(contractResult.Event.Sequence) || progressPayload["through_agent_run_event_sequence"] != float64(contractResult.Event.Sequence+100) {
		t.Fatalf("progress event payload = %s", progressResult.Event.Payload)
	}
	if model.State != StateReady || !model.OwnerActive || model.OwnerAgentID != "agent-versioned" || model.WorkspaceRef != testInput().WorkspaceRef {
		t.Fatalf("TASK-0028 fields changed: %#v", model)
	}
}

func TestVersionedUpdatesRejectVersionAndInputBoundariesWithoutMutation(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	createVersionedTask(t, store, "TASK-reject", "agent-reject")
	if _, err := store.UpdateContract(context.Background(), contractUpdate("TASK-reject", 1, "v2")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateProgress(context.Background(), progressUpdate("TASK-reject", 0, 2, "v1")); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadRecoveredTask(context.Background(), "TASK-reject")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]ContractUpdateInput{
		"stale expected":           contractUpdate("TASK-reject", 1, "overwrite-v2"),
		"future expected":          contractUpdate("TASK-reject", 3, "v4"),
		"skipped version":          {TaskID: "TASK-reject", ExpectedVersion: 2, New: ContractVersion{Version: 4, Schema: versionedSchema, Payload: []byte(`{}`)}},
		"same version replacement": {TaskID: "TASK-reject", ExpectedVersion: 2, New: ContractVersion{Version: 2, Schema: versionedSchema, Payload: []byte(`{"different":true}`)}},
		"bad schema digest":        {TaskID: "TASK-reject", ExpectedVersion: 2, New: ContractVersion{Version: 3, Schema: SchemaReference{ID: "schema", Revision: "1", Digest: "bad"}, Payload: []byte(`{}`)}},
		"invalid payload":          {TaskID: "TASK-reject", ExpectedVersion: 2, New: ContractVersion{Version: 3, Schema: versionedSchema, Payload: []byte(`[]`)}},
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := store.UpdateContract(context.Background(), input)
			var conflict *ConflictError
			var invalid *VersionedInputError
			if !errors.As(err, &conflict) && !errors.As(err, &invalid) {
				t.Fatalf("error = %T %v, want typed rejection", err, err)
			}
			after, err := store.ReadRecoveredTask(context.Background(), "TASK-reject")
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("rejection mutated model: err=%v before=%#v after=%#v", err, before, after)
			}
		})
	}
	progressTests := map[string]ProgressUpdateInput{
		"stale expected":           progressUpdate("TASK-reject", 0, 2, "stale"),
		"future expected":          progressUpdate("TASK-reject", 2, 2, "future"),
		"skipped version":          {TaskID: "TASK-reject", ExpectedVersion: 1, New: ProgressVersion{Version: 3, Schema: versionedSchema, Payload: []byte(`{}`), ThroughTaskEventSequence: 2}},
		"same version replacement": {TaskID: "TASK-reject", ExpectedVersion: 1, New: ProgressVersion{Version: 1, Schema: versionedSchema, Payload: []byte(`{}`), ThroughTaskEventSequence: 2}},
		"invalid schema":           {TaskID: "TASK-reject", ExpectedVersion: 1, New: ProgressVersion{Version: 2, Schema: SchemaReference{}, Payload: []byte(`{}`), ThroughTaskEventSequence: 2}},
		"invalid payload":          {TaskID: "TASK-reject", ExpectedVersion: 1, New: ProgressVersion{Version: 2, Schema: versionedSchema, Payload: []byte(`null`), ThroughTaskEventSequence: 2}},
		"negative watermark":       progressUpdate("TASK-reject", 1, -1, "invalid"),
		"negative agent watermark": {TaskID: "TASK-reject", ExpectedVersion: 1, New: ProgressVersion{Version: 2, Schema: versionedSchema, Payload: []byte(`{}`), ThroughTaskEventSequence: 2, ThroughAgentRunEventSequence: -1}},
	}
	for name, input := range progressTests {
		t.Run("progress "+name, func(t *testing.T) {
			_, err := store.UpdateProgress(context.Background(), input)
			var conflict *ConflictError
			var invalid *VersionedInputError
			if !errors.As(err, &conflict) && !errors.As(err, &invalid) {
				t.Fatalf("error = %T %v, want typed rejection", err, err)
			}
			after, err := store.ReadRecoveredTask(context.Background(), "TASK-reject")
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("progress rejection mutated model: err=%v before=%#v after=%#v", err, before, after)
			}
		})
	}
}

func TestProgressUpdateRejectsRegressingAndFutureWatermarksWithoutMutation(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	createVersionedTask(t, store, "TASK-watermark", "agent-watermark")
	if _, err := store.UpdateProgress(context.Background(), progressUpdate("TASK-watermark", 0, 2, "v1")); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadRecoveredTask(context.Background(), "TASK-watermark")
	if err != nil {
		t.Fatal(err)
	}
	base := ProgressVersion{Version: 2, Schema: versionedSchema, Payload: []byte(`{"progress":"v2"}`), ThroughTaskEventSequence: 2, ThroughAgentRunEventSequence: 102}
	taskRegression := base
	taskRegression.ThroughTaskEventSequence = 1
	agentRegression := base
	agentRegression.ThroughAgentRunEventSequence = 101
	futureTask := base
	futureTask.ThroughTaskEventSequence = before.Events[len(before.Events)-1].Sequence + 1
	tests := map[string]ProgressVersion{
		"task watermark regression":  taskRegression,
		"agent watermark regression": agentRegression,
		"task watermark in future":   futureTask,
	}
	for name, progress := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := store.UpdateProgress(context.Background(), ProgressUpdateInput{TaskID: "TASK-watermark", ExpectedVersion: 1, New: progress})
			var conflict *ConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("error = %T %v, want ConflictError", err, err)
			}
			after, err := store.ReadRecoveredTask(context.Background(), "TASK-watermark")
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("watermark rejection mutated model: err=%v before=%#v after=%#v", err, before, after)
			}
		})
	}
}

func TestVersionedUpdatesRollbackAtEveryStageWithoutGap(t *testing.T) {
	for _, kind := range []string{"contract", "progress"} {
		for _, stage := range []string{"history", "current", "event"} {
			t.Run(kind+"_"+stage, func(t *testing.T) {
				store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
				createVersionedTask(t, store, "TASK-rollback", "agent-rollback")
				before, err := store.ReadRecoveredTask(context.Background(), "TASK-rollback")
				if err != nil {
					t.Fatal(err)
				}
				store.versionedHook = func(tx *sql.Tx, gotKind, gotStage string) error {
					if gotKind != kind || gotStage != stage {
						return nil
					}
					_, err := tx.Exec(`INSERT INTO missing_versioned_table(value) VALUES (1)`)
					return err
				}
				if kind == "contract" {
					_, err = store.UpdateContract(context.Background(), contractUpdate("TASK-rollback", 1, "v2"))
				} else {
					_, err = store.UpdateProgress(context.Background(), progressUpdate("TASK-rollback", 0, 2, "v1"))
				}
				var storageErr *StorageError
				if !errors.As(err, &storageErr) {
					t.Fatalf("forced failure = %T %v", err, err)
				}
				after, err := store.ReadRecoveredTask(context.Background(), "TASK-rollback")
				if err != nil || !reflect.DeepEqual(after, before) {
					t.Fatalf("rollback mismatch: err=%v before=%#v after=%#v", err, before, after)
				}
				store.versionedHook = nil
				if kind == "contract" {
					_, err = store.UpdateContract(context.Background(), contractUpdate("TASK-rollback", 1, "v2"))
				} else {
					_, err = store.UpdateProgress(context.Background(), progressUpdate("TASK-rollback", 0, 2, "v1"))
				}
				if err != nil {
					t.Fatalf("retry without hook: %v", err)
				}
				succeeded, err := store.ReadRecoveredTask(context.Background(), "TASK-rollback")
				if err != nil || succeeded.Events[len(succeeded.Events)-1].Sequence != before.Events[len(before.Events)-1].Sequence+1 {
					t.Fatalf("retry gap/model: %#v err=%v", succeeded, err)
				}
			})
		}
	}
}

func TestConcurrentVersionedUpdatesHaveExactlyOneWinner(t *testing.T) {
	for _, kind := range []string{"contract", "progress"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "control.db")
			stores := []*Store{openTestStore(t, path), openTestStore(t, path)}
			createVersionedTask(t, stores[0], "TASK-race", "agent-race-version")
			start := make(chan struct{})
			errs := make([]error, 2)
			var wait sync.WaitGroup
			for index := range stores {
				wait.Add(1)
				go func(index int) {
					defer wait.Done()
					<-start
					if kind == "contract" {
						_, errs[index] = stores[index].UpdateContract(context.Background(), contractUpdate("TASK-race", 1, fmt.Sprintf("writer-%d", index)))
					} else {
						_, errs[index] = stores[index].UpdateProgress(context.Background(), progressUpdate("TASK-race", 0, 2, fmt.Sprintf("writer-%d", index)))
					}
				}(index)
			}
			close(start)
			wait.Wait()
			successes, conflicts := 0, 0
			for _, err := range errs {
				if err == nil {
					successes++
				} else {
					var conflict *ConflictError
					if errors.As(err, &conflict) {
						conflicts++
					} else {
						t.Fatalf("result = %T %v", err, err)
					}
				}
			}
			if successes != 1 || conflicts != 1 {
				t.Fatalf("success/conflict = %d/%d, errors=%v", successes, conflicts, errs)
			}
			model, err := stores[0].ReadRecoveredTask(context.Background(), "TASK-race")
			if err != nil {
				t.Fatal(err)
			}
			if kind == "contract" && (len(model.ContractHistory) != 2 || len(model.ProgressHistory) != 1) {
				t.Fatalf("contract partial writes: %#v", model)
			}
			if kind == "progress" && (len(model.ProgressHistory) != 2 || len(model.ContractHistory) != 1) {
				t.Fatalf("progress partial writes: %#v", model)
			}
		})
	}
}
