package providercredentials

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokercredentials"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

type trackingBody struct {
	io.Reader
	closed   bool
	closeErr error
}

type failingBody struct{ closed bool }

func (b *failingBody) Read([]byte) (int, error) { return 0, errors.New("secret read detail") }
func (b *failingBody) Close() error {
	b.closed = true
	return nil
}

func (b *trackingBody) Close() error {
	b.closed = true
	return b.closeErr
}

type fakeTransport struct {
	mu       sync.Mutex
	calls    int
	request  *http.Request
	response *http.Response
	err      error
	observe  func(*http.Request)
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.calls++
	f.request = req
	f.mu.Unlock()
	if f.observe != nil {
		f.observe(req)
	}
	return f.response, f.err
}

func (f *fakeTransport) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeTransport) lastRequest() *http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.request
}

func fixtureBundle(t *testing.T) *brokercredentials.Bundle {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("broker credential fixture requires a non-root effective uid")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"github-client-id":       []byte("Iv1.runtime-client\n"),
		"github-installation-id": []byte("123456\n"),
		"github-private-key.pem": pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}),
		"openai-api-key":         []byte("sk-runtime-fixture\n"),
	}
	caCert, caKey := fixtureProxyCA(t)
	files["proxy-ca-cert.pem"] = caCert
	files["proxy-ca-key.pem"] = caKey
	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := brokercredentials.Load(dir)
	if err != nil {
		t.Fatalf("fixture bundle load: %v", err)
	}
	return bundle
}

