// Package brokerexchange composes the existing egress transaction and
// request-scoped upstream forwarder into one synchronous, in-memory call.
// It owns no HTTP entrypoint, transport construction, credential storage, or
// retry policy. Every response sink and forwarding transaction is private to
// one Do invocation.
package brokerexchange

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/upstreamforwarder"
)

const maxResponseBytes = 1 << 20

// Error values are fixed and intentionally contain no request, response,
// credential, capability, URL, provider, or dependency detail.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrDenied       Error = "exchange-denied"
)

// Rules supplies the already constructed policy and capability registry, a
// credential resolver, and an injected upstream RoundTripper. No default
// transport or network client is selected by this package.
type Rules struct {
	Policy             *egresspolicy.Policy
	Registry           *capability.Registry
	Resolver           egresstransaction.CredentialResolver
	Transport          http.RoundTripper
	MaxCredentialBytes int
	Timeout            time.Duration
	MaxResponseBytes   int
}

// Response is the reduced, successful response exposed by the exchange.
// Body is always copied at the exchange boundary.
type Response = upstreamforwarder.Response

// Exchange is immutable after New returns. It retains only long-lived
// dependencies and numeric limits; request, credential, capability, sink,
// and response state are local to each Do call.
type Exchange struct {
	policy             *egresspolicy.Policy
	registry           *capability.Registry
	resolver           egresstransaction.CredentialResolver
	transport          http.RoundTripper
	maxCredentialBytes int
	timeout            time.Duration
	maxResponseBytes   int
}

// Format deliberately avoids exposing dependencies or configuration values.
func (e Exchange) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "brokerexchange.Exchange")
}

// New validates the fixed dependency and resource boundaries. Typed nil
// interface values are rejected before they can reach an existing
// constructor and cause a panic.
func New(r Rules) (*Exchange, error) {
	if r.Policy == nil || r.Registry == nil || isNil(r.Resolver) || isNil(r.Transport) ||
		r.MaxCredentialBytes < 1 || r.MaxCredentialBytes > 4096 ||
		r.Timeout < time.Millisecond || r.Timeout > 30*time.Second ||
		r.MaxResponseBytes < 1 || r.MaxResponseBytes > maxResponseBytes {
		return nil, ErrInvalidRules
	}
	return &Exchange{
		policy: r.Policy, registry: r.Registry, resolver: r.Resolver,
		transport: r.Transport, maxCredentialBytes: r.MaxCredentialBytes,
		timeout: r.Timeout, maxResponseBytes: r.MaxResponseBytes,
	}, nil
}

// Do executes exactly one transaction and one request-scoped forwarder. A
// result is returned only after one valid sink notification and a successful
// transaction. All failures are normalized to zero response and ErrDenied.
func (e *Exchange) Do(subject egresstransaction.Subject, request egresstransaction.Request) (Response, error) {
	if e == nil || e.policy == nil || e.registry == nil || isNil(e.resolver) || isNil(e.transport) ||
		e.maxCredentialBytes < 1 || e.maxCredentialBytes > 4096 ||
		e.timeout < time.Millisecond || e.timeout > 30*time.Second ||
		e.maxResponseBytes < 1 || e.maxResponseBytes > maxResponseBytes {
		return Response{}, ErrDenied
	}

	capture := &captureSink{}
	forwarder, forwardErr := upstreamforwarder.New(upstreamforwarder.Rules{
		Policy: e.policy, Transport: e.transport, Sink: capture,
		Timeout: e.timeout, MaxResponseBytes: e.maxResponseBytes,
	})
	if forwardErr != nil {
		return Response{}, ErrDenied
	}
	transaction, transactionErr := egresstransaction.New(egresstransaction.Rules{
		Policy: e.policy, Registry: e.registry, Resolver: e.resolver,
		Forwarder: forwarder, MaxCredentialBytes: e.maxCredentialBytes,
	})
	if transactionErr != nil {
		return Response{}, ErrDenied
	}
	if executeErr := transaction.Execute(subject, request); executeErr != nil {
		return Response{}, ErrDenied
	}

	result, ok := capture.snapshot()
	if !ok {
		return Response{}, ErrDenied
	}
	return result, nil
}

// captureSink is intentionally private and request-scoped. It rejects a
// second delivery and copies the body both on receipt and on snapshot.
type captureSink struct {
	delivered bool
	response  upstreamforwarder.Response
}

func (s *captureSink) Deliver(response upstreamforwarder.Response) error {
	if s == nil || s.delivered {
		return ErrDenied
	}
	s.delivered = true
	s.response = upstreamforwarder.Response{
		StatusCode: response.StatusCode, ContentType: response.ContentType,
		Body: append([]byte(nil), response.Body...),
	}
	return nil
}

func (s *captureSink) snapshot() (upstreamforwarder.Response, bool) {
	if s == nil || !s.delivered {
		return upstreamforwarder.Response{}, false
	}
	response := s.response
	response.Body = append([]byte(nil), response.Body...)
	return response, true
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
