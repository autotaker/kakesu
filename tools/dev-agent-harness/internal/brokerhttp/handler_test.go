package brokerhttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerexchange"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

var testSubject = egresstransaction.Subject{AgentInstanceID: "agent-1", UID: 1000, WorkspaceID: "workspace-1"}

type fakeResolver struct {
	mu       sync.Mutex
	calls    int
	subject  egresstransaction.Subject
	err      error
	contexts []context.Context
}

func (r *fakeResolver) Resolve(ctx context.Context) (egresstransaction.Subject, error) {
	r.mu.Lock()
	r.calls++
	r.contexts = append(r.contexts, ctx)
	subject, err := r.subject, r.err
	r.mu.Unlock()
	return subject, err
}

func (r *fakeResolver) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

type fakeExchange struct {
	mu       sync.Mutex
	calls    int
	subject  egresstransaction.Subject
	request  egresstransaction.Request
	response brokerexchange.Response
	factory  func(egresstransaction.Request) brokerexchange.Response
	err      error
}

func (e *fakeExchange) Do(subject egresstransaction.Subject, request egresstransaction.Request) (brokerexchange.Response, error) {
	e.mu.Lock()
	e.calls++
	e.subject = subject
	e.request = request
	response, factory, err := e.response, e.factory, e.err
	e.mu.Unlock()
	if factory != nil {
		response = factory(request)
	}
	response.Body = append([]byte(nil), response.Body...)
	return response, err
}

func (e *fakeExchange) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func newHandler(t *testing.T, exchange Exchange, resolver SubjectResolver) *Handler {
	t.Helper()
	handler, err := New(Rules{Exchange: exchange, Resolver: resolver, MaxBodyBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func request(method, target, host string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, target, body)
	r.Host = host
	return r
}

func TestNewBoundsAndZeroDeny(t *testing.T) {
	exchange := &fakeExchange{}
	resolver := &fakeResolver{subject: testSubject}
	base := Rules{Exchange: exchange, Resolver: resolver, MaxBodyBytes: 1}
	for name, rules := range map[string]Rules{
		"zero": {}, "low": func() Rules { r := base; r.MaxBodyBytes = 0; return r }(),
		"high": func() Rules { r := base; r.MaxBodyBytes = maxRequestBodyBytes + 1; return r }(),
	} {
		t.Run(name, func(t *testing.T) {
			if handler, err := New(rules); handler != nil || !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("handler=%p error=%v", handler, err)
			}
		})
	}
	var nilExchange *fakeExchange
	var nilResolver *fakeResolver
	for name, rules := range map[string]Rules{
		"typed nil exchange": {Exchange: nilExchange, Resolver: resolver, MaxBodyBytes: 1},
		"typed nil resolver": {Exchange: exchange, Resolver: nilResolver, MaxBodyBytes: 1},
	} {
		t.Run(name, func(t *testing.T) {
			if handler, err := New(rules); handler != nil || !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("handler=%p error=%v", handler, err)
			}
		})
	}
	var zero Handler
	recorder := httptest.NewRecorder()
	zero.ServeHTTP(recorder, request(http.MethodGet, "/repos/acme/widget", "api.github.com", nil))
	assertDenied(t, recorder)
}

