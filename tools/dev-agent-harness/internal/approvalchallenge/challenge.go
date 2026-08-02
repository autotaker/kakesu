// Package approvalchallenge owns an in-memory, one-time challenge lifecycle.
// A successful verification is evidence returned to a caller; it is not an
// approval-state transition, a grant, or authorization to push.
package approvalchallenge

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	challengeBytes         = 32
	challengeAttempts      = 4
	maxRequestIDBytes      = 128
	maxDigestBytes         = len("sha256:") + sha256.Size*2
	maxOperatorIDBytes     = 128
	maxRPIDBytes           = 253
	maxOriginBytes         = 8 + maxRPIDBytes
	maxAssertionBytes      = 64 * 1024
	maxCredentialIDBytes   = 1024
	maxPendingLimit        = 10_000
	credentialDigestLabel  = "sha256:"
	credentialDigestDomain = "dev-agent-harness/passkey-credential/v1\x00"

	MaxTTL = 10 * time.Minute
)

// ErrorClass is a stable, non-sensitive failure category.
type ErrorClass string

const (
	ClassInvalid      ErrorClass = "invalid"
	ClassRandom       ErrorClass = "random"
	ClassClock        ErrorClass = "clock"
	ClassCapacity     ErrorClass = "capacity"
	ClassClosed       ErrorClass = "closed"
	ClassNotFound     ErrorClass = "not_found"
	ClassExpired      ErrorClass = "expired"
	ClassVerification ErrorClass = "verification"
	ClassInternal     ErrorClass = "internal"
)

// Error never includes a challenge, request value, assertion, credential, or
// lower-level error.
type Error struct{ class ErrorClass }

func (e *Error) Error() string {
	if e == nil {
		return "approval challenge error"
	}
	return "approval challenge " + string(e.class)
}

// Class returns the stable category of e.
func (e *Error) Class() ErrorClass {
	if e == nil {
		return ""
	}
	return e.class
}

// ClassOf returns the stable class for package errors and ClassInternal for an
// unexpected error. It never exposes the unexpected error.
func ClassOf(err error) ErrorClass {
	var target *Error
	if errors.As(err, &target) {
		return target.class
	}
	if err == nil {
		return ""
	}
	return ClassInternal
}

func newError(class ErrorClass) error { return &Error{class: class} }

// Decision is the exact decision bound to a challenge.
type Decision string

const (
	Approve Decision = "approve"
	Deny    Decision = "deny"
)

// Rules are copied and fixed for the lifetime of a Manager.
type Rules struct {
	TTL        time.Duration
	MaxPending int
}

// Request contains caller-controlled values to bind to a fresh challenge.
// Issue validates every field before admitting it.
type Request struct {
	RequestID  string
	Digest     string
	Decision   Decision
	OperatorID string
	RPID       string
	Origin     string
}

// Binding is an immutable caller view of everything a verifier must bind.
type Binding struct {
	challenge  string
	requestID  string
	digest     string
	decision   Decision
	operatorID string
	rpID       string
	origin     string
	issuedAt   time.Time
	expiresAt  time.Time
}

func (b Binding) Challenge() string    { return b.challenge }
func (b Binding) RequestID() string    { return b.requestID }
func (b Binding) Digest() string       { return b.digest }
func (b Binding) Decision() Decision   { return b.decision }
func (b Binding) OperatorID() string   { return b.operatorID }
func (b Binding) RPID() string         { return b.rpID }
func (b Binding) Origin() string       { return b.origin }
func (b Binding) IssuedAt() time.Time  { return b.issuedAt }
func (b Binding) ExpiresAt() time.Time { return b.expiresAt }

// Issued is an immutable view of a newly admitted challenge and its binding.
type Issued struct {
	challenge string
	binding   Binding
}

func (i Issued) Challenge() string { return i.challenge }
func (i Issued) Binding() Binding  { return i.binding }

// Verified contains only the non-secret evidence produced after a verifier
// succeeds. It intentionally omits the challenge, assertion, signature,
// credential public key, and raw credential identifier.
type Verified struct {
	requestID    string
	digest       string
	decision     Decision
	operatorID   string
	credentialID string
	verifiedAt   time.Time
}

