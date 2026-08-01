// Package upstreamforwarder adapts an authorized transaction request to one
// injected upstream exchange and reduces successful responses to a small sink
// value. It deliberately does not own a client, proxy, redirect, or retry.
package upstreamforwarder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

const (
	maxCredentialBytes = 4096
	maxResponseBytes   = 1 << 20
	userAgent          = "kakesu-dev-agent-harness/1"
)

// Error values are fixed and intentionally contain no request, response,
// credential, URL, or underlying transport detail.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrForward      Error = "upstream-forward-failed"
)

// Response is the only value exposed to a response sink. Body is an
// independent copy and is empty for an empty successful response.
type Response struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

// ResponseSink receives one fully validated successful response.
type ResponseSink interface {
	Deliver(Response) error
}

// ResponseSinkFunc adapts a function to ResponseSink.
type ResponseSinkFunc func(Response) error

func (f ResponseSinkFunc) Deliver(response Response) error { return f(response) }

// Rules supplies the existing policy, a trusted injected RoundTripper, a
// request-scoped sink, and bounded request/response execution settings.
type Rules struct {
	Policy           *egresspolicy.Policy
	Transport        http.RoundTripper
	Sink             ResponseSink
	Timeout          time.Duration
	MaxResponseBytes int
}

// Forwarder implements egresstransaction.Forwarder.
type Forwarder struct {
	policy           *egresspolicy.Policy
	transport        http.RoundTripper
	sink             ResponseSink
	timeout          time.Duration
	maxResponseBytes int
}

var _ egresstransaction.Forwarder = (*Forwarder)(nil)

// Format hides all injected dependencies and request/response values from
// ordinary formatting and diagnostics.
func (f Forwarder) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "upstreamforwarder.Forwarder")
}

// New validates all bounds and dependencies. No dependency is copied or
// wrapped; in particular, no default transport is ever selected.
func New(r Rules) (*Forwarder, error) {
	if isNil(r.Policy) || isNil(r.Transport) || isNil(r.Sink) ||
		r.Timeout < time.Millisecond || r.Timeout > 30*time.Second ||
		r.MaxResponseBytes < 1 || r.MaxResponseBytes > maxResponseBytes {
		return nil, ErrInvalidRules
	}
	return &Forwarder{
		policy: r.Policy, transport: r.Transport, sink: r.Sink,
		timeout: r.Timeout, maxResponseBytes: r.MaxResponseBytes,
	}, nil
}

// Forward performs one policy re-evaluation, one injected RoundTrip, and at
// most one sink delivery. It returns only fixed errors.
func (f *Forwarder) Forward(prepared egresstransaction.PreparedRequest) error {
	if f == nil || isNil(f.policy) || isNil(f.transport) || isNil(f.sink) ||
		f.timeout < time.Millisecond || f.timeout > 30*time.Second ||
		f.maxResponseBytes < 1 || f.maxResponseBytes > maxResponseBytes {
		return ErrForward
	}

	scope, decision, err := f.policy.Evaluate(egresspolicy.Request{
		Method: prepared.Method, URL: prepared.URL,
		ContentType: prepared.ContentType, Body: prepared.Body,
	})
	if err != nil || decision == egresspolicy.DecisionDeny || scope != prepared.Scope ||
		!validBearer(prepared.Authorization) ||
		(scope.Provider == "github" && (prepared.ContentType != "" || len(prepared.Body) != 0)) {
		return ErrForward
	}

	// Keep no caller-owned bytes in the request handed to the transport.
	body := append([]byte(nil), prepared.Body...)
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, prepared.Method, prepared.URL, bytes.NewReader(body))
	if err != nil {
		return ErrForward
	}
	request.Header.Set("Authorization", prepared.Authorization)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", userAgent)
	if scope.Provider == "openai" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, roundTripErr := f.transport.RoundTrip(request)
	if roundTripErr != nil || response == nil || response.Body == nil || ctx.Err() != nil {
		closeBody(response)
		return ErrForward
	}

	data, readErr := io.ReadAll(io.LimitReader(response.Body, int64(f.maxResponseBytes)+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || ctx.Err() != nil {
		return ErrForward
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices ||
		len(data) > f.maxResponseBytes {
		return ErrForward
	}
	if (prepared.Method == http.MethodHead || response.StatusCode == http.StatusNoContent) && len(data) != 0 {
		return ErrForward
	}

	contentType := ""
	if len(data) > 0 {
		if !utf8.Valid(data) || !json.Valid(data) {
			return ErrForward
		}
		if !jsonMediaType(response.Header.Get("Content-Type")) {
			return ErrForward
		}
		contentType = "application/json"
	}

	// The second copy keeps sink ownership independent from the read buffer,
	// transport body, and all caller-owned input.
	result := Response{StatusCode: response.StatusCode, ContentType: contentType, Body: append([]byte(nil), data...)}
	if f.sink.Deliver(result) != nil {
		return ErrForward
	}
	return nil
}

func closeBody(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}

func validBearer(value string) bool {
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) || len(value) == len(prefix) || len(value)-len(prefix) > maxCredentialBytes {
		return false
	}
	for i := len(prefix); i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func jsonMediaType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(mediaType)
	return mediaType == "application/json" || (strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json"))
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
