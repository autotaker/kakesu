package brokerexchange

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

var testSubject = egresstransaction.Subject{AgentInstanceID: "agent-1", UID: 1000, WorkspaceID: "workspace-1"}

type fakeResolver struct {
	mu         sync.Mutex
	calls      int
	credential string
	err        error
}

func (r *fakeResolver) Resolve(string, string) (string, error) {
	r.mu.Lock()
	r.calls++
	err, credential := r.err, r.credential
	r.mu.Unlock()
	return credential, err
}

func (r *fakeResolver) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

type fakeTransport struct {
	mu      sync.Mutex
	calls   int
	request *http.Request
	factory func(int) *http.Response
	err     error
}

func (t *fakeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.calls++
	call := t.calls
	t.request = request
	factory, err := t.factory, t.err
	t.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if factory != nil {
		return factory(call), nil
	}
	return &http.Response{
		StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}, nil
}

func (t *fakeTransport) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.calls
}

func (t *fakeTransport) lastRequest() *http.Request {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.request
}

func testPolicy(t *testing.T) *egresspolicy.Policy {
	t.Helper()
	policy, err := egresspolicy.New(egresspolicy.Rules{
		GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"},
		MaxBodyBytes: 4096, MaxOutputTokens: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func testRegistry(t *testing.T, provider string, uses int) (*capability.Registry, string) {
	t.Helper()
	registry, err := capability.New(capability.Rules{PolicyVersion: "policy-v1", MaxTTL: time.Hour, MaxUses: 4})
	if err != nil {
		t.Fatal(err)
	}
	spec := capability.IssueSpec{AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID,
		WorkspaceID: testSubject.WorkspaceID, Provider: provider, TTL: time.Minute, Uses: uses}
	if provider == capability.ProviderGitHub {
		spec.Repository = "acme/widget"
	}
	handle, err := registry.Issue(spec)
	if err != nil {
		t.Fatal(err)
	}
	return registry, handle
}

func newExchange(t *testing.T, registry *capability.Registry, resolver *fakeResolver, transport *fakeTransport) *Exchange {
	t.Helper()
	exchange, err := New(Rules{Policy: testPolicy(t), Registry: registry, Resolver: resolver,
		Transport: transport, MaxCredentialBytes: 128, Timeout: time.Second, MaxResponseBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	return exchange
}

func githubRequest(handle string) egresstransaction.Request {
	return egresstransaction.Request{Method: http.MethodGet, URL: "https://api.github.com/repos/acme/widget/issues",
		Authorization: []string{"Bearer " + handle}}
}

func openAIRequest(handle string) egresstransaction.Request {
	return egresstransaction.Request{Method: http.MethodPost, URL: "https://api.openai.com/v1/responses",
		ContentType: "application/json", Body: []byte(`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":1}`),
		Authorization: []string{"Bearer " + handle}}
}

func TestNewRejectsBoundsAndTypedNil(t *testing.T) {
	registry, _ := testRegistry(t, capability.ProviderGitHub, 1)
	resolver := &fakeResolver{credential: "secret"}
	transport := &fakeTransport{}
	base := Rules{Policy: testPolicy(t), Registry: registry, Resolver: resolver, Transport: transport,
		MaxCredentialBytes: 1, Timeout: time.Millisecond, MaxResponseBytes: 1}
	for name, rules := range map[string]Rules{
		"zero": {}, "credential low": func() Rules { r := base; r.MaxCredentialBytes = 0; return r }(),
		"credential high": func() Rules { r := base; r.MaxCredentialBytes = 4097; return r }(),
		"timeout low":     func() Rules { r := base; r.Timeout = time.Microsecond; return r }(),
		"timeout high":    func() Rules { r := base; r.Timeout = 30*time.Second + time.Nanosecond; return r }(),
		"response low":    func() Rules { r := base; r.MaxResponseBytes = 0; return r }(),
		"response high":   func() Rules { r := base; r.MaxResponseBytes = maxResponseBytes + 1; return r }(),
	} {
		t.Run(name, func(t *testing.T) {
			if exchange, err := New(rules); exchange != nil || !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("New returned exchange=%p error=%v", exchange, err)
			}
		})
	}
	var nilResolver *fakeResolver
	var nilTransport *fakeTransport
	for name, rules := range map[string]Rules{
		"typed nil resolver":  func() Rules { r := base; r.Resolver = nilResolver; return r }(),
		"typed nil transport": func() Rules { r := base; r.Transport = nilTransport; return r }(),
	} {
		t.Run(name, func(t *testing.T) {
			if exchange, err := New(rules); exchange != nil || !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("New returned exchange=%p error=%v", exchange, err)
			}
		})
	}
	var zero Exchange
	response, err := zero.Do(testSubject, egresstransaction.Request{})
	if !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("zero Do response=%#v error=%v", response, err)
	}
	var nilExchange *Exchange
	response, err = nilExchange.Do(testSubject, egresstransaction.Request{})
	if !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("nil Do response=%#v error=%v", response, err)
	}
}

func TestBothProvidersBearerReplacementAndCopies(t *testing.T) {
	for _, provider := range []string{capability.ProviderGitHub, capability.ProviderOpenAI} {
		t.Run(provider, func(t *testing.T) {
			registry, handle := testRegistry(t, provider, 1)
			resolver := &fakeResolver{credential: "upstream-secret"}
			transport := &fakeTransport{}
			exchange := newExchange(t, registry, resolver, transport)
			request := githubRequest(handle)
			if provider == capability.ProviderOpenAI {
				request = openAIRequest(handle)
			}
			beforeBody := append([]byte(nil), request.Body...)
			beforeAuth := append([]string(nil), request.Authorization...)
			response, err := exchange.Do(testSubject, request)
			if err != nil || response.StatusCode != http.StatusOK || response.ContentType != "application/json" || string(response.Body) != `{"ok":true}` {
				t.Fatalf("response=%#v error=%v", response, err)
			}
			if resolver.count() != 1 || transport.count() != 1 {
				t.Fatalf("dependency calls resolver=%d transport=%d", resolver.count(), transport.count())
			}
			upstream := transport.lastRequest()
			if upstream == nil || upstream.Header.Get("Authorization") != "Bearer upstream-secret" {
				t.Fatalf("upstream authorization=%q", upstream.Header.Get("Authorization"))
			}
			if !bytes.Equal(request.Body, beforeBody) || !equalStrings(request.Authorization, beforeAuth) {
				t.Fatal("Do modified caller input")
			}
			response.Body[0] ^= 0xff
			if string(response.Body) == `{"ok":true}` {
				t.Fatal("response body mutation was not visible to local result")
			}
		})
	}
}

func TestAuthorizationAndScopeDenialsDoNotConsume(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	resolver := &fakeResolver{credential: "secret"}
	transport := &fakeTransport{}
	exchange := newExchange(t, registry, resolver, transport)
	malformed := githubRequest(handle)
	malformed.Authorization = []string{"Bearer " + handle, "Bearer duplicate"}
	if response, err := exchange.Do(testSubject, malformed); !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("authorization denial response=%#v error=%v", response, err)
	}
	denied := githubRequest(handle)
	denied.URL = "https://evil.example/repos/acme/widget"
	if response, err := exchange.Do(testSubject, denied); !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("policy denial response=%#v error=%v", response, err)
	}
	wrongSubject := testSubject
	wrongSubject.WorkspaceID = "other-workspace"
	if response, err := exchange.Do(wrongSubject, githubRequest(handle)); !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("scope denial response=%#v error=%v", response, err)
	}
	if resolver.count() != 0 || transport.count() != 0 {
		t.Fatalf("denials reached dependencies resolver=%d transport=%d", resolver.count(), transport.count())
	}
	if response, err := exchange.Do(testSubject, githubRequest(handle)); err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("valid request after denials response=%#v error=%v", response, err)
	}
	if resolver.count() != 1 || transport.count() != 1 {
		t.Fatalf("valid calls resolver=%d transport=%d", resolver.count(), transport.count())
	}

	openAIRegistry, openAIHandle := testRegistry(t, capability.ProviderOpenAI, 1)
	openAIExchange := newExchange(t, openAIRegistry, resolver, transport)
	if response, err := openAIExchange.Do(testSubject, githubRequest(openAIHandle)); !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("provider scope denial response=%#v error=%v", response, err)
	}
	if response, err := openAIExchange.Do(testSubject, openAIRequest(openAIHandle)); err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("valid OpenAI request after scope denial response=%#v error=%v", response, err)
	}
}

