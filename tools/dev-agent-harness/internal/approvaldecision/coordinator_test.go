package approvaldecision

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/approvalchallenge"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/approvalmanifest"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/approvalstate"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCoordinatorSemanticOrderAndExactBinding(t *testing.T) {
	order := []string{}
	state := &fakeState{record: stateRecord{requestID: "request-from-store", digest: testDigest, state: approvalstate.Pending}, order: &order}
	challenges := &fakeChallenges{order: &order}
	coordinator, err := newCoordinator(state, challenges)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := coordinator.Begin("lookup-key", approvalchallenge.Approve, "operator-1", "approval.example.com", "https://approval.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "get,issue" {
		t.Fatalf("begin order = %v", order)
	}
	if issued.RequestID() != state.record.requestID || issued.Digest() != state.record.digest || challenges.request.requestID != state.record.requestID || challenges.request.digest != state.record.digest {
		t.Fatal("Begin did not derive binding from the store record")
	}

	state.calls = nil
	challenges.calls = nil
	order = nil
	result, err := coordinator.Complete(issued.Challenge(), []byte("assertion"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "consume,approve" {
		t.Fatalf("complete order = %v", order)
	}
	if result.RequestID() != state.record.requestID || result.Digest() != state.record.digest || result.State() != approvalstate.Approved || result.ActorID() != "operator-1" || result.CredentialID() == "" {
		t.Fatalf("result mismatch: %#v", result)
	}
}

func TestNewAndNilCoordinatorRejectInvalidDependencies(t *testing.T) {
	manager, err := approvalchallenge.New(approvalchallenge.Rules{TTL: time.Minute, MaxPending: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	verifier := func(approvalchallenge.Binding, []byte) ([]byte, error) { return []byte("credential"), nil }
	for name, construct := range map[string]func() (*Coordinator, error){
		"nil store":     func() (*Coordinator, error) { return New(nil, manager, verifier) },
		"nil manager":   func() (*Coordinator, error) { return New(&approvalstate.Store{}, nil, verifier) },
		"nil verifier":  func() (*Coordinator, error) { return New(&approvalstate.Store{}, manager, nil) },
		"private state": func() (*Coordinator, error) { return newCoordinator(nil, &fakeChallenges{}) },
		"private issue": func() (*Coordinator, error) { return newCoordinator(&fakeState{}, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := construct()
			if got != nil || ClassOf(err) != ClassInvalid || err.Error() != "approval decision invalid" {
				t.Fatalf("New = %#v, %v", got, err)
			}
		})
	}
	var coordinator *Coordinator
	if issued, err := coordinator.Begin("secret-request", approvalchallenge.Approve, "secret-operator", "example.com", "https://example.com"); issued != (Issued{}) || ClassOf(err) != ClassInvalid {
		t.Fatalf("nil Begin = %#v, %v", issued, err)
	}
	if result, err := coordinator.Complete("secret-challenge", []byte("secret-assertion")); result != (Result{}) || ClassOf(err) != ClassInvalid {
		t.Fatalf("nil Complete = %#v, %v", result, err)
	}
}

func TestBeginFailuresAreFixedAndNeverIssueAfterGetFailure(t *testing.T) {
	cases := []struct {
		name   string
		record stateRecord
		getErr error
	}{
		{name: "get failure", record: pendingFakeRecord(), getErr: errFake},
		{name: "expired", record: stateRecord{requestID: "secret-request", digest: testDigest, state: approvalstate.Expired}},
		{name: "cancelled", record: stateRecord{requestID: "secret-request", digest: testDigest, state: approvalstate.Cancelled}},
		{name: "approved", record: stateRecord{requestID: "secret-request", digest: testDigest, state: approvalstate.Approved}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := &fakeState{record: tc.record, err: tc.getErr}
			challenges := &fakeChallenges{}
			coordinator, _ := newCoordinator(state, challenges)
			issued, err := coordinator.Begin("secret-request", approvalchallenge.Approve, "secret-operator", "approval.example.com", "https://approval.example.com")
			if issued != (Issued{}) || ClassOf(err) != ClassBegin || err.Error() != "approval decision begin" || len(challenges.calls) != 0 {
				t.Fatalf("Begin = %#v, %v; issue calls %v", issued, err, challenges.calls)
			}
		})
	}

	state := &fakeState{record: pendingFakeRecord()}
	challenges := &fakeChallenges{err: errFake}
	coordinator, _ := newCoordinator(state, challenges)
	issued, err := coordinator.Begin("secret-request", approvalchallenge.Approve, "secret-operator", "approval.example.com", "https://approval.example.com")
	if issued != (Issued{}) || ClassOf(err) != ClassBegin || strings.Contains(err.Error(), "secret") || len(state.calls) != 1 || len(challenges.calls) != 1 {
		t.Fatalf("Issue failure = %#v, %v; calls %v/%v", issued, err, state.calls, challenges.calls)
	}
}

func TestBeginRejectsMalformedIssueOutput(t *testing.T) {
	validRequest := challengeRequest{requestID: "request-store", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "operator-1", rpID: "approval.example.com", origin: "https://approval.example.com"}
	now := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	valid := challengeIssued{challenge: "opaque", binding: validRequest, issuedAt: now, expiresAt: now.Add(time.Minute)}
	cases := map[string]func(*challengeIssued){
		"empty challenge":   func(i *challengeIssued) { i.challenge = "" },
		"request mismatch":  func(i *challengeIssued) { i.binding.requestID = "wrong" },
		"digest mismatch":   func(i *challengeIssued) { i.binding.digest = strings.Replace(testDigest, "a", "b", 1) },
		"decision mismatch": func(i *challengeIssued) { i.binding.decision = approvalchallenge.Deny },
		"operator mismatch": func(i *challengeIssued) { i.binding.operatorID = "wrong" },
		"rp mismatch":       func(i *challengeIssued) { i.binding.rpID = "wrong.example.com" },
		"origin mismatch":   func(i *challengeIssued) { i.binding.origin = "https://wrong.example.com" },
		"zero issued":       func(i *challengeIssued) { i.issuedAt = time.Time{} },
		"inverted lifetime": func(i *challengeIssued) { i.expiresAt = i.issuedAt },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			malformed := valid
			mutate(&malformed)
			state := &fakeState{record: stateRecord{requestID: validRequest.requestID, digest: validRequest.digest, state: approvalstate.Pending}}
			challenges := challengeFuncs{issueFunc: func(challengeRequest) (challengeIssued, error) { return malformed, nil }}
			coordinator, _ := newCoordinator(state, challenges)
			issued, err := coordinator.Begin("lookup", validRequest.decision, validRequest.operatorID, validRequest.rpID, validRequest.origin)
			if issued != (Issued{}) || ClassOf(err) != ClassBegin {
				t.Fatalf("malformed Issue accepted: %#v, %v", issued, err)
			}
		})
	}
}

