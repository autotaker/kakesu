// Package approvaldecision connects one-shot approval verification to the
// durable approval state transition. A successful result is only a recorded
// decision; it is not a grant or authorization to push.
package approvaldecision

import (
	"errors"
	"strings"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/approvalchallenge"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/approvalstate"
)

// ErrorClass is a stable, non-sensitive coordinator failure category.
type ErrorClass string

const (
	ClassInvalid      ErrorClass = "invalid"
	ClassBegin        ErrorClass = "begin"
	ClassVerification ErrorClass = "verification"
	ClassDecision     ErrorClass = "decision"
)

// Error never includes request, operator, challenge, digest, assertion,
// credential, or lower-level error data.
type Error struct{ class ErrorClass }

func (e *Error) Error() string {
	if e == nil {
		return "approval decision error"
	}
	return "approval decision " + string(e.class)
}

// Class returns the stable category of e.
func (e *Error) Class() ErrorClass {
	if e == nil {
		return ""
	}
	return e.class
}

// ClassOf maps an unexpected error to a non-sensitive decision failure.
func ClassOf(err error) ErrorClass {
	var target *Error
	if errors.As(err, &target) {
		return target.class
	}
	if err == nil {
		return ""
	}
	return ClassDecision
}

func newError(class ErrorClass) error { return &Error{class: class} }

// Issued is an immutable caller view of a challenge and its exact binding.
type Issued struct {
	challenge  string
	requestID  string
	digest     string
	decision   approvalchallenge.Decision
	operatorID string
	rpID       string
	origin     string
	issuedAt   time.Time
	expiresAt  time.Time
}

func (i Issued) Challenge() string                    { return i.challenge }
func (i Issued) RequestID() string                    { return i.requestID }
func (i Issued) Digest() string                       { return i.digest }
func (i Issued) Decision() approvalchallenge.Decision { return i.decision }
func (i Issued) OperatorID() string                   { return i.operatorID }
func (i Issued) RPID() string                         { return i.rpID }
func (i Issued) Origin() string                       { return i.origin }
func (i Issued) IssuedAt() time.Time                  { return i.issuedAt }
func (i Issued) ExpiresAt() time.Time                 { return i.expiresAt }

// Result contains only the durable decision fields and the stable credential
// identifier produced by the fixed verifier.
type Result struct {
	requestID    string
	digest       string
	state        approvalstate.State
	actorID      string
	credentialID string
}

func (r Result) RequestID() string          { return r.requestID }
func (r Result) Digest() string             { return r.digest }
func (r Result) State() approvalstate.State { return r.state }
func (r Result) ActorID() string            { return r.actorID }
func (r Result) CredentialID() string       { return r.credentialID }

type stateRecord struct {
	requestID string
	digest    string
	state     approvalstate.State
	actorID   string
}

type stateOperations interface {
	get(string) (stateRecord, error)
	approve(string, string, string) (stateRecord, error)
	deny(string, string, string) (stateRecord, error)
}

type challengeRequest struct {
	requestID  string
	digest     string
	decision   approvalchallenge.Decision
	operatorID string
	rpID       string
	origin     string
}

type challengeIssued struct {
	challenge string
	binding   challengeRequest
	issuedAt  time.Time
	expiresAt time.Time
}

type challengeVerified struct {
	requestID    string
	digest       string
	decision     approvalchallenge.Decision
	operatorID   string
	credentialID string
}

type challengeOperations interface {
	issue(challengeRequest) (challengeIssued, error)
	consume(string, []byte) (challengeVerified, error)
}

// Coordinator fixes its durable store, challenge manager, and trusted
// verifier for its complete lifetime.
type Coordinator struct {
	state      stateOperations
	challenges challengeOperations
}

// New constructs the only production wiring. Begin and Complete do not accept
// dependency, verifier, digest, state, challenge generator, or clock seams.
func New(store *approvalstate.Store, manager *approvalchallenge.Manager, verifier approvalchallenge.Verifier) (*Coordinator, error) {
	if store == nil || manager == nil || verifier == nil {
		return nil, newError(ClassInvalid)
	}
	return newCoordinator(stateAdapter{store: store}, challengeAdapter{manager: manager, verifier: verifier})
}

func newCoordinator(state stateOperations, challenges challengeOperations) (*Coordinator, error) {
	if state == nil || challenges == nil {
		return nil, newError(ClassInvalid)
	}
	return &Coordinator{state: state, challenges: challenges}, nil
}

// Begin reads the durable record first and issues a challenge only for the
// exact pending request ID and digest returned by that read.
func (c *Coordinator) Begin(requestID string, decision approvalchallenge.Decision, operatorID, rpID, origin string) (Issued, error) {
	if c == nil || c.state == nil || c.challenges == nil {
		return Issued{}, newError(ClassInvalid)
	}
	record, err, panicked := safeGet(c.state, requestID)
	if panicked || err != nil || record.state != approvalstate.Pending {
		return Issued{}, newError(ClassBegin)
	}
	request := challengeRequest{
		requestID:  record.requestID,
		digest:     record.digest,
		decision:   decision,
		operatorID: operatorID,
		rpID:       rpID,
		origin:     origin,
	}
	issued, err, panicked := safeIssue(c.challenges, request)
	if panicked || err != nil || !validIssued(issued, request) {
		return Issued{}, newError(ClassBegin)
	}
	return Issued{
		challenge:  issued.challenge,
		requestID:  issued.binding.requestID,
		digest:     issued.binding.digest,
		decision:   issued.binding.decision,
		operatorID: issued.binding.operatorID,
		rpID:       issued.binding.rpID,
		origin:     issued.binding.origin,
		issuedAt:   issued.issuedAt,
		expiresAt:  issued.expiresAt,
	}, nil
}

