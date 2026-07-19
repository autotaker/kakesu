package control

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

var allTaskStates = []TaskState{
	StateReady, StateRunning, StateWaiting, StateSuspended,
	StateReviewingCompletion, StateCompleted, StateCancelled,
}

func lifecycleInput(taskID, owner string) CreateTaskInput {
	input := testInput()
	input.TaskID = taskID
	input.OwnerAgentID = owner
	return input
}

func transitionInput(taskID string, current, target TaskState) TransitionInput {
	payload := []byte(`{"audit_ref":"test"}`)
	switch allowedTransitions[current][target] {
	case "TaskStarted", "TaskResumed":
		payload = []byte(`{"run_id":"run-test"}`)
	case "TaskCompleted":
		payload = []byte(`{"outcome_ref":"artifact://test/outcome"}`)
	case "TaskCancelled":
		payload = []byte(`{"reason":"test cancellation"}`)
	}
	return TransitionInput{TaskID: taskID, Expected: current, Target: target, Payload: payload}
}

func seedState(t *testing.T, store *Store, taskID string, state TaskState) {
	t.Helper()
	if _, err := store.db.Exec(`UPDATE tasks SET state = ? WHERE task_id = ?`, state, taskID); err != nil {
		t.Fatal(err)
	}
	if state == StateCompleted || state == StateCancelled {
		if _, err := store.db.Exec(`UPDATE task_owners SET released_at = CURRENT_TIMESTAMP WHERE task_id = ?`, taskID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLifecycleAllStatePairs(t *testing.T) {
	allowed, rejected := 0, 0
	for _, current := range allTaskStates {
		for _, target := range allTaskStates {
			name := fmt.Sprintf("%s_to_%s", current, target)
			t.Run(name, func(t *testing.T) {
				store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
				input := lifecycleInput("TASK-pair", "agent-pair")
				if _, err := store.CreateTask(context.Background(), input); err != nil {
					t.Fatal(err)
				}
				seedState(t, store, input.TaskID, current)
				before, err := store.ReadLifecycle(context.Background(), input.TaskID)
				if err != nil {
					t.Fatal(err)
				}
				got, err := store.TransitionTask(context.Background(), transitionInput(input.TaskID, current, target))
				eventType, isAllowed := allowedTransitions[current][target]
				if !isAllowed {
					var transitionErr *TransitionError
					if !errors.As(err, &transitionErr) {
						t.Fatalf("error = %T %v, want TransitionError", err, err)
					}
					after, readErr := store.ReadLifecycle(context.Background(), input.TaskID)
					if readErr != nil || !reflect.DeepEqual(after, before) {
						t.Fatalf("rejected transition mutated state: err=%v before=%#v after=%#v", readErr, before, after)
					}
					rejected++
					return
				}
				if err != nil {
					t.Fatalf("TransitionTask: %v", err)
				}
				if got.State != target || got.ProgressVersion != 0 || len(got.Events) != len(before.Events)+1 {
					t.Fatalf("model = %#v; before = %#v", got, before)
				}
				lastEvent := got.Events[len(got.Events)-1]
				if lastEvent.Type != eventType || lastEvent.Sequence != before.Events[len(before.Events)-1].Sequence+1 {
					t.Fatalf("last event = %#v; before = %#v", lastEvent, before.Events)
				}
				wantActive := target != StateCompleted && target != StateCancelled
				if got.OwnerAgentID != input.OwnerAgentID || got.OwnerActive != wantActive {
					t.Fatalf("owner = %q active=%t, want %q active=%t", got.OwnerAgentID, got.OwnerActive, input.OwnerAgentID, wantActive)
				}
				allowed++
			})
		}
	}
	if allowed != 13 || rejected != 36 {
		t.Fatalf("allowed/rejected = %d/%d, want 13/36", allowed, rejected)
	}
}

func TestLifecycleExpectedCurrentMismatchAndInvalidEdge(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	input := lifecycleInput("TASK-cas", "agent-cas")
	if _, err := store.CreateTask(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadLifecycle(context.Background(), input.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.TransitionTask(context.Background(), transitionInput(input.TaskID, StateRunning, StateWaiting))
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("mismatch = %T %v, want ConflictError", err, err)
	}
	_, err = store.TransitionTask(context.Background(), transitionInput(input.TaskID, StateReady, StateWaiting))
	var transitionErr *TransitionError
	if !errors.As(err, &transitionErr) {
		t.Fatalf("invalid edge = %T %v, want TransitionError", err, err)
	}
	_, err = store.TransitionTask(context.Background(), transitionInput(input.TaskID, StateRunning, StateRunning))
	if !errors.As(err, &conflict) {
		t.Fatalf("invalid edge with mismatch = %T %v, want ConflictError", err, err)
	}
	after, err := store.ReadLifecycle(context.Background(), input.TaskID)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("failed transitions mutated state: err=%v before=%#v after=%#v", err, before, after)
	}
}

func TestLifecycleOwnerRetainedInNonterminalStates(t *testing.T) {
	for _, target := range []TaskState{StateWaiting, StateSuspended, StateReviewingCompletion} {
		t.Run(string(target), func(t *testing.T) {
			store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
			first := lifecycleInput("TASK-first", "agent-shared")
			if _, err := store.CreateTask(context.Background(), first); err != nil {
				t.Fatal(err)
			}
			seedState(t, store, first.TaskID, StateRunning)
			if _, err := store.TransitionTask(context.Background(), transitionInput(first.TaskID, StateRunning, target)); err != nil {
				t.Fatal(err)
			}
			_, err := store.CreateTask(context.Background(), lifecycleInput("TASK-second", "agent-shared"))
			var conflict *ConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("owner reuse = %T %v, want ConflictError", err, err)
			}
			model, err := store.ReadLifecycle(context.Background(), first.TaskID)
			if err != nil || model.State != target || !model.OwnerActive {
				t.Fatalf("first task = %#v, err=%v", model, err)
			}
		})
	}
}

func TestLifecycleTerminalReleaseAllowsOwnerReuse(t *testing.T) {
	for _, test := range []struct {
		name, taskID    string
		current, target TaskState
	}{
		{"completed", "TASK-complete", StateReviewingCompletion, StateCompleted},
		{"cancelled", "TASK-cancel", StateRunning, StateCancelled},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
			if _, err := store.CreateTask(context.Background(), lifecycleInput(test.taskID, "agent-reuse")); err != nil {
				t.Fatal(err)
			}
			seedState(t, store, test.taskID, test.current)
			model, err := store.TransitionTask(context.Background(), transitionInput(test.taskID, test.current, test.target))
			if err != nil || model.OwnerActive {
				t.Fatalf("terminal transition = %#v, err=%v", model, err)
			}
			if _, err := store.CreateTask(context.Background(), lifecycleInput("TASK-reused", "agent-reuse")); err != nil {
				t.Fatalf("reuse owner: %v", err)
			}
			afterReuse, err := store.ReadLifecycle(context.Background(), test.taskID)
			if err != nil || !reflect.DeepEqual(afterReuse, model) {
				t.Fatalf("old terminal task changed after reuse: err=%v before=%#v after=%#v", err, model, afterReuse)
			}
		})
	}
}