func TestProductionIntegrationDurableDecisionAndReplay(t *testing.T) {
	for _, decision := range []approvalchallenge.Decision{approvalchallenge.Approve, approvalchallenge.Deny} {
		t.Run(string(decision), func(t *testing.T) {
			store, record := newRealStoreWithRequest(t, "request-"+string(decision))
			manager, err := approvalchallenge.New(approvalchallenge.Rules{TTL: time.Minute, MaxPending: 2})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = manager.Close() })

			assertion := []byte("opaque-passkey-assertion")
			verifierCalls := 0
			coordinator, err := New(store, manager, func(binding approvalchallenge.Binding, got []byte) ([]byte, error) {
				verifierCalls++
				if binding.RequestID() != record.RequestID() || binding.Digest() != record.Digest() || binding.Decision() != decision || binding.OperatorID() != "operator-real" || binding.RPID() != "approval.example.com" || binding.Origin() != "https://approval.example.com" {
					t.Fatal("production verifier binding mismatch")
				}
				if !bytes.Equal(got, assertion) {
					t.Fatal("production verifier assertion mismatch")
				}
				got[0] = 'X'
				return []byte("raw-credential-real"), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			issued, err := coordinator.Begin(record.RequestID(), decision, "operator-real", "approval.example.com", "https://approval.example.com")
			if err != nil {
				t.Fatal(err)
			}
			if issued.RequestID() != record.RequestID() || issued.Digest() != record.Digest() || issued.Decision() != decision || issued.OperatorID() != "operator-real" {
				t.Fatal("production Begin binding mismatch")
			}

			result, err := coordinator.Complete(issued.Challenge(), assertion)
			if err != nil {
				t.Fatal(err)
			}
			if string(assertion) != "opaque-passkey-assertion" {
				t.Fatal("caller assertion was mutated")
			}
			wantState := approvalstate.Approved
			if decision == approvalchallenge.Deny {
				wantState = approvalstate.Denied
			}
			if result.RequestID() != record.RequestID() || result.Digest() != record.Digest() || result.State() != wantState || result.ActorID() != "operator-real" || !strings.HasPrefix(result.CredentialID(), "sha256:") {
				t.Fatalf("production result mismatch: %#v", result)
			}

			// The durable record, not a reconstructed response, is the recovery
			// source if the successful Complete response is lost.
			durable, err := store.Get(record.RequestID())
			if err != nil || durable.State() != wantState || durable.ActorID() != "operator-real" || durable.Digest() != record.Digest() {
				t.Fatalf("durable reconciliation = %#v, %v", durable, err)
			}
			if replay, err := coordinator.Complete(issued.Challenge(), assertion); replay != (Result{}) || ClassOf(err) != ClassVerification {
				t.Fatalf("replay = %#v, %v", replay, err)
			}
			if verifierCalls != 1 {
				t.Fatalf("verifier calls = %d", verifierCalls)
			}
		})
	}
}