func TestPostConsumeFailuresRemainConsumed(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	resolver := &fakeResolver{err: errors.New("resolver detail")}
	transport := &fakeTransport{}
	exchange := newExchange(t, registry, resolver, transport)
	if response, err := exchange.Do(testSubject, githubRequest(handle)); !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("resolver failure response=%#v error=%v", response, err)
	}
	if _, err := exchange.Do(testSubject, githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("resolver retry error=%v", err)
	}
	if resolver.count() != 1 || transport.count() != 0 {
		t.Fatalf("resolver failure calls resolver=%d transport=%d", resolver.count(), transport.count())
	}
	assertFixedError(t, ErrDenied, handle, githubRequest(handle).URL)

	registry, handle = testRegistry(t, capability.ProviderGitHub, 1)
	resolver = &fakeResolver{credential: "secret"}
	transport = &fakeTransport{err: errors.New("transport detail")}
	exchange = newExchange(t, registry, resolver, transport)
	if response, err := exchange.Do(testSubject, githubRequest(handle)); !zeroResponse(response) || !errors.Is(err, ErrDenied) {
		t.Fatalf("transport failure response=%#v error=%v", response, err)
	}
	if _, err := exchange.Do(testSubject, githubRequest(handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("transport retry error=%v", err)
	}
	if resolver.count() != 1 || transport.count() != 1 {
		t.Fatalf("transport failure calls resolver=%d transport=%d", resolver.count(), transport.count())
	}
	assertFixedError(t, ErrDenied, handle, githubRequest(handle).URL)
}

