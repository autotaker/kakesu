package capabilitycontrol

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

var controlSubject = egresstransaction.Subject{AgentInstanceID: "agent-1", UID: 2001, WorkspaceID: "workspace-1"}

func testRegistry(t *testing.T) *capability.Registry {
	t.Helper()
	registry, err := capability.New(capability.Rules{
		PolicyVersion: "egress-v1", MaxTTL: 10 * time.Minute,
		MaxUses: 16, InitialRevocationEpoch: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func resolverFor(subject egresstransaction.Subject) SubjectResolverFunc {
	return func(context.Context) (egresstransaction.Subject, error) { return subject, nil }
}

func testController(t *testing.T, registry *capability.Registry, subject egresstransaction.Subject, models []string) *Controller {
	t.Helper()
	controller, err := New(Rules{
		Registry: registry, Resolver: resolverFor(subject),
		GitHubRepositories: []string{"octo/repo"}, OpenAIModels: models,
	})
	if err != nil {
		t.Fatal(err)
	}
	return controller
}

func githubConsume(registry *capability.Registry, handle string, subject egresstransaction.Subject) (capability.Grant, error) {
	return registry.Consume(githubRequest(handle, subject))
}

func githubRequest(handle string, subject egresstransaction.Subject) capability.Request {
	return capability.Request{
		Handle: handle, AgentInstanceID: subject.AgentInstanceID, UID: subject.UID,
		WorkspaceID: subject.WorkspaceID, Provider: capability.ProviderGitHub,
		Repository: "octo/repo", Operation: capability.OperationGitHubRESTRead,
		DestinationHost: capability.HostGitHub,
	}
}

func githubGitConsume(registry *capability.Registry, handle string, subject egresstransaction.Subject) (capability.Grant, error) {
	return registry.Consume(githubGitRequest(handle, subject))
}

func githubGitRequest(handle string, subject egresstransaction.Subject) capability.Request {
	return capability.Request{
		Handle: handle, AgentInstanceID: subject.AgentInstanceID, UID: subject.UID,
		WorkspaceID: subject.WorkspaceID, Provider: capability.ProviderGitHub,
		Repository: "octo/repo", Operation: capability.OperationGitHubGitRead,
		DestinationHost: capability.HostGitHubGit,
	}
}

func openAIConsume(registry *capability.Registry, handle string, subject egresstransaction.Subject) (capability.Grant, error) {
	return registry.Consume(openAIRequest(handle, subject))
}

func openAIRequest(handle string, subject egresstransaction.Subject) capability.Request {
	return capability.Request{
		Handle: handle, AgentInstanceID: subject.AgentInstanceID, UID: subject.UID,
		WorkspaceID: subject.WorkspaceID, Provider: capability.ProviderOpenAI,
		Operation: capability.OperationOpenAIResponsesText, DestinationHost: capability.HostOpenAI,
	}
}

func TestGitReadSelectorIssuesOnlyGitReadScope(t *testing.T) {
	registry := testRegistry(t)
	controller := testController(t, registry, controlSubject, []string{"model-a"})
	for _, operations := range [][]string{{"git-read"}, {"github-write"}, {capability.OperationGitHubGitRead, "extra"}} {
		if handle, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo", operations...); handle != "" || !errors.Is(err, ErrDenied) {
			t.Fatalf("operations %q=(%q,%v)", operations, handle, err)
		}
	}
	if handle, err := controller.Issue(context.Background(), capability.ProviderOpenAI, "", capability.OperationGitHubGitRead); handle != "" || !errors.Is(err, ErrDenied) {
		t.Fatalf("OpenAI Git selector=(%q,%v)", handle, err)
	}

	handle, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo", capability.OperationGitHubGitRead)
	if err != nil || !strings.HasPrefix(handle, "cap_") {
		t.Fatalf("Git issue=(%q,%v)", handle, err)
	}
	if _, err := githubConsume(registry, handle, controlSubject); !errors.Is(err, capability.ErrDenied) {
		t.Fatalf("Git grant crossed into REST: %v", err)
	}
	wrongSubject := controlSubject
	wrongSubject.AgentInstanceID = "other-agent"
	if _, err := githubGitConsume(registry, handle, wrongSubject); !errors.Is(err, capability.ErrDenied) {
		t.Fatalf("Git grant crossed subject: %v", err)
	}
	grant, err := githubGitConsume(registry, handle, controlSubject)
	if err != nil || grant.Operation != capability.OperationGitHubGitRead || grant.DestinationHost != capability.HostGitHubGit ||
		grant.ExpiresAt.Sub(grant.IssuedAt) != issueTTL || grant.RemainingUses != 0 {
		t.Fatalf("Git grant=(%+v,%v)", grant, err)
	}
	if _, err := githubGitConsume(registry, handle, controlSubject); !errors.Is(err, capability.ErrDenied) {
		t.Fatal("Git grant was not single use")
	}
}

func TestGitReadScopeMismatchDoesNotSpendItsSingleUse(t *testing.T) {
	registry := testRegistry(t)
	controller := testController(t, registry, controlSubject, []string{"model-a"})
	for _, tc := range []struct {
		name   string
		mutate func(*capability.Request)
	}{
		{name: "agent instance", mutate: func(request *capability.Request) { request.AgentInstanceID = "other-agent" }},
		{name: "UID", mutate: func(request *capability.Request) { request.UID++ }},
		{name: "workspace", mutate: func(request *capability.Request) { request.WorkspaceID = "other-workspace" }},
		{name: "REST scope", mutate: func(request *capability.Request) {
			request.Operation = capability.OperationGitHubRESTRead
			request.DestinationHost = capability.HostGitHub
		}},
		{name: "OpenAI scope", mutate: func(request *capability.Request) {
			request.Provider = capability.ProviderOpenAI
			request.Repository = ""
			request.Operation = capability.OperationOpenAIResponsesText
			request.DestinationHost = capability.HostOpenAI
		}},
		{name: "repository", mutate: func(request *capability.Request) { request.Repository = "other/repo" }},
		{name: "push", mutate: func(request *capability.Request) { request.Operation = "github-git-push" }},
		{name: "write", mutate: func(request *capability.Request) { request.Operation = "github-write" }},
		{name: "API host", mutate: func(request *capability.Request) { request.DestinationHost = capability.HostGitHub }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handle, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo", capability.OperationGitHubGitRead)
			if err != nil {
				t.Fatal(err)
			}
			request := githubGitRequest(handle, controlSubject)
			tc.mutate(&request)
			if grant, consumeErr := registry.Consume(request); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
				t.Fatalf("mismatch consume=(%+v,%v)", grant, consumeErr)
			}
			grant, consumeErr := githubGitConsume(registry, handle, controlSubject)
			if consumeErr != nil || grant.RemainingUses != 0 || grant.Operation != capability.OperationGitHubGitRead ||
				grant.DestinationHost != capability.HostGitHubGit {
				t.Fatalf("single valid consume=(%+v,%v)", grant, consumeErr)
			}
			if grant, consumeErr = githubGitConsume(registry, handle, controlSubject); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
				t.Fatalf("second consume=(%+v,%v)", grant, consumeErr)
			}
		})
	}
}