func TestCompleteRejectsVerificationFailuresBeforeStateMutation(t *testing.T) {
	malformed := []challengeVerified{
		{},
		{requestID: "request", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "operator", credentialID: "sha256:short"},
		{requestID: "request", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "operator", credentialID: "sha256:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"},
		{requestID: "request", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "operator", credentialID: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbg"},
		{requestID: "request", digest: testDigest, decision: approvalchallenge.Decision("other"), operatorID: "operator", credentialID: validCredentialIDForTest()},
	}
	for index, verified := range malformed {
		state := &fakeState{record: pendingFakeRecord()}
		challenges := challengeFuncs{consumeFunc: func(string, []byte) (challengeVerified, error) { return verified, nil }}
		coordinator, _ := newCoordinator(state, challenges)
		result, err := coordinator.Complete("secret-challenge", []byte("secret-assertion"))
		if result != (Result{}) || ClassOf(err) != ClassVerification || len(state.calls) != 0 {
			t.Fatalf("case %d = %#v, %v; state calls %v", index, result, err, state.calls)
		}
	}

	for _, panicAt := range []string{"consume", "verify"} {
		t.Run(panicAt, func(t *testing.T) {
			state := &fakeState{record: pendingFakeRecord()}
			challenges := challengeFuncs{consumeFunc: func(string, []byte) (challengeVerified, error) {
				if panicAt == "consume" {
					panic("secret verifier panic")
				}
				return challengeVerified{}, errFake
			}}
			coordinator, _ := newCoordinator(state, challenges)
			result, err := coordinator.Complete("secret-challenge", []byte("secret-assertion"))
			if result != (Result{}) || ClassOf(err) != ClassVerification || err.Error() != "approval decision verification" || len(state.calls) != 0 {
				t.Fatalf("Complete = %#v, %v; state calls %v", result, err, state.calls)
			}
		})
	}
}

func TestCompleteUsesExactVerifiedDecisionWithoutFallback(t *testing.T) {
	for _, decision := range []approvalchallenge.Decision{approvalchallenge.Approve, approvalchallenge.Deny} {
		t.Run(string(decision), func(t *testing.T) {
			state := &fakeState{record: pendingFakeRecord()}
			challenges := &fakeChallenges{request: challengeRequest{
				requestID: state.record.requestID, digest: state.record.digest, decision: decision, operatorID: "verified-operator",
			}}
			coordinator, _ := newCoordinator(state, challenges)
			result, err := coordinator.Complete("opaque", []byte("assertion"))
			if err != nil {
				t.Fatal(err)
			}
			wantCall, wantState := "approve", approvalstate.Approved
			if decision == approvalchallenge.Deny {
				wantCall, wantState = "deny", approvalstate.Denied
			}
			if strings.Join(state.calls, ",") != wantCall || result.State() != wantState || result.ActorID() != "verified-operator" {
				t.Fatalf("exact decision = calls %v, result %#v", state.calls, result)
			}
		})
	}

	state := &fakeState{record: pendingFakeRecord(), err: errFake}
	challenges := &fakeChallenges{request: challengeRequest{
		requestID: state.record.requestID, digest: state.record.digest, decision: approvalchallenge.Approve, operatorID: "verified-operator",
	}}
	coordinator, _ := newCoordinator(state, challenges)
	result, err := coordinator.Complete("secret-challenge", []byte("secret-assertion"))
	if result != (Result{}) || ClassOf(err) != ClassDecision || err.Error() != "approval decision decision" || strings.Join(state.calls, ",") != "approve" {
		t.Fatalf("state failure = %#v, %v; calls %v", result, err, state.calls)
	}
}

func TestCompleteRejectsInconsistentDurableSuccess(t *testing.T) {
	verified := challengeVerified{requestID: "request", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "operator", credentialID: validCredentialIDForTest()}
	cases := map[string]stateRecord{
		"request": {requestID: "other", digest: testDigest, state: approvalstate.Approved, actorID: "operator"},
		"digest":  {requestID: "request", digest: strings.Replace(testDigest, "a", "b", 1), state: approvalstate.Approved, actorID: "operator"},
		"state":   {requestID: "request", digest: testDigest, state: approvalstate.Denied, actorID: "operator"},
		"actor":   {requestID: "request", digest: testDigest, state: approvalstate.Approved, actorID: "other"},
	}
	for name, record := range cases {
		t.Run(name, func(t *testing.T) {
			state := stateFuncs{approveFunc: func(string, string, string) (stateRecord, error) { return record, nil }}
			challenges := challengeFuncs{consumeFunc: func(string, []byte) (challengeVerified, error) { return verified, nil }}
			coordinator, _ := newCoordinator(state, challenges)
			result, err := coordinator.Complete("challenge", []byte("assertion"))
			if result != (Result{}) || ClassOf(err) != ClassDecision {
				t.Fatalf("inconsistent success = %#v, %v", result, err)
			}
		})
	}
}

func TestTransitionFailureConsumesOnceAndNeverInfersSuccess(t *testing.T) {
	verified := challengeVerified{requestID: "secret-request", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "secret-operator", credentialID: validCredentialIDForTest()}
	var consumeCalls atomic.Int32
	challenges := challengeFuncs{consumeFunc: func(string, []byte) (challengeVerified, error) {
		if consumeCalls.Add(1) != 1 {
			return challengeVerified{}, errFake
		}
		return verified, nil
	}}
	state := &fakeState{record: pendingFakeRecord(), err: errors.New("secret persistence path and digest")}
	coordinator, _ := newCoordinator(state, challenges)

	first, firstErr := coordinator.Complete("secret-challenge", []byte("secret-assertion"))
	second, secondErr := coordinator.Complete("secret-challenge", []byte("secret-assertion"))
	if first != (Result{}) || ClassOf(firstErr) != ClassDecision || second != (Result{}) || ClassOf(secondErr) != ClassVerification {
		t.Fatalf("failure/replay = %#v %v / %#v %v", first, firstErr, second, secondErr)
	}
	if strings.Join(state.calls, ",") != "approve" || consumeCalls.Load() != 2 {
		t.Fatalf("failure calls = state %v consume %d", state.calls, consumeCalls.Load())
	}
	for _, err := range []error{firstErr, secondErr} {
		if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), testDigest) || strings.Contains(err.Error(), errFake.Error()) {
			t.Fatalf("error leaked lower or bound data: %v", err)
		}
	}
}

