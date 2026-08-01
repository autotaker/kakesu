// Package providercredentials resolves trusted broker credentials at the
// narrow boundary between the in-memory broker and an egress transaction.
// It intentionally owns no transport policy, retry, cache, or persistence.
package providercredentials

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokercredentials"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

const (
	maxResponseBytes = 128 * 1024
	maxTokenBytes    = 4096
	maxTokenAge      = 65 * time.Minute
	githubHost       = "api.github.com"
	githubAPIVersion = "2026-03-10"
)

// Error values are deliberately fixed. No request, response, credential,
// URL, parser, or transport detail is exposed to callers.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrResolve      Error = "credential-resolve-failed"
)

// Rules supplies already validated broker credentials and a trusted,
// injected exchange boundary. Timeout is applied to each individual request.
type Rules struct {
	Bundle    *brokercredentials.Bundle
	Transport http.RoundTripper
	Timeout   time.Duration
}

// Resolver implements egresstransaction.CredentialResolver.
type Resolver struct {
	bundle    *brokercredentials.Bundle
	transport http.RoundTripper
	timeout   time.Duration
	now       func() time.Time
}

// Format keeps transport internals and broker secrets out of every ordinary
// diagnostic formatting verb and flag.
func (r Resolver) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "providercredentials.Resolver")
}

var _ egresstransaction.CredentialResolver = (*Resolver)(nil)

// New constructs a production resolver. The resolver never creates or wraps
// a default transport; callers must inject the trusted transport boundary.
func New(r Rules) (*Resolver, error) {
	return newWithClock(r, time.Now)
}

// newWithClock is a package-private deterministic seam for expiry tests.
func newWithClock(r Rules, now func() time.Time) (*Resolver, error) {
	if r.Bundle == nil || r.Transport == nil || r.Timeout < time.Millisecond || r.Timeout > 30*time.Second || now == nil {
		return nil, ErrInvalidRules
	}
	return &Resolver{bundle: r.Bundle, transport: r.Transport, timeout: r.Timeout, now: now}, nil
}

// Resolve returns the broker OpenAI key without network access, or performs a
// single repository-scoped GitHub installation-token exchange.
func (r *Resolver) Resolve(provider, repository string) (credential string, err error) {
	if r == nil || r.bundle == nil || r.transport == nil || r.timeout < time.Millisecond || r.timeout > 30*time.Second || r.now == nil {
		return "", ErrResolve
	}
	switch provider {
	case "openai":
		if repository != "" {
			return "", ErrResolve
		}
		key := r.bundle.OpenAIAPIKey()
		if !visibleASCII(key, 1, maxTokenBytes) {
			return "", ErrResolve
		}
		return key, nil
	case "github":
		if !validRepository(repository) {
			return "", ErrResolve
		}
		return r.resolveGitHub(repository)
	default:
		return "", ErrResolve
	}
}

func (r *Resolver) resolveGitHub(repository string) (credential string, err error) {
	jwt, jwtErr := r.bundle.GitHubAppJWT()
	if jwtErr != nil || !visibleASCII(jwt, 1, maxTokenBytes) {
		return "", ErrResolve
	}
	parts := strings.Split(repository, "/")
	body, marshalErr := json.Marshal(struct {
		Repositories []string `json:"repositories"`
	}{Repositories: []string{parts[1]}})
	if marshalErr != nil {
		return "", ErrResolve
	}
	path := "/app/installations/" + strconv.FormatInt(r.bundle.InstallationID(), 10) + "/access_tokens"
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+githubHost+path, bytes.NewReader(body))
	if requestErr != nil {
		return "", ErrResolve
	}
	request.Header.Set("Authorization", "Bearer "+jwt)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	response, roundTripErr := r.transport.RoundTrip(request)
	if response != nil && response.Body != nil {
		defer func() {
			if response.Body.Close() != nil {
				credential, err = "", ErrResolve
			}
		}()
	}
	if roundTripErr != nil || response == nil || response.Body == nil {
		return "", ErrResolve
	}
	if response.StatusCode != http.StatusCreated || !jsonContentType(response.Header.Get("Content-Type")) {
		return "", ErrResolve
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if readErr != nil || len(data) > maxResponseBytes {
		return "", ErrResolve
	}
	token, expiresAt, ok := parseTokenResponse(data)
	if !ok {
		return "", ErrResolve
	}
	now := r.now().UTC()
	if !expiresAt.After(now) || expiresAt.After(now.Add(maxTokenAge)) {
		return "", ErrResolve
	}
	return token, nil
}

func jsonContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(mediaType, "application/json")
}

func parseTokenResponse(data []byte) (token string, expiresAt time.Time, ok bool) {
	if !utf8.Valid(data) {
		return "", time.Time{}, false
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	first, err := decoder.Token()
	if err != nil {
		return "", time.Time{}, false
	}
	delim, isDelim := first.(json.Delim)
	if !isDelim || delim != '{' {
		return "", time.Time{}, false
	}
	seen := make(map[string]struct{})
	var tokenText, expiresText string
	var hasToken, hasExpires bool
	for decoder.More() {
		field, fieldErr := decoder.Token()
		name, isString := field.(string)
		if fieldErr != nil || !isString {
			return "", time.Time{}, false
		}
		if _, duplicate := seen[name]; duplicate {
			return "", time.Time{}, false
		}
		seen[name] = struct{}{}
		var raw json.RawMessage
		if decoder.Decode(&raw) != nil {
			return "", time.Time{}, false
		}
		switch name {
		case "token":
			if !jsonString(raw, &tokenText) {
				return "", time.Time{}, false
			}
			hasToken = true
		case "expires_at":
			if !jsonString(raw, &expiresText) {
				return "", time.Time{}, false
			}
			hasExpires = true
		}
	}
	end, endErr := decoder.Token()
	if endErr != nil || end != json.Delim('}') {
		return "", time.Time{}, false
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return "", time.Time{}, false
	}
	if !hasToken || !hasExpires || !visibleASCII(tokenText, 1, maxTokenBytes) {
		return "", time.Time{}, false
	}
	parsed, parseErr := time.Parse(time.RFC3339, expiresText)
	if parseErr != nil {
		return "", time.Time{}, false
	}
	return tokenText, parsed.UTC(), true
}

func jsonString(raw json.RawMessage, destination *string) bool {
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, destination) != nil {
		return false
	}
	return true
}

func visibleASCII(value string, min, max int) bool {
	if len(value) < min || len(value) > max {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func validRepository(repository string) bool {
	parts := strings.Split(repository, "/")
	return len(parts) == 2 && validRepositorySegment(parts[0]) && validRepositorySegment(parts[1])
}

func validRepositorySegment(segment string) bool {
	if segment == "" || segment == "." || segment == ".." {
		return false
	}
	if (segment[0] < 'a' || segment[0] > 'z') && (segment[0] < '0' || segment[0] > '9') {
		return false
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
