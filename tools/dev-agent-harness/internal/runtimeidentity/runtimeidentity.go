// Package runtimeidentity resolves the operating-system identity used by one
// broker lifetime.  It has no service lifecycle or persistence responsibility:
// each Resolve call takes one fresh, validated snapshot.
package runtimeidentity

import (
	"crypto/rand"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerlistener"
)

const (
	minIdentifier = 1
	maxIdentifier = 128
	instanceBytes = 16
)

// Error is the fixed diagnostic surface of this package.  It never contains
// a username, workspace, numeric identity, or lower-level error detail.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrDenied       Error = "runtime-identity-denied"
)

// Rules fixes the usernames and workspace that are valid for one resolver.
type Rules struct {
	AgentUser   string
	BrokerUser  string
	WorkspaceID string
}

// Identity is an immutable, validated snapshot of one runtime resolution.
// Its fields are private so callers can only obtain defensive copies.
type Identity struct {
	brokerUID     int
	agentUID      int
	agentGID      int
	agentInstance string
	workspace     string
	subject       brokerlistener.Subject
}

// Format keeps identity values out of diagnostics.
func (Identity) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "runtimeidentity.Identity")
}

// BrokerUID returns the broker's validated non-root UID.
func (i *Identity) BrokerUID() int {
	if !validIdentity(i) {
		return 0
	}
	return i.brokerUID
}

// AgentUID returns the agent's validated UID.
func (i *Identity) AgentUID() int {
	if !validIdentity(i) {
		return 0
	}
	return i.agentUID
}

// AgentGID returns the agent's validated primary GID.
func (i *Identity) AgentGID() int {
	if !validIdentity(i) {
		return 0
	}
	return i.agentGID
}

// AgentInstanceID returns a fresh copy of the service-lifetime instance ID.
func (i *Identity) AgentInstanceID() string {
	if !validIdentity(i) {
		return ""
	}
	return strings.Clone(i.agentInstance)
}

// WorkspaceID returns a fresh copy of the config-fixed workspace ID.
func (i *Identity) WorkspaceID() string {
	if !validIdentity(i) {
		return ""
	}
	return strings.Clone(i.workspace)
}

// Subject returns a fresh copy of the canonical listener subject.
func (i *Identity) Subject() brokerlistener.Subject {
	if !validIdentity(i) {
		return brokerlistener.Subject{}
	}
	return copySubject(i.subject)
}

// Resolver is immutable after construction. Its private seams exist only for
// deterministic package tests; production New supplies the platform adapter.
type Resolver struct {
	agentUser   string
	brokerUser  string
	workspace   string
	lookupUser  userLookup
	lookupGroup groupLookup
	euid        func() int
	entropy     io.Reader
}

// Format keeps resolver internals and configured identity values out of
// diagnostics.
func (Resolver) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "runtimeidentity.Resolver")
}

type account struct {
	UID string
	GID string
}

type group struct{ GID string }

type userLookup func(string) (account, error)
type groupLookup func(string) (group, error)

// New constructs the production resolver. Linux is the only platform with a
// successful adapter; non-Linux adapters fail closed.
func New(r Rules) (*Resolver, error) {
	return newWithDeps(r, platformLookupUser, platformLookupGroup, platformEUID, rand.Reader)
}

// newWithDeps is intentionally package-private: fake lookup, EUID and entropy
// are hermetic test seams rather than an exported production hook.
func newWithDeps(r Rules, lookup userLookup, groups groupLookup, euid func() int, entropy io.Reader) (*Resolver, error) {
	agent, broker, workspace, ok := normalizeRules(r)
	if !ok || lookup == nil || groups == nil || euid == nil || entropy == nil {
		return nil, ErrInvalidRules
	}
	return &Resolver{agentUser: agent, brokerUser: broker, workspace: workspace, lookupUser: lookup, lookupGroup: groups, euid: euid, entropy: entropy}, nil
}

