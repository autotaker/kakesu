// Package approvalstate durably records approval-request decisions. An
// approved record is not a grant and does not authorize a push.
package approvalstate

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/approvalmanifest"
)

const (
	stateVersion    = uint64(1)
	stateName       = "state.json"
	lockName        = "lock"
	temporaryName   = ".state.json.tmp"
	maxActorBytes   = 128
	maxSnapshotBase = 4096
)

// ErrorClass is a stable, non-sensitive failure category.
type ErrorClass string

const (
	ClassInvalid     ErrorClass = "invalid"
	ClassPermission  ErrorClass = "permission"
	ClassLocked      ErrorClass = "locked"
	ClassCorrupt     ErrorClass = "corrupt"
	ClassConflict    ErrorClass = "conflict"
	ClassNotFound    ErrorClass = "not_found"
	ClassDigest      ErrorClass = "digest"
	ClassTransition  ErrorClass = "transition"
	ClassExpired     ErrorClass = "expired"
	ClassClock       ErrorClass = "clock"
	ClassCapacity    ErrorClass = "capacity"
	ClassPersistence ErrorClass = "persistence"
	ClassPoisoned    ErrorClass = "poisoned"
	ClassClosed      ErrorClass = "closed"
	ClassUnsupported ErrorClass = "unsupported"
)

// Error never includes paths, identifiers, manifest contents, or lower errors.
type Error struct{ class ErrorClass }

func (e *Error) Error() string {
	if e == nil {
		return "approval state error"
	}
	return "approval state " + string(e.class)
}

// Class returns the stable category of e.
func (e *Error) Class() ErrorClass {
	if e == nil {
		return ""
	}
	return e.class
}

// ClassOf maps unexpected errors to a non-sensitive persistence failure.
func ClassOf(err error) ErrorClass {
	var target *Error
	if errors.As(err, &target) {
		return target.class
	}
	if err == nil {
		return ""
	}
	return ClassPersistence
}

func newError(class ErrorClass) error { return &Error{class: class} }

// Rules are copied and fixed for the lifetime of a Store.
type Rules struct {
	PolicyVersion   string
	RevocationEpoch uint64
	MaxTTL          time.Duration
	MaxRecords      int
}

// State is the complete set of durable request states.
type State string

const (
	Pending   State = "pending"
	Approved  State = "approved"
	Denied    State = "denied"
	Cancelled State = "cancelled"
	Expired   State = "expired"
	Stale     State = "stale"
)

// Record is an immutable caller view of a durable request.
type Record struct {
	requestID string
	manifest  []byte
	digest    string
	state     State
	createdAt time.Time
	expiresAt time.Time
	decidedAt time.Time
	actorID   string
}

func (r Record) RequestID() string    { return r.requestID }
func (r Record) Digest() string       { return r.digest }
func (r Record) State() State         { return r.state }
func (r Record) CreatedAt() time.Time { return r.createdAt }
func (r Record) ExpiresAt() time.Time { return r.expiresAt }
func (r Record) DecidedAt() time.Time { return r.decidedAt }
func (r Record) ActorID() string      { return r.actorID }
func (r Record) Encoding() []byte     { return append([]byte(nil), r.manifest...) }

// Snapshot is an immutable caller view. Records returns a fresh deep copy.
type Snapshot struct {
	version    uint64
	generation uint64
	observedAt time.Time
	records    []Record
}

func (s Snapshot) Version() uint64       { return s.version }
func (s Snapshot) Generation() uint64    { return s.generation }
func (s Snapshot) ObservedAt() time.Time { return s.observedAt }
func (s Snapshot) Records() []Record     { return cloneRecordList(s.records) }

type persistPhase uint8

const (
	phaseWrite persistPhase = iota + 1
	phaseFileSync
	phaseClose
	phaseRename
	phaseDirectorySync
)

type dependencies struct {
	now  func() time.Time
	hook func(persistPhase) error
}

