package egresstransaction

import (
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
)

var testSubject = Subject{AgentInstanceID: "agent-1", UID: 1000, WorkspaceID: "workspace-1"}

func testPolicy(t *testing.T) *egresspolicy.Policy {
	t.Helper()
	p, err := egresspolicy.New(egresspolicy.Rules{
		GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"},
		MaxBodyBytes: 4096, MaxOutputTokens: 256,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func testRegistry(t *testing.T, provider string, uses int) (*capability.Registry, string) {
	t.Helper()
	r, err := capability.New(capability.Rules{PolicyVersion: "policy-v1", MaxTTL: time.Hour, MaxUses: 4})
	if err != nil {
		t.Fatal(err)
	}
	spec := capability.IssueSpec{AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID,
		WorkspaceID: testSubject.WorkspaceID, Provider: provider, TTL: time.Minute, Uses: uses}
	if provider == capability.ProviderGitHub {
		spec.Repository = "acme/widget"
	}
	handle, err := r.Issue(spec)
	if err != nil {
		t.Fatal(err)
	}
	return r, handle
}

func newTransaction(t *testing.T, registry *capability.Registry, resolver CredentialResolver, forwarder Forwarder) *Transaction {
	t.Helper()
	txn, err := New(Rules{Policy: testPolicy(t), Registry: registry, Resolver: resolver, Forwarder: forwarder, MaxCredentialBytes: 128})
	if err != nil {
		t.Fatal(err)
	}
	return txn
}

func githubRequest(auth string) Request {
	return Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget", Authorization: []string{auth}}
}

func openAIRequest(auth string) Request {
	return Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json",
		Body:          []byte(`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":32}`),
		Authorization: []string{auth}}
}

func gitRequest(auth string) Request {
	return Request{Method: "POST", URL: "https://github.com/acme/widget.git/git-upload-pack", ContentType: egresspolicy.GitUploadPackRequest,
		Body: []byte("0009done\n"), Authorization: []string{auth}}
}

func testGitRegistry(t *testing.T, uses int) (*capability.Registry, string) {
	t.Helper()
	registry, err := capability.New(capability.Rules{PolicyVersion: "policy-v1", MaxTTL: time.Hour, MaxUses: 4})
	if err != nil {
		t.Fatal(err)
	}
	handle, err := registry.Issue(capability.IssueSpec{AgentInstanceID: testSubject.AgentInstanceID, UID: testSubject.UID,
		WorkspaceID: testSubject.WorkspaceID, Provider: capability.ProviderGitHub, Repository: "acme/widget",
		Operation: capability.OperationGitHubGitRead, TTL: time.Minute, Uses: uses})
	if err != nil {
		t.Fatal(err)
	}
	return registry, handle
}

func gitBasic(handle string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+handle))
}

func TestGitBasicCapabilityConsumedThenReplacedOnce(t *testing.T) {
	registry, handle := testGitRegistry(t, 1)
	resolverCalls, forwarderCalls := 0, 0
	var prepared PreparedRequest
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(provider, repository string) (string, error) {
			resolverCalls++
			if provider != capability.ProviderGitHub || repository != "acme/widget" {
				t.Fatalf("resolver scope=(%q,%q)", provider, repository)
			}
			return "real-token", nil
		}),
		ForwarderFunc(func(request PreparedRequest) error { forwarderCalls++; prepared = request; return nil }),
	)
	if err := txn.Execute(testSubject, gitRequest(gitBasic(handle))); err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:real-token"))
	if resolverCalls != 1 || forwarderCalls != 1 || prepared.Authorization != want || strings.Contains(prepared.Authorization, handle) ||
		prepared.Scope.Operation != capability.OperationGitHubGitRead || prepared.Scope.DestinationHost != capability.HostGitHubGit {
		t.Fatalf("prepared=%+v resolver=%d forwarder=%d", prepared, resolverCalls, forwarderCalls)
	}
	if err := txn.Execute(testSubject, gitRequest(gitBasic(handle))); !errors.Is(err, ErrDenied) || resolverCalls != 1 || forwarderCalls != 1 {
		t.Fatalf("reuse=%v resolver=%d forwarder=%d", err, resolverCalls, forwarderCalls)
	}
}