func TestLifecyclePersistsRepresentativePayloadsAcrossReopen(t *testing.T) {
	for _, test := range []struct {
		name, taskID    string
		current, target TaskState
		payload         []byte
	}{
		{"started", "TASK-payload-started", StateReady, StateRunning, []byte(`{"run_id":"run-42"}`)},
		{"completed", "TASK-payload-completed", StateReviewingCompletion, StateCompleted, []byte(`{"outcome_ref":"artifact://task/outcome"}`)},
		{"cancelled", "TASK-payload-cancelled", StateRunning, StateCancelled, []byte(`{"reason":"operator request"}`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "control.db")
			store := openTestStore(t, path)
			if _, err := store.CreateTask(context.Background(), lifecycleInput(test.taskID, "agent-payload")); err != nil {
				t.Fatal(err)
			}
			seedState(t, store, test.taskID, test.current)
			request := TransitionInput{TaskID: test.taskID, Expected: test.current, Target: test.target, Payload: test.payload}
			model, err := store.TransitionTask(context.Background(), request)
			if err != nil || !reflect.DeepEqual(model.Events[len(model.Events)-1].Payload, test.payload) {
				t.Fatalf("transition payload = %q, err=%v", model.Events[len(model.Events)-1].Payload, err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = OpenStore(path)
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := store.ReadLifecycle(context.Background(), test.taskID)
			if err != nil || !reflect.DeepEqual(reopened, model) {
				t.Fatalf("reopened = %#v, want %#v, err=%v", reopened, model, err)
			}
		})
	}
}

func TestLifecycleRejectsEmptyOrInvalidPayloadWithoutMutation(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	input := lifecycleInput("TASK-bad-payload", "agent-bad-payload")
	if _, err := store.CreateTask(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadLifecycle(context.Background(), input.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{
		"empty": nil, "malformed": []byte(`{"run_id":`), "scalar": []byte(`"run-1"`),
		"missing required": []byte(`{}`), "empty required": []byte(`{"run_id":""}`),
	} {
		t.Run(name, func(t *testing.T) {
			request := TransitionInput{TaskID: input.TaskID, Expected: StateReady, Target: StateRunning, Payload: payload}
			_, err := store.TransitionTask(context.Background(), request)
			var payloadErr *EventPayloadError
			if !errors.As(err, &payloadErr) {
				t.Fatalf("error = %T %v, want EventPayloadError", err, err)
			}
			after, err := store.ReadLifecycle(context.Background(), input.TaskID)
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("invalid payload mutated task: err=%v before=%#v after=%#v", err, before, after)
			}
		})
	}
}

func TestLifecycleTerminalTransitionRollbackAtEveryStage(t *testing.T) {
	cases := []struct{ current, target TaskState }{{StateReviewingCompletion, StateCompleted}}
	for _, current := range allTaskStates[:5] {
		cases = append(cases, struct{ current, target TaskState }{current, StateCancelled})
	}
	for _, test := range cases {
		for _, stage := range []string{"state", "event", "owner"} {
			t.Run(string(test.current)+"_to_"+string(test.target)+"_"+stage, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "control.db")
				store := openTestStore(t, path)
				input := lifecycleInput("TASK-rollback", "agent-rollback")
				if _, err := store.CreateTask(context.Background(), input); err != nil {
					t.Fatal(err)
				}
				seedState(t, store, input.TaskID, test.current)
				before, err := store.ReadLifecycle(context.Background(), input.TaskID)
				if err != nil {
					t.Fatal(err)
				}
				store.lifecycleHook = func(tx *sql.Tx, gotStage string) error {
					if gotStage != stage {
						return nil
					}
					_, err := tx.Exec(`INSERT INTO missing_lifecycle_table(value) VALUES (1)`)
					return err
				}
				_, err = store.TransitionTask(context.Background(), transitionInput(input.TaskID, test.current, test.target))
				var storageErr *StorageError
				if !errors.As(err, &storageErr) {
					t.Fatalf("forced failure = %T %v, want StorageError", err, err)
				}
				after, err := store.ReadLifecycle(context.Background(), input.TaskID)
				if err != nil || !reflect.DeepEqual(after, before) {
					t.Fatalf("rollback mismatch: err=%v before=%#v after=%#v", err, before, after)
				}
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
				store, err = OpenStore(path)
				if err != nil {
					t.Fatal(err)
				}
				afterReopen, err := store.ReadLifecycle(context.Background(), input.TaskID)
				if err != nil || !reflect.DeepEqual(afterReopen, before) {
					t.Fatalf("reopen mismatch: err=%v before=%#v after=%#v", err, before, afterReopen)
				}
				succeeded, err := store.TransitionTask(context.Background(), transitionInput(input.TaskID, test.current, test.target))
				if err != nil || succeeded.State != test.target || succeeded.OwnerActive || len(succeeded.Events) != len(before.Events)+1 {
					t.Fatalf("retry without hook = %#v, err=%v", succeeded, err)
				}
				if succeeded.Events[len(succeeded.Events)-1].Sequence != before.Events[len(before.Events)-1].Sequence+1 {
					t.Fatalf("retry introduced sequence gap: before=%#v after=%#v", before.Events, succeeded.Events)
				}
			})
		}
	}
}

