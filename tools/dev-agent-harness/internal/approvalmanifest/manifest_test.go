package approvalmanifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	oldA = "1111111111111111111111111111111111111111"
	newA = "2222222222222222222222222222222222222222"
	oldB = "3333333333333333333333333333333333333333"
	newB = "4444444444444444444444444444444444444444"
)

func validProposal() Proposal {
	return Proposal{
		RequestID: "request-001", AgentID: "agent-001", WorkspaceID: "workspace-001",
		Repository: "openai/agent-harness", Remote: "https://github.com/openai/agent-harness.git",
		RefUpdates: []RefUpdate{
			{Ref: "refs/heads/create", ExpectedOldSHA: zeroObjectID, NewSHA: newA},
			{Ref: "refs/heads/update", ExpectedOldSHA: oldA, NewSHA: newA},
			{Ref: "refs/heads/force", ExpectedOldSHA: oldB, NewSHA: newB, Force: true},
			{Ref: "refs/heads/delete", ExpectedOldSHA: oldA, NewSHA: zeroObjectID, Delete: true},
		},
		PolicyVersion: "push-v1", RevocationEpoch: 7,
		CreatedAt: time.Date(2026, 8, 2, 6, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 8, 2, 6, 5, 0, 0, time.UTC),
	}
}

func TestBuildCanonicalGoldenAndRoundTrip(t *testing.T) {
	p := validProposal()
	m, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"format_version":1,"request_id":"request-001","agent_id":"agent-001","workspace_id":"workspace-001","repository":"openai/agent-harness","remote":"https://github.com/openai/agent-harness.git","ref_updates":[{"ref":"refs/heads/create","expected_old_sha":"0000000000000000000000000000000000000000","new_sha":"2222222222222222222222222222222222222222","force":false,"delete":false},{"ref":"refs/heads/update","expected_old_sha":"1111111111111111111111111111111111111111","new_sha":"2222222222222222222222222222222222222222","force":false,"delete":false},{"ref":"refs/heads/force","expected_old_sha":"3333333333333333333333333333333333333333","new_sha":"4444444444444444444444444444444444444444","force":true,"delete":false},{"ref":"refs/heads/delete","expected_old_sha":"1111111111111111111111111111111111111111","new_sha":"0000000000000000000000000000000000000000","force":false,"delete":true}],"policy_version":"push-v1","revocation_epoch":7,"created_at":"2026-08-02T06:00:00Z","expires_at":"2026-08-02T06:05:00Z"}`
	sum := sha256.Sum256(append([]byte("dev-agent-harness/push-approval-manifest/v1\x00"), []byte(payload)...))
	expectedDigest := "sha256:" + hex.EncodeToString(sum[:])
	expectedEncoding := payload[:len(payload)-1] + `,"request_digest":"` + expectedDigest + `"}`
	if m.Digest() != expectedDigest {
		t.Fatalf("digest mismatch: %q", m.Digest())
	}
	if string(m.Encoding()) != expectedEncoding {
		t.Fatalf("encoding mismatch:\n%s", m.Encoding())
	}
	parsed, err := Parse(m.Encoding())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Encoding(), m.Encoding()) || parsed.Digest() != m.Digest() {
		t.Fatal("round trip changed manifest")
	}
	if got := parsed.RefUpdates(); len(got) != 4 || got[0].Ref != p.RefUpdates[0].Ref || got[3].Ref != p.RefUpdates[3].Ref {
		t.Fatal("ref order changed")
	}
	if m.FormatVersion() != 1 || m.RequestID() != p.RequestID || m.AgentID() != p.AgentID || m.WorkspaceID() != p.WorkspaceID || m.Repository() != p.Repository || m.Remote() != p.Remote || m.PolicyVersion() != p.PolicyVersion || m.RevocationEpoch() != 7 || !m.CreatedAt().Equal(p.CreatedAt) || !m.ExpiresAt().Equal(p.ExpiresAt) {
		t.Fatal("scalar getter mismatch")
	}
}

func TestProposalValidationMatrix(t *testing.T) {
	tooMany := make([]RefUpdate, MaxRefUpdates+1)
	for i := range tooMany {
		tooMany[i] = RefUpdate{Ref: fmt.Sprintf("refs/heads/b%d", i), ExpectedOldSHA: zeroObjectID, NewSHA: newA}
	}
	cases := []struct {
		name   string
		field  Field
		index  int
		mutate func(*Proposal)
	}{
		{"request missing", FieldRequestID, -1, func(p *Proposal) { p.RequestID = "" }},
		{"request too long", FieldRequestID, -1, func(p *Proposal) { p.RequestID = strings.Repeat("r", maxRequestIDBytes+1) }},
		{"agent non ascii", FieldAgentID, -1, func(p *Proposal) { p.AgentID = "agent-é" }},
		{"agent too long", FieldAgentID, -1, func(p *Proposal) { p.AgentID = strings.Repeat("a", maxAgentIDBytes+1) }},
		{"workspace nul", FieldWorkspaceID, -1, func(p *Proposal) { p.WorkspaceID = "workspace\x00x" }},
		{"workspace too long", FieldWorkspaceID, -1, func(p *Proposal) { p.WorkspaceID = strings.Repeat("w", maxWorkspaceIDBytes+1) }},
		{"policy too long", FieldPolicyVersion, -1, func(p *Proposal) { p.PolicyVersion = strings.Repeat("p", maxPolicyVersionBytes+1) }},
		{"repository uppercase", FieldRepository, -1, func(p *Proposal) { p.Repository = "OpenAI/agent-harness" }},
		{"repository shape", FieldRepository, -1, func(p *Proposal) { p.Repository = "owner/repo/extra" }},
		{"repository leading punctuation", FieldRepository, -1, func(p *Proposal) { p.Repository = "-owner/repo" }},
		{"repository too long", FieldRepository, -1, func(p *Proposal) { p.Repository = strings.Repeat("o", maxRepositoryBytes) + "/r" }},
		{"remote mismatch", FieldRemote, -1, func(p *Proposal) { p.Remote = "https://github.com/other/repo.git" }},
		{"no refs", FieldRefUpdates, -1, func(p *Proposal) { p.RefUpdates = nil }},
		{"too many refs", FieldRefUpdates, -1, func(p *Proposal) { p.RefUpdates = tooMany }},
		{"duplicate ref", FieldRef, 1, func(p *Proposal) { p.RefUpdates[1].Ref = p.RefUpdates[0].Ref }},
		{"tag ref", FieldRef, 0, func(p *Proposal) { p.RefUpdates[0].Ref = "refs/tags/v1" }},
		{"unsafe ref", FieldRef, 0, func(p *Proposal) { p.RefUpdates[0].Ref = "refs/heads/a..b" }},
		{"ref too long", FieldRef, 0, func(p *Proposal) { p.RefUpdates[0].Ref = "refs/heads/" + strings.Repeat("b", maxRefBytes) }},
		{"uppercase old", FieldExpectedOldSHA, 1, func(p *Proposal) { p.RefUpdates[1].ExpectedOldSHA = strings.Repeat("A", objectIDBytes) }},
		{"short new", FieldNewSHA, 1, func(p *Proposal) { p.RefUpdates[1].NewSHA = "1234" }},
		{"create force", FieldForce, 0, func(p *Proposal) { p.RefUpdates[0].Force = true }},
		{"delete flag missing", FieldDelete, 3, func(p *Proposal) { p.RefUpdates[3].Delete = false }},
		{"delete force", FieldForce, 3, func(p *Proposal) { p.RefUpdates[3].Force = true }},
		{"zero zero", FieldNewSHA, 0, func(p *Proposal) { p.RefUpdates[0].NewSHA = zeroObjectID }},
		{"no op", FieldNewSHA, 1, func(p *Proposal) { p.RefUpdates[1].NewSHA = oldA }},
		{"created location", FieldCreatedAt, -1, func(p *Proposal) { p.CreatedAt = p.CreatedAt.In(time.FixedZone("UTC", 0)) }},
		{"created fraction", FieldCreatedAt, -1, func(p *Proposal) { p.CreatedAt = p.CreatedAt.Add(time.Nanosecond) }},
		{"created out of range", FieldCreatedAt, -1, func(p *Proposal) { p.CreatedAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) }},
		{"expires order", FieldExpiresAt, -1, func(p *Proposal) { p.ExpiresAt = p.CreatedAt }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validProposal()
			tc.mutate(&p)
			_, err := Build(p)
			assertError(t, err, ClassInvalid, tc.field, tc.index)
			assertDoesNotLeak(t, err, proposalValues(p)...)
		})
	}
	for count := 1; count <= MaxRefUpdates; count += MaxRefUpdates - 1 {
		p := validProposal()
		p.RefUpdates = tooMany[:count]
		if _, err := Build(p); err != nil {
			t.Fatalf("%d refs rejected: %v", count, err)
		}
	}
}

func TestDigestBindsEveryValueAndRefOrder(t *testing.T) {
	base := validProposal()
	manifest, _ := Build(base)
	original := manifest.Digest()
	mutations := []func(*Proposal){
		func(p *Proposal) { p.RequestID = "request-002" }, func(p *Proposal) { p.AgentID = "agent-002" },
		func(p *Proposal) { p.WorkspaceID = "workspace-002" },
		func(p *Proposal) {
			p.Repository = "openai/agent-harness-two"
			p.Remote = "https://github.com/openai/agent-harness-two.git"
		},
		func(p *Proposal) { p.PolicyVersion = "push-v2" }, func(p *Proposal) { p.RevocationEpoch++ },
		func(p *Proposal) { p.CreatedAt = p.CreatedAt.Add(-time.Second) }, func(p *Proposal) { p.ExpiresAt = p.ExpiresAt.Add(time.Second) },
		func(p *Proposal) { p.RefUpdates[1].Ref = "refs/heads/update-two" }, func(p *Proposal) { p.RefUpdates[1].ExpectedOldSHA = oldB },
		func(p *Proposal) { p.RefUpdates[1].NewSHA = newB }, func(p *Proposal) { p.RefUpdates[1].Force = true },
		func(p *Proposal) { p.RefUpdates[3].Delete = false; p.RefUpdates[3].NewSHA = newB },
		func(p *Proposal) { p.RefUpdates[0], p.RefUpdates[1] = p.RefUpdates[1], p.RefUpdates[0] },
	}
	for i, mutate := range mutations {
		p := validProposal()
		mutate(&p)
		got, err := Build(p)
		if err != nil {
			t.Fatalf("mutation %d invalid: %v", i, err)
		}
		if got.Digest() == original {
			t.Fatalf("mutation %d did not change digest", i)
		}
	}
	second, _ := Build(base)
	if second.Digest() != original || !bytes.Equal(second.Encoding(), manifest.Encoding()) {
		t.Fatal("same proposal was not deterministic")
	}
}

func TestDigestEncoderBindsDeleteBit(t *testing.T) {
	p := validProposal()
	original, err := buildValidated(p)
	if err != nil {
		t.Fatal(err)
	}
	// The opposite delete bit is intentionally not a valid public proposal.
	// Bypass semantic validation here only to prove the digest encoder binds it.
	p.RefUpdates[3].Delete = false
	mutated, err := buildValidated(p)
	if err != nil {
		t.Fatal(err)
	}
	if mutated.Digest() == original.Digest() {
		t.Fatal("delete bit was omitted from digest")
	}
}

func TestParseRejectsTamperingAndNonCanonicalJSON(t *testing.T) {
	m, _ := Build(validProposal())
	canonical := string(m.Encoding())
	mutations := []struct {
		name  string
		class ErrorClass
		make  func(string) string
	}{
		{"unknown", ClassParse, func(s string) string { return strings.Replace(s, `"request_id":`, `"unknown":1,"request_id":`, 1) }},
		{"duplicate top", ClassDuplicateKey, func(s string) string {
			return strings.Replace(s, `"request_id":`, `"request_id":"request-001","request_id":`, 1)
		}},
		{"duplicate nested", ClassDuplicateKey, func(s string) string {
			return strings.Replace(s, `"ref":"refs/heads/create"`, `"ref":"refs/heads/create","ref":"refs/heads/create"`, 1)
		}},
		{"missing", ClassParse, func(s string) string { return strings.Replace(s, `"agent_id":"agent-001",`, "", 1) }},
		{"wrong nested shape", ClassParse, func(s string) string {
			start := strings.Index(s, `"ref_updates":`)
			end := strings.Index(s[start:], `],"policy_version"`) + start + 1
			return s[:start] + `"ref_updates":{}` + s[end:]
		}},
		{"space", ClassNonCanonical, func(s string) string { return strings.Replace(s, `{"format_version"`, `{ "format_version"`, 1) }},
		{"key order", ClassNonCanonical, func(s string) string {
			return strings.Replace(s, `"request_id":"request-001","agent_id":"agent-001"`, `"agent_id":"agent-001","request_id":"request-001"`, 1)
		}},
		{"escape", ClassNonCanonical, func(s string) string { return strings.Replace(s, "request-001", `request-00\u0031`, 1) }},
		{"number", ClassParse, func(s string) string { return strings.Replace(s, `"revocation_epoch":7`, `"revocation_epoch":7.0`, 1) }},
		{"fractional time", ClassNonCanonical, func(s string) string { return strings.Replace(s, "06:00:00Z", "06:00:00.0Z", 1) }},
		{"offset time", ClassInvalid, func(s string) string { return strings.Replace(s, "06:00:00Z", "06:00:00+00:00", 1) }},
		{"digest tamper", ClassDigest, tamperDigest},
		{"digest uppercase", ClassDigest, func(s string) string {
			marker := `"request_digest":"`
			index := strings.Index(s, marker) + len(marker)
			end := strings.Index(s[index:], `"`) + index
			return s[:index] + strings.ToUpper(s[index:end]) + s[end:]
		}},
		{"trailing", ClassParse, func(s string) string { return s + `{}` }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			raw := tc.make(canonical)
			_, err := Parse([]byte(raw))
			if ClassOf(err) != tc.class {
				t.Fatalf("class=%s err=%v", ClassOf(err), err)
			}
			values := append(proposalValues(validProposal()), raw, canonical, m.Digest())
			assertDoesNotLeak(t, err, values...)
		})
	}
	oversized := bytes.Repeat([]byte("x"), MaxEncodingBytes+1)
	if _, err := Parse(oversized); ClassOf(err) != ClassParse {
		t.Fatal("oversized encoding accepted")
	} else {
		assertDoesNotLeak(t, err, string(oversized), canonical, m.Digest())
	}
	for _, raw := range [][]byte{nil, {}, []byte(`null`), []byte(`[]`), []byte(`{"format_version":1`)} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("malformed accepted: %q", raw)
		} else {
			assertDoesNotLeak(t, err, string(raw), canonical, m.Digest())
		}
	}
}