func TestGitBasicMalformedDoesNotSpendCapability(t *testing.T) {
	registry, handle := testGitRegistry(t, 1)
	resolverCalls, forwarderCalls := 0, 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { resolverCalls++; return "real-token", nil }),
		ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }),
	)
	encode := func(value string) string { return "Basic " + base64.StdEncoding.EncodeToString([]byte(value)) }
	for _, value := range []string{
		"Bearer " + handle,
		"token " + handle,
		"basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+handle)),
		"Basic  " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+handle)),
		"Basic !!!",
		"Basic " + strings.TrimSuffix(base64.StdEncoding.EncodeToString([]byte("x-access-token:"+handle)), "="),
		encode("other:" + handle),
		encode("x-access-token:"),
		encode("x-access-token:not-a-handle"),
		encode("x-access-token:" + handle + ":extra"),
		encode("x-access-token:\n" + handle),
	} {
		if err := txn.Execute(testSubject, gitRequest(value)); !errors.Is(err, ErrDenied) {
			t.Fatalf("auth %q err=%v", value, err)
		}
	}
	duplicate := gitRequest(gitBasic(handle))
	duplicate.Authorization = []string{gitBasic(handle), gitBasic(handle)}
	if err := txn.Execute(testSubject, duplicate); !errors.Is(err, ErrDenied) {
		t.Fatalf("duplicate auth=%v", err)
	}
	if resolverCalls != 0 || forwarderCalls != 0 {
		t.Fatalf("malformed reached resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
	if err := txn.Execute(testSubject, gitRequest(gitBasic(handle))); err != nil {
		t.Fatalf("malformed Basic spent handle: %v", err)
	}
}

func TestGitResolverFailureConsumesWithoutRetryOrLeak(t *testing.T) {
	registry, handle := testGitRegistry(t, 1)
	resolverCalls, forwarderCalls := 0, 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) {
			resolverCalls++
			return "", errors.New("real-token-secret lower URL")
		}),
		ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }),
	)
	authorization := gitBasic(handle)
	for attempt := 0; attempt < 2; attempt++ {
		err := txn.Execute(testSubject, gitRequest(authorization))
		if !errors.Is(err, ErrDenied) || strings.Contains(err.Error(), handle) || strings.Contains(err.Error(), "real-token-secret") || strings.Contains(err.Error(), "URL") {
			t.Fatalf("attempt=%d err=%v", attempt, err)
		}
	}
	if resolverCalls != 1 || forwarderCalls != 0 {
		t.Fatalf("resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
}

func TestNewRulesAndZeroDependenciesFailClosed(t *testing.T) {
	resolver := CredentialResolverFunc(func(string, string) (string, error) { return "secret", nil })
	forwarder := ForwarderFunc(func(PreparedRequest) error { return nil })
	registry, _ := testRegistry(t, capability.ProviderGitHub, 1)
	policy := testPolicy(t)
	for _, rules := range []Rules{
		{}, {Policy: policy, Registry: registry, Resolver: resolver, Forwarder: forwarder},
		{Policy: policy, Registry: registry, Resolver: resolver, Forwarder: forwarder, MaxCredentialBytes: 4097},
		{Policy: policy, Registry: registry, Resolver: resolver, Forwarder: forwarder, MaxCredentialBytes: -1},
	} {
		if txn, err := New(rules); txn != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("New(%#v)=(%p,%v), want fixed Rules error", rules, txn, err)
		}
	}
	zeroPolicy := &egresspolicy.Policy{}
	zeroRegistry := &capability.Registry{}
	if err := newTransaction(t, zeroRegistry, resolver, forwarder).Execute(testSubject, githubRequest("Bearer cap_bad")); !errors.Is(err, ErrDenied) {
		t.Fatalf("zero registry error=%v", err)
	}
	txn, err := New(Rules{Policy: zeroPolicy, Registry: registry, Resolver: resolver, Forwarder: forwarder, MaxCredentialBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer cap_bad")); !errors.Is(err, ErrDenied) {
		t.Fatalf("zero policy error=%v", err)
	}
}

func TestExecuteBothProvidersAndCopiesBody(t *testing.T) {
	for _, tc := range []struct {
		provider string
		request  func(string) Request
		wantRepo string
	}{
		{capability.ProviderGitHub, githubRequest, "acme/widget"},
		{capability.ProviderOpenAI, openAIRequest, ""},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			registry, handle := testRegistry(t, tc.provider, 1)
			var got PreparedRequest
			var calls int
			txn := newTransaction(t, registry,
				CredentialResolverFunc(func(provider, repository string) (string, error) {
					if provider != tc.provider || repository != tc.wantRepo {
						t.Fatalf("resolver scope=(%q,%q)", provider, repository)
					}
					return "upstream-secret", nil
				}),
				ForwarderFunc(func(req PreparedRequest) error { got = req; calls++; return nil }),
			)
			req := tc.request("Bearer " + handle)
			before := append([]byte(nil), req.Body...)
			if err := txn.Execute(testSubject, req); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || got.Authorization != "Bearer upstream-secret" || got.Scope.Repository != tc.wantRepo ||
				strings.Contains(got.Authorization, handle) {
				t.Fatalf("forwarder request=%#v calls=%d", got, calls)
			}
			if len(got.Body) > 0 {
				got.Body[0] = 'X'
			}
			if string(req.Body) != string(before) {
				t.Fatal("Execute or Forwarder modified caller body")
			}
		})
	}
}