func fixtureProxyCA(t *testing.T) ([]byte, []byte) {
	t.Helper()
	now := time.Now().UTC()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "provider-fixture-ca"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(2 * time.Hour), IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func fixedResolver(t *testing.T, transport http.RoundTripper, now time.Time) *Resolver {
	t.Helper()
	resolver, err := newWithClock(Rules{Bundle: fixtureBundle(t), Transport: transport, Timeout: time.Second}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

func responseBody(data string) *trackingBody {
	return &trackingBody{Reader: strings.NewReader(data)}
}

func createdResponse(body io.ReadCloser, contentType string) *http.Response {
	return &http.Response{StatusCode: http.StatusCreated, Header: http.Header{"Content-Type": []string{contentType}}, Body: body}
}

func TestNewAndFormatFailClosed(t *testing.T) {
	transport := &fakeTransport{}
	bundle := fixtureBundle(t)
	for _, rules := range []Rules{
		{}, {Bundle: bundle, Transport: transport, Timeout: 0},
		{Bundle: bundle, Transport: transport, Timeout: 31 * time.Second},
		{Bundle: bundle, Transport: nil, Timeout: time.Second},
	} {
		if resolver, err := New(rules); resolver != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("New(%#v)=(%p,%v), want invalid-rules", rules, resolver, err)
		}
	}
	resolver := fixedResolver(t, transport, time.Now())
	for _, value := range []any{resolver, *resolver} {
		for _, format := range []string{"%v", "%+v", "%#v", "%q", "%s"} {
			formatted := fmt.Sprintf(format, value)
			if formatted != "providercredentials.Resolver" || strings.Contains(formatted, "runtime") {
				t.Fatalf("format %s leaked resolver: %q", format, formatted)
			}
		}
	}
	var zero *Resolver
	if value, err := zero.Resolve("openai", ""); value != "" || !errors.Is(err, ErrResolve) {
		t.Fatalf("nil receiver=(%q,%v)", value, err)
	}
}

func TestOpenAINoNetworkAndScopeRejection(t *testing.T) {
	transport := &fakeTransport{}
	resolver := fixedResolver(t, transport, time.Now())
	if value, err := resolver.Resolve("openai", ""); err != nil || value != "sk-runtime-fixture" {
		t.Fatalf("OpenAI resolve=(%q,%v)", value, err)
	}
	if transport.callCount() != 0 {
		t.Fatal("OpenAI resolution reached transport")
	}
	for _, tc := range [][2]string{{"openai", "acme/widget"}, {"azure", ""}, {"github", "acme/Widget"}, {"github", "acme/widget/child"}, {"github", "acme//widget"}, {"github", "../widget"}} {
		if value, err := resolver.Resolve(tc[0], tc[1]); value != "" || !errors.Is(err, ErrResolve) {
			t.Fatalf("Resolve(%q,%q)=(%q,%v)", tc[0], tc[1], value, err)
		}
	}
	if transport.callCount() != 0 {
		t.Fatal("rejected provider reached transport")
	}
}

func TestGitHubExchangeBindsRequestAndClosesBody(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	body := responseBody(`{"token":"ghs-runtime-token","expires_at":"2026-08-01T13:05:00Z","unknown":{"ok":true}}`)
	transport := &fakeTransport{response: createdResponse(body, "application/json; charset=utf-8")}
	resolver := fixedResolver(t, transport, now)
	value, err := resolver.Resolve("github", "acme/widget")
	if err != nil || value != "ghs-runtime-token" {
		t.Fatalf("GitHub resolve=(%q,%v)", value, err)
	}
	if transport.callCount() != 1 || !body.closed {
		t.Fatalf("calls=%d bodyClosed=%v", transport.callCount(), body.closed)
	}
	request := transport.lastRequest()
	if request == nil || request.Method != http.MethodPost || request.URL.String() != "https://api.github.com/app/installations/123456/access_tokens" {
		t.Fatalf("request binding=%v", request)
	}
	if request.Header.Get("Accept") != "application/vnd.github+json" || request.Header.Get("Content-Type") != "application/json" || request.Header.Get("X-GitHub-Api-Version") != githubAPIVersion || !strings.HasPrefix(request.Header.Get("Authorization"), "Bearer ") {
		t.Fatalf("headers=%v", request.Header)
	}
	requestBody, readErr := io.ReadAll(request.Body)
	if readErr != nil || string(requestBody) != `{"repositories":["widget"]}` {
		t.Fatalf("request body=%q err=%v", requestBody, readErr)
	}
	deadline, hasDeadline := request.Context().Deadline()
	if !hasDeadline || time.Until(deadline) <= 0 {
		t.Fatalf("missing request deadline: %v", deadline)
	}
	if strings.Contains(string(requestBody), value) || strings.Contains(request.Header.Get("Authorization"), value) {
		t.Fatal("installation token leaked into request")
	}
}

func TestGitHubResponseBoundariesAndCloseOnErrors(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name         string
		response     *http.Response
		transportErr error
		wantClose    bool
	}{
		{"status", &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: responseBody(`{"token":"x","expires_at":"2026-08-01T13:00:00Z"}`)}, nil, true},
		{"nil-response", nil, nil, false},
		{"nil-body", &http.Response{StatusCode: http.StatusCreated, Header: http.Header{"Content-Type": []string{"application/json"}}}, nil, false},
		{"content-type", createdResponse(responseBody(`{"token":"x","expires_at":"2026-08-01T13:00:00Z"}`), "text/plain"), nil, true},
		{"duplicate", createdResponse(responseBody(`{"token":"x","token":"y","expires_at":"2026-08-01T13:00:00Z"}`), "application/json"), nil, true},
		{"trailing", createdResponse(responseBody(`{"token":"x","expires_at":"2026-08-01T13:00:00Z"} null`), "application/json"), nil, true},
		{"missing-token", createdResponse(responseBody(`{"expires_at":"2026-08-01T13:00:00Z"}`), "application/json"), nil, true},
		{"missing-expiry", createdResponse(responseBody(`{"token":"x"}`), "application/json"), nil, true},
		{"empty-token", createdResponse(responseBody(`{"token":"","expires_at":"2026-08-01T13:00:00Z"}`), "application/json"), nil, true},
		{"wrong-token", createdResponse(responseBody(`{"token":"\n","expires_at":"2026-08-01T13:00:00Z"}`), "application/json"), nil, true},
		{"expiry-now", createdResponse(responseBody(`{"token":"x","expires_at":"2026-08-01T12:00:00Z"}`), "application/json"), nil, true},
		{"expiry-too-late", createdResponse(responseBody(`{"token":"x","expires_at":"2026-08-01T13:05:01Z"}`), "application/json"), nil, true},
		{"oversize", createdResponse(responseBody(`{"token":"x","expires_at":"2026-08-01T13:00:00Z","padding":"`+strings.Repeat("a", maxResponseBytes)+`"}`), "application/json"), nil, true},
		{"transport-error-with-response", createdResponse(responseBody(`{"token":"x","expires_at":"2026-08-01T13:00:00Z"}`), "application/json"), errors.New("secret transport detail"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *trackingBody
			if tc.response != nil {
				body, _ = tc.response.Body.(*trackingBody)
			}
			transport := &fakeTransport{response: tc.response, err: tc.transportErr}
			resolver := fixedResolver(t, transport, now)
			value, err := resolver.Resolve("github", "acme/widget")
			if value != "" || !errors.Is(err, ErrResolve) || transport.callCount() != 1 {
				t.Fatalf("resolve=(%q,%v) calls=%d", value, err, transport.callCount())
			}
			if tc.wantClose && body != nil && !body.closed {
				t.Fatal("response body was not closed")
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("transport detail leaked")
			}
		})
	}
	readFailure := &failingBody{}
	readTransport := &fakeTransport{response: createdResponse(readFailure, "application/json")}
	if value, err := fixedResolver(t, readTransport, now).Resolve("github", "acme/widget"); value != "" || !errors.Is(err, ErrResolve) || !readFailure.closed {
		t.Fatalf("read failure=(%q,%v), closed=%v", value, err, readFailure.closed)
	}
	closeFailure := responseBody(`{"token":"x","expires_at":"2026-08-01T13:00:00Z"}`)
	closeFailure.closeErr = errors.New("secret close detail")
	closeTransport := &fakeTransport{response: createdResponse(closeFailure, "application/json")}
	if value, err := fixedResolver(t, closeTransport, now).Resolve("github", "acme/widget"); value != "" || !errors.Is(err, ErrResolve) || !closeFailure.closed {
		t.Fatalf("close failure=(%q,%v), closed=%v", value, err, closeFailure.closed)
	}
	for _, size := range []int{maxTokenBytes, maxTokenBytes + 1} {
		tokenBody := responseBody(`{"token":"` + strings.Repeat("x", size) + `","expires_at":"2026-08-01T13:00:00Z"}`)
		tokenTransport := &fakeTransport{response: createdResponse(tokenBody, "application/json")}
		value, err := fixedResolver(t, tokenTransport, now).Resolve("github", "acme/widget")
		if size == maxTokenBytes && (err != nil || len(value) != size) {
			t.Fatalf("token boundary %d=(%q,%v)", size, value, err)
		}
		if size == maxTokenBytes+1 && (value != "" || !errors.Is(err, ErrResolve)) {
			t.Fatalf("oversize token accepted=(%q,%v)", value, err)
		}
	}
}

