package capabilitycontrol

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	return registry.Consume(capability.Request{
		Handle: handle, AgentInstanceID: subject.AgentInstanceID, UID: subject.UID,
		WorkspaceID: subject.WorkspaceID, Provider: capability.ProviderGitHub,
		Repository: "octo/repo", Operation: capability.OperationGitHubRESTRead,
		DestinationHost: capability.HostGitHub,
	})
}

func githubGitConsume(registry *capability.Registry, handle string, subject egresstransaction.Subject) (capability.Grant, error) {
	return registry.Consume(capability.Request{
		Handle: handle, AgentInstanceID: subject.AgentInstanceID, UID: subject.UID,
		WorkspaceID: subject.WorkspaceID, Provider: capability.ProviderGitHub,
		Repository: "octo/repo", Operation: capability.OperationGitHubGitRead,
		DestinationHost: capability.HostGitHubGit,
	})
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

func TestIssueUsesPeerSubjectAllowlistAndFixedLease(t *testing.T) {
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
	grant, err := githubConsume(registry, handle, controlSubject)
	if err != nil || grant.RemainingUses != 0 || grant.ExpiresAt.Sub(grant.IssuedAt) != issueTTL {
		t.Fatalf("grant=%+v err=%v", grant, err)
	}
	if _, err := githubConsume(registry, handle, controlSubject); !errors.Is(err, capability.ErrDenied) {
		t.Fatal("fixed single-use capability was reusable")
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
}

func TestIssuedScopeMismatchDoesNotSpend(t *testing.T) {
	registry := testRegistry(t)
	controller := testController(t, registry, controlSubject, []string{"model-a"})
	for _, mutate := range []func(*capability.Request){
		func(request *capability.Request) { request.AgentInstanceID = "other-agent" },
		func(request *capability.Request) { request.UID++ },
		func(request *capability.Request) { request.WorkspaceID = "other-workspace" },
		func(request *capability.Request) { request.Provider = capability.ProviderOpenAI },
		func(request *capability.Request) { request.Repository = "other/repo" },
		func(request *capability.Request) { request.Operation = capability.OperationOpenAIResponsesText },
		func(request *capability.Request) { request.DestinationHost = capability.HostOpenAI },
	} {
		handle, err := controller.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
		if err != nil {
			t.Fatal(err)
		}
		request := capability.Request{
			Handle: handle, AgentInstanceID: controlSubject.AgentInstanceID, UID: controlSubject.UID,
			WorkspaceID: controlSubject.WorkspaceID, Provider: capability.ProviderGitHub,
			Repository: "octo/repo", Operation: capability.OperationGitHubRESTRead,
			DestinationHost: capability.HostGitHub,
		}
		mutate(&request)
		if _, err := registry.Consume(request); !errors.Is(err, capability.ErrDenied) {
			t.Fatalf("mismatch err=%v", err)
		}
		if _, err := githubConsume(registry, handle, controlSubject); err != nil {
			t.Fatalf("mismatch spent capability: %v", err)
		}
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
	if err := owner.Revoke(context.Background(), expired); !errors.Is(err, ErrDenied) {
		t.Fatalf("expired revoke err=%v", err)
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
