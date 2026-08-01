// Package egresspolicy contains the pure allowlist decision used at the
// harness' egress boundary.  It does not open connections, inspect
// credentials, or perform any other I/O.
package egresspolicy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	// MaxURLBytes is the maximum raw URL size examined by Authorize.
	MaxURLBytes = 4 * 1024

	// Decision values are intentionally small, stable strings.  An allow
	// decision does not itself authorize a network connection.
	DecisionDeny                Decision = "deny"
	DecisionGitHubRESTRead      Decision = "allow/github-rest-read"
	DecisionOpenAIResponsesText Decision = "allow/openai-responses-text"
)

// Decision is the result of a policy check.
type Decision string

// Scope is the canonical provider scope produced by the same evaluation as
// an allow decision.  A zero Scope is returned whenever the request is denied.
// Callers must not derive these fields from the URL themselves.
type Scope struct {
	Provider        string
	Repository      string
	Operation       string
	DestinationHost string
}

// Error is a fixed, comparable policy error.  Constants prevent callers from
// replacing the package's sentinel values after they have been published.
type Error string

func (e Error) Error() string { return string(e) }

// Rules describes the two intentionally small provider surfaces understood by
// this package.  All slices are copied by New.
type Rules struct {
	GitHubRepositories []string
	OpenAIModels       []string
	MaxBodyBytes       int
	MaxOutputTokens    int
}

// Request is the complete value examined by Authorize.  URL is the raw URL
// supplied by the caller; it is never normalized as a basis for allowing a
// request.
type Request struct {
	Method      string
	URL         string
	ContentType string
	Body        []byte
}

// Policy is immutable after New returns.  Its fields are deliberately
// unexported so caller-owned slices cannot become aliases of policy state.
type Policy struct {
	githubRepositories map[string]struct{}
	openAIModels       map[string]struct{}
	maxBodyBytes       int
	maxOutputTokens    int
}

const (
	// ErrInvalidRules is returned by New for every invalid Rules value.  Its
	// text is fixed and contains no caller input.
	ErrInvalidRules Error = "invalid-rules"
	// ErrDenied is returned for every non-allow request.  Its text is fixed and
	// contains no URL, body, parser, or operating-system detail.
	ErrDenied Error = "request-denied"
)

// New validates Rules and returns an immutable policy.  The caller may reuse
// or mutate the Rules slices immediately after this function returns.
func New(r Rules) (*Policy, error) {
	if len(r.GitHubRepositories) == 0 || len(r.OpenAIModels) == 0 ||
		r.MaxBodyBytes <= 0 || r.MaxOutputTokens <= 0 {
		return nil, ErrInvalidRules
	}

	github := make(map[string]struct{}, len(r.GitHubRepositories))
	for _, repository := range r.GitHubRepositories {
		if !validRepository(repository) {
			return nil, ErrInvalidRules
		}
		if _, exists := github[repository]; exists {
			return nil, ErrInvalidRules
		}
		github[repository] = struct{}{}
	}

	models := make(map[string]struct{}, len(r.OpenAIModels))
	for _, model := range r.OpenAIModels {
		if !validModel(model) {
			return nil, ErrInvalidRules
		}
		if _, exists := models[model]; exists {
			return nil, ErrInvalidRules
		}
		models[model] = struct{}{}
	}

	return &Policy{
		githubRepositories: github,
		openAIModels:       models,
		maxBodyBytes:       r.MaxBodyBytes,
		maxOutputTokens:    r.MaxOutputTokens,
	}, nil
}

// Authorize returns one of the fixed provider allow decisions or the fixed
// deny decision and error.  A nil or zero Policy always denies.
func (p *Policy) Authorize(req Request) (Decision, error) {
	_, decision, err := p.Evaluate(req)
	return decision, err
}

// Evaluate evaluates req once and returns both the existing decision and
// its canonical scope.  It preserves Authorize's fixed deny behavior.
func (p *Policy) Evaluate(req Request) (Scope, Decision, error) {
	if p == nil || len(p.githubRepositories) == 0 || len(p.openAIModels) == 0 ||
		p.maxBodyBytes <= 0 || p.maxOutputTokens <= 0 {
		return Scope{}, DecisionDeny, ErrDenied
	}

	if scope, decision, ok := p.authorizeGitHub(req.Method, req.URL); ok {
		return scope, decision, nil
	}
	if scope, decision, ok := p.authorizeOpenAI(req.Method, req.URL, req.ContentType, req.Body); ok {
		return scope, decision, nil
	}
	return Scope{}, DecisionDeny, ErrDenied
}

func (p *Policy) authorizeGitHub(method, rawURL string) (Scope, Decision, bool) {
	if method != "GET" && method != "HEAD" {
		return Scope{}, DecisionDeny, false
	}
	u, ok := parseCanonicalURL(rawURL, "api.github.com", "/")
	if !ok {
		return Scope{}, DecisionDeny, false
	}
	segments, ok := canonicalPathSegments(u.Path)
	if !ok || len(segments) < 3 || segments[0] != "repos" {
		return Scope{}, DecisionDeny, false
	}
	// owner and repository are validated as canonical identifiers in New;
	// requiring the path values to use the same restricted alphabet prevents
	// parser or Unicode normalization from becoming an allowlist match.
	if !validRepoSegment(segments[1]) || !validRepoSegment(segments[2]) {
		return Scope{}, DecisionDeny, false
	}
	for _, child := range segments[3:] {
		if !validChildSegment(child) {
			return Scope{}, DecisionDeny, false
		}
	}
	repository := segments[1] + "/" + segments[2]
	if _, allowed := p.githubRepositories[repository]; !allowed {
		return Scope{}, DecisionDeny, false
	}
	return Scope{Provider: "github", Repository: repository, Operation: "github-rest-read", DestinationHost: "api.github.com"}, DecisionGitHubRESTRead, true
}