func TestSuccessMappingAndHeaderAllowlist(t *testing.T) {
	resolver := &fakeResolver{subject: testSubject}
	exchange := &fakeExchange{response: brokerexchange.Response{StatusCode: http.StatusCreated, ContentType: "application/json", Body: []byte(`{"ok":true}`)}}
	handler := newHandler(t, exchange, resolver)
	body := []byte(`{"model":"gpt-5-mini"}`)
	req := request(http.MethodPost, "/v1/responses", "api.openai.com", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer cap_test")
	req.Header.Add("Authorization", "token cap_second")
	req.Header.Set("Forwarded", "for=attacker")
	req = req.WithContext(context.WithValue(req.Context(), "identity", "trusted"))
	beforeBody := append([]byte(nil), body...)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated || recorder.Body.String() != `{"ok":true}` {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control=%q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("nosniff=%q", got)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type=%q", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != "11" {
		t.Fatalf("content-length=%q", got)
	}
	for key := range recorder.Header() {
		if key != "Cache-Control" && key != "X-Content-Type-Options" && key != "Content-Type" && key != "Content-Length" {
			t.Fatalf("unexpected response header %q", key)
		}
	}
	if !bytes.Equal(body, beforeBody) || resolver.count() != 1 || exchange.count() != 1 {
		t.Fatalf("calls or input changed resolver=%d exchange=%d body=%q", resolver.count(), exchange.count(), body)
	}
	exchange.mu.Lock()
	captured := exchange.request
	exchange.mu.Unlock()
	if captured.URL != "https://api.openai.com/v1/responses" || captured.ContentType != "application/json" ||
		len(captured.Authorization) != 2 || string(captured.Body) != string(body) {
		t.Fatalf("mapped request=%#v", captured)
	}
	captured.Body[0] = 'X'
	if !bytes.Equal(body, beforeBody) {
		t.Fatal("mapped body aliases caller")
	}
}

func TestGitDiscoveryQueryAndBinaryResponseMapping(t *testing.T) {
	resolver := &fakeResolver{subject: testSubject}
	binary := []byte{0x00, 0xff, 'g', 'i', 't'}
	exchange := &fakeExchange{response: brokerexchange.Response{StatusCode: http.StatusOK, ContentType: egresspolicy.GitUploadPackAdvertise, Body: binary}}
	handler := newHandler(t, exchange, resolver)
	req := request(http.MethodGet, "/acme/widget.git/info/refs?service=git-upload-pack", "github.com", nil)
	req.Header.Set("Authorization", "Basic opaque")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || !bytes.Equal(recorder.Body.Bytes(), binary) || recorder.Header().Get("Content-Type") != egresspolicy.GitUploadPackAdvertise {
		t.Fatalf("status=%d body=%v headers=%v", recorder.Code, recorder.Body.Bytes(), recorder.Header())
	}
	exchange.mu.Lock()
	captured := exchange.request
	exchange.mu.Unlock()
	if captured.URL != "https://github.com/acme/widget.git/info/refs?service=git-upload-pack" || captured.Method != http.MethodGet || len(captured.Authorization) != 1 {
		t.Fatalf("mapped=%+v", captured)
	}

	for _, target := range []string{
		"/acme/widget.git/info/refs?service=git-receive-pack",
		"/acme/widget.git/info/refs?service=git-upload-pack&x=1",
		"/acme/widget.git/info/refs?service%3Dgit-upload-pack",
		"/acme/widget.git/info/refs?",
		"/repos/acme/widget?service=git-upload-pack",
	} {
		deniedResolver := &fakeResolver{subject: testSubject}
		deniedExchange := &fakeExchange{}
		denied := newHandler(t, deniedExchange, deniedResolver)
		recorder := httptest.NewRecorder()
		denied.ServeHTTP(recorder, request(http.MethodGet, target, "github.com", nil))
		assertDenied(t, recorder)
		if deniedResolver.count() != 0 || deniedExchange.count() != 0 {
			t.Fatalf("target %q reached dependencies", target)
		}
	}
}

func TestProtocolAndFramingDenialsDoNotReachDependencies(t *testing.T) {
	resolver := &fakeResolver{subject: testSubject}
	exchange := &fakeExchange{response: brokerexchange.Response{StatusCode: http.StatusOK}}
	handler := newHandler(t, exchange, resolver)
	cases := []struct {
		name string
		edit func(*http.Request)
	}{
		{"absolute", func(r *http.Request) { r.URL.Scheme = "https"; r.URL.Host = r.Host }},
		{"nil url", func(r *http.Request) { r.URL = nil }},
		{"opaque", func(r *http.Request) { r.URL.Opaque = "opaque" }},
		{"userinfo", func(r *http.Request) { r.URL.User = url.UserPassword("user", "secret") }},
		{"fragment", func(r *http.Request) { r.URL.Fragment = "fragment" }},
		{"raw path", func(r *http.Request) { r.URL.RawPath = "/repos/acme/widget" }},
		{"query", func(r *http.Request) { r.URL.RawQuery = "x=1"; r.RequestURI += "?x=1" }},
		{"percent path", func(r *http.Request) { r.URL.Path = "/repos/acme/%77idget"; r.RequestURI = r.URL.Path }},
		{"http10", func(r *http.Request) { r.Proto = "HTTP/1.0"; r.ProtoMajor = 1; r.ProtoMinor = 0 }},
		{"http2", func(r *http.Request) { r.Proto = "HTTP/2.0"; r.ProtoMajor = 2; r.ProtoMinor = 0 }},
		{"connect", func(r *http.Request) { r.Method = http.MethodConnect }},
		{"transfer", func(r *http.Request) { r.TransferEncoding = []string{"chunked"} }},
		{"trailer", func(r *http.Request) { r.Header.Set("Trailer", "X-Test") }},
		{"trailer map", func(r *http.Request) { r.Trailer = http.Header{"X-Test": []string{"value"}} }},
		{"upgrade", func(r *http.Request) { r.Header.Set("Connection", "Upgrade") }},
		{"duplicate content length", func(r *http.Request) { r.Header["Content-Length"] = []string{"0", "0"} }},
		{"length mismatch", func(r *http.Request) { r.ContentLength = 2 }},
		{"unknown length", func(r *http.Request) { r.ContentLength = -1 }},
		{"body too large", func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", 4097)))
			r.ContentLength = 4097
		}},
		{"read error", func(r *http.Request) { r.Body = errorBody{}; r.ContentLength = 1 }},
		{"empty host", func(r *http.Request) { r.Host = "" }},
		{"overlong host", func(r *http.Request) { r.Host = strings.Repeat("a", 254) }},
		{"host header", func(r *http.Request) { r.Header.Set("Host", "evil.example") }},
		{"duplicate content type", func(r *http.Request) { r.Header["Content-Type"] = []string{"application/json", "application/json"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := request(http.MethodGet, "/repos/acme/widget", "api.github.com", nil)
			tc.edit(req)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			assertDenied(t, recorder)
			if resolver.count() != 0 || exchange.count() != 0 {
				t.Fatalf("dependencies reached resolver=%d exchange=%d", resolver.count(), exchange.count())
			}
		})
	}
}

func TestFailureIsEmptyFixed403AndOutputIsolation(t *testing.T) {
	resolver := &fakeResolver{subject: testSubject, err: errors.New("secret resolver detail")}
	exchange := &fakeExchange{response: brokerexchange.Response{StatusCode: http.StatusOK, Body: []byte(`{"ok":true}`)}}
	handler := newHandler(t, exchange, resolver)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request(http.MethodGet, "/repos/acme/widget", "api.github.com", nil))
	assertDenied(t, recorder)
	if exchange.count() != 0 {
		t.Fatal("resolver failure reached exchange")
	}
	resolver.err = nil
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request(http.MethodGet, "/repos/acme/widget", "api.github.com", nil))
	if first.Code != http.StatusOK || first.Body.String() != `{"ok":true}` {
		t.Fatalf("success status=%d body=%q", first.Code, first.Body.String())
	}
}