func TestIssueUsesPeerSubjectAllowlistAndFixedAPIBudget(t *testing.T) {
	registry := testRegistry(t)
	controller := testController(t, registry, controlSubject, []string{"model-a"})

	for _, request := range []struct{ provider, repository string }{
		{"github", "other/repo"}, {"github", "Octo/repo"},
		{"github", "octo/repo/extra"}, {"openai", "octo/repo"},
		{"other", ""},
	} {
		if handle, err := controller.Issue(context.Background(), request.provider, request.repository); handle != "" || !errors.Is(err, ErrDenied) {
			t.Fatalf("denied issue returned handle=%q err=%v", handle, err)
		}
	}

	handle, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
	if err != nil || !strings.HasPrefix(handle, "cap_") || len(handle) != 47 {
		t.Fatalf("issue=(%q,%v)", handle, err)
	}
	wrong := controlSubject
	wrong.WorkspaceID = "caller-supplied"
	if _, err := githubConsume(registry, handle, wrong); !errors.Is(err, capability.ErrDenied) {
		t.Fatalf("caller subject mismatch err=%v", err)
	}
	for use := 1; use <= apiUses; use++ {
		grant, consumeErr := githubConsume(registry, handle, controlSubject)
		if consumeErr != nil || grant.RemainingUses != apiUses-use || grant.ExpiresAt.Sub(grant.IssuedAt) != issueTTL {
			t.Fatalf("use %d grant=%+v err=%v", use, grant, consumeErr)
		}
	}
	if _, err := githubConsume(registry, handle, controlSubject); !errors.Is(err, capability.ErrDenied) {
		t.Fatal("seventeenth GitHub REST use succeeded")
	}
}

