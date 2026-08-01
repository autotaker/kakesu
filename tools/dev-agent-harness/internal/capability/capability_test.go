package capability

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

func testRules() Rules {
	return Rules{PolicyVersion: "policy-v1", MaxTTL: time.Hour, MaxUses: 3, InitialRevocationEpoch: 7}
}

func testSpec(provider string) IssueSpec {
	return IssueSpec{
		AgentInstanceID: "agent-1",
		UID:             1000,
		WorkspaceID:     "workspace-1",
		Provider:        provider,
		Repository:      "acme/widget",
		TTL:             10 * time.Minute,
		Uses:            2,
	}
}

type repeatingReader struct {
	chunks [][]byte
	index  int
}

func (r *repeatingReader) Read(p []byte) (int, error) {
	if r.index >= len(r.chunks) {
		return 0, io.EOF
	}
	chunk := r.chunks[r.index]
	r.index++
	copy(p, chunk)
	return len(chunk), nil
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func newTestRegistry(t *testing.T, reader io.Reader, now *time.Time) *Registry {
	t.Helper()
	r, err := newWithDeps(testRules(), reader, func() time.Time { return *now })
	if err != nil {
		t.Fatalf("newWithDeps: %v", err)
	}
	return r
}

func fixedChunk(value byte) []byte { return bytes.Repeat([]byte{value}, 32) }

func mustIssue(t *testing.T, r *Registry, spec IssueSpec) string {
	t.Helper()
	handle, err := r.Issue(spec)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return handle
}

func githubRequest(handle string) Request {
	return Request{Handle: handle, AgentInstanceID: "agent-1", UID: 1000, WorkspaceID: "workspace-1", Provider: ProviderGitHub, Repository: "acme/widget", Operation: OperationGitHubRESTRead, DestinationHost: HostGitHub}
}

func openAIRequest(handle string) Request {
	return Request{Handle: handle, AgentInstanceID: "agent-1", UID: 1000, WorkspaceID: "workspace-1", Provider: ProviderOpenAI, Operation: OperationOpenAIResponsesText, DestinationHost: HostOpenAI}
}

func TestRulesAndIssueScope(t *testing.T) {
	for _, rules := range []Rules{
		{},
		{PolicyVersion: "bad value", MaxTTL: time.Hour, MaxUses: 1},
		{PolicyVersion: "p", MaxTTL: 24*time.Hour + time.Nanosecond, MaxUses: 1},
		{PolicyVersion: "p", MaxTTL: time.Hour, MaxUses: maxRulesUses + 1},
	} {
		if r, err := New(rules); r != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("New(%#v) = (%p, %v), want fixed Rules error", rules, r, err)
		}
	}

	clock := fixedTime
	r := newTestRegistry(t, &repeatingReader{chunks: [][]byte{fixedChunk(1), fixedChunk(2)}}, &clock)
	for _, spec := range []IssueSpec{
		{AgentInstanceID: "agent", UID: 0, WorkspaceID: "workspace", Provider: ProviderOpenAI, TTL: time.Minute, Uses: 1},
		{AgentInstanceID: "agent", UID: 1, WorkspaceID: "workspace", Provider: "other", TTL: time.Minute, Uses: 1},
		{AgentInstanceID: "agent", UID: 1, WorkspaceID: "workspace", Provider: ProviderGitHub, Repository: "ACME/widget", TTL: time.Minute, Uses: 1},
		{AgentInstanceID: "agent", UID: 1, WorkspaceID: "workspace", Provider: ProviderOpenAI, Repository: "acme/widget", TTL: time.Minute, Uses: 1},
		{AgentInstanceID: "agent", UID: 1, WorkspaceID: "workspace", Provider: ProviderOpenAI, TTL: 2 * time.Hour, Uses: 1},
	} {
		if handle, err := r.Issue(spec); handle != "" || !errors.Is(err, ErrIssue) {
			t.Fatalf("Issue(%#v) = (%q, %v), want fixed issue error", spec, handle, err)
		}
	}

	githubHandle := mustIssue(t, r, testSpec(ProviderGitHub))
	openAISpec := testSpec(ProviderOpenAI)
	openAISpec.Repository = ""
	openAIHandle := mustIssue(t, r, openAISpec)
	if !strings.HasPrefix(githubHandle, "cap_") || len(githubHandle) != 47 || githubHandle == openAIHandle {
		t.Fatalf("unexpected opaque handles: %q %q", githubHandle, openAIHandle)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(githubHandle, "cap_"))
	if err != nil || len(raw) != 32 || strings.Contains(githubHandle, "=") {
		t.Fatalf("handle is not unpadded base64url: %q", githubHandle)
	}
	if _, ok := r.entries[sha256.Sum256([]byte(githubHandle))]; !ok {
		t.Fatal("issued handle digest missing")
	}
	for _, e := range r.entries {
		if bytes.Contains([]byte(e.scope.agentInstanceID), []byte(githubHandle)) {
			t.Fatal("raw handle retained in entry")
		}
	}
}