func TestResolverExchangeAndResponseFailuresAreEmpty403(t *testing.T) {
	cases := []struct {
		name     string
		resolver error
		err      error
		response brokerexchange.Response
	}{
		{name: "resolver", resolver: errors.New("resolver secret")},
		{name: "exchange", err: errors.New("exchange secret")},
		{name: "status", response: brokerexchange.Response{StatusCode: http.StatusBadGateway}},
		{name: "content type", response: brokerexchange.Response{StatusCode: http.StatusOK, ContentType: "text/plain"}},
		{name: "oversize", response: brokerexchange.Response{StatusCode: http.StatusOK, Body: bytes.Repeat([]byte{'x'}, maxResponseBytes+1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &fakeResolver{subject: testSubject, err: tc.resolver}
			exchange := &fakeExchange{response: tc.response, err: tc.err}
			handler := newHandler(t, exchange, resolver)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request(http.MethodGet, "/repos/acme/widget", "api.github.com", nil))
			assertDenied(t, recorder)
			if fmt.Sprintf("%v", handler) != "brokerhttp.Handler" || strings.Contains(fmt.Sprintf("%+v", handler), "secret") {
				t.Fatalf("handler format leaked details: %q", fmt.Sprintf("%+v", handler))
			}
		})
	}
}

func TestConcurrentRequestsHaveIndependentResponses(t *testing.T) {
	resolver := &fakeResolver{subject: testSubject}
	exchange := &fakeExchange{factory: func(request egresstransaction.Request) brokerexchange.Response {
		return brokerexchange.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(fmt.Sprintf(`{"path":%q}`, request.URL))}
	}}
	handler := newHandler(t, exchange, resolver)
	results := make(chan string, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request(http.MethodGet, fmt.Sprintf("/repos/acme/widget/issues/%d", i), "api.github.com", nil))
			results <- recorder.Body.String()
		}(i)
	}
	wg.Wait()
	close(results)
	if len(results) != 8 {
		t.Fatalf("results=%d", len(results))
	}
	seen := make(map[string]struct{})
	for result := range results {
		seen[result] = struct{}{}
	}
	if len(seen) != 8 {
		t.Fatalf("concurrent responses mixed: %d distinct", len(seen))
	}
}

func assertDenied(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusForbidden || recorder.Body.Len() != 0 || recorder.Header().Get("Content-Length") != "0" ||
		recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("deny response status=%d body=%q headers=%v", recorder.Code, recorder.Body.String(), recorder.Header())
	}
	if len(recorder.Header()) != 3 {
		t.Fatalf("deny headers=%v", recorder.Header())
	}
}