func TestOpenAIGateDoesNotAcceptModelScope(t *testing.T) {
	registry := testRegistry(t)
	disabled := testController(t, registry, controlSubject, nil)
	if handle, err := disabled.Issue(context.Background(), capability.ProviderOpenAI, ""); handle != "" || !errors.Is(err, ErrDenied) {
		t.Fatalf("empty model gate=(%q,%v)", handle, err)
	}

	enabled := testController(t, registry, controlSubject, []string{"model-value-is-only-a-gate"})
	handle, err := enabled.Issue(context.Background(), capability.ProviderOpenAI, "")
	if err != nil {
		t.Fatal(err)
	}
	grant, err := registry.Consume(capability.Request{
		Handle: handle, AgentInstanceID: controlSubject.AgentInstanceID, UID: controlSubject.UID,
		WorkspaceID: controlSubject.WorkspaceID, Provider: capability.ProviderOpenAI,
		Operation: capability.OperationOpenAIResponsesText, DestinationHost: capability.HostOpenAI,
	})
	if err != nil || grant.Repository != "" || grant.Provider != capability.ProviderOpenAI {
		t.Fatalf("openai grant=%+v err=%v", grant, err)
	}
	if grant.RemainingUses != apiUses-1 || grant.ExpiresAt.Sub(grant.IssuedAt) != issueTTL {
		t.Fatalf("OpenAI lease=%+v", grant)
	}
}

func TestAPIScopeSelectorsHaveFixedSixteenUseBudget(t *testing.T) {
	for _, tc := range []struct {
		name       string
		provider   string
		repository string
		operations []string
		consume    func(*capability.Registry, string, egresstransaction.Subject) (capability.Grant, error)
		wantOp     string
		wantHost   string
	}{
		{
			name: "GitHub default", provider: capability.ProviderGitHub, repository: "octo/repo",
			consume: githubConsume, wantOp: capability.OperationGitHubRESTRead, wantHost: capability.HostGitHub,
		},
		{
			name: "GitHub explicit REST", provider: capability.ProviderGitHub, repository: "octo/repo",
			operations: []string{capability.OperationGitHubRESTRead}, consume: githubConsume,
			wantOp: capability.OperationGitHubRESTRead, wantHost: capability.HostGitHub,
		},
		{
			name: "OpenAI default", provider: capability.ProviderOpenAI, consume: openAIConsume,
			wantOp: capability.OperationOpenAIResponsesText, wantHost: capability.HostOpenAI,
		},
		{
			name: "OpenAI explicit Responses", provider: capability.ProviderOpenAI,
			operations: []string{capability.OperationOpenAIResponsesText}, consume: openAIConsume,
			wantOp: capability.OperationOpenAIResponsesText, wantHost: capability.HostOpenAI,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := testRegistry(t)
			controller := testController(t, registry, controlSubject, []string{"model-a"})
			handle, err := controller.Issue(context.Background(), tc.provider, tc.repository, tc.operations...)
			if err != nil || !strings.HasPrefix(handle, "cap_") {
				t.Fatalf("issue=(%q,%v)", handle, err)
			}
			for use := 1; use <= apiUses; use++ {
				grant, consumeErr := tc.consume(registry, handle, controlSubject)
				if consumeErr != nil || grant.RemainingUses != apiUses-use || grant.Operation != tc.wantOp ||
					grant.DestinationHost != tc.wantHost || grant.ExpiresAt.Sub(grant.IssuedAt) != issueTTL {
					t.Fatalf("use %d grant=%+v err=%v", use, grant, consumeErr)
				}
			}
			if grant, consumeErr := tc.consume(registry, handle, controlSubject); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
				t.Fatalf("seventeenth consume=(%+v,%v)", grant, consumeErr)
			}
		})
	}
}