func TestGitHubTimeoutIsSingleExchange(t *testing.T) {
	transport := &fakeTransport{}
	transport.observe = func(request *http.Request) {
		<-request.Context().Done()
	}
	resolver, err := newWithClock(Rules{Bundle: fixtureBundle(t), Transport: transport, Timeout: time.Millisecond}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	value, resolveErr := resolver.Resolve("github", "acme/widget")
	if value != "" || !errors.Is(resolveErr, ErrResolve) || transport.callCount() != 1 {
		t.Fatalf("timeout=(%q,%v), calls=%d", value, resolveErr, transport.callCount())
	}
}

func TestResolverTransactionIntegration(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	body := responseBody(`{"token":"ghs-transaction","expires_at":"2026-08-01T13:00:00Z"}`)
	transport := &fakeTransport{response: createdResponse(body, "application/json")}
	resolver := fixedResolver(t, transport, now)
	policy, err := egresspolicy.New(egresspolicy.Rules{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 4096, MaxOutputTokens: 128})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := capability.New(capability.Rules{PolicyVersion: "policy-v1", MaxTTL: time.Minute, MaxUses: 1})
	if err != nil {
		t.Fatal(err)
	}
	const agent, workspace = "agent-1", "workspace-1"
	handle, err := registry.Issue(capability.IssueSpec{AgentInstanceID: agent, UID: 1000, WorkspaceID: workspace, Provider: capability.ProviderGitHub, Repository: "acme/widget", TTL: time.Minute, Uses: 1})
	if err != nil {
		t.Fatal(err)
	}
	forwarded := 0
	txn, err := egresstransaction.New(egresstransaction.Rules{Policy: policy, Registry: registry, Resolver: resolver, Forwarder: egresstransaction.ForwarderFunc(func(req egresstransaction.PreparedRequest) error {
		forwarded++
		if req.Authorization != "Bearer ghs-transaction" {
			t.Fatalf("forwarded authorization=%q", req.Authorization)
		}
		return nil
	}), MaxCredentialBytes: 128})
	if err != nil {
		t.Fatal(err)
	}
	request := egresstransaction.Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget", Authorization: []string{"Bearer " + handle}}
	denied := request
	denied.URL = "https://evil.example/repos/acme/widget"
	if err := txn.Execute(egresstransaction.Subject{AgentInstanceID: agent, UID: 1000, WorkspaceID: workspace}, denied); !errors.Is(err, egresstransaction.ErrDenied) {
		t.Fatalf("denied transaction=%v", err)
	}
	if transport.callCount() != 0 || forwarded != 0 {
		t.Fatal("invalid capability reached resolver or forwarder")
	}
	if err := txn.Execute(egresstransaction.Subject{AgentInstanceID: agent, UID: 1000, WorkspaceID: workspace}, request); err != nil {
		t.Fatalf("valid transaction=%v", err)
	}
	if transport.callCount() != 1 || forwarded != 1 {
		t.Fatalf("valid calls transport=%d forwarder=%d", transport.callCount(), forwarded)
	}
}

func TestResolverTransactionOpenAIIntegration(t *testing.T) {
	transport := &fakeTransport{}
	resolver := fixedResolver(t, transport, time.Now())
	policy, err := egresspolicy.New(egresspolicy.Rules{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 4096, MaxOutputTokens: 128})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := capability.New(capability.Rules{PolicyVersion: "policy-v1", MaxTTL: time.Minute, MaxUses: 1})
	if err != nil {
		t.Fatal(err)
	}
	const agent, workspace = "agent-openai", "workspace-openai"
	handle, err := registry.Issue(capability.IssueSpec{AgentInstanceID: agent, UID: 1000, WorkspaceID: workspace, Provider: capability.ProviderOpenAI, TTL: time.Minute, Uses: 1})
	if err != nil {
		t.Fatal(err)
	}
	var gotAuthorization string
	txn, err := egresstransaction.New(egresstransaction.Rules{Policy: policy, Registry: registry, Resolver: resolver, Forwarder: egresstransaction.ForwarderFunc(func(req egresstransaction.PreparedRequest) error {
		gotAuthorization = req.Authorization
		return nil
	}), MaxCredentialBytes: 128})
	if err != nil {
		t.Fatal(err)
	}
	request := egresstransaction.Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: []byte(`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":32}`), Authorization: []string{"Bearer " + handle}}
	if err := txn.Execute(egresstransaction.Subject{AgentInstanceID: agent, UID: 1000, WorkspaceID: workspace}, request); err != nil {
		t.Fatalf("OpenAI transaction=%v", err)
	}
	if transport.callCount() != 0 || gotAuthorization != "Bearer sk-runtime-fixture" {
		t.Fatalf("OpenAI calls=%d authorization=%q", transport.callCount(), gotAuthorization)
	}
}