func TestRealExchangeBothProviders(t *testing.T) {
	for _, provider := range []string{capability.ProviderGitHub, capability.ProviderOpenAI} {
		t.Run(provider, func(t *testing.T) {
			policy, err := egresspolicy.New(egresspolicy.Rules{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 4096, MaxOutputTokens: 32})
			if err != nil {
				t.Fatal(err)
			}
			registry, err := capability.New(capability.Rules{PolicyVersion: "v1", MaxTTL: time.Hour, MaxUses: 2})
			if err != nil {
				t.Fatal(err)
			}
			spec := capability.IssueSpec{AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID, WorkspaceID: testSubject.WorkspaceID, Provider: provider, TTL: time.Minute, Uses: 1}
			if provider == capability.ProviderGitHub {
				spec.Repository = "acme/widget"
			}
			handle, err := registry.Issue(spec)
			if err != nil {
				t.Fatal(err)
			}
			transport := &httpRoundTripper{}
			resolver := egresstransaction.CredentialResolverFunc(func(string, string) (string, error) { return "secret", nil })
			exchange, err := brokerexchange.New(brokerexchange.Rules{Policy: policy, Registry: registry, Resolver: resolver, Transport: transport, MaxCredentialBytes: 128, Timeout: time.Second, MaxResponseBytes: 1024})
			if err != nil {
				t.Fatal(err)
			}
			handler := newHandler(t, exchange, &fakeResolver{subject: testSubject})
			var req *http.Request
			if provider == capability.ProviderGitHub {
				req = request(http.MethodGet, "/repos/acme/widget/issues", "api.github.com", nil)
				req.Header.Set("Authorization", "Bearer "+handle)
			} else {
				req = request(http.MethodPost, "/v1/responses", "api.openai.com", strings.NewReader(`{"model":"gpt-5-mini","input":"hi","store":false,"stream":false,"max_output_tokens":1}`))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+handle)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK || recorder.Body.String() != `{"ok":true}` || transport.calls != 1 {
				t.Fatalf("status=%d body=%q transport=%d", recorder.Code, recorder.Body.String(), transport.calls)
			}
		})
	}
}

func TestRealExchangeGitUploadPack(t *testing.T) {
	policy, err := egresspolicy.New(egresspolicy.Rules{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 4096, MaxOutputTokens: 32})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := capability.New(capability.Rules{PolicyVersion: "v1", MaxTTL: time.Hour, MaxUses: 1})
	if err != nil {
		t.Fatal(err)
	}
	handle, err := registry.Issue(capability.IssueSpec{AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID, WorkspaceID: testSubject.WorkspaceID, Provider: capability.ProviderGitHub, Repository: "acme/widget", Operation: capability.OperationGitHubGitRead, TTL: time.Minute, Uses: 1})
	if err != nil {
		t.Fatal(err)
	}
	transport := &httpRoundTripper{}
	exchange, err := brokerexchange.New(brokerexchange.Rules{
		Policy: policy, Registry: registry,
		Resolver: egresstransaction.CredentialResolverFunc(func(provider, repository string) (string, error) {
			if provider != "github" || repository != "acme/widget" {
				t.Fatalf("resolver scope=%s/%s", provider, repository)
			}
			return "real-token", nil
		}),
		Transport: transport, MaxCredentialBytes: 128, Timeout: time.Second, MaxResponseBytes: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(t, exchange, &fakeResolver{subject: testSubject})
	body := []byte("0009done\n")
	req := request(http.MethodPost, "/acme/widget.git/git-upload-pack", "github.com", bytes.NewReader(body))
	req.Header.Set("Content-Type", egresspolicy.GitUploadPackRequest)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:"+handle)))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || !bytes.Equal(recorder.Body.Bytes(), []byte{0x00, 0xff, 'g', 'i', 't'}) || recorder.Header().Get("Content-Type") != egresspolicy.GitUploadPackResult {
		t.Fatalf("status=%d body=%v headers=%v", recorder.Code, recorder.Body.Bytes(), recorder.Header())
	}
	wantBasic := "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:real-token"))
	if transport.calls != 1 || transport.authorization != wantBasic || transport.url != "https://github.com/acme/widget.git/git-upload-pack" || transport.contentType != egresspolicy.GitUploadPackRequest {
		t.Fatalf("transport=%+v", transport)
	}
}