func TestUseBudgetPolicyFailsClosedForUnknownOperation(t *testing.T) {
	for operation, want := range map[string]int{
		"":                                      apiUses,
		capability.OperationGitHubRESTRead:      apiUses,
		capability.OperationOpenAIResponsesText: apiUses,
		capability.OperationGitHubGitRead:       gitReadUses,
		"github-write":                          0,
		"github-graphql":                        0,
		"openai-files-upload":                   0,
	} {
		if got := usesForOperation(operation); got != want {
			t.Fatalf("usesForOperation(%q)=%d want=%d", operation, got, want)
		}
	}
}

func TestAPIUseBudgetIsConsumedAtomically(t *testing.T) {
	for _, tc := range []struct {
		name       string
		provider   string
		repository string
		consume    func(*capability.Registry, string, egresstransaction.Subject) (capability.Grant, error)
	}{
		{name: "GitHub REST", provider: capability.ProviderGitHub, repository: "octo/repo", consume: githubConsume},
		{name: "OpenAI Responses", provider: capability.ProviderOpenAI, consume: openAIConsume},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := testRegistry(t)
			controller := testController(t, registry, controlSubject, []string{"model-a"})
			handle, err := controller.Issue(context.Background(), tc.provider, tc.repository)
			if err != nil {
				t.Fatal(err)
			}
			const attempts = apiUses * 2
			results := make(chan int, attempts)
			var workers sync.WaitGroup
			workers.Add(attempts)
			for attempt := 0; attempt < attempts; attempt++ {
				go func() {
					defer workers.Done()
					grant, consumeErr := tc.consume(registry, handle, controlSubject)
					if consumeErr == nil {
						results <- grant.RemainingUses
						return
					}
					if !errors.Is(consumeErr, capability.ErrDenied) {
						results <- -2
						return
					}
					results <- -1
				}()
			}
			workers.Wait()
			close(results)

			var remaining []int
			denied := 0
			for result := range results {
				if result >= 0 {
					remaining = append(remaining, result)
				} else if result == -1 {
					denied++
				} else {
					t.Fatal("consume returned a non-fixed error")
				}
			}
			sort.Ints(remaining)
			if len(remaining) != apiUses || denied != attempts-apiUses {
				t.Fatalf("success=%d denied=%d remaining=%v", len(remaining), denied, remaining)
			}
			for index, value := range remaining {
				if value != index {
					t.Fatalf("remaining sequence=%v", remaining)
				}
			}
		})
	}
}

func TestGitHubAPIScopeMismatchAndCrossScopeDoNotSpend(t *testing.T) {
	registry := testRegistry(t)
	controller := testController(t, registry, controlSubject, []string{"model-a"})
	for _, tc := range []struct {
		name   string
		mutate func(*capability.Request)
	}{
		{name: "agent instance", mutate: func(request *capability.Request) { request.AgentInstanceID = "other-agent" }},
		{name: "UID", mutate: func(request *capability.Request) { request.UID++ }},
		{name: "workspace", mutate: func(request *capability.Request) { request.WorkspaceID = "other-workspace" }},
		{name: "provider", mutate: func(request *capability.Request) { request.Provider = capability.ProviderOpenAI }},
		{name: "repository", mutate: func(request *capability.Request) { request.Repository = "other/repo" }},
		{name: "Git read operation", mutate: func(request *capability.Request) {
			request.Operation = capability.OperationGitHubGitRead
			request.DestinationHost = capability.HostGitHubGit
		}},
		{name: "write operation", mutate: func(request *capability.Request) { request.Operation = "github-write" }},
		{name: "GraphQL operation", mutate: func(request *capability.Request) { request.Operation = "github-graphql" }},
		{name: "OpenAI operation", mutate: func(request *capability.Request) { request.Operation = capability.OperationOpenAIResponsesText }},
		{name: "GitHub Git host", mutate: func(request *capability.Request) { request.DestinationHost = capability.HostGitHubGit }},
		{name: "OpenAI host", mutate: func(request *capability.Request) { request.DestinationHost = capability.HostOpenAI }},
		{name: "other host", mutate: func(request *capability.Request) { request.DestinationHost = "uploads.github.com" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handle, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
			if err != nil {
				t.Fatal(err)
			}
			request := githubRequest(handle, controlSubject)
			tc.mutate(&request)
			if grant, consumeErr := registry.Consume(request); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
				t.Fatalf("mismatch consume=(%+v,%v)", grant, consumeErr)
			}
			for use := 1; use <= apiUses; use++ {
				grant, consumeErr := githubConsume(registry, handle, controlSubject)
				if consumeErr != nil || grant.RemainingUses != apiUses-use {
					t.Fatalf("mismatch spent budget at use %d: grant=%+v err=%v", use, grant, consumeErr)
				}
			}
		})
	}
}