func TestProviderScopesConsumeAndMismatchDoesNotSpend(t *testing.T) {
	clock := fixedTime
	r := newTestRegistry(t, &repeatingReader{chunks: [][]byte{fixedChunk(1), fixedChunk(2), fixedChunk(3)}}, &clock)
	gh := mustIssue(t, r, testSpec(ProviderGitHub))
	r.policyVersion = "policy-v2"
	if _, err := r.Consume(githubRequest(gh)); !errors.Is(err, ErrDenied) {
		t.Fatalf("policy mismatch error = %v", err)
	}
	r.policyVersion = "policy-v1"
	gh = mustIssue(t, r, testSpec(ProviderGitHub))
	grant, err := r.Consume(githubRequest(gh))
	if err != nil || grant.Operation != OperationGitHubRESTRead || grant.DestinationHost != HostGitHub || grant.RemainingUses != 1 || grant.RevocationEpoch != 7 {
		t.Fatalf("GitHub consume = (%#v, %v)", grant, err)
	}
	mismatch := githubRequest(gh)
	mismatch.DestinationHost = "evil.example"
	if _, err := r.Consume(mismatch); !errors.Is(err, ErrDenied) {
		t.Fatalf("scope mismatch error = %v", err)
	}
	grant, err = r.Consume(githubRequest(gh))
	if err != nil || grant.RemainingUses != 0 {
		t.Fatalf("remaining consume = (%#v, %v)", grant, err)
	}
	if _, err := r.Consume(githubRequest(gh)); !errors.Is(err, ErrDenied) {
		t.Fatal("last-use handle was reusable")
	}

	openAISpec := testSpec(ProviderOpenAI)
	openAISpec.Repository = ""
	oi := mustIssue(t, r, openAISpec)
	grant, err = r.Consume(openAIRequest(oi))
	if err != nil || grant.Provider != ProviderOpenAI || grant.Repository != "" || grant.Operation != OperationOpenAIResponsesText || grant.DestinationHost != HostOpenAI {
		t.Fatalf("OpenAI consume = (%#v, %v)", grant, err)
	}
	for _, bad := range []Request{
		openAIRequest("cap_bad"),
		func() Request { req := openAIRequest(oi); req.Provider = "github"; return req }(),
		func() Request { req := openAIRequest(oi); req.Operation += "-extra"; return req }(),
	} {
		if _, err := r.Consume(bad); !errors.Is(err, ErrDenied) {
			t.Fatalf("bad request %#v error = %v", bad, err)
		}
	}
}

func TestExpiryRevokeAndEpoch(t *testing.T) {
	clock := fixedTime
	r := newTestRegistry(t, &repeatingReader{chunks: [][]byte{fixedChunk(3), fixedChunk(4), fixedChunk(5)}}, &clock)
	spec := testSpec(ProviderGitHub)
	spec.Uses = 1
	handle := mustIssue(t, r, spec)
	clock = fixedTime.Add(10 * time.Minute)
	if _, err := r.Consume(githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("expiry boundary error = %v", err)
	}
	if err := r.Revoke(handle); !errors.Is(err, ErrDenied) {
		t.Fatalf("expired revoke error = %v", err)
	}

	clock = fixedTime
	second := mustIssue(t, r, testSpec(ProviderGitHub))
	if err := r.Revoke(second); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := r.Consume(githubRequest(second)); !errors.Is(err, ErrDenied) {
		t.Fatal("revoked handle consumed")
	}

	third := mustIssue(t, r, testSpec(ProviderGitHub))
	if err := r.AdvanceRevocationEpoch(8); err != nil {
		t.Fatalf("AdvanceRevocationEpoch: %v", err)
	}
	if _, err := r.Consume(githubRequest(third)); !errors.Is(err, ErrDenied) {
		t.Fatal("epoch-invalidated handle consumed")
	}
	for _, epoch := range []uint64{8, 7} {
		if err := r.AdvanceRevocationEpoch(epoch); !errors.Is(err, ErrDenied) {
			t.Fatalf("epoch %d error = %v", epoch, err)
		}
	}
}

func TestEntropyAndCollisionAreBounded(t *testing.T) {
	clock := fixedTime
	r := newTestRegistry(t, failingReader{}, &clock)
	if handle, err := r.Issue(testSpec(ProviderGitHub)); handle != "" || !errors.Is(err, ErrIssue) || len(r.entries) != 0 {
		t.Fatalf("entropy failure = (%q, %v), entries=%d", handle, err, len(r.entries))
	}

	chunk := fixedChunk(9)
	reader := &repeatingReader{chunks: [][]byte{chunk, chunk, chunk, chunk, chunk, fixedChunk(10)}}
	r = newTestRegistry(t, reader, &clock)
	first := mustIssue(t, r, testSpec(ProviderGitHub))
	if first == "" || len(r.entries) != 1 {
		t.Fatalf("first collision fixture issue failed: %q entries=%d", first, len(r.entries))
	}
	if handle, err := r.Issue(testSpec(ProviderGitHub)); handle == "" || err != nil || len(r.entries) != 2 {
		t.Fatalf("collision retry did not recover: %q %v entries=%d", handle, err, len(r.entries))
	}

	reader = &repeatingReader{chunks: [][]byte{chunk, chunk, chunk, chunk, chunk, chunk}}
	r = newTestRegistry(t, reader, &clock)
	mustIssue(t, r, testSpec(ProviderGitHub))
	if handle, err := r.Issue(testSpec(ProviderGitHub)); handle != "" || !errors.Is(err, ErrIssue) || len(r.entries) != 1 {
		t.Fatalf("bounded collision = (%q, %v), entries=%d", handle, err, len(r.entries))
	}
}