func (p *Policy) authorizeOpenAI(method, rawURL, contentType string, body []byte) (Scope, Decision, bool) {
	if method != "POST" || contentType != "application/json" || len(body) == 0 || len(body) > p.maxBodyBytes {
		return Scope{}, DecisionDeny, false
	}
	u, ok := parseCanonicalURL(rawURL, "api.openai.com", "/v1/responses")
	if !ok || u.Path != "/v1/responses" || !utf8.Valid(body) {
		return Scope{}, DecisionDeny, false
	}
	if !strictOpenAIRequest(body, p.openAIModels, p.maxOutputTokens) {
		return Scope{}, DecisionDeny, false
	}
	return Scope{Provider: "openai", Operation: "openai-responses-text", DestinationHost: "api.openai.com"}, DecisionOpenAIResponsesText, true
}

// parseCanonicalURL performs structural checks without using parser
// normalization as an allowlist key.  Percent-encoding is rejected before
// parsing, so Path and the authority are the raw canonical forms.
func parseCanonicalURL(raw, host, path string) (*url.URL, bool) {
	if len(raw) == 0 || len(raw) > MaxURLBytes || strings.IndexByte(raw, '%') >= 0 || strings.IndexByte(raw, '#') >= 0 || !asciiURL(raw) {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil || u.String() != raw || u.Scheme != "https" || u.Host != host && u.Host != host+":443" ||
		u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery ||
		u.Fragment != "" || u.RawFragment != "" || u.RawPath != "" {
		return nil, false
	}
	if u.Path == "" || (path != "/" && u.Path != path) {
		return nil, false
	}
	return u, true
}

func canonicalPathSegments(path string) ([]string, bool) {
	if path == "" || path[0] != '/' || !asciiPath(path) {
		return nil, false
	}
	parts := strings.Split(path[1:], "/")
	if len(parts) == 0 {
		return nil, false
	}
	return parts, true
}

func validRepository(repository string) bool {
	parts := strings.Split(repository, "/")
	return len(parts) == 2 && validRepoSegment(parts[0]) && validRepoSegment(parts[1])
}

func validRepoSegment(segment string) bool {
	if segment == "" || segment == "." || segment == ".." {
		return false
	}
	if segment[0] < 'a' || segment[0] > 'z' {
		if segment[0] < '0' || segment[0] > '9' {
			return false
		}
	}
	for i := 0; i < len(segment); i++ {
		c := segment[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func validChildSegment(segment string) bool {
	return segment != "" && segment != "." && segment != ".." && asciiPath(segment) && !strings.ContainsRune(segment, '\\')
}

func validModel(model string) bool {
	if model == "" || !asciiIdentifier(model[0], true) {
		return false
	}
	for i := 1; i < len(model); i++ {
		if !asciiIdentifier(model[i], false) {
			return false
		}
	}
	return true
}

func asciiIdentifier(c byte, first bool) bool {
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
		return true
	}
	return !first && (c == '.' || c == '_' || c == '-')
}

func asciiURL(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] == 0x7f || value[i] >= 0x80 {
			return false
		}
	}
	return true
}

func asciiPath(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] == 0x7f || value[i] >= 0x80 {
			return false
		}
	}
	return true
}

type openAIRequest struct {
	Model           *string `json:"model"`
	Input           *string `json:"input"`
	Instructions    *string `json:"instructions"`
	Store           *bool   `json:"store"`
	Stream          *bool   `json:"stream"`
	MaxOutputTokens *int    `json:"max_output_tokens"`
}

func strictOpenAIRequest(body []byte, models map[string]struct{}, maxOutputTokens int) bool {
	if !scanSingleJSONObject(body) {
		return false
	}
	var request openAIRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return false
	}
	if request.Model == nil || request.Input == nil || request.Store == nil || request.Stream == nil || request.MaxOutputTokens == nil ||
		*request.Input == "" || *request.Store || *request.Stream || *request.MaxOutputTokens <= 0 || *request.MaxOutputTokens > maxOutputTokens {
		return false
	}
	if _, ok := models[*request.Model]; !ok {
		return false
	}
	// Instructions is optional, but if present DisallowUnknownFields and the
	// string pointer ensure it is a string (rather than null, object, or array).
	if request.Instructions == nil && hasJSONField(body, "instructions") {
		return false
	}
	return true
}

// scanSingleJSONObject verifies that the top-level value is one object,
// rejects duplicate keys at every nesting level, and rejects trailing JSON.
func scanSingleJSONObject(body []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return false
	}
	if !scanObjectContents(decoder) {
		return false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return false
	}
	return true
}

func scanObjectContents(decoder *json.Decoder) bool {
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return false
		}
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
		if !scanJSONValue(decoder) {
			return false
		}
	}
	end, err := decoder.Token()
	return err == nil && end == json.Delim('}')
}

func scanJSONValue(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	switch delimiter := token.(type) {
	case json.Delim:
		switch delimiter {
		case '{':
			return scanObjectContents(decoder)
		case '[':
			for decoder.More() {
				if !scanJSONValue(decoder) {
					return false
				}
			}
			end, err := decoder.Token()
			return err == nil && end == json.Delim(']')
		default:
			return false
		}
	default:
		return true
	}
}

// hasJSONField is only used to distinguish an omitted optional string from
// an explicit JSON null.  The duplicate-key pass and strict decoder still
// perform all structural and type checks.
func hasJSONField(body []byte, wanted string) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return false
	}
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, ok := keyToken.(string)
		if err != nil || !ok {
			return false
		}
		if key == wanted {
			return true
		}
		if !scanJSONValue(decoder) {
			return false
		}
	}
	return false
}