func TestOpenAIAPIScopeMismatchAndCrossScopeDoNotSpend(t *testing.T) {
	registry := testRegistry(t)
	controller := testController(t, registry, controlSubject, []string{"model-a"})
	for _, tc := range []struct {
		name   string
		mutate func(*capability.Request)
	}{
		{name: "agent instance", mutate: func(request *capability.Request) { request.AgentInstanceID = "other-agent" }},
		{name: "UID", mutate: func(request *capability.Request) { request.UID++ }},
		{name: "workspace", mutate: func(request *capability.Request) { request.WorkspaceID = "other-workspace" }},
		{name: "provider", mutate: func(request *capability.Request) { request.Provider = capability.ProviderGitHub }},
		{name: "repository", mutate: func(request *capability.Request) { request.Repository = "octo/repo" }},
		{name: "GitHub REST operation", mutate: func(request *capability.Request) { request.Operation = capability.OperationGitHubRESTRead }},
		{name: "GitHub Git operation", mutate: func(request *capability.Request) { request.Operation = capability.OperationGitHubGitRead }},
		{name: "admin operation", mutate: func(request *capability.Request) { request.Operation = "openai-admin" }},
		{name: "files operation", mutate: func(request *capability.Request) { request.Operation = "openai-files-upload" }},
		{name: "GitHub API host", mutate: func(request *capability.Request) { request.DestinationHost = capability.HostGitHub }},
		{name: "upload host", mutate: func(request *capability.Request) { request.DestinationHost = "uploads.openai.com" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handle, err := controller.Issue(context.Background(), capability.ProviderOpenAI, "")
			if err != nil {
				t.Fatal(err)
			}
			request := openAIRequest(handle, controlSubject)
			tc.mutate(&request)
			if grant, consumeErr := registry.Consume(request); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
				t.Fatalf("mismatch consume=(%+v,%v)", grant, consumeErr)
			}
			for use := 1; use <= apiUses; use++ {
				grant, consumeErr := openAIConsume(registry, handle, controlSubject)
				if consumeErr != nil || grant.RemainingUses != apiUses-use {
					t.Fatalf("mismatch spent budget at use %d: grant=%+v err=%v", use, grant, consumeErr)
				}
			}
		})
	}
}