func tamperDigest(encoding string) string {
	marker := `"request_digest":"sha256:`
	index := strings.Index(encoding, marker) + len(marker)
	replacement := byte('0')
	if encoding[index] == replacement {
		replacement = '1'
	}
	encoded := []byte(encoding)
	encoded[index] = replacement
	return string(encoded)
}

func TestOwnershipCopiesAndConcurrentGetters(t *testing.T) {
	p := validProposal()
	m, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	original := m.Encoding()
	digest := m.Digest()
	p.RefUpdates[0].Ref = "refs/heads/mutated"
	got := m.RefUpdates()
	got[0].Ref = "refs/heads/mutated-again"
	encoding := m.Encoding()
	encoding[0] = 'x'
	if !bytes.Equal(m.Encoding(), original) || m.Digest() != digest || m.RefUpdates()[0].Ref != "refs/heads/create" {
		t.Fatal("Build retained mutable storage")
	}
	raw := append([]byte(nil), original...)
	parsed, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	raw[0] = 'x'
	if !bytes.Equal(parsed.Encoding(), original) {
		t.Fatal("Parse retained raw input")
	}
	var group sync.WaitGroup
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 100; j++ {
				copy := parsed.Encoding()
				copy[0] = 'x'
				refs := parsed.RefUpdates()
				refs[0].Ref = "changed"
				if parsed.Digest() != digest {
					t.Error("digest changed")
				}
			}
		}()
	}
	group.Wait()
}

