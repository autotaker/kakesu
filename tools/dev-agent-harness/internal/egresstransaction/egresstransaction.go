// Package egresstransaction joins the pure egress policy and opaque
// capability registry.  It has no transport, persistence, or credential
// storage responsibilities.
package egresstransaction

import (
	"encoding/base64"
	"strings"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
)

// Error is a fixed transaction error.  Execute intentionally maps every
// failure (including resolver and forwarder details) to ErrDenied.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrDenied       Error = "transaction-denied"
)

// Rules contains only in-memory dependencies.  MaxCredentialBytes must be
// between one and 4096 inclusive.
type Rules struct {
	Policy             *egresspolicy.Policy
	Registry           *capability.Registry
	Resolver           CredentialResolver
	Forwarder          Forwarder
	MaxCredentialBytes int
}

// Subject is the canonical identity bound to both policy and capability
// evaluation. UID must be non-root and the string fields must be non-empty.
type Subject struct {
	AgentInstanceID string
	UID             int
	WorkspaceID     string
}

// Request is caller-owned input. Execute does not retain or modify any of its
// fields or slices.
type Request struct {
	Method        string
	URL           string
	ContentType   string
	Body          []byte
	Authorization []string
}

// PreparedRequest is the only value handed to a trusted Forwarder. Body is an
// independent copy; Authorization contains only the resolved upstream value
// in the scheme required by Scope and never the opaque capability handle.
type PreparedRequest struct {
	Method        string
	URL           string
	ContentType   string
	Body          []byte
	Scope         egresspolicy.Scope
	Authorization string
}

// CredentialResolver obtains an upstream credential after a capability has
// been consumed. Implementations must not be supplied with an opaque handle.
type CredentialResolver interface {
	Resolve(provider, repository string) (string, error)
}

// CredentialResolverFunc adapts a function to CredentialResolver.
type CredentialResolverFunc func(provider, repository string) (string, error)

func (f CredentialResolverFunc) Resolve(provider, repository string) (string, error) {
	return f(provider, repository)
}

// Forwarder synchronously receives one prepared request. This package does
// not implement forwarding or any network operation.
type Forwarder interface {
	Forward(PreparedRequest) error
}

// ForwarderFunc adapts a function to Forwarder.
type ForwarderFunc func(PreparedRequest) error

func (f ForwarderFunc) Forward(req PreparedRequest) error { return f(req) }

// Transaction is immutable after New. It does not retain request,
// Authorization, capability, or credential values between Execute calls.
type Transaction struct {
	policy             *egresspolicy.Policy
	registry           *capability.Registry
	resolver           CredentialResolver
	forwarder          Forwarder
	maxCredentialBytes int
}

// New validates dependency presence and credential length bounds. A non-nil
// zero Policy or Registry is accepted and fails closed during Execute.
func New(r Rules) (*Transaction, error) {
	if r.Policy == nil || r.Registry == nil || r.Resolver == nil || r.Forwarder == nil ||
		r.MaxCredentialBytes < 1 || r.MaxCredentialBytes > 4096 {
		return nil, ErrInvalidRules
	}
	return &Transaction{
		policy: r.Policy, registry: r.Registry, resolver: r.Resolver,
		forwarder: r.Forwarder, maxCredentialBytes: r.MaxCredentialBytes,
	}, nil
}

// Execute performs exactly one ordered transaction: policy allow, strict
// Authorization extraction, capability Consume, credential resolution and
// credential validation, then one synchronous Forwarder call. A successful
// call returns nil and no credential-bearing value.
func (t *Transaction) Execute(subject Subject, req Request) error {
	if t == nil || t.policy == nil || t.registry == nil || t.resolver == nil || t.forwarder == nil {
		return ErrDenied
	}
	scope, decision, err := t.policy.Evaluate(egresspolicy.Request{
		Method: req.Method, URL: req.URL, ContentType: req.ContentType, Body: req.Body,
	})
	if err != nil || decision == egresspolicy.DecisionDeny {
		return ErrDenied
	}
	handle, ok := extractCapability(scope, req.Authorization)
	if !ok {
		return ErrDenied
	}
	grant, err := t.registry.Consume(capability.Request{
		Handle: handle, AgentInstanceID: subject.AgentInstanceID, UID: subject.UID,
		WorkspaceID: subject.WorkspaceID, Provider: scope.Provider,
		Repository: scope.Repository, Operation: scope.Operation,
		DestinationHost: scope.DestinationHost,
	})
	if err != nil || !grantMatches(grant, subject, scope) {
		return ErrDenied
	}
	credential, err := t.resolver.Resolve(scope.Provider, scope.Repository)
	if err != nil || !validCredential(credential, t.maxCredentialBytes) {
		return ErrDenied
	}
	authorization := "Bearer " + credential
	if scope.Operation == capability.OperationGitHubGitRead {
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+credential))
	}
	prepared := PreparedRequest{
		Method: req.Method, URL: req.URL, ContentType: req.ContentType,
		Body: append([]byte(nil), req.Body...), Scope: scope,
		Authorization: authorization,
	}
	if err := t.forwarder.Forward(prepared); err != nil {
		return ErrDenied
	}
	return nil
}

func grantMatches(grant capability.Grant, subject Subject, scope egresspolicy.Scope) bool {
	return grant.AgentInstanceID == subject.AgentInstanceID && grant.UID == subject.UID &&
		grant.WorkspaceID == subject.WorkspaceID && grant.Provider == scope.Provider &&
		grant.Repository == scope.Repository && grant.Operation == scope.Operation &&
		grant.DestinationHost == scope.DestinationHost
}

func extractCapability(scope egresspolicy.Scope, values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	value := values[0]
	if scope.Operation == capability.OperationGitHubGitRead {
		return extractGitBasicCapability(value)
	}
	if strings.Count(value, " ") != 1 {
		return "", false
	}
	var prefix string
	switch scope.Provider {
	case capability.ProviderOpenAI:
		prefix = "Bearer "
	case capability.ProviderGitHub:
		if strings.HasPrefix(value, "Bearer ") {
			prefix = "Bearer "
		} else if strings.HasPrefix(value, "token ") {
			prefix = "token "
		} else {
			return "", false
		}
	default:
		return "", false
	}
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}
	handle := strings.TrimPrefix(value, prefix)
	if handle == "" || !strings.HasPrefix(handle, "cap_") {
		return "", false
	}
	for i := 0; i < len(handle); i++ {
		if handle[i] < 0x21 || handle[i] > 0x7e {
			return "", false
		}
	}
	return handle, true
}

func extractGitBasicCapability(value string) (string, bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(value, prefix) || strings.Count(value, " ") != 1 {
		return "", false
	}
	encoded := strings.TrimPrefix(value, prefix)
	if encoded == "" || len(encoded) > 8*1024 {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != encoded ||
		strings.Count(string(decoded), ":") != 1 || !strings.HasPrefix(string(decoded), "x-access-token:") {
		return "", false
	}
	handle := strings.TrimPrefix(string(decoded), "x-access-token:")
	if !canonicalHandle(handle) {
		return "", false
	}
	return handle, true
}

func canonicalHandle(handle string) bool {
	if len(handle) != len("cap_")+43 || !strings.HasPrefix(handle, "cap_") {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(handle, "cap_"))
	return err == nil && len(raw) == 32 && "cap_"+base64.RawURLEncoding.EncodeToString(raw) == handle
}

func validCredential(value string, max int) bool {
	if len(value) < 1 || len(value) > max || len(value) > 4096 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}