func TestResponseIsolationUnderConcurrentDo(t *testing.T) {
	transport := &fakeTransport{factory: func(call int) *http.Response {
		return &http.Response{StatusCode: http.StatusOK + call%2, Header: http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"call":%d}`, call)))}
	}}
	resolver := &fakeResolver{credential: "secret"}
	registry, _ := testRegistry(t, capability.ProviderGitHub, 1)
	handles := make([]string, 8)
	var issueErr error
	for i := range handles {
		handles[i], issueErr = registry.Issue(capability.IssueSpec{AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID,
			WorkspaceID: testSubject.WorkspaceID, Provider: capability.ProviderGitHub, Repository: "acme/widget", TTL: time.Minute, Uses: 1})
		if issueErr != nil {
			t.Fatal(issueErr)
		}
	}
	exchange := newExchange(t, registry, resolver, transport)
	results := make(chan Response, len(handles))
	errs := make(chan error, len(handles))
	var wg sync.WaitGroup
	for _, handle := range handles {
		wg.Add(1)
		go func(handle string) {
			defer wg.Done()
			response, err := exchange.Do(testSubject, githubRequest(handle))
			results <- response
			errs <- err
		}(handle)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Do error=%v", err)
		}
	}
	seen := make(map[string]struct{})
	for response := range results {
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices || len(response.Body) == 0 {
			t.Fatalf("invalid concurrent response=%#v", response)
		}
		seen[string(response.Body)] = struct{}{}
	}
	if len(seen) != len(handles) {
		t.Fatalf("responses mixed or aliased: got %d distinct of %d", len(seen), len(handles))
	}
}

func TestCaptureSinkRequiresOneDeliveryAndCopiesBody(t *testing.T) {
	sink := &captureSink{}
	if _, ok := sink.snapshot(); ok {
		t.Fatal("empty sink reported a response")
	}
	body := []byte(`{"ok":true}`)
	if err := sink.Deliver(Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: body}); err != nil {
		t.Fatal(err)
	}
	body[0] = 'x'
	if err := sink.Deliver(Response{StatusCode: http.StatusOK}); !errors.Is(err, ErrDenied) {
		t.Fatalf("second delivery error=%v", err)
	}
	response, ok := sink.snapshot()
	if !ok || string(response.Body) != `{"ok":true}` {
		t.Fatalf("snapshot=%#v ok=%v", response, ok)
	}
	response.Body[0] = 'x'
	second, ok := sink.snapshot()
	if !ok || string(second.Body) != `{"ok":true}` {
		t.Fatalf("snapshot body aliased internal state: %#v ok=%v", second, ok)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func zeroResponse(response Response) bool {
	return response.StatusCode == 0 && response.ContentType == "" && len(response.Body) == 0
}

func assertFixedError(t *testing.T, err error, secret, url string) {
	t.Helper()
	formatted := fmt.Sprintf("%+v", err)
	if err == nil || err.Error() != string(ErrDenied) || fmt.Sprintf("%v", err) != string(ErrDenied) || formatted != string(ErrDenied) {
		t.Fatalf("non-fixed error=%q formatted=%q", err, formatted)
	}
	for _, value := range []string{secret, url, "resolver detail", "transport detail"} {
		if strings.Contains(err.Error(), value) || strings.Contains(formatted, value) {
			t.Fatalf("error leaked %q: %q", value, err)
		}
	}
}
