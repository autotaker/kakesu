package approvalchallenge

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIssueConsumeSemanticLifecycle(t *testing.T) {
	now := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	random := bytes.NewReader(append(bytes.Repeat([]byte{0x11}, challengeBytes), bytes.Repeat([]byte{0x22}, challengeBytes)...))
	m, err := newWithDependencies(Rules{TTL: 2 * time.Minute, MaxPending: 2}, dependencies{random: random, now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	request := validTestRequest()
	issued, err := m.Issue(request)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(issued.Challenge())
	if err != nil || len(raw) != challengeBytes {
		t.Fatalf("challenge is not 32-byte raw base64url: %q, %v", issued.Challenge(), err)
	}
	binding := issued.Binding()
	if binding.Challenge() != issued.Challenge() || binding.RequestID() != request.RequestID || binding.Digest() != request.Digest || binding.Decision() != request.Decision || binding.OperatorID() != request.OperatorID || binding.RPID() != request.RPID || binding.Origin() != request.Origin || !binding.IssuedAt().Equal(now) || !binding.ExpiresAt().Equal(now.Add(2*time.Minute)) {
		t.Fatal("issued binding mismatch")
	}

	assertion := []byte("opaque-assertion")
	credential := []byte("credential-01")
	verified, err := m.Consume(issued.Challenge(), assertion, func(got Binding, gotAssertion []byte) ([]byte, error) {
		if got != binding || !bytes.Equal(gotAssertion, assertion) {
			t.Fatal("verifier input mismatch")
		}
		return credential, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(append([]byte(credentialDigestDomain), credential...))
	if verified.RequestID() != request.RequestID || verified.Digest() != request.Digest || verified.Decision() != request.Decision || verified.OperatorID() != request.OperatorID || verified.CredentialID() != "sha256:"+hex.EncodeToString(sum[:]) || !verified.VerifiedAt().Equal(now) {
		t.Fatal("verified result mismatch")
	}
	if _, err := m.Consume(issued.Challenge(), assertion, func(Binding, []byte) ([]byte, error) { return credential, nil }); ClassOf(err) != ClassNotFound {
		t.Fatalf("replay class = %q, err = %v", ClassOf(err), err)
	}
	request.Decision = Deny
	denied, err := m.Issue(request)
	if err != nil || denied.Binding().Decision() != Deny || denied.Challenge() == issued.Challenge() {
		t.Fatalf("deny issue = %#v, %v", denied, err)
	}
}

func validTestRequest() Request {
	return Request{
		RequestID:  "request-001",
		Digest:     "sha256:" + string(bytes.Repeat([]byte{'a'}, sha256.Size*2)),
		Decision:   Approve,
		OperatorID: "operator-001",
		RPID:       "approval.example.com",
		Origin:     "https://approval.example.com",
	}
}

func TestNewAndRulesValidation(t *testing.T) {
	valid := Rules{TTL: time.Minute, MaxPending: 1}
	cases := []Rules{
		{},
		{TTL: -time.Second, MaxPending: 1},
		{TTL: time.Nanosecond, MaxPending: 1},
		{TTL: MaxTTL + time.Second, MaxPending: 1},
		{TTL: time.Minute, MaxPending: 0},
		{TTL: time.Minute, MaxPending: maxPendingLimit + 1},
	}
	for _, rules := range cases {
		if manager, err := New(rules); manager != nil || ClassOf(err) != ClassInvalid {
			t.Fatalf("New(%+v) = %#v, %v", rules, manager, err)
		}
	}
	if manager, err := newWithDependencies(valid, dependencies{}); manager != nil || ClassOf(err) != ClassInvalid {
		t.Fatalf("nil dependencies = %#v, %v", manager, err)
	}
	if manager, err := newWithDependencies(valid, dependencies{random: bytes.NewReader(nil), now: nil}); manager != nil || ClassOf(err) != ClassInvalid {
		t.Fatalf("nil clock = %#v, %v", manager, err)
	}

	manager, err := New(valid)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(issued.Challenge())
	if err != nil || len(decoded) != challengeBytes || strings.Contains(issued.Challenge(), "=") {
		t.Fatalf("production challenge shape = %q (%d bytes), %v", issued.Challenge(), len(decoded), err)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueRejectsInvalidRequestWithoutReadingRandom(t *testing.T) {
	now := time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)
	randomValue := bytes.Repeat([]byte{0x33}, challengeBytes)
	reader := &countingReader{reader: bytes.NewReader(randomValue)}
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, reader, func() time.Time { return now })

	cases := []struct {
		name   string
		mutate func(*Request)
	}{
		{"empty request", func(r *Request) { r.RequestID = "" }},
		{"long request", func(r *Request) { r.RequestID = strings.Repeat("r", maxRequestIDBytes+1) }},
		{"request whitespace", func(r *Request) { r.RequestID = " request" }},
		{"request unicode", func(r *Request) { r.RequestID = "request-é" }},
		{"digest label", func(r *Request) { r.Digest = "sha512:" + strings.Repeat("a", 64) }},
		{"digest short", func(r *Request) { r.Digest = "sha256:abcd" }},
		{"digest uppercase", func(r *Request) { r.Digest = "sha256:" + strings.Repeat("A", 64) }},
		{"digest non hex", func(r *Request) { r.Digest = "sha256:" + strings.Repeat("z", 64) }},
		{"decision empty", func(r *Request) { r.Decision = "" }},
		{"decision spelling", func(r *Request) { r.Decision = "approved" }},
		{"operator empty", func(r *Request) { r.OperatorID = "" }},
		{"operator long", func(r *Request) { r.OperatorID = strings.Repeat("o", maxOperatorIDBytes+1) }},
		{"operator control", func(r *Request) { r.OperatorID = "operator\nsecret" }},
		{"rp uppercase", func(r *Request) { r.RPID = "Approval.example.com"; r.Origin = "https://Approval.example.com" }},
		{"rp ip", func(r *Request) { r.RPID = "192.0.2.1"; r.Origin = "https://192.0.2.1" }},
		{"rp leading hyphen", func(r *Request) { r.RPID = "-approval.example.com"; r.Origin = "https://-approval.example.com" }},
		{"rp trailing hyphen", func(r *Request) { r.RPID = "approval-.example.com"; r.Origin = "https://approval-.example.com" }},
		{"rp empty label", func(r *Request) { r.RPID = "approval..example.com"; r.Origin = "https://approval..example.com" }},
		{"rp trailing dot", func(r *Request) { r.RPID = "approval.example.com."; r.Origin = "https://approval.example.com." }},
		{"rp unicode", func(r *Request) { r.RPID = "äpproval.example.com"; r.Origin = "https://äpproval.example.com" }},
		{"origin http", func(r *Request) { r.Origin = "http://approval.example.com" }},
		{"origin path", func(r *Request) { r.Origin += "/" }},
		{"origin query", func(r *Request) { r.Origin += "?x=1" }},
		{"origin fragment", func(r *Request) { r.Origin += "#x" }},
		{"origin port", func(r *Request) { r.Origin += ":443" }},
		{"origin user", func(r *Request) { r.Origin = "https://u@approval.example.com" }},
		{"origin subdomain", func(r *Request) { r.Origin = "https://phone.approval.example.com" }},
		{"origin suffix", func(r *Request) { r.Origin = "https://example.com" }},
		{"origin escaped", func(r *Request) { r.Origin = "https://approval%2eexample.com" }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := validTestRequest()
			test.mutate(&request)
			issued, err := manager.Issue(request)
			if issued != (Issued{}) || ClassOf(err) != ClassInvalid {
				t.Fatalf("Issue = %#v, %v", issued, err)
			}
			assertDoesNotLeak(t, err, requestValues(request)...)
		})
	}
	if reader.reads != 0 || len(manager.pending) != 0 {
		t.Fatalf("invalid requests changed state: reads=%d pending=%d", reader.reads, len(manager.pending))
	}
	issued, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	if issued.Challenge() != base64.RawURLEncoding.EncodeToString(randomValue) || reader.reads != 1 {
		t.Fatalf("random was consumed before valid issue: challenge=%q reads=%d", issued.Challenge(), reader.reads)
	}
}

func TestIssueRandomFailureCollisionAndCapacity(t *testing.T) {
	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	rules := Rules{TTL: time.Minute, MaxPending: 2}
	for _, test := range []struct {
		name   string
		reader io.Reader
	}{
		{"short", bytes.NewReader(bytes.Repeat([]byte{1}, challengeBytes-1))},
		{"error", errorReader{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := mustTestManager(t, rules, test.reader, func() time.Time { return now })
			issued, err := manager.Issue(validTestRequest())
			if issued != (Issued{}) || ClassOf(err) != ClassRandom || len(manager.pending) != 0 {
				t.Fatalf("Issue = %#v, %v, pending=%d", issued, err, len(manager.pending))
			}
			assertDoesNotLeak(t, err, requestValues(validTestRequest())...)
		})
	}

	collisionBytes := bytes.Repeat([]byte{0x44}, challengeBytes*(challengeAttempts+1))
	manager := mustTestManager(t, rules, bytes.NewReader(collisionBytes), func() time.Time { return now })
	first, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.Issue(validTestRequest())
	if second != (Issued{}) || ClassOf(err) != ClassInternal || len(manager.pending) != 1 {
		t.Fatalf("collision Issue = %#v, %v, pending=%d", second, err, len(manager.pending))
	}
	assertDoesNotLeak(t, err, first.Challenge())

	capacity := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(bytes.Repeat([]byte{0x55}, challengeBytes*2)), func() time.Time { return now })
	if _, err := capacity.Issue(validTestRequest()); err != nil {
		t.Fatal(err)
	}
	if issued, err := capacity.Issue(validTestRequest()); issued != (Issued{}) || ClassOf(err) != ClassCapacity || len(capacity.pending) != 1 {
		t.Fatalf("capacity Issue = %#v, %v", issued, err)
	}
}

func TestConsumeCopiesInputsAndReturnsOnlyStableEvidence(t *testing.T) {
	now := time.Date(2026, 8, 2, 11, 0, 0, 123456789, time.UTC)
	clock := newTestClock(now)
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(bytes.Repeat([]byte{0x66}, challengeBytes)), clock.Now)
	request := validTestRequest()
	issued, err := manager.Issue(request)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Binding().IssuedAt().Nanosecond() != 0 {
		t.Fatal("public issued time was not normalized")
	}

	assertion := []byte("assertion-owned-by-caller")
	originalAssertion := append([]byte(nil), assertion...)
	credential := []byte("credential-owned-by-verifier")
	originalCredential := append([]byte(nil), credential...)
	verified, err := manager.Consume(issued.Challenge(), assertion, func(binding Binding, copyOfAssertion []byte) ([]byte, error) {
		if binding != issued.Binding() {
			t.Fatal("binding changed")
		}
		copyOfAssertion[0] = 'X'
		return credential, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(assertion, originalAssertion) {
		t.Fatal("verifier mutated caller assertion")
	}
	expectedCredentialID := stableCredentialID(originalCredential)
	for index := range credential {
		credential[index] = 'X'
	}
	for index := range assertion {
		assertion[index] = 'Y'
	}
	if verified.CredentialID() != expectedCredentialID || verified.VerifiedAt().Nanosecond() != 0 {
		t.Fatal("verified evidence followed caller-owned mutation")
	}
	representation := fmt.Sprintf("%#v", verified)
	for _, secret := range []string{issued.Challenge(), string(originalAssertion), string(originalCredential)} {
		if strings.Contains(representation, secret) {
			t.Fatalf("verified result leaked %q", secret)
		}
	}
	if len(manager.pending) != 0 {
		t.Fatal("manager retained consumed challenge")
	}
}

func TestConsumeFailurePanicAndInvalidVerifierOutputAreOneShot(t *testing.T) {
	tests := []struct {
		name      string
		assertion []byte
		verifier  Verifier
		leaks     []string
	}{
		{"error", []byte("assertion-error-secret"), func(Binding, []byte) ([]byte, error) { return nil, errors.New("lower-verifier-secret") }, []string{"assertion-error-secret", "lower-verifier-secret"}},
		{"panic", []byte("assertion-panic-secret"), func(Binding, []byte) ([]byte, error) { panic("panic-value-secret") }, []string{"assertion-panic-secret", "panic-value-secret"}},
		{"nil verifier", []byte("assertion-nil-secret"), nil, []string{"assertion-nil-secret"}},
		{"empty assertion", nil, func(Binding, []byte) ([]byte, error) { t.Fatal("verifier called"); return nil, nil }, nil},
		{"large assertion", make([]byte, maxAssertionBytes+1), func(Binding, []byte) ([]byte, error) { t.Fatal("verifier called"); return nil, nil }, nil},
		{"empty credential", []byte("assertion-empty-credential"), func(Binding, []byte) ([]byte, error) { return nil, nil }, nil},
		{"large credential", []byte("assertion-large-credential"), func(Binding, []byte) ([]byte, error) { return make([]byte, maxCredentialIDBytes+1), nil }, nil},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			now := time.Date(2026, 8, 2, 12, index, 0, 0, time.UTC)
			manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(bytes.Repeat([]byte{byte(index + 1)}, challengeBytes)), func() time.Time { return now })
			issued, err := manager.Issue(validTestRequest())
			if err != nil {
				t.Fatal(err)
			}
			verified, err := manager.Consume(issued.Challenge(), test.assertion, test.verifier)
			if verified != (Verified{}) || ClassOf(err) != ClassVerification {
				t.Fatalf("Consume = %#v, %v", verified, err)
			}
			assertDoesNotLeak(t, err, append(test.leaks, issued.Challenge())...)
			if _, err := manager.Consume(issued.Challenge(), []byte("retry"), successfulVerifier); ClassOf(err) != ClassNotFound {
				t.Fatalf("replay class = %q, err=%v", ClassOf(err), err)
			}
		})
	}
}