func (v Verified) RequestID() string     { return v.requestID }
func (v Verified) Digest() string        { return v.digest }
func (v Verified) Decision() Decision    { return v.decision }
func (v Verified) OperatorID() string    { return v.operatorID }
func (v Verified) CredentialID() string  { return v.credentialID }
func (v Verified) VerifiedAt() time.Time { return v.verifiedAt }

// Verifier is a trusted seam for a later WebAuthn implementation. The manager
// passes value-copied binding data and a fresh assertion copy. The returned raw
// credential identifier is hashed and is never retained or returned.
type Verifier func(Binding, []byte) ([]byte, error)

type dependencies struct {
	random io.Reader
	now    func() time.Time
}

// Manager serializes admission, expiry, reservation, and close operations for
// one process. It starts no goroutines and persists nothing.
type Manager struct {
	mu      sync.Mutex
	rules   Rules
	pending map[string]Binding
	random  io.Reader
	now     func() time.Time
	lastNow time.Time
	closed  bool
}

// New constructs a production manager using crypto/rand and the system clock.
// Callers cannot inject a challenge, random source, or clock.
func New(rules Rules) (*Manager, error) {
	return newWithDependencies(rules, dependencies{random: rand.Reader, now: time.Now})
}

func newWithDependencies(rules Rules, deps dependencies) (*Manager, error) {
	if !validRules(rules) || deps.random == nil || deps.now == nil {
		return nil, newError(ClassInvalid)
	}
	return &Manager{
		rules:   rules,
		pending: make(map[string]Binding, rules.MaxPending),
		random:  deps.random,
		now:     deps.now,
	}, nil
}

// Issue validates and binds a request to a fresh opaque challenge.
func (m *Manager) Issue(request Request) (Issued, error) {
	if m == nil {
		return Issued{}, newError(ClassClosed)
	}
	if !validRequest(request) {
		return Issued{}, newError(ClassInvalid)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Issued{}, newError(ClassClosed)
	}
	now, err := m.trustedNowLocked()
	if err != nil {
		return Issued{}, err
	}
	m.purgeDueLocked(now, "")
	if len(m.pending) >= m.rules.MaxPending {
		return Issued{}, newError(ClassCapacity)
	}

	for attempt := 0; attempt < challengeAttempts; attempt++ {
		raw := make([]byte, challengeBytes)
		if _, err := io.ReadFull(m.random, raw); err != nil {
			return Issued{}, newError(ClassRandom)
		}
		challenge := base64.RawURLEncoding.EncodeToString(raw)
		if _, exists := m.pending[challenge]; exists {
			continue
		}
		binding := Binding{
			challenge:  challenge,
			requestID:  request.RequestID,
			digest:     request.Digest,
			decision:   request.Decision,
			operatorID: request.OperatorID,
			rpID:       request.RPID,
			origin:     request.Origin,
			issuedAt:   now,
			expiresAt:  now.Add(m.rules.TTL),
		}
		m.pending[challenge] = binding
		return Issued{challenge: challenge, binding: binding}, nil
	}
	return Issued{}, newError(ClassInternal)
}