func TestRealExchangeGitPushDeniedBeforeResolverAndTransport(t *testing.T) {
	policy, err := egresspolicy.New(egresspolicy.Rules{
		GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"},
		MaxBodyBytes: 4096, MaxOutputTokens: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := capability.New(capability.Rules{PolicyVersion: "v1", MaxTTL: time.Hour, MaxUses: 1})
	if err != nil {
		t.Fatal(err)
	}
	handle, err := registry.Issue(capability.IssueSpec{
		AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID,
		WorkspaceID: testSubject.WorkspaceID, Provider: capability.ProviderGitHub,
		Repository: "acme/widget", Operation: capability.OperationGitHubGitRead,
		TTL: time.Minute, Uses: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolverCalls := 0
	transport := &httpRoundTripper{}
	exchange, err := brokerexchange.New(brokerexchange.Rules{
		Policy: policy, Registry: registry,
		Resolver: egresstransaction.CredentialResolverFunc(func(string, string) (string, error) {
			resolverCalls++
			return "real-token", nil
		}),
		Transport: transport, MaxCredentialBytes: 128, Timeout: time.Second, MaxResponseBytes: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(t, exchange, &fakeResolver{subject: testSubject})
	authorization := "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+handle))

	push := request(http.MethodPost, "/acme/widget.git/git-receive-pack", "github.com", strings.NewReader("push"))
	push.Header.Set("Content-Type", "application/x-git-receive-pack-request")
	push.Header.Set("Authorization", authorization)
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, push)
	assertDenied(t, denied)
	if resolverCalls != 0 || transport.calls != 0 {
		t.Fatalf("push reached resolver=%d transport=%d", resolverCalls, transport.calls)
	}

	upload := request(http.MethodPost, "/acme/widget.git/git-upload-pack", "github.com", strings.NewReader("0009done\n"))
	upload.Header.Set("Content-Type", egresspolicy.GitUploadPackRequest)
	upload.Header.Set("Authorization", authorization)
	success := httptest.NewRecorder()
	handler.ServeHTTP(success, upload)
	if success.Code != http.StatusOK || resolverCalls != 1 || transport.calls != 1 {
		t.Fatalf("valid upload status=%d resolver=%d transport=%d body=%v", success.Code, resolverCalls, transport.calls, success.Body.Bytes())
	}
}

func TestRealExchangeGitDiscoveryUsesAdvertisementMedia(t *testing.T) {
	policy, err := egresspolicy.New(egresspolicy.Rules{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 4096, MaxOutputTokens: 32})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := capability.New(capability.Rules{PolicyVersion: "v1", MaxTTL: time.Hour, MaxUses: 1})
	if err != nil {
		t.Fatal(err)
	}
	handle, err := registry.Issue(capability.IssueSpec{AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID, WorkspaceID: testSubject.WorkspaceID, Provider: capability.ProviderGitHub, Repository: "acme/widget", Operation: capability.OperationGitHubGitRead, TTL: time.Minute, Uses: 1})
	if err != nil {
		t.Fatal(err)
	}
	transport := &httpRoundTripper{}
	exchange, err := brokerexchange.New(brokerexchange.Rules{Policy: policy, Registry: registry,
		Resolver:  egresstransaction.CredentialResolverFunc(func(string, string) (string, error) { return "real-token", nil }),
		Transport: transport, MaxCredentialBytes: 128, Timeout: time.Second, MaxResponseBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(t, exchange, &fakeResolver{subject: testSubject})
	discovery := request(http.MethodGet, "/acme/widget.git/info/refs?service=git-upload-pack", "github.com", nil)
	discovery.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:"+handle)))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, discovery)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != egresspolicy.GitUploadPackAdvertise || transport.calls != 1 || transport.contentType != "" {
		t.Fatalf("status=%d headers=%v transport=%+v", recorder.Code, recorder.Header(), transport)
	}
}

type httpRoundTripper struct {
	mu            sync.Mutex
	calls         int
	authorization string
	url           string
	contentType   string
}

type errorBody struct{}

func (errorBody) Read([]byte) (int, error) { return 0, errors.New("read secret") }
func (errorBody) Close() error             { return nil }

func (t *httpRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.calls++
	t.authorization = request.Header.Get("Authorization")
	t.url = request.URL.String()
	t.contentType = request.Header.Get("Content-Type")
	t.mu.Unlock()
	if request.Header.Get("Accept") == egresspolicy.GitUploadPackAdvertise {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{egresspolicy.GitUploadPackAdvertise}}, Body: io.NopCloser(bytes.NewReader([]byte{0x00, 0xff, 'g', 'i', 't'}))}, nil
	}
	if request.Header.Get("Accept") == egresspolicy.GitUploadPackResult {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{egresspolicy.GitUploadPackResult}}, Body: io.NopCloser(bytes.NewReader([]byte{0x00, 0xff, 'g', 'i', 't'}))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
}