func TestConsumeUnknownAndReplayNeverInvokeVerifier(t *testing.T) {
	now := time.Date(2026, 8, 2, 13, 0, 0, 0, time.UTC)
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(bytes.Repeat([]byte{0x77}, challengeBytes)), func() time.Time { return now })
	var calls atomic.Int32
	verifier := func(Binding, []byte) ([]byte, error) {
		calls.Add(1)
		return []byte("credential"), nil
	}
	if _, err := manager.Consume("unknown-secret-token", []byte("assertion"), verifier); ClassOf(err) != ClassNotFound {
		t.Fatalf("unknown class = %q", ClassOf(err))
	} else {
		assertDoesNotLeak(t, err, "unknown-secret-token")
	}
	issued, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Consume(issued.Challenge(), []byte("assertion"), verifier); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Consume(issued.Challenge(), []byte("assertion"), verifier); ClassOf(err) != ClassNotFound {
		t.Fatalf("replay class = %q", ClassOf(err))
	}
	if calls.Load() != 1 {
		t.Fatalf("verifier calls = %d", calls.Load())
	}
}

func TestExpiryPriorityAndCapacityPurge(t *testing.T) {
	base := time.Date(2026, 8, 2, 14, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		offset time.Duration
		want   ErrorClass
	}{
		{"before", time.Minute - time.Nanosecond, ""},
		{"exact", time.Minute, ClassExpired},
		{"after", time.Minute + time.Nanosecond, ClassExpired},
	} {
		t.Run(test.name, func(t *testing.T) {
			clock := newTestClock(base)
			manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(bytes.Repeat([]byte{0x81}, challengeBytes)), clock.Now)
			issued, err := manager.Issue(validTestRequest())
			if err != nil {
				t.Fatal(err)
			}
			clock.Set(base.Add(test.offset))
			var calls atomic.Int32
			_, err = manager.Consume(issued.Challenge(), []byte("assertion"), func(Binding, []byte) ([]byte, error) {
				calls.Add(1)
				return []byte("credential"), nil
			})
			if ClassOf(err) != test.want {
				t.Fatalf("class = %q, want %q, err=%v", ClassOf(err), test.want, err)
			}
			wantCalls := int32(1)
			if test.want != "" {
				wantCalls = 0
			}
			if calls.Load() != wantCalls {
				t.Fatalf("verifier calls=%d, want %d", calls.Load(), wantCalls)
			}
		})
	}

	clock := newTestClock(base)
	random := append(bytes.Repeat([]byte{0x82}, challengeBytes), bytes.Repeat([]byte{0x83}, challengeBytes)...)
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(random), clock.Now)
	old, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	clock.Set(base.Add(time.Minute))
	fresh, err := manager.Issue(validTestRequest())
	if err != nil || fresh.Challenge() == old.Challenge() {
		t.Fatalf("due-first capacity recovery = %#v, %v", fresh, err)
	}
	if _, err := manager.Consume(old.Challenge(), []byte("assertion"), successfulVerifier); ClassOf(err) != ClassNotFound {
		t.Fatalf("purged old class = %q", ClassOf(err))
	}
}