func TestStateFailureMatrixAlwaysReturnsEmptyFixedDecisionError(t *testing.T) {
	verified := challengeVerified{requestID: "secret-request", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "secret-operator", credentialID: validCredentialIDForTest()}
	for _, name := range []string{"expired", "terminal", "digest", "persistence", "poisoned"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			state := stateFuncs{approveFunc: func(string, string, string) (stateRecord, error) {
				calls++
				return stateRecord{}, errors.New("secret lower " + name)
			}}
			challenges := challengeFuncs{consumeFunc: func(string, []byte) (challengeVerified, error) { return verified, nil }}
			coordinator, _ := newCoordinator(state, challenges)
			result, err := coordinator.Complete("secret-challenge", []byte("secret-assertion"))
			if result != (Result{}) || ClassOf(err) != ClassDecision || err.Error() != "approval decision decision" || calls != 1 {
				t.Fatalf("failure = %#v, %v; calls %d", result, err, calls)
			}
		})
	}
}

func TestCoordinatorCopiesAssertionAtPrivateBoundary(t *testing.T) {
	assertion := []byte("caller-owned-assertion")
	verified := challengeVerified{requestID: "request", digest: testDigest, decision: approvalchallenge.Approve, operatorID: "operator", credentialID: validCredentialIDForTest()}
	challenges := challengeFuncs{consumeFunc: func(_ string, got []byte) (challengeVerified, error) {
		got[0] = 'X'
		return verified, nil
	}}
	state := stateFuncs{approveFunc: func(requestID, digest, actorID string) (stateRecord, error) {
		return stateRecord{requestID: requestID, digest: digest, state: approvalstate.Approved, actorID: actorID}, nil
	}}
	coordinator, _ := newCoordinator(state, challenges)
	result, err := coordinator.Complete("challenge", assertion)
	if err != nil || result.State() != approvalstate.Approved {
		t.Fatalf("Complete = %#v, %v", result, err)
	}
	if string(assertion) != "caller-owned-assertion" {
		t.Fatal("private challenge operation mutated caller assertion")
	}
}