func TestSubjectBoundRevokeAndExpiredUnknownDenials(t *testing.T) {
	registry := testRegistry(t)
	owner := testController(t, registry, controlSubject, []string{"model-a"})
	handle, err := owner.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
	if err != nil {
		t.Fatal(err)
	}
	for _, otherSubject := range []egresstransaction.Subject{
		{AgentInstanceID: "other-agent", UID: controlSubject.UID, WorkspaceID: controlSubject.WorkspaceID},
		{AgentInstanceID: controlSubject.AgentInstanceID, UID: controlSubject.UID + 1, WorkspaceID: controlSubject.WorkspaceID},
		{AgentInstanceID: controlSubject.AgentInstanceID, UID: controlSubject.UID, WorkspaceID: "other-workspace"},
	} {
		other := testController(t, registry, otherSubject, []string{"model-a"})
		if err := other.Revoke(context.Background(), handle); !errors.Is(err, ErrDenied) {
			t.Fatalf("wrong subject revoke err=%v", err)
		}
	}
	if err := owner.Revoke(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	if _, err := githubConsume(registry, handle, controlSubject); !errors.Is(err, capability.ErrDenied) {
		t.Fatal("revoked capability consumed")
	}
	for _, bad := range []string{"cap_bad", "cap_" + strings.Repeat("A", 43)} {
		if err := owner.Revoke(context.Background(), bad); !errors.Is(err, ErrDenied) {
			t.Fatalf("bad revoke err=%v", err)
		}
	}

	expired, err := registry.Issue(capability.IssueSpec{
		AgentInstanceID: controlSubject.AgentInstanceID, UID: controlSubject.UID,
		WorkspaceID: controlSubject.WorkspaceID, Provider: capability.ProviderGitHub,
		Repository: "octo/repo", TTL: time.Nanosecond, Uses: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if grant, consumeErr := githubConsume(registry, expired, controlSubject); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
		t.Fatalf("expired consume=(%+v,%v)", grant, consumeErr)
	}
	if err := owner.Revoke(context.Background(), expired); !errors.Is(err, ErrDenied) {
		t.Fatalf("expired revoke err=%v", err)
	}
}

func TestAPICapabilityRevokeAndEpochRemainEffective(t *testing.T) {
	registry := testRegistry(t)
	controller := testController(t, registry, controlSubject, []string{"model-a"})

	revoked, err := controller.Issue(context.Background(), capability.ProviderOpenAI, "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := openAIConsume(registry, revoked, controlSubject)
	if err != nil || first.RemainingUses != apiUses-1 {
		t.Fatalf("first consume=(%+v,%v)", first, err)
	}
	if err := controller.Revoke(context.Background(), revoked); err != nil {
		t.Fatal(err)
	}
	if grant, consumeErr := openAIConsume(registry, revoked, controlSubject); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
		t.Fatalf("revoked consume=(%+v,%v)", grant, consumeErr)
	}

	invalidated, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.AdvanceRevocationEpoch(2); err != nil {
		t.Fatal(err)
	}
	if grant, consumeErr := githubConsume(registry, invalidated, controlSubject); !errors.Is(consumeErr, capability.ErrDenied) || grant != (capability.Grant{}) {
		t.Fatalf("old epoch consume=(%+v,%v)", grant, consumeErr)
	}
	current, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
	if err != nil {
		t.Fatal(err)
	}
	grant, err := githubConsume(registry, current, controlSubject)
	if err != nil || grant.RevocationEpoch != 2 || grant.RemainingUses != apiUses-1 {
		t.Fatalf("current epoch consume=(%+v,%v)", grant, err)
	}
}

func TestContextFailuresAndDiagnosticsAreFixed(t *testing.T) {
	registry := testRegistry(t)
	for name, resolver := range map[string]SubjectResolver{
		"error": SubjectResolverFunc(func(context.Context) (egresstransaction.Subject, error) {
			return egresstransaction.Subject{}, errors.New("secret lower detail")
		}),
		"corrupt": resolverFor(egresstransaction.Subject{AgentInstanceID: "bad value", UID: 0, WorkspaceID: "secret"}),
	} {
		t.Run(name, func(t *testing.T) {
			controller, err := New(Rules{Registry: registry, Resolver: resolver, GitHubRepositories: []string{"octo/repo"}})
			if err != nil {
				t.Fatal(err)
			}
			handle, issueErr := controller.Issue(context.Background(), "github", "octo/repo")
			if handle != "" || issueErr != ErrDenied || issueErr.Error() != "capability-control-denied" {
				t.Fatalf("issue=(%q,%v)", handle, issueErr)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	controller := testController(t, registry, controlSubject, nil)
	if handle, err := controller.Issue(ctx, "github", "octo/repo"); handle != "" || err != ErrDenied {
		t.Fatalf("cancelled issue=(%q,%v)", handle, err)
	}
	if got := fmt.Sprintf("%+v", *controller); got != "capabilitycontrol.Controller" {
		t.Fatalf("format=%q", got)
	}
}

func TestRulesRejectInvalidDependenciesAndRepositories(t *testing.T) {
	registry := testRegistry(t)
	var nilResolver *testNilResolver
	for _, rules := range []Rules{
		{}, {Registry: registry}, {Registry: registry, Resolver: nilResolver},
		{Registry: registry, Resolver: resolverFor(controlSubject), GitHubRepositories: []string{"Octo/repo"}},
		{Registry: registry, Resolver: resolverFor(controlSubject), GitHubRepositories: []string{"octo/repo", "octo/repo"}},
	} {
		if controller, err := New(rules); controller != nil || err != ErrInvalidRules {
			t.Fatalf("New(%+v)=(%p,%v)", rules, controller, err)
		}
	}
}

type testNilResolver struct{}

func (*testNilResolver) Resolve(context.Context) (egresstransaction.Subject, error) {
	return egresstransaction.Subject{}, nil
}
