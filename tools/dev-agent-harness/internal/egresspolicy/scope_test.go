package egresspolicy

import (
	"errors"
	"testing"
)

func TestEvaluateReturnsCanonicalScopeAndAuthorizeCompatibility(t *testing.T) {
	p, err := New(testRules())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		req      Request
		want     Scope
		decision Decision
	}{
		{"github", Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget"}, Scope{Provider: "github", Repository: "acme/widget", Operation: "github-rest-read", DestinationHost: "api.github.com"}, DecisionGitHubRESTRead},
		{"github-git-discovery", Request{Method: "GET", URL: "https://github.com/acme/widget.git/info/refs?service=git-upload-pack"}, Scope{Provider: "github", Repository: "acme/widget", Operation: OperationGitHubGitRead, DestinationHost: GitHubGitHost}, DecisionGitHubGitRead},
		{"github-git-upload", Request{Method: "POST", URL: "https://github.com/acme/widget.git/git-upload-pack", ContentType: GitUploadPackRequest, Body: []byte("0000")}, Scope{Provider: "github", Repository: "acme/widget", Operation: OperationGitHubGitRead, DestinationHost: GitHubGitHost}, DecisionGitHubGitRead},
		{"openai", Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: []byte(`{"model":"gpt-5-mini","input":"hi","store":false,"stream":false,"max_output_tokens":1}`)}, Scope{Provider: "openai", Operation: "openai-responses-text", DestinationHost: "api.openai.com"}, DecisionOpenAIResponsesText},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scope, decision, err := p.Evaluate(tc.req)
			if err != nil || decision != tc.decision || scope != tc.want {
				t.Fatalf("Evaluate=(%#v,%q,%v)", scope, decision, err)
			}
			oldDecision, oldErr := p.Authorize(tc.req)
			if oldDecision != decision || oldErr != nil {
				t.Fatalf("Authorize=(%q,%v), Evaluate=(%q,%v)", oldDecision, oldErr, decision, err)
			}
		})
	}
	deniedScope, deniedDecision, deniedErr := p.Evaluate(Request{Method: "GET", URL: "https://evil.example/secret"})
	if deniedScope != (Scope{}) || deniedDecision != DecisionDeny || !errors.Is(deniedErr, ErrDenied) {
		t.Fatalf("deny Evaluate=(%#v,%q,%v)", deniedScope, deniedDecision, deniedErr)
	}
}
