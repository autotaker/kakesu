package upstreamforwarder

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

type fakeTransport struct {
	mu              sync.Mutex
	calls           int
	request         *http.Request
	response        *http.Response
	responseFactory func() *http.Response
	err             error
	mutate          bool
	wait            bool
}

func (t *fakeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.calls++
	t.request = request
	response, err, wait, mutate, factory := t.response, t.err, t.wait, t.mutate, t.responseFactory
	t.mu.Unlock()
	if factory != nil {
		response = factory()
	}
	if mutate && request.Body != nil {
		_, _ = io.ReadAll(request.Body)
	}
	if wait {
		<-request.Context().Done()
		return nil, request.Context().Err()
	}
	return response, err
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

type fakeBody struct {
	reader io.Reader
	mu     sync.Mutex
	closed int
	err    error
}

func (b *fakeBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *fakeBody) Close() error {
	b.mu.Lock()
	b.closed++
	b.mu.Unlock()
	return b.err
}
func (b *fakeBody) closeCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

type fakeSink struct {
	mu       sync.Mutex
	calls    int
	response Response
	err      error
}

func (s *fakeSink) Deliver(response Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.response = response
	s.response.Body = append([]byte(nil), response.Body...)
	if response.Body != nil {
		response.Body[0] ^= 0xff
	}
	return s.err
}
func (s *fakeSink) snapshot() (int, Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls, s.response
}

func testPolicy(t *testing.T) *egresspolicy.Policy {
	t.Helper()
	policy, err := egresspolicy.New(egresspolicy.Rules{
		GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"},
		MaxBodyBytes: 4096, MaxOutputTokens: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func githubPrepared(t *testing.T, method string) egresstransaction.PreparedRequest {
	t.Helper()
	req := egresstransaction.PreparedRequest{Method: method, URL: "https://api.github.com/repos/acme/widget/issues", Authorization: "Bearer secret"}
	scope, decision, err := testPolicy(t).Evaluate(egresspolicy.Request{Method: method, URL: req.URL})
	if err != nil || decision == egresspolicy.DecisionDeny {
		t.Fatal("fixture policy denied")
	}
	req.Scope = scope
	return req
}

func openAIPrepared(t *testing.T) egresstransaction.PreparedRequest {
	t.Helper()
	body := []byte(`{"model":"gpt-5-mini","input":"hi","store":false,"stream":false,"max_output_tokens":1}`)
	req := egresstransaction.PreparedRequest{Method: http.MethodPost, URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: body, Authorization: "Bearer secret"}
	scope, decision, err := testPolicy(t).Evaluate(egresspolicy.Request{Method: req.Method, URL: req.URL, ContentType: req.ContentType, Body: req.Body})
	if err != nil || decision == egresspolicy.DecisionDeny {
		t.Fatal("fixture policy denied")
	}
	req.Scope = scope
	return req
}

func gitPrepared(t *testing.T, method string) egresstransaction.PreparedRequest {
	t.Helper()
	req := egresstransaction.PreparedRequest{Method: method, Authorization: "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:real-token"))}
	if method == http.MethodGet {
		req.URL = "https://github.com/acme/widget.git/info/refs?service=git-upload-pack"
	} else {
		req.URL = "https://github.com/acme/widget.git/git-upload-pack"
		req.ContentType = egresspolicy.GitUploadPackRequest
		req.Body = []byte("0009done\n")
	}
	scope, decision, err := testPolicy(t).Evaluate(egresspolicy.Request{Method: req.Method, URL: req.URL, ContentType: req.ContentType, Body: req.Body})
	if err != nil || decision != egresspolicy.DecisionGitHubGitRead {
		t.Fatalf("Git fixture policy=(%+v,%q,%v)", scope, decision, err)
	}
	req.Scope = scope
	return req
}

func TestGitUploadPackHeadersAndBinaryResponses(t *testing.T) {
	for _, tc := range []struct {
		method       string
		responseType string
		requestType  string
	}{
		{http.MethodGet, egresspolicy.GitUploadPackAdvertise, ""},
		{http.MethodPost, egresspolicy.GitUploadPackResult, egresspolicy.GitUploadPackRequest},
	} {
		t.Run(tc.method, func(t *testing.T) {
			binary := []byte{0x00, 0xff, 'g', 'i', 't'}
			body := &fakeBody{reader: bytes.NewReader(binary)}
			transport := &fakeTransport{response: response(http.StatusOK, body, tc.responseType)}
			sink := &fakeSink{}
			forwarder := makeForwarder(t, transport, sink, 64)
			prepared := gitPrepared(t, tc.method)
			callerBody := append([]byte(nil), prepared.Body...)
			if err := forwarder.Forward(prepared); err != nil {
				t.Fatal(err)
			}
			upstream := transport.lastRequest()
			if transport.count() != 1 || upstream.URL.String() != prepared.URL || upstream.Method != tc.method ||
				upstream.Header.Get("Authorization") != prepared.Authorization || upstream.Header.Get("Accept") != tc.responseType ||
				upstream.Header.Get("Content-Type") != tc.requestType || upstream.Header.Get("User-Agent") != userAgent {
				t.Fatalf("upstream=%+v headers=%v calls=%d", upstream, upstream.Header, transport.count())
			}
			calls, got := sink.snapshot()
			if calls != 1 || got.StatusCode != http.StatusOK || got.ContentType != tc.responseType || !bytes.Equal(got.Body, binary) || body.closeCount() != 1 {
				t.Fatalf("sink=%d %+v closes=%d", calls, got, body.closeCount())
			}
			if !bytes.Equal(prepared.Body, callerBody) {
				t.Fatal("caller body changed")
			}
		})
	}
}

func TestGitResponseAndPreparedBoundariesFailBeforeDelivery(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		media       []string
		body        []byte
		max         int
		mutate      func(*egresstransaction.PreparedRequest)
		wantNetwork bool
	}{
		{name: "status", status: http.StatusCreated, media: []string{egresspolicy.GitUploadPackResult}, body: []byte("git"), max: 64, wantNetwork: true},
		{name: "empty", status: http.StatusOK, media: []string{egresspolicy.GitUploadPackResult}, max: 64, wantNetwork: true},
		{name: "wrong media", status: http.StatusOK, media: []string{egresspolicy.GitUploadPackAdvertise}, body: []byte("git"), max: 64, wantNetwork: true},
		{name: "media parameter", status: http.StatusOK, media: []string{egresspolicy.GitUploadPackResult + "; charset=binary"}, body: []byte("git"), max: 64, wantNetwork: true},
		{name: "duplicate media", status: http.StatusOK, media: []string{egresspolicy.GitUploadPackResult, egresspolicy.GitUploadPackResult}, body: []byte("git"), max: 64, wantNetwork: true},
		{name: "oversize", status: http.StatusOK, media: []string{egresspolicy.GitUploadPackResult}, body: []byte("toolong"), max: 3, wantNetwork: true},
		{name: "bearer", mutate: func(request *egresstransaction.PreparedRequest) { request.Authorization = "Bearer secret" }},
		{name: "handle basic", mutate: func(request *egresstransaction.PreparedRequest) {
			request.Authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:cap_"+string(bytes.Repeat([]byte{'A'}, 43))))
		}},
		{name: "receive pack", mutate: func(request *egresstransaction.PreparedRequest) {
			request.URL = "https://github.com/acme/widget.git/git-receive-pack"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": tc.media}, Body: io.NopCloser(bytes.NewReader(tc.body))}
			transport := &fakeTransport{response: response}
			sink := &fakeSink{}
			max := tc.max
			if max == 0 {
				max = 64
			}
			forwarder := makeForwarder(t, transport, sink, max)
			prepared := gitPrepared(t, http.MethodPost)
			if tc.mutate != nil {
				tc.mutate(&prepared)
			}
			if err := forwarder.Forward(prepared); !errors.Is(err, ErrForward) {
				t.Fatalf("error=%v", err)
			}
			if (transport.count() == 1) != tc.wantNetwork {
				t.Fatalf("transport=%d wantNetwork=%v", transport.count(), tc.wantNetwork)
			}
			if calls, _ := sink.snapshot(); calls != 0 {
				t.Fatalf("sink calls=%d", calls)
			}
		})
	}
}

func makeForwarder(t *testing.T, transport *fakeTransport, sink *fakeSink, max int) *Forwarder {
	t.Helper()
	forwarder, err := New(Rules{Policy: testPolicy(t), Transport: transport, Sink: sink, Timeout: time.Second, MaxResponseBytes: max})
	if err != nil {
		t.Fatal(err)
	}
	return forwarder
}

func response(status int, body io.ReadCloser, contentType string) *http.Response {
	return &http.Response{StatusCode: status, Body: body, Header: http.Header{"Content-Type": []string{contentType}}}
}

func TestNewRulesAndZeroReceiver(t *testing.T) {
	transport := &fakeTransport{}
	sink := &fakeSink{}
	base := Rules{Policy: testPolicy(t), Transport: transport, Sink: sink, Timeout: time.Second, MaxResponseBytes: 1}
	for _, tc := range []struct {
		name  string
		rules Rules
	}{
		{"zero", Rules{}}, {"no policy", Rules{Transport: transport, Sink: sink, Timeout: time.Second, MaxResponseBytes: 1}},
		{"no transport", Rules{Policy: base.Policy, Sink: sink, Timeout: time.Second, MaxResponseBytes: 1}},
		{"no sink", Rules{Policy: base.Policy, Transport: transport, Timeout: time.Second, MaxResponseBytes: 1}},
		{"short timeout", Rules{Policy: base.Policy, Transport: transport, Sink: sink, Timeout: 999 * time.Microsecond, MaxResponseBytes: 1}},
		{"long timeout", Rules{Policy: base.Policy, Transport: transport, Sink: sink, Timeout: 31 * time.Second, MaxResponseBytes: 1}},
		{"zero limit", Rules{Policy: base.Policy, Transport: transport, Sink: sink, Timeout: time.Second}},
		{"large limit", Rules{Policy: base.Policy, Transport: transport, Sink: sink, Timeout: time.Second, MaxResponseBytes: maxResponseBytes + 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.rules); !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	var zero Forwarder
	if err := zero.Forward(egresstransaction.PreparedRequest{}); !errors.Is(err, ErrForward) {
		t.Fatalf("zero Forward error=%v", err)
	}
	var nilForwarder *Forwarder
	if err := nilForwarder.Forward(egresstransaction.PreparedRequest{}); !errors.Is(err, ErrForward) {
		t.Fatalf("nil Forward error=%v", err)
	}
}

func TestGitHubSuccessHeadersAndOwnership(t *testing.T) {
	body := &fakeBody{reader: bytes.NewReader([]byte(`{"ok":true}`))}
	transport := &fakeTransport{response: response(http.StatusOK, body, "application/vnd.github+json; charset=utf-8"), mutate: true}
	sink := &fakeSink{}
	forwarder := makeForwarder(t, transport, sink, 1024)
	request := githubPrepared(t, http.MethodGet)
	if err := forwarder.Forward(request); err != nil {
		t.Fatal(err)
	}
	if transport.count() != 1 {
		t.Fatalf("transport calls=%d", transport.count())
	}
	if body.closeCount() != 1 {
		t.Fatalf("body closes=%d", body.closeCount())
	}
	if calls, got := sink.snapshot(); calls != 1 || got.StatusCode != http.StatusOK || got.ContentType != "application/json" || string(got.Body) != `{"ok":true}` {
		t.Fatalf("sink=%d %#v", calls, got)
	}
	upstream := transport.lastRequest()
	if upstream == nil || upstream.Header.Get("Authorization") != "Bearer secret" || upstream.Header.Get("Accept") != "application/json" || upstream.Header.Get("User-Agent") != userAgent || upstream.Header.Get("Content-Type") != "" {
		t.Fatalf("headers=%v", upstream.Header)
	}
	keys := make([]string, 0, len(upstream.Header))
	for key := range upstream.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if got, want := fmt.Sprint(keys), fmt.Sprint([]string{"Accept", "Authorization", "User-Agent"}); got != want {
		t.Fatalf("github header keys=%s", got)
	}
	if string(request.Body) != "" {
		t.Fatal("caller body changed")
	}
	if len(sink.response.Body) != 0 {
		sink.response.Body[0] ^= 0xff
	}
}

func TestOpenAISuccessAndHeaderAllowlist(t *testing.T) {
	body := &fakeBody{reader: bytes.NewReader([]byte(`{"id":"x"}`))}
	transport := &fakeTransport{response: response(http.StatusCreated, body, "application/json")}
	sink := &fakeSink{}
	forwarder := makeForwarder(t, transport, sink, 1024)
	request := openAIPrepared(t)
	caller := append([]byte(nil), request.Body...)
	if err := forwarder.Forward(request); err != nil {
		t.Fatal(err)
	}
	upstream := transport.lastRequest()
	if upstream.Header.Get("Content-Type") != "application/json" || upstream.Header.Get("Authorization") != "Bearer secret" || upstream.Header.Get("Accept") != "application/json" || upstream.Header.Get("Cookie") != "" {
		t.Fatalf("headers=%v", upstream.Header)
	}
	keys := make([]string, 0, len(upstream.Header))
	for key := range upstream.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if got, want := fmt.Sprint(keys), fmt.Sprint([]string{"Accept", "Authorization", "Content-Type", "User-Agent"}); got != want {
		t.Fatalf("openai header keys=%s", got)
	}
	if !bytes.Equal(request.Body, caller) {
		t.Fatal("caller body changed")
	}
}

func TestReevaluationAndBearerRejectBeforeTransport(t *testing.T) {
	transport := &fakeTransport{}
	sink := &fakeSink{}
	forwarder := makeForwarder(t, transport, sink, 1024)
	base := githubPrepared(t, http.MethodGet)
	for _, tc := range []struct {
		name string
		edit func(*egresstransaction.PreparedRequest)
	}{
		{"scope", func(r *egresstransaction.PreparedRequest) { r.Scope.Repository = "other/repo" }},
		{"authorization", func(r *egresstransaction.PreparedRequest) { r.Authorization = "Bearer secret\nleak" }},
		{"prefix", func(r *egresstransaction.PreparedRequest) { r.Authorization = "token secret" }},
		{"empty bearer", func(r *egresstransaction.PreparedRequest) { r.Authorization = "Bearer " }},
		{"long bearer", func(r *egresstransaction.PreparedRequest) {
			r.Authorization = "Bearer " + string(bytes.Repeat([]byte{'x'}, maxCredentialBytes+1))
		}},
		{"github body", func(r *egresstransaction.PreparedRequest) { r.Body = []byte("agent-controlled") }},
		{"github content type", func(r *egresstransaction.PreparedRequest) { r.ContentType = "application/json" }},
		{"url", func(r *egresstransaction.PreparedRequest) { r.URL = "https://evil.example/" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := base
			tc.edit(&request)
			if err := forwarder.Forward(request); !errors.Is(err, ErrForward) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if transport.count() != 0 {
		t.Fatalf("transport calls=%d", transport.count())
	}
	if calls, _ := sink.snapshot(); calls != 0 {
		t.Fatalf("sink calls=%d", calls)
	}
	openAI := openAIPrepared(t)
	for _, tc := range []struct {
		name string
		edit func(*egresstransaction.PreparedRequest)
	}{
		{"content type", func(r *egresstransaction.PreparedRequest) { r.ContentType = "text/plain" }},
		{"body", func(r *egresstransaction.PreparedRequest) { r.Body = []byte("{}") }},
		{"empty bearer", func(r *egresstransaction.PreparedRequest) { r.Authorization = "Bearer " }},
		{"long bearer", func(r *egresstransaction.PreparedRequest) {
			r.Authorization = "Bearer " + string(bytes.Repeat([]byte{'x'}, maxCredentialBytes+1))
		}},
	} {
		t.Run("openai "+tc.name, func(t *testing.T) {
			request := openAI
			tc.edit(&request)
			if err := forwarder.Forward(request); !errors.Is(err, ErrForward) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if transport.count() != 0 {
		t.Fatalf("OpenAI transport calls=%d", transport.count())
	}
}

func TestResponseValidationAndClose(t *testing.T) {
	cases := []struct {
		name        string
		method      string
		status      int
		contentType string
		data        string
		max         int
		readErr     error
		closeErr    error
		wantSuccess bool
	}{
		{"empty", http.MethodGet, 204, "", "", 64, nil, nil, true},
		{"head empty", http.MethodHead, 200, "", "", 64, nil, nil, true},
		{"head body", http.MethodHead, 200, "application/json", `{}`, 64, nil, nil, false},
		{"204 body", http.MethodGet, 204, "application/json", `{}`, 64, nil, nil, false},
		{"json", http.MethodGet, 200, "application/json", `{"ok":true}`, 64, nil, nil, true},
		{"vendor json", http.MethodGet, 200, "application/problem+json", `{}`, 64, nil, nil, true},
		{"bad type", http.MethodGet, 200, "text/plain", `{}`, 64, nil, nil, false},
		{"bad utf8", http.MethodGet, 200, "application/json", string([]byte{0xff}), 64, nil, nil, false},
		{"bad json", http.MethodGet, 200, "application/json", `{`, 64, nil, nil, false},
		{"oversize", http.MethodGet, 200, "application/json", `{}`, 1, nil, nil, false},
		{"status", http.MethodGet, 500, "application/json", `{}`, 64, nil, nil, false},
		{"read", http.MethodGet, 200, "application/json", `{}`, 64, errors.New("detail"), nil, false},
		{"close", http.MethodGet, 200, "application/json", `{}`, 64, nil, errors.New("detail"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := &fakeBody{reader: &errorReader{data: []byte(tc.data), err: tc.readErr}, err: tc.closeErr}
			transport := &fakeTransport{response: response(tc.status, body, tc.contentType)}
			sink := &fakeSink{}
			forwarder := makeForwarder(t, transport, sink, tc.max)
			request := githubPrepared(t, tc.method)
			if err := forwarder.Forward(request); (err == nil) != tc.wantSuccess {
				t.Fatalf("error=%v wantSuccess=%v", err, tc.wantSuccess)
			}
			if body.closeCount() != 1 {
				t.Fatalf("close=%d", body.closeCount())
			}
			calls, _ := sink.snapshot()
			if (calls == 1) != tc.wantSuccess {
				t.Fatalf("sink calls=%d", calls)
			}
		})
	}
}

func TestNilResponseBodyIsRejected(t *testing.T) {
	transport := &fakeTransport{response: response(http.StatusOK, nil, "application/json")}
	sink := &fakeSink{}
	forwarder := makeForwarder(t, transport, sink, 64)
	if err := forwarder.Forward(githubPrepared(t, http.MethodGet)); !errors.Is(err, ErrForward) {
		t.Fatalf("error=%v", err)
	}
	if calls, _ := sink.snapshot(); calls != 0 {
		t.Fatalf("sink calls=%d", calls)
	}
}

type errorReader struct {
	data []byte
	err  error
}

func (r *errorReader) Read(p []byte) (int, error) {
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	if r.err != nil {
		err := r.err
		r.err = nil
		return 0, err
	}
	return 0, io.EOF
}

func TestResponseAndTransportErrorClose(t *testing.T) {
	body := &fakeBody{reader: bytes.NewReader([]byte(`{}`))}
	transport := &fakeTransport{response: response(200, body, "application/json"), err: errors.New("secret transport detail")}
	sink := &fakeSink{}
	forwarder := makeForwarder(t, transport, sink, 64)
	if err := forwarder.Forward(githubPrepared(t, http.MethodGet)); !errors.Is(err, ErrForward) {
		t.Fatalf("error=%v", err)
	}
	if body.closeCount() != 1 {
		t.Fatalf("close=%d", body.closeCount())
	}
	if calls, _ := sink.snapshot(); calls != 0 {
		t.Fatalf("sink calls=%d", calls)
	}
	if stringsContains(errString(forwarder.Forward(githubPrepared(t, http.MethodGet))), "secret") {
		t.Fatal("underlying detail leaked")
	}
}

func TestSinkErrorIsFixedAndNotRetried(t *testing.T) {
	body := &fakeBody{reader: bytes.NewReader([]byte(`{}`))}
	transport := &fakeTransport{response: response(http.StatusOK, body, "application/json")}
	sink := &fakeSink{err: errors.New("sink secret")}
	forwarder := makeForwarder(t, transport, sink, 64)
	if err := forwarder.Forward(githubPrepared(t, http.MethodGet)); !errors.Is(err, ErrForward) {
		t.Fatalf("error=%v", err)
	}
	if transport.count() != 1 || body.closeCount() != 1 {
		t.Fatalf("transport=%d close=%d", transport.count(), body.closeCount())
	}
	if calls, _ := sink.snapshot(); calls != 1 {
		t.Fatalf("sink calls=%d", calls)
	}
	if stringsContains(ErrForward.Error(), "secret") || stringsContains(fmt.Sprintf("%+v %#v", forwarder, forwarder), "secret") {
		t.Fatal("fixed error or format leaked detail")
	}
}

func TestTimeoutCancellation(t *testing.T) {
	transport := &fakeTransport{wait: true}
	sink := &fakeSink{}
	forwarder, err := New(Rules{Policy: testPolicy(t), Transport: transport, Sink: sink, Timeout: time.Millisecond, MaxResponseBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	if err := forwarder.Forward(githubPrepared(t, http.MethodGet)); !errors.Is(err, ErrForward) {
		t.Fatalf("error=%v", err)
	}
	if calls, _ := sink.snapshot(); calls != 0 {
		t.Fatalf("sink calls=%d", calls)
	}
}

func TestConcurrentForwardUsesRequestLocalValues(t *testing.T) {
	transport := &fakeTransport{responseFactory: func() *http.Response {
		return response(http.StatusOK, &fakeBody{reader: bytes.NewReader([]byte(`{"ok":true}`))}, "application/json")
	}}
	sink := &fakeSink{}
	forwarder := makeForwarder(t, transport, sink, 64)
	prepared := githubPrepared(t, http.MethodGet)
	const count = 20
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := forwarder.Forward(prepared); err != nil {
				t.Errorf("Forward=%v", err)
			}
		}()
	}
	wg.Wait()
	if transport.count() != count {
		t.Fatalf("transport calls=%d", transport.count())
	}
	if calls, _ := sink.snapshot(); calls != count {
		t.Fatalf("sink calls=%d", calls)
	}
}

func stringsContains(value, needle string) bool { return bytes.Contains([]byte(value), []byte(needle)) }
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