func TestClockRollbackBeforeAndAfterVerifier(t *testing.T) {
	base := time.Date(2026, 8, 2, 15, 0, 0, 900_000_000, time.UTC)
	clock := newTestClock(base)
	random := append(bytes.Repeat([]byte{0x91}, challengeBytes), bytes.Repeat([]byte{0x92}, challengeBytes)...)
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 2}, bytes.NewReader(random), clock.Now)
	issued, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	clock.Set(base.Add(-100 * time.Millisecond))
	if next, err := manager.Issue(validTestRequest()); next != (Issued{}) || ClassOf(err) != ClassClock {
		t.Fatalf("same-second rollback Issue = %#v, %v", next, err)
	}
	if _, err := manager.Consume(issued.Challenge(), []byte("assertion"), successfulVerifier); ClassOf(err) != ClassClock {
		t.Fatalf("same-second rollback Consume class = %q", ClassOf(err))
	}
	clock.Set(base)
	if _, err := manager.Consume(issued.Challenge(), []byte("assertion"), successfulVerifier); err != nil {
		t.Fatalf("challenge changed during pre-reservation rollback: %v", err)
	}

	clock.Set(base.Add(time.Second))
	second, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Consume(second.Challenge(), []byte("assertion"), func(Binding, []byte) ([]byte, error) {
		clock.Set(base.Add(500 * time.Millisecond))
		return []byte("credential"), nil
	})
	if ClassOf(err) != ClassClock {
		t.Fatalf("post-verifier rollback class = %q, err=%v", ClassOf(err), err)
	}
	clock.Set(base.Add(time.Second))
	if _, err := manager.Consume(second.Challenge(), []byte("assertion"), successfulVerifier); ClassOf(err) != ClassNotFound {
		t.Fatalf("post-verifier rollback replay class = %q", ClassOf(err))
	}
}