func TestProviderAuthorizationSchemes(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	forwarded := 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { forwarded++; return nil }),
	)
	if err := txn.Execute(testSubject, githubRequest("token "+handle)); err != nil {
		t.Fatalf("GitHub token scheme=%v", err)
	}
	openAIRegistry, openAIHandle := testRegistry(t, capability.ProviderOpenAI, 1)
	openAITxn := newTransaction(t, openAIRegistry,
		CredentialResolverFunc(func(string, string) (string, error) { return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { forwarded++; return nil }),
	)
	if err := openAITxn.Execute(testSubject, openAIRequest("token "+openAIHandle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("OpenAI token scheme=%v", err)
	}
	if forwarded != 1 {
		t.Fatalf("OpenAI token reached Forwarder: calls=%d", forwarded)
	}
}

func TestPolicyAndScopeDenialsDoNotSpendHandle(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	resolverCalls, forwarderCalls := 0, 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { resolverCalls++; return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }),
	)
	denied := githubRequest("Bearer " + handle)
	denied.URL = "https://evil.example/repos/acme/widget"
	if err := txn.Execute(testSubject, denied); !errors.Is(err, ErrDenied) {
		t.Fatalf("policy denial=%v", err)
	}
	if resolverCalls != 0 || forwarderCalls != 0 {
		t.Fatalf("policy denial reached dependencies resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); err != nil {
		t.Fatalf("valid request after policy denial=%v", err)
	}
	if resolverCalls != 1 || forwarderCalls != 1 {
		t.Fatalf("policy denial reached dependencies resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}

	registry, handle = testRegistry(t, capability.ProviderGitHub, 1)
	txn = newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { resolverCalls++; return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }),
	)
	wrongSubject := testSubject
	wrongSubject.WorkspaceID = "other-workspace"
	if err := txn.Execute(wrongSubject, githubRequest("Bearer "+handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("subject mismatch=%v", err)
	}
	if resolverCalls != 1 || forwarderCalls != 1 {
		t.Fatalf("subject mismatch reached dependencies resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); err != nil {
		t.Fatalf("handle was spent by subject mismatch=%v", err)
	}
	if resolverCalls != 2 || forwarderCalls != 2 {
		t.Fatalf("subject-valid dependency count resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
	registry, handle = testRegistry(t, capability.ProviderGitHub, 1)
	txn = newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { resolverCalls++; return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }),
	)
	if err := txn.Execute(testSubject, openAIRequest("Bearer "+handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("scope mismatch=%v", err)
	}
	if resolverCalls != 2 || forwarderCalls != 2 {
		t.Fatalf("scope mismatch reached dependencies resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); err != nil {
		t.Fatalf("handle was spent by scope mismatch=%v", err)
	}
	if resolverCalls != 3 || forwarderCalls != 3 {
		t.Fatalf("scope-valid dependency count resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
}

func TestNewRejectsEachNilDependencyAndNilTransaction(t *testing.T) {
	registry, _ := testRegistry(t, capability.ProviderGitHub, 1)
	policy := testPolicy(t)
	resolver := CredentialResolverFunc(func(string, string) (string, error) { return "secret", nil })
	forwarder := ForwarderFunc(func(PreparedRequest) error { return nil })
	cases := []Rules{
		{Registry: registry, Resolver: resolver, Forwarder: forwarder, MaxCredentialBytes: 1},
		{Policy: policy, Resolver: resolver, Forwarder: forwarder, MaxCredentialBytes: 1},
		{Policy: policy, Registry: registry, Forwarder: forwarder, MaxCredentialBytes: 1},
		{Policy: policy, Registry: registry, Resolver: resolver, MaxCredentialBytes: 1},
	}
	for i, rules := range cases {
		if txn, err := New(rules); txn != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("nil dependency case %d=(%p,%v)", i, txn, err)
		}
	}
	var nilTxn *Transaction
	if err := nilTxn.Execute(testSubject, Request{}); !errors.Is(err, ErrDenied) {
		t.Fatalf("nil Transaction Execute=%v", err)
	}
}

func TestCredentialLengthBoundsAndAuthorizationSliceUnchanged(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	forwarderCalls := 0
	txn, err := New(Rules{
		Policy: testPolicy(t), Registry: registry,
		Resolver:  CredentialResolverFunc(func(string, string) (string, error) { return "12345", nil }),
		Forwarder: ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }), MaxCredentialBytes: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	authorization := []string{"Bearer " + handle}
	req := githubRequest(authorization[0])
	req.Authorization = authorization
	before := append([]string(nil), authorization...)
	if err := txn.Execute(testSubject, req); !errors.Is(err, ErrDenied) || forwarderCalls != 0 {
		t.Fatalf("overlong credential=%v calls=%d", err, forwarderCalls)
	}
	if len(authorization) != len(before) || authorization[0] != before[0] {
		t.Fatal("Execute modified Authorization slice")
	}

	registry, handle = testRegistry(t, capability.ProviderGitHub, 1)
	txn, err = New(Rules{
		Policy: testPolicy(t), Registry: registry,
		Resolver:  CredentialResolverFunc(func(string, string) (string, error) { return "1234", nil }),
		Forwarder: ForwarderFunc(func(PreparedRequest) error { return nil }), MaxCredentialBytes: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); err != nil {
		t.Fatalf("credential boundary=%v", err)
	}
}

func TestExecuteRejectsAuthorizationBoundariesBeforeConsume(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	resolverCalls, forwarderCalls := 0, 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { resolverCalls++; return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }),
	)
	for _, value := range []string{"bearer " + handle, "Bearer  " + handle, "Bearer " + handle + " ", "Bearer " + handle + "\n", "token " + handle + " extra"} {
		if err := txn.Execute(testSubject, githubRequest(value)); !errors.Is(err, ErrDenied) {
			t.Fatalf("auth %q error=%v", value, err)
		}
	}
	bad := githubRequest("Bearer " + handle)
	bad.Authorization = []string{"Bearer " + handle, "Bearer " + handle}
	if err := txn.Execute(testSubject, bad); !errors.Is(err, ErrDenied) {
		t.Fatalf("multiple auth error=%v", err)
	}
	if resolverCalls != 0 || forwarderCalls != 0 {
		t.Fatalf("calls after auth denials resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); err != nil {
		t.Fatalf("valid handle was spent by auth denials: %v", err)
	}
}

func TestExecuteConsumesBeforeResolverAndNeverRetries(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	resolverCalls, forwarderCalls := 0, 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { resolverCalls++; return "", errors.New("secret detail") }),
		ForwarderFunc(func(PreparedRequest) error { forwarderCalls++; return nil }),
	)
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("resolver failure=%v", err)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); !errors.Is(err, ErrDenied) {
		t.Fatalf("consumed handle retry=%v", err)
	}
	if resolverCalls != 1 || forwarderCalls != 0 {
		t.Fatalf("failure order resolver=%d forwarder=%d", resolverCalls, forwarderCalls)
	}
}

func TestCredentialValidationAndForwarderFailureAreFixed(t *testing.T) {
	for _, credential := range []string{"", "has space", "has\ttab", "has\nnewline", "bad\x00", "é"} {
		registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
		calls := 0
		txn := newTransaction(t, registry,
			CredentialResolverFunc(func(string, string) (string, error) { return credential, nil }),
			ForwarderFunc(func(PreparedRequest) error { calls++; return nil }),
		)
		if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); !errors.Is(err, ErrDenied) || calls != 0 {
			t.Fatalf("credential %q error=%v calls=%d", credential, err, calls)
		}
	}
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	calls := 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { calls++; return errors.New("network detail") }),
	)
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); !errors.Is(err, ErrDenied) || calls != 1 {
		t.Fatalf("forwarder failure=%v calls=%d", err, calls)
	}
	if err := txn.Execute(testSubject, githubRequest("Bearer "+handle)); !errors.Is(err, ErrDenied) || calls != 1 {
		t.Fatalf("forwarder retry=%v calls=%d", err, calls)
	}
	if strings.Contains(ErrDenied.Error(), "network detail") {
		t.Fatal("fixed error contains forwarder detail")
	}
}

func TestConcurrentSingleUseReachesResolverAndForwarderOnce(t *testing.T) {
	registry, handle := testRegistry(t, capability.ProviderGitHub, 1)
	var mu sync.Mutex
	resolverCalls, forwarderCalls := 0, 0
	txn := newTransaction(t, registry,
		CredentialResolverFunc(func(string, string) (string, error) { mu.Lock(); resolverCalls++; mu.Unlock(); return "secret", nil }),
		ForwarderFunc(func(PreparedRequest) error { mu.Lock(); forwarderCalls++; mu.Unlock(); return nil }),
	)
	const workers = 16
	var wg sync.WaitGroup
	results := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- txn.Execute(testSubject, githubRequest("Bearer "+handle)) }()
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrDenied) {
			t.Fatalf("concurrent error=%v", err)
		}
	}
	if successes != 1 || resolverCalls != 1 || forwarderCalls != 1 {
		t.Fatalf("successes=%d resolver=%d forwarder=%d", successes, resolverCalls, forwarderCalls)
	}
}
