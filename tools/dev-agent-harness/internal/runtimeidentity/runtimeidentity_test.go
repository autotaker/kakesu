package runtimeidentity

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerlistener"
)

type lookupFixture struct {
	users      map[string]account
	groups     map[string]group
	userCalls  map[string]int
	groupCalls map[string]int
}

func (f *lookupFixture) user(name string) (account, error) {
	f.userCalls[name]++
	value, ok := f.users[name]
	if !ok {
		return account{}, errors.New("missing")
	}
	return value, nil
}

func (f *lookupFixture) group(name string) (group, error) {
	f.groupCalls[name]++
	value, ok := f.groups[name]
	if !ok {
		return group{}, errors.New("missing")
	}
	return value, nil
}

type exactReader struct {
	data    []byte
	calls   int
	lengths []int
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("entropy failure") }

func (r *exactReader) Read(dst []byte) (int, error) {
	r.calls++
	r.lengths = append(r.lengths, len(dst))
	if len(r.data) < len(dst) {
		copy(dst, r.data)
		return len(r.data), nil
	}
	copy(dst, r.data[:len(dst)])
	return len(dst), nil
}

func validFixture() (*lookupFixture, *exactReader) {
	return &lookupFixture{
		users: map[string]account{
			"agent":  {UID: "1001", GID: "2001"},
			"broker": {UID: "1000", GID: "2000"},
		},
		groups:    map[string]group{"agent": {GID: "2001"}},
		userCalls: make(map[string]int), groupCalls: make(map[string]int),
	}, &exactReader{data: bytes.Repeat([]byte{0xab}, instanceBytes)}
}

func newFixtureResolver(t *testing.T, fixture *lookupFixture, entropy *exactReader) *Resolver {
	t.Helper()
	r, err := newWithDeps(Rules{AgentUser: "agent", BrokerUser: "broker", WorkspaceID: "workspace-1"}, fixture.user, fixture.group, func() int { return 1000 }, entropy)
	if err != nil {
		t.Fatalf("newWithDeps: %v", err)
	}
	return r
}

func TestResolveProducesOneConsistentSnapshot(t *testing.T) {
	fixture, entropy := validFixture()
	r := newFixtureResolver(t, fixture, entropy)
	identity, err := r.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if identity.BrokerUID() != 1000 || identity.AgentUID() != 1001 || identity.AgentGID() != 2001 {
		t.Fatalf("unexpected numeric identity: broker=%d agent=%d gid=%d", identity.BrokerUID(), identity.AgentUID(), identity.AgentGID())
	}
	if got := identity.AgentInstanceID(); got != "agent-"+strings.Repeat("ab", instanceBytes) {
		t.Fatalf("instance=%q", got)
	}
	if identity.WorkspaceID() != "workspace-1" {
		t.Fatalf("workspace=%q", identity.WorkspaceID())
	}
	wantSubject := brokerlistener.Subject{AgentInstanceID: identity.AgentInstanceID(), UID: 1001, WorkspaceID: "workspace-1"}
	if got := identity.Subject(); got != wantSubject {
		t.Fatalf("subject=%#v want %#v", got, wantSubject)
	}
	if fixture.userCalls["agent"] != 1 || fixture.userCalls["broker"] != 1 || fixture.groupCalls["agent"] != 1 {
		t.Fatalf("lookup counts: users=%v groups=%v", fixture.userCalls, fixture.groupCalls)
	}
	if entropy.calls != 1 || len(entropy.lengths) != 1 || entropy.lengths[0] != instanceBytes {
		t.Fatalf("entropy calls=%d lengths=%v", entropy.calls, entropy.lengths)
	}
}

func TestResolveUsesFreshIDsAndDefensiveSubjectCopy(t *testing.T) {
	fixture, entropy := validFixture()
	r := newFixtureResolver(t, fixture, entropy)
	one, err := r.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	twoEntropy := &exactReader{data: bytes.Repeat([]byte{0xcd}, instanceBytes)}
	r.entropy = twoEntropy
	two, err := r.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if one.AgentInstanceID() == two.AgentInstanceID() {
		t.Fatal("Resolve reused instance ID")
	}
	subject := one.Subject()
	subject.AgentInstanceID = "agent-mutated"
	subject.WorkspaceID = "workspace-mutated"
	if got := one.Subject(); got.AgentInstanceID == subject.AgentInstanceID || got.WorkspaceID == subject.WorkspaceID {
		t.Fatalf("subject mutation affected identity: %#v", got)
	}
	if got := one.AgentInstanceID(); got != "agent-"+strings.Repeat("ab", instanceBytes) {
		t.Fatalf("instance mutation affected identity: %q", got)
	}
}