// Store owns a process lock and serializes all state operations.
type Store struct {
	mu         sync.Mutex
	root       *os.Root
	rules      Rules
	lock       *os.File
	dir        *os.File
	records    map[string]Record
	generation uint64
	observed   time.Time
	lastNow    time.Time
	now        func() time.Time
	hook       func(persistPhase) error
	closed     bool
	poisoned   bool
}

// Open locks an existing owner-only directory and validates any snapshot.
func Open(root string, rules Rules) (*Store, error) {
	return openWith(root, rules, dependencies{now: time.Now})
}

func openWith(root string, rules Rules, deps dependencies) (*Store, error) {
	if err := validateRules(rules); err != nil || deps.now == nil {
		return nil, newError(ClassInvalid)
	}
	if !platformSupported() {
		return nil, newError(ClassUnsupported)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, newError(ClassInvalid)
	}
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, newError(ClassInvalid)
	}
	if info.Mode().Perm() != 0o700 || !ownedByCurrentUser(info) {
		return nil, newError(ClassPermission)
	}
	dir, err := openDirectoryNoFollow(root)
	if err != nil {
		if ClassOf(err) == ClassUnsupported {
			return nil, err
		}
		return nil, newError(ClassPermission)
	}
	if fi, statErr := dir.Stat(); statErr != nil || !os.SameFile(info, fi) {
		dir.Close()
		return nil, newError(ClassInvalid)
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		dir.Close()
		return nil, newError(ClassInvalid)
	}
	if fi, statErr := rootHandle.Stat("."); statErr != nil || !os.SameFile(info, fi) {
		rootHandle.Close()
		dir.Close()
		return nil, newError(ClassInvalid)
	}
	if _, err := rootHandle.Lstat(temporaryName); err == nil {
		rootHandle.Close()
		dir.Close()
		return nil, newError(ClassCorrupt)
	} else if !errors.Is(err, os.ErrNotExist) {
		rootHandle.Close()
		dir.Close()
		return nil, newError(ClassPermission)
	}
	lock, err := openProcessLock(rootHandle)
	if err != nil {
		rootHandle.Close()
		dir.Close()
		return nil, err
	}
	s := &Store{root: rootHandle, rules: rules, lock: lock, dir: dir, records: make(map[string]Record), now: deps.now, hook: deps.hook}
	if err := s.load(); err != nil {
		releaseProcessLock(lock)
		rootHandle.Close()
		dir.Close()
		return nil, err
	}
	return s, nil
}

func validateRules(r Rules) error {
	if !validScalar(r.PolicyVersion, 128) || r.MaxTTL <= 0 || r.MaxTTL%time.Second != 0 || r.MaxRecords <= 0 || r.MaxRecords > 10000 {
		return newError(ClassInvalid)
	}
	return nil
}

