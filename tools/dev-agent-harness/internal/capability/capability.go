// Package capability stores short-lived, opaque references to fixed provider
// scopes. It has no credential, transport, or persistence responsibilities.
package capability

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	ProviderGitHub = "github"
	ProviderOpenAI = "openai"

	OperationGitHubRESTRead      = "github-rest-read"
	OperationOpenAIResponsesText = "openai-responses-text"
	HostGitHub                   = "api.github.com"
	HostOpenAI                   = "api.openai.com"

	maxRulesTTL  = 24 * time.Hour
	maxRulesUses = 10_000
	maxIssueTry  = 5 // first issuance plus four collision retries
)

// Error is a fixed, comparable registry error. Constants prevent callers
// from replacing a published sentinel value.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrIssue        Error = "capability-issue-denied"
	ErrDenied       Error = "capability-denied"
)

// Rules bounds all entries in a Registry.
type Rules struct {
	PolicyVersion          string
	MaxTTL                 time.Duration
	MaxUses                int
	InitialRevocationEpoch uint64
}

// IssueSpec describes a provider-specific scope. Operation and destination
// host are fixed by Provider and are intentionally absent from this type.
type IssueSpec struct {
	AgentInstanceID string
	UID             int
	WorkspaceID     string
	Provider        string
	Repository      string
	TTL             time.Duration
	Uses            int
}

// Request is the scope presented when a capability is consumed.
type Request struct {
	Handle          string
	AgentInstanceID string
	UID             int
	WorkspaceID     string
	Provider        string
	Repository      string
	Operation       string
	DestinationHost string
}

// Grant is a copy of the validated scope and remaining lease state. It does
// not contain the capability handle or any credential.
type Grant struct {
	AgentInstanceID string
	UID             int
	WorkspaceID     string
	Provider        string
	Repository      string
	Operation       string
	DestinationHost string
	IssuedAt        time.Time
	ExpiresAt       time.Time
	RemainingUses   int
	PolicyVersion   string
	RevocationEpoch uint64
}

type scope struct {
	agentInstanceID string
	uid             int
	workspaceID     string
	provider        string
	repository      string
	operation       string
	destinationHost string
}

type entry struct {
	digest          [sha256.Size]byte
	scope           scope
	issuedAt        time.Time
	expiresAt       time.Time
	issuedElapsed   time.Duration
	deadlineElapsed time.Duration
	remainingUses   int
	policyVersion   string
	revocationEpoch uint64
}

type clockReading struct {
	wall    time.Time
	elapsed time.Duration
}

// Registry is an in-memory capability registry. Its zero value is unusable;
// construct one with New.
type Registry struct {
	mu              sync.Mutex
	entries         map[[sha256.Size]byte]entry
	policyVersion   string
	maxTTL          time.Duration
	maxUses         int
	revocationEpoch uint64
	random          io.Reader
	now             func() clockReading
}

// New constructs a production registry using crypto/rand and a monotonic
// reading derived from time.Now.
func New(r Rules) (*Registry, error) {
	origin := time.Now()
	return newWithClockDeps(r, rand.Reader, func() clockReading {
		current := time.Now()
		return clockReading{wall: current, elapsed: current.Sub(origin)}
	})
}

// newWithDeps is package-private so deterministic entropy and clocks remain a
// test dependency rather than an exported production hook.
func newWithDeps(r Rules, random io.Reader, now func() time.Time) (*Registry, error) {
	if now == nil {
		return nil, ErrInvalidRules
	}
	var origin time.Time
	var initialized bool
	return newWithClockDeps(r, random, func() clockReading {
		wall := now()
		if !initialized {
			origin = wall
			initialized = true
		}
		return clockReading{wall: wall, elapsed: wall.Sub(origin)}
	})
}

func newWithClockDeps(r Rules, random io.Reader, now func() clockReading) (*Registry, error) {
	if !validRules(r) || random == nil || now == nil {
		return nil, ErrInvalidRules
	}
	return &Registry{
		entries:         make(map[[sha256.Size]byte]entry),
		policyVersion:   r.PolicyVersion,
		maxTTL:          r.MaxTTL,
		maxUses:         r.MaxUses,
		revocationEpoch: r.InitialRevocationEpoch,
		random:          random,
		now:             now,
	}, nil
}

// Issue creates an opaque reference and stores only its digest and fixed
// scope. Random collisions are retried at most four times after the first
// attempt.
func (r *Registry) Issue(spec IssueSpec) (string, error) {
	if r == nil {
		return "", ErrIssue
	}
	if !r.validSpec(spec) {
		return "", ErrIssue
	}
	s := makeScope(spec)
	reading := r.now()
	issued := reading.wall
	expires := issued.Add(spec.TTL)
	deadline := reading.elapsed + spec.TTL

	for attempt := 0; attempt < maxIssueTry; attempt++ {
		var randomBytes [32]byte
		if _, err := io.ReadFull(r.random, randomBytes[:]); err != nil {
			return "", ErrIssue
		}
		handle := "cap_" + base64.RawURLEncoding.EncodeToString(randomBytes[:])
		digest := sha256.Sum256([]byte(handle))
		r.mu.Lock()
		if _, exists := r.entries[digest]; exists {
			r.mu.Unlock()
			continue
		}
		r.entries[digest] = entry{
			digest:          digest,
			scope:           s,
			issuedAt:        issued,
			expiresAt:       expires,
			issuedElapsed:   reading.elapsed,
			deadlineElapsed: deadline,
			remainingUses:   spec.Uses,
			policyVersion:   r.policyVersion,
			revocationEpoch: r.revocationEpoch,
		}
		r.mu.Unlock()
		return handle, nil
	}
	return "", ErrIssue
}