func TestResolveRejectsIdentityBoundariesWithoutPartialResults(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*lookupFixture)
		euid    int
		entropy []byte
	}{
		{name: "agent zero", mutate: func(f *lookupFixture) { f.users["agent"] = account{UID: "0", GID: "2001"} }, euid: 1000},
		{name: "broker root", mutate: func(f *lookupFixture) { f.users["broker"] = account{UID: "0", GID: "2000"} }, euid: 0},
		{name: "broker euid mismatch", mutate: func(*lookupFixture) {}, euid: 1002},
		{name: "same uid", mutate: func(f *lookupFixture) { f.users["agent"] = account{UID: "1000", GID: "2001"} }, euid: 1000},
		{name: "primary group mismatch", mutate: func(f *lookupFixture) { f.groups["agent"] = group{GID: "2002"} }, euid: 1000},
		{name: "noncanonical uid", mutate: func(f *lookupFixture) { f.users["agent"] = account{UID: "01001", GID: "2001"} }, euid: 1000},
		{name: "entropy short", mutate: func(*lookupFixture) {}, euid: 1000, entropy: []byte{1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture, entropy := validFixture()
			tc.mutate(fixture)
			if tc.entropy != nil {
				entropy.data = tc.entropy
			}
			r, err := newWithDeps(Rules{AgentUser: "agent", BrokerUser: "broker", WorkspaceID: "workspace-1"}, fixture.user, fixture.group, func() int { return tc.euid }, entropy)
			if err != nil {
				t.Fatal(err)
			}
			if identity, err := r.Resolve(); identity != nil || !errors.Is(err, ErrDenied) {
				t.Fatalf("identity=%#v err=%v", identity, err)
			}
		})
	}
	fixture, entropy := validFixture()
	lookupError := func(name string) (account, error) {
		if name == "agent" {
			return account{}, errors.New("lookup failure")
		}
		return fixture.user(name)
	}
	r, err := newWithDeps(Rules{AgentUser: "agent", BrokerUser: "broker", WorkspaceID: "workspace-1"}, lookupError, fixture.group, func() int { return 1000 }, entropy)
	if err != nil {
		t.Fatal(err)
	}
	if identity, resolveErr := r.Resolve(); identity != nil || !errors.Is(resolveErr, ErrDenied) {
		t.Fatalf("lookup error returned partial identity=%#v err=%v", identity, resolveErr)
	}
}

func TestConstructorAndCorruptionUseFixedErrors(t *testing.T) {
	badRules := []Rules{
		{},
		{AgentUser: "agent", BrokerUser: "agent", WorkspaceID: "workspace"},
		{AgentUser: "agent", BrokerUser: "broker", WorkspaceID: "_workspace"},
	}
	fixture, entropy := validFixture()
	for _, rules := range badRules {
		if resolver, err := newWithDeps(rules, fixture.user, fixture.group, func() int { return 1000 }, entropy); resolver != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("rules=%#v resolver=%#v err=%v", rules, resolver, err)
		}
	}
	r := newFixtureResolver(t, fixture, entropy)
	r.lookupUser = nil
	if identity, err := r.Resolve(); identity != nil || !errors.Is(err, ErrDenied) {
		t.Fatalf("corrupt resolver identity=%#v err=%v", identity, err)
	}
	identity := &Identity{brokerUID: 1000, agentUID: 1001, agentGID: 2001, agentInstance: "agent-" + strings.Repeat("ab", instanceBytes), workspace: "workspace-1", subject: brokerlistener.Subject{AgentInstanceID: "agent-" + strings.Repeat("ab", instanceBytes), UID: 1001, WorkspaceID: "workspace-1"}}
	for _, format := range []string{"%v", "%+v", "%#v", "%q", "%s"} {
		formatted := fmt.Sprintf(format, identity)
		if formatted != "runtimeidentity.Identity" {
			t.Fatalf("identity format %s=%q", format, formatted)
		}
		if formattedResolver := fmt.Sprintf(format, r); formattedResolver != "runtimeidentity.Resolver" {
			t.Fatalf("resolver format %s=%q", format, formattedResolver)
		}
	}
	if fmt.Sprintf("%v", ErrDenied) != "runtime-identity-denied" {
		t.Fatal("unexpected fixed diagnostic")
	}
	for _, corrupted := range []*Identity{nil, {}, {brokerUID: 1000, agentUID: 1001, agentGID: 2001, agentInstance: "agent-" + strings.Repeat("zz", instanceBytes), workspace: "workspace-1", subject: brokerlistener.Subject{AgentInstanceID: "agent-" + strings.Repeat("zz", instanceBytes), UID: 1001, WorkspaceID: "workspace-1"}}} {
		if corrupted.BrokerUID() != 0 || corrupted.AgentUID() != 0 || corrupted.AgentGID() != 0 || corrupted.AgentInstanceID() != "" || corrupted.WorkspaceID() != "" || corrupted.Subject() != (brokerlistener.Subject{}) {
			t.Fatalf("corrupt identity accessor leaked data: %#v", corrupted)
		}
	}
	fixture, _ = validFixture()
	if resolver, err := newWithDeps(Rules{AgentUser: "agent", BrokerUser: "broker", WorkspaceID: "workspace"}, fixture.user, fixture.group, func() int { return 1000 }, failingReader{}); err != nil || resolver == nil {
		t.Fatalf("failed to construct entropy-error resolver: resolver=%#v err=%v", resolver, err)
	} else if got, resolveErr := resolver.Resolve(); got != nil || !errors.Is(resolveErr, ErrDenied) {
		t.Fatalf("entropy error returned partial identity=%#v err=%v", got, resolveErr)
	}
}

func TestParseIDCanonicalAndLossless(t *testing.T) {
	for _, value := range []string{"", "0", "00", "01", "+1", " 1", "-1", "4294967296"} {
		if got, ok := parseID(value); ok || got != 0 {
			t.Errorf("parseID(%q)=%d,%v", value, got, ok)
		}
	}
	for _, value := range []string{"1", "1001", "4294967295"} {
		if got, ok := parseID(value); !ok || got <= 0 {
			t.Errorf("parseID(%q)=%d,%v", value, got, ok)
		}
	}
}
