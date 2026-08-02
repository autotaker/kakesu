// Package capabilitycontrol issues and revokes narrowly scoped capabilities
// for a subject already authenticated by the listener. It owns no protocol,
// credential, transport, listener, retry, or persistence responsibilities.
package capabilitycontrol

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

const (
	issueTTL        = 5 * time.Minute
	issueUses       = 1
	maxRepositories = 32
	maxModels       = 32
)

// Error values are deliberately fixed. They never contain a subject, handle,
// repository, provider, allowlist value, or lower-level error.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrDenied       Error = "capability-control-denied"
)

// SubjectResolver obtains the private peer identity from context. Production
// uses brokerlistener.Resolver; there is intentionally no subject setter.
type SubjectResolver interface {
	Resolve(context.Context) (egresstransaction.Subject, error)
}

// SubjectResolverFunc adapts a context-only function for bounded tests.
type SubjectResolverFunc func(context.Context) (egresstransaction.Subject, error)

func (f SubjectResolverFunc) Resolve(ctx context.Context) (egresstransaction.Subject, error) {
	return f(ctx)
}

// Rules contains the one shared in-memory Registry, the peer-context
// resolver, and configuration-derived issuance gates.
type Rules struct {
	Registry           *capability.Registry
	Resolver           SubjectResolver
	GitHubRepositories []string
	OpenAIModels       []string
}

// Controller is immutable after New. The model values are not retained:
// their non-empty presence is only an issuance gate, while request model
// authorization remains in the existing egress policy.
type Controller struct {
	registry      *capability.Registry
	resolver      SubjectResolver
	repositories  map[string]struct{}
	openAIEnabled bool
}

// New validates and copies the repository allowlist. OpenAI model names are
// deliberately not interpreted here.
func New(r Rules) (*Controller, error) {
	if r.Registry == nil || isNil(r.Resolver) || len(r.GitHubRepositories) > maxRepositories || len(r.OpenAIModels) > maxModels {
		return nil, ErrInvalidRules
	}
	repositories := make(map[string]struct{}, len(r.GitHubRepositories))
	for _, repository := range r.GitHubRepositories {
		if !canonicalRepository(repository) {
			return nil, ErrInvalidRules
		}
		if _, duplicate := repositories[repository]; duplicate {
			return nil, ErrInvalidRules
		}
		repositories[strings.Clone(repository)] = struct{}{}
	}
	return &Controller{
		registry: r.Registry, resolver: r.Resolver, repositories: repositories,
		openAIEnabled: len(r.OpenAIModels) > 0,
	}, nil
}

// Format exposes only a stable type name.
func (c Controller) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "capabilitycontrol.Controller")
}

// Issue accepts only the provider, its minimum repository scope, and an
// optional explicit operation selector. The omitted selector preserves the
// existing REST/OpenAI behavior; Git Smart HTTP is never selected implicitly.
// Subject, TTL, uses, and host remain fixed by trusted dependencies.
func (c *Controller) Issue(ctx context.Context, provider, repository string, operations ...string) (string, error) {
	if c == nil || c.registry == nil || isNil(c.resolver) || isNilContext(ctx) || ctx.Err() != nil {
		return "", ErrDenied
	}
	if len(operations) > 1 {
		return "", ErrDenied
	}
	operation := ""
	if len(operations) == 1 {
		operation = operations[0]
	}
	switch provider {
	case capability.ProviderGitHub:
		if operation != "" && operation != capability.OperationGitHubRESTRead && operation != capability.OperationGitHubGitRead {
			return "", ErrDenied
		}
		if !canonicalRepository(repository) {
			return "", ErrDenied
		}
		if _, allowed := c.repositories[repository]; !allowed {
			return "", ErrDenied
		}
	case capability.ProviderOpenAI:
		if repository != "" || !c.openAIEnabled || operation != "" && operation != capability.OperationOpenAIResponsesText {
			return "", ErrDenied
		}
	default:
		return "", ErrDenied
	}
	subject, err := c.resolver.Resolve(ctx)
	if err != nil || !validSubject(subject) || ctx.Err() != nil {
		return "", ErrDenied
	}
	handle, err := c.registry.Issue(capability.IssueSpec{
		AgentInstanceID: subject.AgentInstanceID,
		UID:             subject.UID,
		WorkspaceID:     subject.WorkspaceID,
		Provider:        provider,
		Repository:      repository,
		Operation:       operation,
		TTL:             issueTTL,
		Uses:            issueUses,
	})
	if err != nil || handle == "" {
		return "", ErrDenied
	}
	return handle, nil
}

// Revoke removes one canonical handle only for the subject resolved from the
// same listener-bound context.
func (c *Controller) Revoke(ctx context.Context, handle string) error {
	if c == nil || c.registry == nil || isNil(c.resolver) || isNilContext(ctx) || ctx.Err() != nil {
		return ErrDenied
	}
	subject, err := c.resolver.Resolve(ctx)
	if err != nil || !validSubject(subject) || ctx.Err() != nil {
		return ErrDenied
	}
	if err := c.registry.RevokeForSubject(capability.RevokeRequest{
		Handle: handle, AgentInstanceID: subject.AgentInstanceID,
		UID: subject.UID, WorkspaceID: subject.WorkspaceID,
	}); err != nil {
		return ErrDenied
	}
	return nil
}

func validSubject(subject egresstransaction.Subject) bool {
	return subject.UID > 0 && validIdentifier(subject.AgentInstanceID, 128) && validIdentifier(subject.WorkspaceID, 128)
}

func canonicalRepository(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && canonicalRepositoryPart(parts[0]) && canonicalRepositoryPart(parts[1])
}

func canonicalRepositoryPart(value string) bool {
	if value == "" || value == "." || value == ".." || !lowerAlphaNumeric(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		char := value[index]
		if lowerAlphaNumeric(char) || char == '.' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func lowerAlphaNumeric(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

func validIdentifier(value string, max int) bool {
	if len(value) == 0 || len(value) > max || !alphaNumeric(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		char := value[index]
		if !alphaNumeric(char) && char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func alphaNumeric(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9'
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func isNilContext(ctx context.Context) bool {
	return isNil(ctx)
}
