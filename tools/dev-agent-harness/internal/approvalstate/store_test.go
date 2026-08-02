package approvalstate

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/approvalmanifest"
)

const (
	testOld = "1111111111111111111111111111111111111111"
	testNew = "2222222222222222222222222222222222222222"
)

var testNow = time.Date(2026, 8, 2, 6, 2, 0, 0, time.UTC)

func testRules() Rules {
	return Rules{PolicyVersion: "push-v1", RevocationEpoch: 7, MaxTTL: 10 * time.Minute, MaxRecords: 8}
}

func testManifest(t *testing.T, id string, created, expires time.Time) *approvalmanifest.Manifest {
	return testManifestFor(t, id, created, expires, testRules())
}

func testManifestFor(t *testing.T, id string, created, expires time.Time, rules Rules) *approvalmanifest.Manifest {
	t.Helper()
	m, err := approvalmanifest.Build(approvalmanifest.Proposal{
		RequestID: id, AgentID: "agent-001", WorkspaceID: "workspace-001",
		Repository: "openai/agent-harness", Remote: "https://github.com/openai/agent-harness.git",
		RefUpdates:    []approvalmanifest.RefUpdate{{Ref: "refs/heads/main", ExpectedOldSHA: testOld, NewSHA: testNew}},
		PolicyVersion: rules.PolicyVersion, RevocationEpoch: rules.RevocationEpoch, CreatedAt: created, ExpiresAt: expires,
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func testRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func openTest(t *testing.T, root string, now *time.Time, hook func(persistPhase) error) *Store {
	t.Helper()
	s, err := openWith(root, testRules(), dependencies{now: func() time.Time { return *now }, hook: hook})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func requireClass(t *testing.T, err error, want ErrorClass) {
	t.Helper()
	if ClassOf(err) != want {
		t.Fatalf("class=%q want=%q err=%v", ClassOf(err), want, err)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func stateRoot(t *testing.T, raw []byte, mode os.FileMode) string {
	t.Helper()
	root := testRoot(t)
	must(t, os.WriteFile(filepath.Join(root, stateName), raw, mode))
	return root
}

func rejectOpen(t *testing.T, root string, rules Rules, want ErrorClass) {
	t.Helper()
	_, err := Open(root, rules)
	requireClass(t, err, want)
}

func TestOpenRootLockAndClose(t *testing.T) {
	root := testRoot(t)
	now := testNow
	s := openTest(t, root, &now, nil)
	rejectOpen(t, root, testRules(), ClassLocked)
	must(t, s.Close())
	must(t, s.Close())
	s2, err := Open(root, testRules())
	must(t, err)
	must(t, s2.Close())
	rejectOpen(t, "relative", testRules(), ClassInvalid)
}

func TestCreateTransitionsCopiesAndRestart(t *testing.T) {
	root := testRoot(t)
	now := testNow
	s := openTest(t, root, &now, nil)
	m := testManifest(t, "request-001", now.Add(-time.Minute), now.Add(4*time.Minute))
	raw := m.Encoding()
	r, err := s.Create(raw)
	if err != nil || r.State() != Pending || r.Digest() != m.Digest() {
		t.Fatalf("create=%v %v", r.State(), err)
	}
	raw[0] = 'x'
	copyOut := r.Encoding()
	copyOut[0] = 'x'
	got, err := s.Get(m.RequestID())
	if err != nil || !bytes.Equal(got.Encoding(), m.Encoding()) {
		t.Fatal("caller mutation reached store")
	}
	approved, err := s.Approve(m.RequestID(), m.Digest(), "actor-001")
	if err != nil || approved.State() != Approved || approved.ActorID() != "actor-001" {
		t.Fatalf("approve=%v %v", approved.State(), err)
	}
	_, err = s.Deny(m.RequestID(), m.Digest(), "actor-002")
	requireClass(t, err, ClassTransition)
	must(t, s.Close())
	reopened := openTest(t, root, &now, nil)
	got, err = reopened.Get(m.RequestID())
	if err != nil || got.State() != Approved || got.ActorID() != "actor-001" {
		t.Fatalf("restart=%v actor=%q err=%v", got.State(), got.ActorID(), err)
	}
	snap, err := reopened.Snapshot()
	if err != nil || snap.Version() != 1 || snap.Generation() != 2 || len(snap.Records()) != 1 {
		t.Fatalf("snapshot=%+v err=%v", snap, err)
	}
}

func TestCreateValidationCapacityAndOrdering(t *testing.T) {
	root := testRoot(t)
	now := testNow
	rules := testRules()
	rules.MaxRecords = 2
	s, err := openWith(root, rules, dependencies{now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, id := range []string{"request-b", "request-a"} {
		m := testManifest(t, id, now.Add(-time.Minute), now.Add(time.Minute))
		if _, err := s.Create(m.Encoding()); err != nil {
			t.Fatal(err)
		}
	}
	snap, _ := s.Snapshot()
	if got := snap.Records(); got[0].RequestID() != "request-a" || got[1].RequestID() != "request-b" {
		t.Fatal("records not sorted")
	}
	m := testManifest(t, "request-c", now.Add(-time.Minute), now.Add(time.Minute))
	_, err = s.Create(m.Encoding())
	requireClass(t, err, ClassCapacity)
	for _, tc := range []struct {
		name string
		raw  func(*testing.T, time.Time) []byte
		want ErrorClass
	}{
		{"noncanonical", func(t *testing.T, n time.Time) []byte {
			return append(testManifest(t, "bad-noncanonical", n.Add(-time.Minute), n.Add(time.Minute)).Encoding(), '\n')
		}, ClassInvalid},
		{"policy", func(t *testing.T, n time.Time) []byte {
			r := testRules()
			r.PolicyVersion = "push-v2"
			return testManifestFor(t, "bad-policy", n.Add(-time.Minute), n.Add(time.Minute), r).Encoding()
		}, ClassInvalid},
		{"epoch", func(t *testing.T, n time.Time) []byte {
			r := testRules()
			r.RevocationEpoch++
			return testManifestFor(t, "bad-epoch", n.Add(-time.Minute), n.Add(time.Minute), r).Encoding()
		}, ClassInvalid},
		{"future", func(t *testing.T, n time.Time) []byte {
			return testManifest(t, "bad-future", n.Add(time.Second), n.Add(time.Minute)).Encoding()
		}, ClassExpired},
		{"expired", func(t *testing.T, n time.Time) []byte {
			return testManifest(t, "bad-expired", n.Add(-time.Minute), n).Encoding()
		}, ClassExpired},
		{"ttl", func(t *testing.T, n time.Time) []byte {
			return testManifest(t, "bad-ttl", n.Add(-time.Minute), n.Add(10*time.Minute)).Encoding()
		}, ClassExpired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := testNow
			st := openTest(t, testRoot(t), &n, nil)
			_, err := st.Create(tc.raw(t, n))
			requireClass(t, err, tc.want)
		})
	}
	n := testNow
	duplicateStore := openTest(t, testRoot(t), &n, nil)
	duplicate := testManifest(t, "duplicate", n.Add(-time.Minute), n.Add(time.Minute))
	if _, err := duplicateStore.Create(duplicate.Encoding()); err != nil {
		t.Fatal(err)
	}
	_, err = duplicateStore.Create(duplicate.Encoding())
	requireClass(t, err, ClassConflict)
	_, err = duplicateStore.Approve(duplicate.RequestID(), duplicate.Digest(), "bad actor")
	requireClass(t, err, ClassInvalid)
	_, err = duplicateStore.MarkStale(duplicate.RequestID(), duplicate.Digest())
	requireClass(t, err, ClassTransition)
}

func TestExpiryPrecedenceAndClockRollback(t *testing.T) {
	root := testRoot(t)
	now := testNow
	s := openTest(t, root, &now, nil)
	m := testManifest(t, "request-expire", now.Add(-time.Minute), now.Add(time.Minute))
	if _, err := s.Create(m.Encoding()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	_, err := s.Approve(m.RequestID(), m.Digest(), "actor-001")
	requireClass(t, err, ClassExpired)
	r, err := s.Get(m.RequestID())
	if err != nil || r.State() != Expired {
		t.Fatalf("state=%v err=%v", r.State(), err)
	}
	now = now.Add(-time.Second)
	_, err = s.Get(m.RequestID())
	requireClass(t, err, ClassClock)

	now2 := testNow
	s2 := openTest(t, testRoot(t), &now2, nil)
	for _, id := range []string{"approved-get", "approved-sweep"} {
		m := testManifest(t, id, now2.Add(-time.Minute), now2.Add(time.Minute))
		if _, err := s2.Create(m.Encoding()); err != nil {
			t.Fatal(err)
		}
		if _, err := s2.Approve(id, m.Digest(), "actor"); err != nil {
			t.Fatal(err)
		}
	}
	now2 = now2.Add(time.Minute)
	got, err := s2.Get("approved-get")
	if err != nil || got.State() != Expired {
		t.Fatalf("Get active expiry=%v %v", got.State(), err)
	}
	if count, err := s2.ExpireDue(); err != nil || count != 1 {
		t.Fatalf("ExpireDue=%d %v", count, err)
	}
	snapshot, _ := s2.Snapshot()
	for _, record := range snapshot.Records() {
		if active(record.State()) {
			t.Fatal("active due record returned")
		}
	}
}

func TestCancelDenyStaleAndDigest(t *testing.T) {
	for _, tc := range []struct {
		name string
		act  func(*Store, *approvalmanifest.Manifest) (Record, error)
		want State
	}{
		{"deny", func(s *Store, m *approvalmanifest.Manifest) (Record, error) {
			return s.Deny(m.RequestID(), m.Digest(), "actor")
		}, Denied},
		{"cancel", func(s *Store, m *approvalmanifest.Manifest) (Record, error) {
			return s.Cancel(m.RequestID(), m.Digest())
		}, Cancelled},
		{"stale", func(s *Store, m *approvalmanifest.Manifest) (Record, error) {
			if _, err := s.Approve(m.RequestID(), m.Digest(), "actor"); err != nil {
				return Record{}, err
			}
			return s.MarkStale(m.RequestID(), m.Digest())
		}, Stale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := testNow
			s := openTest(t, testRoot(t), &now, nil)
			m := testManifest(t, "request-"+tc.name, now.Add(-time.Minute), now.Add(time.Minute))
			if _, err := s.Create(m.Encoding()); err != nil {
				t.Fatal(err)
			}
			_, err := s.Cancel(m.RequestID(), strings.Repeat("0", len(m.Digest())))
			requireClass(t, err, ClassDigest)
			r, err := tc.act(s, m)
			if err != nil || r.State() != tc.want {
				t.Fatalf("state=%v err=%v", r.State(), err)
			}
		})
	}
}

func TestPersistenceFailuresAndPoison(t *testing.T) {
	for _, tc := range []struct {
		phase  persistPhase
		class  ErrorClass
		poison bool
	}{{phaseWrite, ClassPersistence, false}, {phaseFileSync, ClassPersistence, false}, {phaseClose, ClassPersistence, false}, {phaseRename, ClassPoisoned, true}, {phaseDirectorySync, ClassPoisoned, true}} {
		t.Run(string(rune(tc.phase+'0')), func(t *testing.T) {
			root := testRoot(t)
			now := testNow
			s := openTest(t, root, &now, nil)
			baseline := testManifest(t, "request-baseline", now.Add(-time.Minute), now.Add(time.Minute))
			if _, err := s.Create(baseline.Encoding()); err != nil {
				t.Fatal(err)
			}
			before, err := s.Snapshot()
			must(t, err)
			diskBefore, err := os.ReadFile(filepath.Join(root, stateName))
			must(t, err)
			s.hook = func(p persistPhase) error {
				if p == tc.phase {
					return os.ErrInvalid
				}
				return nil
			}
			m := testManifest(t, "request-fail", now.Add(-time.Minute), now.Add(time.Minute))
			_, err = s.Create(m.Encoding())
			requireClass(t, err, tc.class)
			if !tc.poison {
				after, snapErr := s.Snapshot()
				must(t, snapErr)
				diskAfter, readErr := os.ReadFile(filepath.Join(root, stateName))
				must(t, readErr)
				if after.Generation() != before.Generation() || !reflect.DeepEqual(after.Records(), before.Records()) || !bytes.Equal(diskBefore, diskAfter) {
					t.Fatal("pre-rename failure changed memory or disk")
				}
			}
			_, err = s.Get(m.RequestID())
			if tc.poison {
				requireClass(t, err, ClassPoisoned)
			} else {
				requireClass(t, err, ClassNotFound)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			reopened := openTest(t, root, &now, nil)
			got, getErr := reopened.Get(m.RequestID())
			if tc.phase == phaseDirectorySync {
				if getErr != nil || got.State() != Pending {
					t.Fatalf("post-rename snapshot missing: state=%v err=%v", got.State(), getErr)
				}
			} else {
				requireClass(t, getErr, ClassNotFound)
			}
		})
	}
}

func TestOpenNodesAndCorruptionRejected(t *testing.T) {
	temp := testRoot(t)
	must(t, os.WriteFile(filepath.Join(temp, temporaryName), []byte("x"), 0o600))
	rejectOpen(t, temp, testRules(), ClassCorrupt)
	badJSON := testRoot(t)
	must(t, os.WriteFile(filepath.Join(badJSON, stateName), []byte("{}\n"), 0o600))
	rejectOpen(t, badJSON, testRules(), ClassCorrupt)
	target := testRoot(t)
	rootLink := filepath.Join(t.TempDir(), "root-link")
	must(t, os.Symlink(target, rootLink))
	nodes := []struct {
		name, node string
		mode       os.FileMode
		want       ErrorClass
	}{
		{"root symlink", "", 0, ClassInvalid}, {"state symlink", stateName, os.ModeSymlink, ClassPermission},
		{"lock symlink", lockName, os.ModeSymlink, ClassPermission}, {"root mode", "", 0o755, ClassPermission},
		{"lock mode", lockName, 0o644, ClassPermission},
	}
	for _, tc := range nodes {
		t.Run(tc.name, func(t *testing.T) {
			root := rootLink
			if tc.name != "root symlink" {
				root = testRoot(t)
				if tc.mode&os.ModeSymlink != 0 {
					must(t, os.Symlink(filepath.Join(t.TempDir(), "target"), filepath.Join(root, tc.node)))
				} else if tc.node == "" {
					must(t, os.Chmod(root, tc.mode))
				} else {
					must(t, os.WriteFile(filepath.Join(root, tc.node), nil, tc.mode))
				}
			}
			rejectOpen(t, root, testRules(), tc.want)
		})
	}

	baseRoot := testRoot(t)
	now := testNow
	base := openTest(t, baseRoot, &now, nil)
	m := testManifest(t, "request-corrupt", now.Add(-time.Minute), now.Add(time.Minute))
	_, err := base.Create(m.Encoding())
	must(t, err)
	must(t, base.Close())
	canonical, err := os.ReadFile(filepath.Join(baseRoot, stateName))
	must(t, err)
	mutations := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"unknown", func(b []byte) []byte { return append(append([]byte(nil), b[:len(b)-1]...), []byte(`,"unknown":1}`)...) }},
		{"duplicate", func(b []byte) []byte {
			return bytes.Replace(b, []byte(`{"format_version":1`), []byte(`{"format_version":1,"format_version":1`), 1)
		}},
		{"trailing", func(b []byte) []byte { return append(append([]byte(nil), b...), []byte(`{}`)...) }},
		{"whitespace", func(b []byte) []byte { return append(append([]byte(nil), b...), '\n') }},
		{"record mismatch", func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"request_id":"request-corrupt"`), []byte(`"request_id":"request-other"`), 1)
		}},
		{"partial", func(b []byte) []byte { return append([]byte(nil), b[:len(b)/2]...) }},
		{"oversize", func([]byte) []byte {
			limit := maxSnapshotBase + testRules().MaxRecords*(approvalmanifest.MaxEncodingBytes+1024)
			return bytes.Repeat([]byte("x"), limit+1)
		}},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			rejectOpen(t, stateRoot(t, tc.mutate(canonical), 0o600), testRules(), ClassCorrupt)
		})
	}
	for _, mode := range []os.FileMode{0o644, 0o400} {
		t.Run("state mode", func(t *testing.T) { rejectOpen(t, stateRoot(t, canonical, mode), testRules(), ClassPermission) })
	}
	mismatch := testRules()
	mismatch.RevocationEpoch++
	rejectOpen(t, stateRoot(t, canonical, 0o600), mismatch, ClassCorrupt)
}

func TestRootRenameDoesNotRedirectStore(t *testing.T) {
	parent := t.TempDir()
	rootPath := filepath.Join(parent, "state")
	must(t, os.Mkdir(rootPath, 0o700))
	now := testNow
	store := openTest(t, rootPath, &now, nil)
	original := filepath.Join(parent, "original")
	replacement := filepath.Join(parent, "replacement")
	must(t, os.Rename(rootPath, original))
	must(t, os.Mkdir(replacement, 0o700))
	must(t, os.Symlink(replacement, rootPath))
	m := testManifest(t, "request-root-race", now.Add(-time.Minute), now.Add(time.Minute))
	if _, err := store.Create(m.Encoding()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(original, stateName)); err != nil {
		t.Fatal("original root did not receive state")
	}
	if _, err := os.Stat(filepath.Join(replacement, stateName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("replacement root received state")
	}
}

func TestConcurrentOperationsAndClose(t *testing.T) {
	root := testRoot(t)
	now := testNow
	s := openTest(t, root, &now, nil)
	decision := testManifest(t, "request-decision", now.Add(-time.Minute), now.Add(time.Minute))
	if _, err := s.Create(decision.Encoding()); err != nil {
		t.Fatal(err)
	}
	manifests := make([]*approvalmanifest.Manifest, 6)
	for i := range manifests {
		manifests[i] = testManifest(t, "request-00"+string(rune('1'+i)), now.Add(-time.Minute), now.Add(time.Minute))
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, m := range manifests {
		m := m
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := s.Create(m.Encoding()); err != nil && ClassOf(err) != ClassClosed {
				t.Errorf("create: %v", err)
				return
			}
			_, _ = s.Get(m.RequestID())
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); <-start; _, _ = s.Approve(decision.RequestID(), decision.Digest(), "actor") }()
	wg.Add(1)
	go func() { defer wg.Done(); <-start; _, _ = s.ExpireDue(); _, _ = s.Snapshot(); _ = s.Close() }()
	close(start)
	wg.Wait()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := s.ExpireDue()
	requireClass(t, err, ClassClosed)
}

func TestErrorsDoNotLeakValues(t *testing.T) {
	root := testRoot(t)
	now := testNow
	s := openTest(t, root, &now, nil)
	secrets := []string{root, "request-secret", "actor-secret", "owner/repo", "refs/heads/secret", testOld, "sha256:secret", "raw-secret", os.ErrPermission.Error()}
	classes := []ErrorClass{ClassInvalid, ClassPermission, ClassLocked, ClassCorrupt, ClassConflict, ClassNotFound, ClassDigest, ClassTransition, ClassExpired, ClassClock, ClassCapacity, ClassPersistence, ClassPoisoned, ClassClosed, ClassUnsupported}
	for _, class := range classes {
		for _, secret := range secrets {
			if strings.Contains(newError(class).Error(), secret) {
				t.Fatalf("%s leaked", class)
			}
		}
	}
	_, err := s.Get("request-secret")
	for _, secret := range secrets {
		if err != nil && strings.Contains(err.Error(), secret) {
			t.Fatalf("runtime error leaked: %v", err)
		}
	}
	var nilStore *Store
	_, err = nilStore.Get("request-secret")
	requireClass(t, err, ClassClosed)
	_, err = nilStore.ExpireDue()
	requireClass(t, err, ClassClosed)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = s.Snapshot()
	requireClass(t, err, ClassClosed)

	n := testNow
	copyStore := openTest(t, testRoot(t), &n, nil)
	m := testManifest(t, "copy-snapshot", n.Add(-time.Minute), n.Add(time.Minute))
	if _, err := copyStore.Create(m.Encoding()); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := copyStore.Snapshot()
	records := snapshot.Records()
	records[0].requestID = "mutated"
	records[0].manifest[0] = 'x'
	again, _ := copyStore.Snapshot()
	if again.Records()[0].RequestID() != m.RequestID() || !bytes.Equal(again.Records()[0].Encoding(), m.Encoding()) {
		t.Fatal("snapshot getter leaked ownership")
	}
}
