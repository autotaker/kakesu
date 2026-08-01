package egresspolicy

import (
	"errors"
	"strings"
	"testing"
)

func testRules() Rules {
	return Rules{
		GitHubRepositories: []string{"acme/widget"},
		OpenAIModels:       []string{"gpt-5-mini"},
		MaxBodyBytes:       1024,
		MaxOutputTokens:    256,
	}
}

func mustPolicy(t *testing.T) *Policy {
	t.Helper()
	p, err := New(testRules())
	if err != nil {
		t.Fatalf("New(valid): %v", err)
	}
	return p
}

func authorizeDenied(t *testing.T, p *Policy, req Request) {
	t.Helper()
	decision, err := p.Authorize(req)
	if decision != DecisionDeny || !errors.Is(err, ErrDenied) {
		t.Fatalf("Authorize(%#v) = (%q, %v), want deny/%v", req, decision, err, ErrDenied)
	}
}

func TestNewRejectsInvalidRulesAndCopiesSlices(t *testing.T) {
	valid := testRules()
	cases := []Rules{
		{},
		{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 0, MaxOutputTokens: 1},
		{GitHubRepositories: []string{"acme/widget", "acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 1, MaxOutputTokens: 1},
		{GitHubRepositories: []string{"Acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 1, MaxOutputTokens: 1},
		{GitHubRepositories: []string{"-acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 1, MaxOutputTokens: 1},
		{GitHubRepositories: []string{"acme/_widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 1, MaxOutputTokens: 1},
		{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt 5"}, MaxBodyBytes: 1, MaxOutputTokens: 1},
		{GitHubRepositories: []string{"acme/widget"}, OpenAIModels: []string{"gpt-5-mini"}, MaxBodyBytes: 1, MaxOutputTokens: -1},
	}
	for i, rules := range cases {
		t.Run("invalid-"+string(rune('a'+i)), func(t *testing.T) {
			if p, err := New(rules); p != nil || !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("New(%#v) = (%p, %v), want fixed rules error", rules, p, err)
			}
		})
	}

	p, err := New(valid)
	if err != nil {
		t.Fatal(err)
	}
	valid.GitHubRepositories[0] = "evil/changed"
	valid.OpenAIModels[0] = "other-model"
	if _, err := p.Authorize(Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget"}); err != nil {
		t.Fatalf("policy changed after Rules mutation: %v", err)
	}
	validBody := []byte(`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":32}`)
	if decision, err := p.Authorize(Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: validBody}); decision != DecisionOpenAIResponsesText || err != nil {
		t.Fatalf("OpenAI policy changed after Rules mutation: decision=%q err=%v", decision, err)
	}
}

func TestAuthorizeGitHubCanonicalSurface(t *testing.T) {
	p := mustPolicy(t)
	for _, method := range []string{"GET", "HEAD"} {
		for _, rawURL := range []string{
			"https://api.github.com/repos/acme/widget",
			"https://api.github.com:443/repos/acme/widget/issues/1",
		} {
			decision, err := p.Authorize(Request{Method: method, URL: rawURL})
			if decision != DecisionGitHubRESTRead || err != nil {
				t.Fatalf("Authorize(%s %s) = (%q, %v), want github allow", method, rawURL, decision, err)
			}
		}
	}

	denied := []string{
		"http://api.github.com/repos/acme/widget",
		"https://evil.example/repos/acme/widget",
		"https://api.github.com:444/repos/acme/widget",
		"https://api.github.com/repos/acme/widget?x=1",
		"https://api.github.com/repos/acme/widget?",
		"https://api.github.com/repos/acme/widget#fragment",
		"https://api.github.com/repos/acme/widget ",
		"https://user@api.github.com/repos/acme/widget",
		"https://api.github.com/repos/acme%2Fwidget",
		"https://api.github.com/repos/acme/widget/.",
		"https://api.github.com/repos/acme/widget/..",
		"https://api.github.com/repos/acme/widget/",
		"https://api.github.com/repos/acme//widget",
		"https://api.github.com/repos/-acme/widget",
		"https://api.github.com/repos/acme/other",
		"https://api.github.com/users/acme",
		"https://api.github.com/graphql",
	}
	for _, rawURL := range denied {
		t.Run(rawURL, func(t *testing.T) {
			authorizeDenied(t, p, Request{Method: "GET", URL: rawURL})
		})
	}
	authorizeDenied(t, p, Request{Method: "POST", URL: "https://api.github.com/repos/acme/widget"})
	authorizeDenied(t, p, Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget/\u00e9"})
	authorizeDenied(t, p, Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget/\\child"})
	authorizeDenied(t, p, Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget#"})
	authorizeDenied(t, p, Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget" + strings.Repeat("x", MaxURLBytes)})
}

func TestAuthorizeOpenAIResponsesTextStrictBody(t *testing.T) {
	p := mustPolicy(t)
	validBodies := []string{
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","instructions":"be concise","store":false,"stream":false,"max_output_tokens":32}`,
	}
	for _, body := range validBodies {
		decision, err := p.Authorize(Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: []byte(body)})
		if decision != DecisionOpenAIResponsesText || err != nil {
			t.Fatalf("valid OpenAI body = (%q, %v), want text allow", decision, err)
		}
	}

	deniedBodies := []string{
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":32} trailing`,
		`{"model":"gpt-5-mini",`,
		`{"model":"gpt-5-mini","model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":32,"tools":[]}`,
		`{"model":"not-allowed","input":"hello","store":false,"stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"","store":false,"stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":[],"store":false,"stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","store":true,"stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":true,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false}`,
		`{"model":"gpt-5-mini","input":"hello","stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","store":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":0}`,
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":1.5}`,
		`{"model":"gpt-5-mini","input":"hello","store":false,"stream":false,"max_output_tokens":257}`,
		`{"model":"gpt-5-mini","input":"hello","instructions":null,"store":false,"stream":false,"max_output_tokens":32}`,
		`{"model":"gpt-5-mini","input":"hello","instructions":[],"store":false,"stream":false,"max_output_tokens":32}`,
	}
	for i, body := range deniedBodies {
		t.Run("body-"+string(rune('a'+i)), func(t *testing.T) {
			authorizeDenied(t, p, Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: []byte(body)})
		})
	}

	for _, req := range []Request{
		{Method: "POST", URL: "https://api.openai.com/v1/responses?x=1", ContentType: "application/json", Body: []byte(validBodies[0])},
		{Method: "POST", URL: "https://evil.example/v1/responses", ContentType: "application/json", Body: []byte(validBodies[0])},
		{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json; charset=utf-8", Body: []byte(validBodies[0])},
		{Method: "GET", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: []byte(validBodies[0])},
		{Method: "POST", URL: "https://api.openai.com:444/v1/responses", ContentType: "application/json", Body: []byte(validBodies[0])},
	} {
		authorizeDenied(t, p, req)
	}
	authorizeDenied(t, p, Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: []byte(strings.Repeat("x", p.maxBodyBytes+1))})

	body := []byte(validBodies[0])
	before := append([]byte(nil), body...)
	if _, err := p.Authorize(Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: body}); err != nil {
		t.Fatal(err)
	}
	if string(body) != string(before) {
		t.Fatal("Authorize modified Request body")
	}
	if _, err := p.Authorize(Request{Method: "POST", URL: "https://api.openai.com/v1/responses", ContentType: "application/json", Body: []byte{0xff}}); !errors.Is(err, ErrDenied) {
		t.Fatalf("invalid UTF-8 error = %v, want fixed deny", err)
	}
}

func TestAuthorizeDenyErrorDoesNotLeakInput(t *testing.T) {
	p := mustPolicy(t)
	sentinel := "credential-do-not-leak"
	decision, err := p.Authorize(Request{Method: "POST", URL: "https://evil.example/" + sentinel, ContentType: "application/json", Body: []byte(`{"model":"` + sentinel + `"}`)})
	if decision != DecisionDeny || !errors.Is(err, ErrDenied) || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("deny leaked input: decision=%q err=%v", decision, err)
	}
	var nilPolicy *Policy
	authorizeDenied(t, nilPolicy, Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget"})
	authorizeDenied(t, &Policy{}, Request{Method: "GET", URL: "https://api.github.com/repos/acme/widget"})
}