// Consume atomically reserves a live challenge before invoking verifier. Every
// known challenge is one-shot even if input is malformed, verification fails,
// verifier panics, Close races with verification, or the post-callback clock
// rolls back.
func (m *Manager) Consume(challenge string, assertion []byte, verifier Verifier) (Verified, error) {
	if m == nil {
		return Verified{}, newError(ClassClosed)
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return Verified{}, newError(ClassClosed)
	}
	now, err := m.trustedNowLocked()
	if err != nil {
		m.mu.Unlock()
		return Verified{}, err
	}
	dueTarget := m.purgeDueLocked(now, challenge)
	if dueTarget {
		m.mu.Unlock()
		return Verified{}, newError(ClassExpired)
	}
	binding, found := m.pending[challenge]
	if !found {
		m.mu.Unlock()
		return Verified{}, newError(ClassNotFound)
	}
	delete(m.pending, challenge)
	m.mu.Unlock()

	if len(assertion) == 0 || len(assertion) > maxAssertionBytes || verifier == nil {
		return Verified{}, newError(ClassVerification)
	}
	credential, ok := invokeVerifier(verifier, binding, append([]byte(nil), assertion...))
	if !ok || len(credential) == 0 || len(credential) > maxCredentialIDBytes {
		return Verified{}, newError(ClassVerification)
	}
	credentialCopy := append([]byte(nil), credential...)
	credentialID := stableCredentialID(credentialCopy)

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Verified{}, newError(ClassClosed)
	}
	verifiedAt, err := m.trustedNowLocked()
	if err != nil {
		return Verified{}, err
	}
	return Verified{
		requestID:    binding.requestID,
		digest:       binding.digest,
		decision:     binding.decision,
		operatorID:   binding.operatorID,
		credentialID: credentialID,
		verifiedAt:   verifiedAt,
	}, nil
}

// Close atomically rejects new work and discards every pending challenge. It is
// idempotent and does not wait for or interrupt an in-flight verifier.
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	m.pending = nil
	return nil
}

func (m *Manager) trustedNowLocked() (time.Time, error) {
	raw := m.now().UTC()
	if raw.IsZero() || raw.Year() < 1 || raw.Year() > 9999 || (!m.lastNow.IsZero() && raw.Before(m.lastNow)) {
		return time.Time{}, newError(ClassClock)
	}
	m.lastNow = raw
	return raw.Truncate(time.Second), nil
}

func (m *Manager) purgeDueLocked(now time.Time, target string) bool {
	targetDue := false
	for challenge, binding := range m.pending {
		if !now.Before(binding.expiresAt) {
			if challenge == target {
				targetDue = true
			}
			delete(m.pending, challenge)
		}
	}
	return targetDue
}

func invokeVerifier(verifier Verifier, binding Binding, assertion []byte) (credential []byte, ok bool) {
	defer func() {
		if recover() != nil {
			credential = nil
			ok = false
		}
	}()
	credential, err := verifier(binding, assertion)
	if err != nil {
		return nil, false
	}
	return credential, true
}

func stableCredentialID(raw []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(credentialDigestDomain))
	_, _ = h.Write(raw)
	return credentialDigestLabel + hex.EncodeToString(h.Sum(nil))
}

func validRules(r Rules) bool {
	return r.TTL > 0 && r.TTL <= MaxTTL && r.TTL%time.Second == 0 && r.MaxPending > 0 && r.MaxPending <= maxPendingLimit
}

func validRequest(r Request) bool {
	return validScalar(r.RequestID, maxRequestIDBytes) &&
		validDigest(r.Digest) &&
		(r.Decision == Approve || r.Decision == Deny) &&
		validScalar(r.OperatorID, maxOperatorIDBytes) &&
		validRPID(r.RPID) &&
		validOrigin(r.Origin, r.RPID)
}

func validScalar(value string, maximum int) bool {
	if len(value) == 0 || len(value) > maximum || strings.TrimSpace(value) != value {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x21 || value[index] > 0x7e {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	if len(value) != maxDigestBytes || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for index := len("sha256:"); index < len(value); index++ {
		if (value[index] < '0' || value[index] > '9') && (value[index] < 'a' || value[index] > 'f') {
			return false
		}
	}
	return true
}

func validRPID(value string) bool {
	if len(value) == 0 || len(value) > maxRPIDBytes || net.ParseIP(value) != nil || strings.HasSuffix(value, ".") {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for index := 0; index < len(label); index++ {
			character := label[index]
			if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func validOrigin(raw, rpID string) bool {
	if len(raw) == 0 || len(raw) > maxOriginBytes || raw != "https://"+rpID {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Host != rpID || u.Hostname() != rpID || u.Port() != "" {
		return false
	}
	return u.Path == "" && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.RawFragment == "" && u.String() == raw
}