func TestErrorsDoNotLeakInputValues(t *testing.T) {
	p := validProposal()
	p.RefUpdates[0].NewSHA = "SECRET-SHA"
	_, err := Build(p)
	assertError(t, err, ClassInvalid, FieldNewSHA, 0)
	for _, secret := range []string{p.Repository, p.Remote, p.AgentID, p.RefUpdates[0].Ref, "SECRET-SHA"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q", secret)
		}
	}
	if ClassOf(fmt.Errorf("foreign detail")) != ClassInternal {
		t.Fatal("foreign error was not collapsed")
	}
}

func proposalValues(p Proposal) []string {
	values := []string{p.RequestID, p.AgentID, p.WorkspaceID, p.Repository, p.Remote, p.PolicyVersion,
		p.CreatedAt.Format(time.RFC3339Nano), p.ExpiresAt.Format(time.RFC3339Nano)}
	for _, update := range p.RefUpdates {
		values = append(values, update.Ref, update.ExpectedOldSHA, update.NewSHA)
	}
	return values
}

func assertDoesNotLeak(t *testing.T, err error, values ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error for non-leak assertion")
	}
	for _, value := range values {
		if value != "" && strings.Contains(err.Error(), value) {
			t.Fatalf("error exposed caller value of length %d", len(value))
		}
	}
}

func TestParseBoundedArbitraryBytesNeverPanics(t *testing.T) {
	for length := 0; length < 1024; length++ {
		raw := make([]byte, length)
		for i := range raw {
			raw[i] = byte((i*31 + length*17) % 256)
		}
		parsed, err := Parse(raw)
		if err == nil {
			if !bytes.Equal(parsed.Encoding(), raw) {
				t.Fatal("successful parse normalized bytes")
			}
			reparsed, err := Parse(parsed.Encoding())
			if err != nil || reparsed.Digest() != parsed.Digest() {
				t.Fatal("successful parse was not stable")
			}
		}
	}
}

func assertError(t *testing.T, err error, class ErrorClass, field Field, index int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if ClassOf(err) != class {
		t.Fatalf("class=%q error=%v", ClassOf(err), err)
	}
	gotField, gotIndex, indexed := LocationOf(err)
	if gotField != field {
		t.Fatalf("field=%q want=%q", gotField, field)
	}
	if index >= 0 && (!indexed || gotIndex != index) {
		t.Fatalf("index=(%d,%v) want=%d", gotIndex, indexed, index)
	}
	if index < 0 && indexed {
		t.Fatalf("unexpected index %d", gotIndex)
	}
}