func TestConcurrentSingleUse(t *testing.T) {
	clock := fixedTime
	r := newTestRegistry(t, &repeatingReader{chunks: [][]byte{fixedChunk(11)}}, &clock)
	spec := testSpec(ProviderGitHub)
	spec.Uses = 1
	handle := mustIssue(t, r, spec)
	const workers = 16
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.Consume(githubRequest(handle))
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrDenied) {
			t.Fatalf("concurrent consume error = %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent successes = %d, want 1", successes)
	}
}

func TestFixedErrorsDoNotLeakAndInputsStayUnchanged(t *testing.T) {
	clock := fixedTime
	r := newTestRegistry(t, &repeatingReader{chunks: [][]byte{fixedChunk(12)}}, &clock)
	spec := testSpec(ProviderGitHub)
	before := spec
	handle := mustIssue(t, r, spec)
	if spec != before {
		t.Fatal("Issue modified IssueSpec")
	}
	req := githubRequest(handle)
	req.AgentInstanceID = "credential-not-in-error"
	reqBefore := req
	_, err := r.Consume(req)
	if !errors.Is(err, ErrDenied) || strings.Contains(err.Error(), "credential-not-in-error") {
		t.Fatalf("fixed deny leaked input: %v", err)
	}
	if req != reqBefore {
		t.Fatal("Consume modified Request")
	}
	if _, err := (*Registry)(nil).Consume(githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("nil registry error = %v", err)
	}
	if err := (*Registry)(nil).AdvanceRevocationEpoch(1); !errors.Is(err, ErrDenied) {
		t.Fatalf("nil registry epoch error = %v", err)
	}
	if err := (&Registry{}).AdvanceRevocationEpoch(1); !errors.Is(err, ErrDenied) {
		t.Fatalf("zero registry epoch error = %v", err)
	}
}

func TestProductionLifecycleKeepsMonotonicInternalTimes(t *testing.T) {
	r, err := New(testRules())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	handle := mustIssue(t, r, testSpec(ProviderGitHub))
	digest, ok := digestForHandle(handle)
	if !ok {
		t.Fatal("issued handle was not canonical")
	}
	e, ok := r.entries[digest]
	if !ok {
		t.Fatal("issued entry missing")
	}
	if !strings.Contains(e.issuedAt.String(), "m=") || !strings.Contains(e.expiresAt.String(), "m=") {
		t.Fatalf("production lifecycle lost monotonic reading: issued=%q expires=%q", e.issuedAt, e.expiresAt)
	}
	grant, err := r.Consume(githubRequest(handle))
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if grant.IssuedAt.Location() != time.UTC || grant.ExpiresAt.Location() != time.UTC ||
		strings.Contains(grant.IssuedAt.String(), "m=") || strings.Contains(grant.ExpiresAt.String(), "m=") {
		t.Fatalf("Grant times were not UTC wall-clock values: issued=%q expires=%q", grant.IssuedAt, grant.ExpiresAt)
	}
	if !grant.IssuedAt.Equal(e.issuedAt.UTC()) || !grant.ExpiresAt.Equal(e.expiresAt.UTC()) {
		t.Fatal("Grant times do not match the lifecycle entry")
	}
}

func TestMonotonicExpirySurvivesWallClockRollback(t *testing.T) {
	reading := clockReading{wall: fixedTime, elapsed: 0}
	r, err := newWithClockDeps(testRules(), &repeatingReader{chunks: [][]byte{fixedChunk(13)}}, func() clockReading { return reading })
	if err != nil {
		t.Fatalf("newWithClockDeps: %v", err)
	}
	handle := mustIssue(t, r, testSpec(ProviderGitHub))
	reading.wall = fixedTime.Add(-time.Hour)
	reading.elapsed = 10 * time.Minute
	if _, err := r.Consume(githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("monotonic expiry after wall rollback = %v", err)
	}
	reading.wall = fixedTime
	if _, err := r.Consume(githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("expired handle revived after wall restoration: %v", err)
	}

	reading = clockReading{wall: fixedTime, elapsed: 0}
	r, err = newWithClockDeps(testRules(), &repeatingReader{chunks: [][]byte{fixedChunk(14)}}, func() clockReading { return reading })
	if err != nil {
		t.Fatalf("newWithClockDeps (wall-forward): %v", err)
	}
	handle = mustIssue(t, r, testSpec(ProviderGitHub))
	reading.wall = fixedTime.Add(11 * time.Minute)
	reading.elapsed = time.Minute
	if _, err := r.Consume(githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("wall-forward expiry was not fail-closed: %v", err)
	}
	reading.wall = fixedTime
	if _, err := r.Consume(githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("wall-forward expired handle revived: %v", err)
	}
}