// Complete consumes the challenge with the fixed verifier before attempting
// exactly one durable Approve or Deny transition. A failed transition leaves
// the challenge consumed and returns no result.
func (c *Coordinator) Complete(challenge string, assertion []byte) (Result, error) {
	if c == nil || c.state == nil || c.challenges == nil {
		return Result{}, newError(ClassInvalid)
	}
	verified, err, panicked := safeConsume(c.challenges, challenge, append([]byte(nil), assertion...))
	if panicked || err != nil || !validVerified(verified) {
		return Result{}, newError(ClassVerification)
	}

	var record stateRecord
	switch verified.decision {
	case approvalchallenge.Approve:
		record, err, panicked = safeApprove(c.state, verified.requestID, verified.digest, verified.operatorID)
	case approvalchallenge.Deny:
		record, err, panicked = safeDeny(c.state, verified.requestID, verified.digest, verified.operatorID)
	default:
		return Result{}, newError(ClassVerification)
	}
	if panicked || err != nil || !matchesTransition(record, verified) {
		return Result{}, newError(ClassDecision)
	}
	return Result{
		requestID:    record.requestID,
		digest:       record.digest,
		state:        record.state,
		actorID:      record.actorID,
		credentialID: verified.credentialID,
	}, nil
}

func validVerified(v challengeVerified) bool {
	return v.requestID != "" && v.digest != "" && v.operatorID != "" &&
		validCredentialID(v.credentialID)
}

func validIssued(issued challengeIssued, request challengeRequest) bool {
	return issued.challenge != "" && issued.binding == request &&
		!issued.issuedAt.IsZero() && issued.issuedAt.Before(issued.expiresAt)
}

func validCredentialID(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for index := len("sha256:"); index < len(value); index++ {
		if (value[index] < '0' || value[index] > '9') && (value[index] < 'a' || value[index] > 'f') {
			return false
		}
	}
	return true
}

func matchesTransition(record stateRecord, verified challengeVerified) bool {
	expected := approvalstate.Approved
	if verified.decision == approvalchallenge.Deny {
		expected = approvalstate.Denied
	}
	return record.requestID == verified.requestID && record.digest == verified.digest &&
		record.actorID == verified.operatorID && record.state == expected
}

type stateAdapter struct{ store *approvalstate.Store }

func (a stateAdapter) get(requestID string) (stateRecord, error) {
	record, err := a.store.Get(requestID)
	return fromStateRecord(record), err
}

func (a stateAdapter) approve(requestID, digest, actorID string) (stateRecord, error) {
	record, err := a.store.Approve(requestID, digest, actorID)
	return fromStateRecord(record), err
}

func (a stateAdapter) deny(requestID, digest, actorID string) (stateRecord, error) {
	record, err := a.store.Deny(requestID, digest, actorID)
	return fromStateRecord(record), err
}

func fromStateRecord(record approvalstate.Record) stateRecord {
	return stateRecord{
		requestID: record.RequestID(),
		digest:    record.Digest(),
		state:     record.State(),
		actorID:   record.ActorID(),
	}
}

type challengeAdapter struct {
	manager  *approvalchallenge.Manager
	verifier approvalchallenge.Verifier
}

func (a challengeAdapter) issue(request challengeRequest) (challengeIssued, error) {
	issued, err := a.manager.Issue(approvalchallenge.Request{
		RequestID:  request.requestID,
		Digest:     request.digest,
		Decision:   request.decision,
		OperatorID: request.operatorID,
		RPID:       request.rpID,
		Origin:     request.origin,
	})
	if err != nil {
		return challengeIssued{}, err
	}
	binding := issued.Binding()
	return challengeIssued{
		challenge: issued.Challenge(),
		binding: challengeRequest{
			requestID:  binding.RequestID(),
			digest:     binding.Digest(),
			decision:   binding.Decision(),
			operatorID: binding.OperatorID(),
			rpID:       binding.RPID(),
			origin:     binding.Origin(),
		},
		issuedAt:  binding.IssuedAt(),
		expiresAt: binding.ExpiresAt(),
	}, nil
}

func (a challengeAdapter) consume(challenge string, assertion []byte) (challengeVerified, error) {
	verified, err := a.manager.Consume(challenge, assertion, a.verifier)
	if err != nil {
		return challengeVerified{}, err
	}
	return challengeVerified{
		requestID:    verified.RequestID(),
		digest:       verified.Digest(),
		decision:     verified.Decision(),
		operatorID:   verified.OperatorID(),
		credentialID: verified.CredentialID(),
	}, nil
}

func safeGet(state stateOperations, requestID string) (record stateRecord, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			record, err, panicked = stateRecord{}, nil, true
		}
	}()
	record, err = state.get(requestID)
	return record, err, false
}

func safeIssue(challenges challengeOperations, request challengeRequest) (issued challengeIssued, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			issued, err, panicked = challengeIssued{}, nil, true
		}
	}()
	issued, err = challenges.issue(request)
	return issued, err, false
}

func safeConsume(challenges challengeOperations, challenge string, assertion []byte) (verified challengeVerified, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			verified, err, panicked = challengeVerified{}, nil, true
		}
	}()
	verified, err = challenges.consume(challenge, assertion)
	return verified, err, false
}

func safeApprove(state stateOperations, requestID, digest, actorID string) (record stateRecord, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			record, err, panicked = stateRecord{}, nil, true
		}
	}()
	record, err = state.approve(requestID, digest, actorID)
	return record, err, false
}

func safeDeny(state stateOperations, requestID, digest, actorID string) (record stateRecord, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			record, err, panicked = stateRecord{}, nil, true
		}
	}()
	record, err = state.deny(requestID, digest, actorID)
	return record, err, false
}