func TestResponseParserAllowsUnknownFieldAndExactExpiryBoundaries(t *testing.T) {
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		when time.Time
		ok   bool
	}{
		{"just-after-now", base.Add(time.Nanosecond), true},
		{"at-65-minutes", base.Add(maxTokenAge), true},
		{"after-65-minutes", base.Add(maxTokenAge + time.Nanosecond), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expires := tc.when.UTC().Format(time.RFC3339Nano)
			body := responseBody(`{"token":"x","expires_at":"` + expires + `","new_field":[1,2,3]}`)
			transport := &fakeTransport{response: createdResponse(body, "application/json")}
			resolver := fixedResolver(t, transport, base)
			value, err := resolver.Resolve("github", "acme/widget")
			if tc.ok && (err != nil || value != "x") {
				t.Fatalf("boundary resolve=(%q,%v)", value, err)
			}
			if !tc.ok && (value != "" || !errors.Is(err, ErrResolve)) {
				t.Fatalf("boundary accepted=(%q,%v)", value, err)
			}
		})
	}
	if _, _, ok := parseTokenResponse([]byte(`[]`)); ok {
		t.Fatal("array response accepted")
	}
	if _, _, ok := parseTokenResponse([]byte(`{"token":null,"expires_at":"2026-08-01T13:00:00Z"}`)); ok {
		t.Fatal("null token accepted")
	}
	if _, _, ok := parseTokenResponse([]byte(`{"token":"x","expires_at":null}`)); ok {
		t.Fatal("null expiry accepted")
	}
	if value, _, ok := parseTokenResponse([]byte(`{"token":"x","expires_at":"2026-08-01T13:00:00Z"}`)); !ok || value != "x" {
		t.Fatal("valid parser response rejected")
	}
}