func TestVerifierCanReenterManagerWithoutReusingReservedChallenge(t *testing.T) {
	now := time.Date(2026, 8, 2, 16, 0, 0, 0, time.UTC)
	random := append(bytes.Repeat([]byte{0xa1}, challengeBytes), bytes.Repeat([]byte{0xa2}, challengeBytes)...)
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 2}, bytes.NewReader(random), func() time.Time { return now })
	issued, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	verified, err := manager.Consume(issued.Challenge(), []byte("assertion"), func(Binding, []byte) ([]byte, error) {
		if _, err := manager.Consume(issued.Challenge(), []byte("nested"), successfulVerifier); ClassOf(err) != ClassNotFound {
			t.Fatalf("nested replay class = %q", ClassOf(err))
		}
		if _, err := manager.Issue(validTestRequest()); err != nil {
			t.Fatalf("reentrant Issue: %v", err)
		}
		return []byte("credential"), nil
	})
	if err != nil || verified.RequestID() != validTestRequest().RequestID || len(manager.pending) != 1 {
		t.Fatalf("outer Consume = %#v, %v, pending=%d", verified, err, len(manager.pending))
	}
}

func TestConcurrentConsumeRunsVerifierOnce(t *testing.T) {
	now := time.Date(2026, 8, 2, 17, 0, 0, 0, time.UTC)
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(bytes.Repeat([]byte{0xb1}, challengeBytes)), func() time.Time { return now })
	issued, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	const consumers = 16
	start := make(chan struct{})
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	results := make(chan error, consumers)
	var calls atomic.Int32
	verifier := func(Binding, []byte) ([]byte, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-release
		return []byte("credential"), nil
	}
	var group sync.WaitGroup
	for index := 0; index < consumers; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := manager.Consume(issued.Challenge(), []byte("assertion"), verifier)
			results <- err
		}()
	}
	close(start)
	waitSignal(t, entered, "verifier entry")
	close(release)
	group.Wait()
	close(results)
	successes, notFound := 0, 0
	for err := range results {
		switch ClassOf(err) {
		case "":
			successes++
		case ClassNotFound:
			notFound++
		default:
			t.Fatalf("unexpected class %q: %v", ClassOf(err), err)
		}
	}
	if successes != 1 || notFound != consumers-1 || calls.Load() != 1 {
		t.Fatalf("success=%d not_found=%d calls=%d", successes, notFound, calls.Load())
	}
}