// Consume atomically checks scope, expiry, epoch, and remaining uses. Only a
// successful check decrements uses.
func (r *Registry) Consume(req Request) (Grant, error) {
	if r == nil {
		return Grant{}, ErrDenied
	}
	digest, ok := digestForHandle(req.Handle)
	if !ok {
		return Grant{}, ErrDenied
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[digest]
	if !ok {
		return Grant{}, ErrDenied
	}
	reading := r.now()
	// Round(0) strips any monotonic reading for the independent wall-clock
	// fail-closed check; the lifecycle deadline check remains monotonic.
	expired := reading.elapsed >= e.deadlineElapsed || !reading.wall.Round(0).Before(e.expiresAt.Round(0))
	if e.revocationEpoch != r.revocationEpoch || e.policyVersion != r.policyVersion || !scopeMatches(e.scope, req) || reading.elapsed < e.issuedElapsed || expired || e.remainingUses <= 0 {
		if e.revocationEpoch != r.revocationEpoch || e.policyVersion != r.policyVersion || expired || e.remainingUses <= 0 {
			delete(r.entries, digest)
		}
		return Grant{}, ErrDenied
	}
	e.remainingUses--
	grant := Grant{
		AgentInstanceID: e.scope.agentInstanceID,
		UID:             e.scope.uid,
		WorkspaceID:     e.scope.workspaceID,
		Provider:        e.scope.provider,
		Repository:      e.scope.repository,
		Operation:       e.scope.operation,
		DestinationHost: e.scope.destinationHost,
		IssuedAt:        e.issuedAt.UTC(),
		ExpiresAt:       e.expiresAt.UTC(),
		RemainingUses:   e.remainingUses,
		PolicyVersion:   e.policyVersion,
		RevocationEpoch: e.revocationEpoch,
	}
	if e.remainingUses == 0 {
		delete(r.entries, digest)
	} else {
		r.entries[digest] = e
	}
	return grant, nil
}

// Revoke removes one capability. Unknown or malformed handles are denied.
func (r *Registry) Revoke(handle string) error {
	if r == nil {
		return ErrDenied
	}
	digest, ok := digestForHandle(handle)
	if !ok {
		return ErrDenied
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[digest]; !exists {
		return ErrDenied
	}
	delete(r.entries, digest)
	return nil
}

// AdvanceRevocationEpoch invalidates every existing entry and accepts only a
// strictly greater epoch.
func (r *Registry) AdvanceRevocationEpoch(epoch uint64) error {
	if r == nil || r.entries == nil || r.random == nil || r.now == nil || r.maxTTL <= 0 || r.maxUses <= 0 {
		return ErrDenied
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if epoch <= r.revocationEpoch {
		return ErrDenied
	}
	r.revocationEpoch = epoch
	r.entries = make(map[[sha256.Size]byte]entry)
	return nil
}

func (r *Registry) validSpec(spec IssueSpec) bool {
	if !validIdentifier(spec.AgentInstanceID, 128) || spec.UID <= 0 || !validIdentifier(spec.WorkspaceID, 128) ||
		spec.TTL <= 0 || spec.TTL > r.maxTTL || spec.Uses <= 0 || spec.Uses > r.maxUses {
		return false
	}
	switch spec.Provider {
	case ProviderGitHub:
		return validRepository(spec.Repository)
	case ProviderOpenAI:
		return spec.Repository == ""
	default:
		return false
	}
}

func validRules(r Rules) bool {
	return validIdentifier(r.PolicyVersion, 64) && r.MaxTTL > 0 && r.MaxTTL <= maxRulesTTL && r.MaxUses > 0 && r.MaxUses <= maxRulesUses
}

func makeScope(spec IssueSpec) scope {
	operation, host := OperationOpenAIResponsesText, HostOpenAI
	if spec.Provider == ProviderGitHub {
		operation, host = OperationGitHubRESTRead, HostGitHub
	}
	return scope{agentInstanceID: spec.AgentInstanceID, uid: spec.UID, workspaceID: spec.WorkspaceID, provider: spec.Provider, repository: spec.Repository, operation: operation, destinationHost: host}
}

func scopeMatches(s scope, req Request) bool {
	return s.agentInstanceID == req.AgentInstanceID && s.uid == req.UID && s.workspaceID == req.WorkspaceID &&
		s.provider == req.Provider && s.repository == req.Repository && s.operation == req.Operation && s.destinationHost == req.DestinationHost
}

func digestForHandle(handle string) ([sha256.Size]byte, bool) {
	var zero [sha256.Size]byte
	if len(handle) != len("cap_")+43 || !strings.HasPrefix(handle, "cap_") {
		return zero, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(handle[len("cap_"):])
	if err != nil || len(raw) != 32 || "cap_"+base64.RawURLEncoding.EncodeToString(raw) != handle {
		return zero, false
	}
	return sha256.Sum256([]byte(handle)), true
}

func validIdentifier(value string, max int) bool {
	if len(value) == 0 || len(value) > max || !asciiAlphaNumeric(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if !asciiAlphaNumeric(c) && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func asciiAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func validRepository(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && validRepositoryPart(parts[0]) && validRepositoryPart(parts[1])
}

func validRepositoryPart(value string) bool {
	if value == "" || value == "." || value == ".." || !asciiAlphaNumeric(value[0]) || value[0] >= 'A' && value[0] <= 'Z' {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}