func TestLifecycleStateConstraintRejectsUnknownValue(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "control.db"))
	input := lifecycleInput("TASK-constraint", "agent-constraint")
	if _, err := store.CreateTask(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadLifecycle(context.Background(), input.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec(`UPDATE tasks SET state = 'unknown' WHERE task_id = ?`, input.TaskID)
	var sqliteConflict *ConflictError
	if classified := classifyCreateError(input.TaskID, err); !errors.As(classified, &sqliteConflict) {
		t.Fatalf("constraint = %T %v, want typed ConflictError", classified, classified)
	}
	after, err := store.ReadLifecycle(context.Background(), input.TaskID)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("constraint failure mutated task: err=%v before=%#v after=%#v", err, before, after)
	}
}

func TestConcurrentStoresEnforceOneActiveTaskPerAgent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.db")
	first := openTestStore(t, path)
	second := openTestStore(t, path)
	stores := []*Store{first, second}
	inputs := []CreateTaskInput{
		lifecycleInput("TASK-race-a", "agent-race"),
		lifecycleInput("TASK-race-b", "agent-race"),
	}
	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range stores {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = stores[i].CreateTask(context.Background(), inputs[i])
		}(i)
	}
	close(start)
	wg.Wait()
	successes, conflicts := 0, 0
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		var conflict *ConflictError
		if errors.As(err, &conflict) {
			conflicts++
			continue
		}
		t.Fatalf("concurrent result = %T %v", err, err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes/conflicts = %d/%d, errors=%v", successes, conflicts, errs)
	}
	assertTableCounts(t, first.db, 1)
}