// Resolve performs exactly one lookup for each configured user and the
// same-named agent group, then one 16-byte entropy read. Any failure returns
// no partial identity and the same fixed error.
func (r *Resolver) Resolve() (*Identity, error) {
	if !validResolver(r) {
		return nil, ErrDenied
	}
	agent, err := r.lookupUser(r.agentUser)
	if err != nil {
		return nil, ErrDenied
	}
	broker, err := r.lookupUser(r.brokerUser)
	if err != nil {
		return nil, ErrDenied
	}
	agentGroup, err := r.lookupGroup(r.agentUser)
	if err != nil {
		return nil, ErrDenied
	}
	agentUID, ok := parseID(agent.UID)
	if !ok {
		return nil, ErrDenied
	}
	brokerUID, ok := parseID(broker.UID)
	if !ok {
		return nil, ErrDenied
	}
	agentGID, ok := parseID(agent.GID)
	if !ok {
		return nil, ErrDenied
	}
	groupGID, ok := parseID(agentGroup.GID)
	if !ok || groupGID != agentGID || agentUID == brokerUID || brokerUID != r.euid() || brokerUID <= 0 {
		return nil, ErrDenied
	}
	var entropy [instanceBytes]byte
	n, readErr := r.entropy.Read(entropy[:])
	if readErr != nil || n != len(entropy) {
		return nil, ErrDenied
	}
	instance := "agent-" + fmt.Sprintf("%x", entropy[:])
	identity := &Identity{
		brokerUID: brokerUID, agentUID: agentUID, agentGID: agentGID,
		agentInstance: instance, workspace: strings.Clone(r.workspace),
		subject: brokerlistener.Subject{AgentInstanceID: instance, UID: agentUID, WorkspaceID: strings.Clone(r.workspace)},
	}
	if !validIdentity(identity) {
		return nil, ErrDenied
	}
	return identity, nil
}

func normalizeRules(r Rules) (string, string, string, bool) {
	return strings.Clone(r.AgentUser), strings.Clone(r.BrokerUser), strings.Clone(r.WorkspaceID), validUsername(r.AgentUser) && validUsername(r.BrokerUser) && r.AgentUser != r.BrokerUser && validIdentifier(r.WorkspaceID)
}

func validResolver(r *Resolver) bool {
	return r != nil && validUsername(r.agentUser) && validUsername(r.brokerUser) && r.agentUser != r.brokerUser && validIdentifier(r.workspace) && r.lookupUser != nil && r.lookupGroup != nil && r.euid != nil && r.entropy != nil
}

func validIdentity(i *Identity) bool {
	if i == nil || !validNumeric(i.brokerUID) || !validNumeric(i.agentUID) || !validNumeric(i.agentGID) || i.agentUID == i.brokerUID || !validIdentifier(i.workspace) || i.subject.UID != i.agentUID || i.subject.AgentInstanceID != i.agentInstance || i.subject.WorkspaceID != i.workspace {
		return false
	}
	if len(i.agentInstance) != len("agent-")+instanceBytes*2 || !strings.HasPrefix(i.agentInstance, "agent-") {
		return false
	}
	for _, c := range i.agentInstance[len("agent-"):] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func validUsername(value string) bool {
	if len(value) == 0 || len(value) > 32 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if i == 0 {
			if !isAlpha(c) && c != '_' {
				return false
			}
		} else if !isAlpha(c) && !isDigit(c) && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func validIdentifier(value string) bool {
	if len(value) < minIdentifier || len(value) > maxIdentifier {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if i == 0 {
			if !isAlpha(c) && !isDigit(c) {
				return false
			}
		} else if !isAlpha(c) && !isDigit(c) && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func parseID(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != value || uint64(int(n)) != n || int(n) <= 0 {
		return 0, false
	}
	return int(n), true
}

func validNumeric(value int) bool { return value > 0 && uint64(value) <= uint64(^uint32(0)) }

func copySubject(subject brokerlistener.Subject) brokerlistener.Subject {
	return brokerlistener.Subject{AgentInstanceID: strings.Clone(subject.AgentInstanceID), UID: subject.UID, WorkspaceID: strings.Clone(subject.WorkspaceID)}
}

func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