func (s *Store) load() error {
	info, err := s.root.Lstat(stateName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return newError(ClassPermission)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ownedByCurrentUser(info) {
		return newError(ClassPermission)
	}
	limit := int64(maxSnapshotBase + s.rules.MaxRecords*(approvalmanifest.MaxEncodingBytes+1024))
	if info.Size() <= 0 || info.Size() > limit {
		return newError(ClassCorrupt)
	}
	f, err := s.root.Open(stateName)
	if err != nil {
		return newError(ClassPermission)
	}
	defer f.Close()
	current, currentErr := s.root.Lstat(stateName)
	actual, statErr := f.Stat()
	if currentErr != nil || statErr != nil || !os.SameFile(info, current) || !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Mode().Perm() != 0o600 || !ownedByCurrentUser(actual) {
		return newError(ClassCorrupt)
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return newError(ClassCorrupt)
	}
	w, records, observed, err := decodeSnapshot(raw, s.rules)
	if err != nil {
		return err
	}
	s.generation, s.observed, s.lastNow, s.records = w.Generation, observed, observed, records
	return nil
}

// Create reparses canonical manifest bytes and durably creates a pending record.
func (s *Store) Create(raw []byte) (Record, error) {
	if s == nil {
		return Record{}, newError(ClassClosed)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.usable(); err != nil {
		return Record{}, err
	}
	now, err := s.trustedNow()
	if err != nil {
		return Record{}, err
	}
	m, err := approvalmanifest.Parse(raw)
	if err != nil || !bytes.Equal(raw, m.Encoding()) {
		return Record{}, newError(ClassInvalid)
	}
	if m.PolicyVersion() != s.rules.PolicyVersion || m.RevocationEpoch() != s.rules.RevocationEpoch {
		return Record{}, newError(ClassInvalid)
	}
	if m.CreatedAt().After(now) || !now.Before(m.ExpiresAt()) || m.ExpiresAt().Sub(m.CreatedAt()) > s.rules.MaxTTL {
		return Record{}, newError(ClassExpired)
	}
	if _, exists := s.records[m.RequestID()]; exists {
		return Record{}, newError(ClassConflict)
	}
	if len(s.records) >= s.rules.MaxRecords {
		return Record{}, newError(ClassCapacity)
	}
	r := Record{requestID: m.RequestID(), manifest: append([]byte(nil), raw...), digest: m.Digest(), state: Pending, createdAt: m.CreatedAt(), expiresAt: m.ExpiresAt()}
	next := cloneRecords(s.records)
	next[r.requestID] = r
	if err := s.commit(next, now); err != nil {
		return Record{}, err
	}
	return cloneRecord(r), nil
}

// Get returns a copy, first durably expiring the requested active record if due.
func (s *Store) Get(requestID string) (Record, error) {
	if s == nil {
		return Record{}, newError(ClassClosed)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.usable(); err != nil {
		return Record{}, err
	}
	now, err := s.trustedNow()
	if err != nil {
		return Record{}, err
	}
	r, ok := s.records[requestID]
	if !ok {
		return Record{}, newError(ClassNotFound)
	}
	if active(r.state) && !now.Before(r.expiresAt) {
		wasPending := r.state == Pending
		r.state, r.decidedAt = Expired, now
		if wasPending {
			r.actorID = ""
		}
		next := cloneRecords(s.records)
		next[requestID] = r
		if err := s.commit(next, now); err != nil {
			return Record{}, err
		}
	}
	return cloneRecord(r), nil
}

// Approve records a verified actor decision. It does not issue authorization.
func (s *Store) Approve(requestID, digest, actorID string) (Record, error) {
	return s.transition(requestID, digest, Approved, actorID)
}

// Deny records a verified actor decision.
func (s *Store) Deny(requestID, digest, actorID string) (Record, error) {
	return s.transition(requestID, digest, Denied, actorID)
}

// Cancel cancels a pending request without accepting caller-provided state/time.
func (s *Store) Cancel(requestID, digest string) (Record, error) {
	return s.transition(requestID, digest, Cancelled, "")
}

// MarkStale invalidates an approved request after an upstream policy event.
func (s *Store) MarkStale(requestID, digest string) (Record, error) {
	return s.transition(requestID, digest, Stale, "")
}

func (s *Store) transition(requestID, digest string, target State, actor string) (Record, error) {
	if s == nil {
		return Record{}, newError(ClassClosed)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.usable(); err != nil {
		return Record{}, err
	}
	if (target == Approved || target == Denied) && !validScalar(actor, maxActorBytes) {
		return Record{}, newError(ClassInvalid)
	}
	now, err := s.trustedNow()
	if err != nil {
		return Record{}, err
	}
	r, ok := s.records[requestID]
	if !ok {
		return Record{}, newError(ClassNotFound)
	}
	if active(r.state) && !now.Before(r.expiresAt) {
		wasPending := r.state == Pending
		r.state, r.decidedAt = Expired, now
		if wasPending {
			r.actorID = ""
		}
		next := cloneRecords(s.records)
		next[requestID] = r
		if err := s.commit(next, now); err != nil {
			return Record{}, err
		}
		return Record{}, newError(ClassExpired)
	}
	if !digestEqual(r.digest, digest) {
		return Record{}, newError(ClassDigest)
	}
	allowed := (r.state == Pending && (target == Approved || target == Denied || target == Cancelled)) || (r.state == Approved && target == Stale)
	if !allowed {
		return Record{}, newError(ClassTransition)
	}
	r.state, r.decidedAt = target, now
	if target == Approved || target == Denied {
		r.actorID = actor
	}
	next := cloneRecords(s.records)
	next[requestID] = r
	if err := s.commit(next, now); err != nil {
		return Record{}, err
	}
	return cloneRecord(r), nil
}

// ExpireDue durably expires all pending or approved records due at trusted now.
func (s *Store) ExpireDue() (int, error) {
	if s == nil {
		return 0, newError(ClassClosed)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.usable(); err != nil {
		return 0, err
	}
	now, err := s.trustedNow()
	if err != nil {
		return 0, err
	}
	next, count := expiredRecords(s.records, now)
	if count == 0 {
		return 0, nil
	}
	if err := s.commit(next, now); err != nil {
		return 0, err
	}
	return count, nil
}

// Snapshot returns a deep copy without reading caller-controlled storage.
func (s *Store) Snapshot() (Snapshot, error) {
	if s == nil {
		return Snapshot{}, newError(ClassClosed)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.usable(); err != nil {
		return Snapshot{}, err
	}
	now, err := s.trustedNow()
	if err != nil {
		return Snapshot{}, err
	}
	if next, count := expiredRecords(s.records, now); count > 0 {
		if err := s.commit(next, now); err != nil {
			return Snapshot{}, err
		}
	}
	ids := sortedIDs(s.records)
	records := make([]Record, 0, len(ids))
	for _, id := range ids {
		records = append(records, cloneRecord(s.records[id]))
	}
	return Snapshot{version: stateVersion, generation: s.generation, observedAt: s.observed, records: records}, nil
}

// Close releases the process lock once. Repeated calls are harmless.
func (s *Store) Close() error {
	if s == nil {
		return newError(ClassClosed)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	lock, root, dir := s.lock, s.root, s.dir
	s.lock, s.root, s.dir = nil, nil, nil
	if err := releaseProcessLock(lock); err != nil {
		if root != nil {
			_ = root.Close()
		}
		if dir != nil {
			_ = dir.Close()
		}
		return newError(ClassPersistence)
	}
	if root != nil {
		if err := root.Close(); err != nil {
			if dir != nil {
				_ = dir.Close()
			}
			return newError(ClassPersistence)
		}
	}
	if dir != nil {
		if err := dir.Close(); err != nil {
			return newError(ClassPersistence)
		}
	}
	return nil
}

func (s *Store) usable() error {
	if s.closed {
		return newError(ClassClosed)
	}
	if s.poisoned {
		return newError(ClassPoisoned)
	}
	return nil
}

func (s *Store) trustedNow() (time.Time, error) {
	raw := s.now()
	now := raw.UTC().Truncate(time.Second)
	if now.Year() < 1 || now.Year() > 9999 {
		return time.Time{}, newError(ClassClock)
	}
	if !s.lastNow.IsZero() && now.Before(s.lastNow) {
		return time.Time{}, newError(ClassClock)
	}
	s.lastNow = now
	return now, nil
}

type snapshotWire struct {
	FormatVersion uint64       `json:"format_version"`
	Generation    uint64       `json:"generation"`
	ObservedAt    string       `json:"observed_at"`
	Records       []recordWire `json:"records"`
}

type recordWire struct {
	RequestID string          `json:"request_id"`
	Manifest  json.RawMessage `json:"manifest"`
	Digest    string          `json:"digest"`
	State     State           `json:"state"`
	CreatedAt string          `json:"created_at"`
	ExpiresAt string          `json:"expires_at"`
	DecidedAt string          `json:"decided_at,omitempty"`
	ActorID   string          `json:"actor_id,omitempty"`
}

func (s *Store) commit(next map[string]Record, observed time.Time) error {
	if s.generation == ^uint64(0) {
		return newError(ClassCapacity)
	}
	w := snapshotWire{FormatVersion: stateVersion, Generation: s.generation + 1, ObservedAt: formatTime(observed)}
	for _, id := range sortedIDs(next) {
		r := next[id]
		wr := recordWire{RequestID: r.requestID, Manifest: append(json.RawMessage(nil), r.manifest...), Digest: r.digest, State: r.state, CreatedAt: formatTime(r.createdAt), ExpiresAt: formatTime(r.expiresAt), ActorID: r.actorID}
		if !r.decidedAt.IsZero() {
			wr.DecidedAt = formatTime(r.decidedAt)
		}
		w.Records = append(w.Records, wr)
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return newError(ClassPersistence)
	}
	if err := s.persist(raw); err != nil {
		return err
	}
	s.records, s.generation, s.observed = next, w.Generation, observed
	return nil
}

func (s *Store) persist(raw []byte) error {
	f, err := s.root.OpenFile(temporaryName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return newError(ClassPersistence)
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = s.root.Remove(temporaryName)
		return newError(ClassPersistence)
	}
	created, lstatErr := s.root.Lstat(temporaryName)
	info, statErr := f.Stat()
	if lstatErr != nil || statErr != nil || !os.SameFile(created, info) || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ownedByCurrentUser(info) {
		_ = f.Close()
		_ = s.root.Remove(temporaryName)
		return newError(ClassPersistence)
	}
	cleanup := func() { _ = f.Close(); _ = s.root.Remove(temporaryName) }
	if s.fail(phaseWrite) || writeAll(f, raw) != nil {
		cleanup()
		return newError(ClassPersistence)
	}
	if s.fail(phaseFileSync) || f.Sync() != nil {
		cleanup()
		return newError(ClassPersistence)
	}
	if err := f.Close(); err != nil || s.fail(phaseClose) {
		_ = s.root.Remove(temporaryName)
		return newError(ClassPersistence)
	}
	if s.fail(phaseRename) || s.root.Rename(temporaryName, stateName) != nil {
		s.poisoned = true
		_ = s.root.Remove(temporaryName)
		return newError(ClassPoisoned)
	}
	if s.fail(phaseDirectorySync) || s.dir.Sync() != nil {
		s.poisoned = true
		return newError(ClassPoisoned)
	}
	return nil
}

func (s *Store) fail(phase persistPhase) bool { return s.hook != nil && s.hook(phase) != nil }

func writeAll(w io.Writer, raw []byte) error {
	for len(raw) > 0 {
		n, err := w.Write(raw)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		raw = raw[n:]
	}
	return nil
}

func decodeSnapshot(raw []byte, rules Rules) (snapshotWire, map[string]Record, time.Time, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var w snapshotWire
	if err := decoder.Decode(&w); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return w, nil, time.Time{}, newError(ClassCorrupt)
	}
	canonical, err := json.Marshal(w)
	if err != nil || !bytes.Equal(raw, canonical) || w.FormatVersion != stateVersion || w.Generation == 0 || w.Records == nil || len(w.Records) > rules.MaxRecords {
		return w, nil, time.Time{}, newError(ClassCorrupt)
	}
	observed, ok := parseTime(w.ObservedAt)
	if !ok {
		return w, nil, time.Time{}, newError(ClassCorrupt)
	}
	records := make(map[string]Record, len(w.Records))
	previous := ""
	for i, wr := range w.Records {
		if (i > 0 && wr.RequestID <= previous) || wr.RequestID == "" {
			return w, nil, time.Time{}, newError(ClassCorrupt)
		}
		previous = wr.RequestID
		m, err := approvalmanifest.Parse(wr.Manifest)
		if err != nil || !bytes.Equal(wr.Manifest, m.Encoding()) || m.RequestID() != wr.RequestID || m.Digest() != wr.Digest || m.PolicyVersion() != rules.PolicyVersion || m.RevocationEpoch() != rules.RevocationEpoch {
			return w, nil, time.Time{}, newError(ClassCorrupt)
		}
		created, cok := parseTime(wr.CreatedAt)
		expires, eok := parseTime(wr.ExpiresAt)
		if !cok || !eok || !created.Equal(m.CreatedAt()) || !expires.Equal(m.ExpiresAt()) || expires.Sub(created) > rules.MaxTTL || observed.Before(created) {
			return w, nil, time.Time{}, newError(ClassCorrupt)
		}
		decided, shapeOK := validateWireState(wr, created, expires, observed)
		if !shapeOK {
			return w, nil, time.Time{}, newError(ClassCorrupt)
		}
		records[wr.RequestID] = Record{requestID: wr.RequestID, manifest: append([]byte(nil), wr.Manifest...), digest: wr.Digest, state: wr.State, createdAt: created, expiresAt: expires, decidedAt: decided, actorID: wr.ActorID}
	}
	return w, records, observed, nil
}

func validateWireState(w recordWire, created, expires, observed time.Time) (time.Time, bool) {
	if w.State == Pending {
		return time.Time{}, w.DecidedAt == "" && w.ActorID == "" && observed.Before(expires)
	}
	decided, ok := parseTime(w.DecidedAt)
	if !ok || decided.After(observed) || decided.Before(created) {
		return time.Time{}, false
	}
	switch w.State {
	case Approved, Denied:
		return decided, validScalar(w.ActorID, maxActorBytes) && decided.Before(expires)
	case Cancelled:
		return decided, w.ActorID == "" && decided.Before(expires)
	case Stale:
		return decided, validScalar(w.ActorID, maxActorBytes) && decided.Before(expires)
	case Expired:
		return decided, (w.ActorID == "" || validScalar(w.ActorID, maxActorBytes)) && !decided.Before(expires)
	default:
		return time.Time{}, false
	}
}

func formatTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05Z") }

func parseTime(raw string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02T15:04:05Z", raw)
	return t, err == nil && formatTime(t) == raw
}

func validScalar(v string, max int) bool {
	if len(v) == 0 || len(v) > max || strings.TrimSpace(v) != v {
		return false
	}
	for i := 0; i < len(v); i++ {
		if v[i] < 0x21 || v[i] > 0x7e {
			return false
		}
	}
	return true
}

func active(state State) bool { return state == Pending || state == Approved }

func expiredRecords(records map[string]Record, now time.Time) (map[string]Record, int) {
	next := cloneRecords(records)
	count := 0
	for id, r := range next {
		if active(r.state) && !now.Before(r.expiresAt) {
			wasPending := r.state == Pending
			r.state, r.decidedAt = Expired, now
			if wasPending {
				r.actorID = ""
			}
			next[id], count = r, count+1
		}
	}
	return next, count
}

func digestEqual(expected, actual string) bool {
	return len(expected) == len(actual) && subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func cloneRecord(r Record) Record { r.manifest = append([]byte(nil), r.manifest...); return r }

func cloneRecords(in map[string]Record) map[string]Record {
	out := make(map[string]Record, len(in))
	for k, v := range in {
		out[k] = cloneRecord(v)
	}
	return out
}

func cloneRecordList(in []Record) []Record {
	out := make([]Record, len(in))
	for i := range in {
		out[i] = cloneRecord(in[i])
	}
	return out
}

func sortedIDs(records map[string]Record) []string {
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