func TestProductionVerifierPanicConsumesChallenge(t *testing.T) {
	store, record := newRealStoreWithRequest(t, "request-panic")
	manager, err := approvalchallenge.New(approvalchallenge.Rules{TTL: time.Minute, MaxPending: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	coordinator, err := New(store, manager, func(approvalchallenge.Binding, []byte) ([]byte, error) {
		panic("secret verifier panic")
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := coordinator.Begin(record.RequestID(), approvalchallenge.Approve, "operator", "approval.example.com", "https://approval.example.com")
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := coordinator.Complete(issued.Challenge(), []byte("secret-assertion"))
		if result != (Result{}) || ClassOf(err) != ClassVerification || strings.Contains(err.Error(), "secret") {
			t.Fatalf("attempt %d = %#v, %v", attempt, result, err)
		}
	}
	durable, err := store.Get(record.RequestID())
	if err != nil || durable.State() != approvalstate.Pending {
		t.Fatalf("verifier panic mutated state: %#v, %v", durable, err)
	}
}

func TestProductionOpposedChallengesFirstDurableDecisionWins(t *testing.T) {
	store, record := newRealStoreWithRequest(t, "request-opposed")
	manager, err := approvalchallenge.New(approvalchallenge.Rules{TTL: time.Minute, MaxPending: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	coordinator, err := New(store, manager, func(approvalchallenge.Binding, []byte) ([]byte, error) {
		return []byte("credential-race"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	approve, err := coordinator.Begin(record.RequestID(), approvalchallenge.Approve, "operator-approve", "approval.example.com", "https://approval.example.com")
	if err != nil {
		t.Fatal(err)
	}
	deny, err := coordinator.Begin(record.RequestID(), approvalchallenge.Deny, "operator-deny", "approval.example.com", "https://approval.example.com")
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	type outcome struct {
		result Result
		err    error
	}
	outcomes := make(chan outcome, 2)
	for _, issued := range []Issued{approve, deny} {
		issued := issued
		go func() {
			<-start
			result, err := coordinator.Complete(issued.Challenge(), []byte("assertion"))
			outcomes <- outcome{result: result, err: err}
		}()
	}
	close(start)
	successes, failures := 0, 0
	for range 2 {
		outcome := <-outcomes
		if outcome.err == nil {
			successes++
		} else if outcome.result == (Result{}) && ClassOf(outcome.err) == ClassDecision {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("opposed outcomes success=%d failure=%d", successes, failures)
	}
	durable, err := store.Get(record.RequestID())
	if err != nil || (durable.State() != approvalstate.Approved && durable.State() != approvalstate.Denied) {
		t.Fatalf("durable first winner = %#v, %v", durable, err)
	}
}

func TestProductionConcurrentBeginAndCompleteRemainsStoreArbitrated(t *testing.T) {
	store, record := newRealStoreWithRequest(t, "request-begin-complete")
	manager, err := approvalchallenge.New(approvalchallenge.Rules{TTL: time.Minute, MaxPending: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	coordinator, err := New(store, manager, func(approvalchallenge.Binding, []byte) ([]byte, error) {
		return []byte("credential-concurrent"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	approve, err := coordinator.Begin(record.RequestID(), approvalchallenge.Approve, "operator-approve", "approval.example.com", "https://approval.example.com")
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	var completeResult Result
	var completeErr error
	var denyIssued Issued
	var denyBeginErr error
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		completeResult, completeErr = coordinator.Complete(approve.Challenge(), []byte("assertion"))
	}()
	go func() {
		defer wait.Done()
		<-start
		denyIssued, denyBeginErr = coordinator.Begin(record.RequestID(), approvalchallenge.Deny, "operator-deny", "approval.example.com", "https://approval.example.com")
	}()
	close(start)
	wait.Wait()
	if completeErr != nil || completeResult.State() != approvalstate.Approved {
		t.Fatalf("approve complete = %#v, %v", completeResult, completeErr)
	}
	if denyBeginErr == nil {
		if result, err := coordinator.Complete(denyIssued.Challenge(), []byte("assertion")); result != (Result{}) || ClassOf(err) != ClassDecision {
			t.Fatalf("stale concurrent challenge = %#v, %v", result, err)
		}
	} else if ClassOf(denyBeginErr) != ClassBegin {
		t.Fatalf("deny Begin error = %v", denyBeginErr)
	}
	durable, err := store.Get(record.RequestID())
	if err != nil || durable.State() != approvalstate.Approved || durable.ActorID() != "operator-approve" {
		t.Fatalf("concurrent durable result = %#v, %v", durable, err)
	}
}

func newRealStoreWithRequest(t *testing.T, requestID string) (*approvalstate.Store, approvalstate.Record) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	rules := approvalstate.Rules{PolicyVersion: "push-v1", RevocationEpoch: 7, MaxTTL: 10 * time.Minute, MaxRecords: 8}
	store, err := approvalstate.Open(root, rules)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Now().UTC().Truncate(time.Second)
	manifest, err := approvalmanifest.Build(approvalmanifest.Proposal{
		RequestID: requestID, AgentID: "agent-001", WorkspaceID: "workspace-001",
		Repository: "openai/agent-harness", Remote: "https://github.com/openai/agent-harness.git",
		RefUpdates: []approvalmanifest.RefUpdate{{
			Ref: "refs/heads/main", ExpectedOldSHA: "1111111111111111111111111111111111111111", NewSHA: "2222222222222222222222222222222222222222",
		}},
		PolicyVersion: rules.PolicyVersion, RevocationEpoch: rules.RevocationEpoch,
		CreatedAt: now.Add(-time.Second), ExpiresAt: now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.Create(manifest.Encoding())
	if err != nil {
		t.Fatal(err)
	}
	return store, record
}

type fakeState struct {
	record stateRecord
	err    error
	calls  []string
	order  *[]string
}

func (f *fakeState) get(string) (stateRecord, error) {
	f.calls = append(f.calls, "get")
	f.recordOrder("get")
	return f.record, f.err
}

func (f *fakeState) approve(requestID, digest, actorID string) (stateRecord, error) {
	f.calls = append(f.calls, "approve")
	f.recordOrder("approve")
	if f.err != nil {
		return stateRecord{}, f.err
	}
	f.record = stateRecord{requestID: requestID, digest: digest, state: approvalstate.Approved, actorID: actorID}
	return f.record, nil
}

func (f *fakeState) deny(requestID, digest, actorID string) (stateRecord, error) {
	f.calls = append(f.calls, "deny")
	f.recordOrder("deny")
	if f.err != nil {
		return stateRecord{}, f.err
	}
	f.record = stateRecord{requestID: requestID, digest: digest, state: approvalstate.Denied, actorID: actorID}
	return f.record, nil
}

func (f *fakeState) recordOrder(operation string) {
	if f.order != nil {
		*f.order = append(*f.order, operation)
	}
}

type fakeChallenges struct {
	request challengeRequest
	err     error
	calls   []string
	order   *[]string
}

func (f *fakeChallenges) issue(request challengeRequest) (challengeIssued, error) {
	f.calls = append(f.calls, "issue")
	f.recordOrder("issue")
	f.request = request
	if f.err != nil {
		return challengeIssued{}, f.err
	}
	now := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	return challengeIssued{challenge: "opaque-challenge", binding: request, issuedAt: now, expiresAt: now.Add(time.Minute)}, nil
}

func (f *fakeChallenges) consume(string, []byte) (challengeVerified, error) {
	f.calls = append(f.calls, "consume")
	f.recordOrder("consume")
	if f.err != nil {
		return challengeVerified{}, f.err
	}
	return challengeVerified{
		requestID:    f.request.requestID,
		digest:       f.request.digest,
		decision:     f.request.decision,
		operatorID:   f.request.operatorID,
		credentialID: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}, nil
}

func (f *fakeChallenges) recordOrder(operation string) {
	if f.order != nil {
		*f.order = append(*f.order, operation)
	}
}

type stateFuncs struct {
	getFunc     func(string) (stateRecord, error)
	approveFunc func(string, string, string) (stateRecord, error)
	denyFunc    func(string, string, string) (stateRecord, error)
}

func (f stateFuncs) get(requestID string) (stateRecord, error) {
	if f.getFunc == nil {
		return stateRecord{}, errFake
	}
	return f.getFunc(requestID)
}

func (f stateFuncs) approve(requestID, digest, actorID string) (stateRecord, error) {
	if f.approveFunc == nil {
		return stateRecord{}, errFake
	}
	return f.approveFunc(requestID, digest, actorID)
}

func (f stateFuncs) deny(requestID, digest, actorID string) (stateRecord, error) {
	if f.denyFunc == nil {
		return stateRecord{}, errFake
	}
	return f.denyFunc(requestID, digest, actorID)
}

type challengeFuncs struct {
	issueFunc   func(challengeRequest) (challengeIssued, error)
	consumeFunc func(string, []byte) (challengeVerified, error)
}

func (f challengeFuncs) issue(request challengeRequest) (challengeIssued, error) {
	if f.issueFunc == nil {
		return challengeIssued{}, errFake
	}
	return f.issueFunc(request)
}

func (f challengeFuncs) consume(challenge string, assertion []byte) (challengeVerified, error) {
	if f.consumeFunc == nil {
		return challengeVerified{}, errFake
	}
	return f.consumeFunc(challenge, assertion)
}

func pendingFakeRecord() stateRecord {
	return stateRecord{requestID: "secret-request", digest: testDigest, state: approvalstate.Pending}
}

func validCredentialIDForTest() string {
	return "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
}

var errFake = errors.New("sensitive fake failure")