func TestCloseDuringVerifierSuppressesResultAndRestartRejectsOldChallenge(t *testing.T) {
	now := time.Date(2026, 8, 2, 18, 0, 0, 0, time.UTC)
	random := append(bytes.Repeat([]byte{0xc1}, challengeBytes), bytes.Repeat([]byte{0xc2}, challengeBytes)...)
	manager := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 2}, bytes.NewReader(random), func() time.Time { return now })
	inFlight, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	pending, err := manager.Issue(validTestRequest())
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := manager.Consume(inFlight.Challenge(), []byte("assertion"), func(Binding, []byte) ([]byte, error) {
			close(entered)
			<-release
			return []byte("credential"), nil
		})
		result <- err
	}()
	waitSignal(t, entered, "in-flight verifier")
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if len(manager.pending) != 0 {
		t.Fatal("Close retained pending challenges")
	}
	if _, err := manager.Issue(validTestRequest()); ClassOf(err) != ClassClosed {
		t.Fatalf("Issue after Close class = %q", ClassOf(err))
	}
	if _, err := manager.Consume(pending.Challenge(), []byte("assertion"), successfulVerifier); ClassOf(err) != ClassClosed {
		t.Fatalf("Consume after Close class = %q", ClassOf(err))
	}
	close(release)
	select {
	case err := <-result:
		if ClassOf(err) != ClassClosed {
			t.Fatalf("in-flight result class = %q, err=%v", ClassOf(err), err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight verifier did not finish")
	}

	restarted := mustTestManager(t, Rules{TTL: time.Minute, MaxPending: 1}, bytes.NewReader(bytes.Repeat([]byte{0xc3}, challengeBytes)), func() time.Time { return now })
	if _, err := restarted.Consume(inFlight.Challenge(), []byte("assertion"), successfulVerifier); ClassOf(err) != ClassNotFound {
		t.Fatalf("new manager accepted old challenge: %v", err)
	}
}

func TestNilAndFixedErrorBehavior(t *testing.T) {
	var manager *Manager
	if _, err := manager.Issue(validTestRequest()); ClassOf(err) != ClassClosed {
		t.Fatalf("nil Issue class = %q", ClassOf(err))
	}
	if _, err := manager.Consume("secret-challenge", []byte("secret-assertion"), successfulVerifier); ClassOf(err) != ClassClosed {
		t.Fatalf("nil Consume class = %q", ClassOf(err))
	} else {
		assertDoesNotLeak(t, err, "secret-challenge", "secret-assertion")
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	if ClassOf(nil) != "" || ClassOf(errors.New("lower-secret")) != ClassInternal {
		t.Fatal("ClassOf contract mismatch")
	}
	var nilError *Error
	if nilError.Error() != "approval challenge error" || nilError.Class() != "" {
		t.Fatal("nil Error contract mismatch")
	}
}

type countingReader struct {
	reader io.Reader
	reads  int
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	r.reads++
	return r.reader.Read(buffer)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("random-source-secret") }

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock(now time.Time) *testClock { return &testClock{now: now} }

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

func mustTestManager(t *testing.T, rules Rules, random io.Reader, now func() time.Time) *Manager {
	t.Helper()
	manager, err := newWithDependencies(rules, dependencies{random: random, now: now})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func successfulVerifier(Binding, []byte) ([]byte, error) {
	return []byte("credential"), nil
}

func requestValues(request Request) []string {
	return []string{request.RequestID, request.Digest, string(request.Decision), request.OperatorID, request.RPID, request.Origin}
}

func assertDoesNotLeak(t *testing.T, err error, values ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, value := range values {
		if value != "" && strings.Contains(err.Error(), value) {
			t.Fatalf("error %q leaked %q", err, value)
		}
	}
}

func waitSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", description)
	}
}
