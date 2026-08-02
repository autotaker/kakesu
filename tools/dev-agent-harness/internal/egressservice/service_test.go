package egressservice

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokercredentials"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerexchange"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerhttp"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerlistener"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capability"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/capabilitycontrol"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/config"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/connectsession"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/peerbinder"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/providercredentials"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/socketactivation"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/upstreamtransport"
)

type testListener struct{}

type fakeAuthority struct{}

func (fakeAuthority) Issue(string) (tls.Certificate, error) {
	return tls.Certificate{}, errors.New("not used")
}
func (fakeAuthority) PublicCertificatePEM() []byte { return []byte("not-used") }

func (*testListener) Accept() (net.Conn, error) { return nil, errors.New("not used") }
func (*testListener) Close() error              { return nil }
func (*testListener) Addr() net.Addr            { return testAddr("test") }

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

func serviceTestConfig() *config.Config {
	return &config.Config{
		Version:  1,
		Paths:    config.Paths{ConfigDir: "/etc/dev-agent", StateDir: "/var/lib/dev-agent", RuntimeDir: "/run/dev-agent"},
		Users:    config.Users{Agent: "dev-agent", Runtime: "dev-runtime", Broker: "dev-broker"},
		Identity: config.Identity{WorkspaceID: "workspace-1"}, Network: config.Network{Default: "deny"},
		Egress: config.Egress{GitHubRepositories: []string{"octo/repo"}, OpenAIModels: []string{"gpt-4o-mini"}},
	}
}

func serviceTestIdentity() identitySnapshot {
	subject := brokerlistener.Subject{AgentInstanceID: "agent-instance", UID: 2001, WorkspaceID: "workspace-1"}
	return identitySnapshot{brokerUID: 2000, agentUID: 2001, agentGID: 2002, agentInstance: subject.AgentInstanceID, workspace: subject.WorkspaceID, subject: subject}
}

func TestServeWithFactoryExactOrderAndIdentityWiring(t *testing.T) {
	var events []string
	var gotPath string
	wantIdentity := serviceTestIdentity()
	wantConfig := serviceTestConfig()
	factory := serviceFactory{
		loadConfig: func(path string) (*config.Config, error) {
			events = append(events, "config")
			gotPath = path
			return wantConfig, nil
		},
		resolveIdentity: func(c *config.Config) (identitySnapshot, error) {
			events = append(events, "identity")
			if c != wantConfig {
				t.Fatal("config was replaced")
			}
			return wantIdentity, nil
		},
		loadCredentials: func(path string) (credentialSnapshot, error) {
			events = append(events, "credentials")
			gotPath = path
			return credentialSnapshot{bundle: &brokercredentials.Bundle{}, authority: fakeAuthority{}}, nil
		},
		buildGraph: func(input graphInput) (runtimeGraph, error) {
			events = append(events, "graph")
			if input.identity != wantIdentity || input.config != wantConfig {
				t.Fatal("graph did not receive the same snapshots")
			}
			return runtimeGraph{
				take: func() (net.Listener, error) { events = append(events, "take"); return &testListener{}, nil },
				serve: func(ctx context.Context, listener net.Listener) error {
					events = append(events, "serve")
					if listener == nil || ctx == nil {
						t.Fatal("invalid serve inputs")
					}
					return nil
				},
			}, nil
		},
	}
	if err := serveWithFactory(context.Background(), "/config/path", factory); err != nil {
		t.Fatalf("serveWithFactory: %v", err)
	}
	if gotPath != filepath.Join(wantConfig.Paths.ConfigDir, "credentials") {
		t.Fatalf("credential path=%q", gotPath)
	}
	if strings.Join(events, ",") != "config,identity,credentials,graph,take,serve" {
		t.Fatalf("events=%v", events)
	}
}

func TestServeWithFactoryStopsAfterEveryFailure(t *testing.T) {
	stages := []string{"config", "identity", "credentials", "graph", "take", "serve"}
	for _, failed := range stages {
		t.Run(failed, func(t *testing.T) {
			var events []string
			factory := serviceFactory{
				loadConfig: func(string) (*config.Config, error) {
					events = append(events, "config")
					if failed == "config" {
						return nil, errors.New("secret config detail")
					}
					return serviceTestConfig(), nil
				},
				resolveIdentity: func(*config.Config) (identitySnapshot, error) {
					events = append(events, "identity")
					if failed == "identity" {
						return identitySnapshot{}, errors.New("identity detail")
					}
					return serviceTestIdentity(), nil
				},
				loadCredentials: func(string) (credentialSnapshot, error) {
					events = append(events, "credentials")
					if failed == "credentials" {
						return credentialSnapshot{}, errors.New("credential detail")
					}
					return credentialSnapshot{bundle: &brokercredentials.Bundle{}, authority: fakeAuthority{}}, nil
				},
				buildGraph: func(graphInput) (runtimeGraph, error) {
					events = append(events, "graph")
					if failed == "graph" {
						return runtimeGraph{}, errors.New("constructor detail")
					}
					return runtimeGraph{
						take: func() (net.Listener, error) {
							events = append(events, "take")
							if failed == "take" {
								return nil, errors.New("fd detail")
							}
							return &testListener{}, nil
						},
						serve: func(context.Context, net.Listener) error {
							events = append(events, "serve")
							if failed == "serve" {
								return errors.New("serve detail")
							}
							return nil
						},
					}, nil
				},
			}
			if err := serveWithFactory(context.Background(), "/config/path", factory); err != ErrStart {
				t.Fatalf("error=%v, want fixed ErrStart", err)
			}
			for _, later := range stages {
				if later == failed {
					break
				}
				if !containsEvent(events, later) {
					t.Fatalf("events=%v missing %s", events, later)
				}
			}
			for _, later := range stages {
				if later == failed {
					continue
				}
				if stageIndex(events, later) > stageIndex(events, failed) {
					t.Fatalf("failure %s reached later stage %s: %v", failed, later, events)
				}
			}
			if strings.Contains(errString(ErrStart), "detail") {
				t.Fatal("fixed error leaked cause")
			}
		})
	}
}

func TestProductionGraphConstructorRulesAndIdentityWiring(t *testing.T) {
	input := graphInput{config: serviceTestConfig(), identity: serviceTestIdentity(), credentials: credentialSnapshot{bundle: &brokercredentials.Bundle{}, authority: fakeAuthority{}}}
	var policyRules egresspolicy.Rules
	var registryRules capability.Rules
	var controlRules capabilitycontrol.Rules
	var providerRules providercredentials.Rules
	var exchangeRules brokerexchange.Rules
	var handlerRules brokerhttp.Rules
	var sessionRules connectsession.Rules
	var binderRules peerbinder.Rules
	var serverRules brokerlistener.Rules
	var receiverRules socketactivation.Rules
	var events []string
	var policyPtr *egresspolicy.Policy
	var registryPtr *capability.Registry
	var controlPtr *capabilitycontrol.Controller
	var transportPtr *upstreamtransport.Transport
	var credentialsPtr *providercredentials.Resolver
	var exchangePtr *brokerexchange.Exchange
	var handlerPtr *brokerhttp.Handler
	var sessionPtr *connectsession.Session
	var binderPtr *peerbinder.Binder
	constructors := constructorSet{
		policy: func(r egresspolicy.Rules) (*egresspolicy.Policy, error) {
			events = append(events, "policy")
			policyRules = r
			policyPtr, _ = egresspolicy.New(r)
			return policyPtr, nil
		},
		registry: func(r capability.Rules) (*capability.Registry, error) {
			events = append(events, "registry")
			registryRules = r
			registryPtr, _ = capability.New(r)
			return registryPtr, nil
		},
		control: func(r capabilitycontrol.Rules) (*capabilitycontrol.Controller, error) {
			events = append(events, "control")
			controlRules = r
			r.Resolver = capabilitycontrol.SubjectResolverFunc(func(context.Context) (egresstransaction.Subject, error) {
				return input.identity.subject, nil
			})
			controlPtr, _ = capabilitycontrol.New(r)
			return controlPtr, nil
		},
		transport: func() *upstreamtransport.Transport {
			events = append(events, "transport")
			transportPtr = upstreamtransport.New()
			return transportPtr
		},
		credentials: func(r providercredentials.Rules) (*providercredentials.Resolver, error) {
			events = append(events, "credentials")
			providerRules = r
			credentialsPtr = &providercredentials.Resolver{}
			return credentialsPtr, nil
		},
		exchange: func(r brokerexchange.Rules) (*brokerexchange.Exchange, error) {
			events = append(events, "exchange")
			exchangeRules = r
			exchangePtr = &brokerexchange.Exchange{}
			return exchangePtr, nil
		},
		handler: func(r brokerhttp.Rules) (*brokerhttp.Handler, error) {
			events = append(events, "handler")
			handlerRules = r
			handlerPtr = &brokerhttp.Handler{}
			return handlerPtr, nil
		},
		session: func(r connectsession.Rules) (*connectsession.Session, error) {
			events = append(events, "session")
			sessionRules = r
			sessionPtr = &connectsession.Session{}
			return sessionPtr, nil
		},
		binder: func(r peerbinder.Rules) (*peerbinder.Binder, error) {
			events = append(events, "binder")
			binderRules = r
			binderPtr = &peerbinder.Binder{}
			return binderPtr, nil
		},
		server: func(r brokerlistener.Rules) (*brokerlistener.Server, error) {
			events = append(events, "server")
			serverRules = r
			return &brokerlistener.Server{}, nil
		},
		receiver: func(r socketactivation.Rules) (*socketactivation.Receiver, error) {
			events = append(events, "receiver")
			receiverRules = r
			return &socketactivation.Receiver{}, nil
		},
	}
	if _, err := buildProductionGraphWith(input, constructors); err != nil {
		t.Fatalf("constructor seam returned error: %v", err)
	}
	if policyRules.MaxBodyBytes != 64*1024 || policyRules.MaxOutputTokens != 4096 || len(policyRules.GitHubRepositories) != 1 || len(policyRules.OpenAIModels) != 1 {
		t.Fatalf("policy rules=%+v", policyRules)
	}
	if registryRules.PolicyVersion != "egress-v1" || registryRules.MaxTTL != 10*time.Minute || registryRules.MaxUses != 16 || registryRules.InitialRevocationEpoch != 1 {
		t.Fatalf("registry rules=%+v", registryRules)
	}
	if _, err := registryPtr.Consume(capability.Request{Handle: "cap_" + strings.Repeat("A", 43), AgentInstanceID: input.identity.agentInstance, UID: input.identity.agentUID, WorkspaceID: input.identity.workspace, Provider: capability.ProviderOpenAI, Operation: capability.OperationOpenAIResponsesText, DestinationHost: capability.HostOpenAI}); err != capability.ErrDenied {
		t.Fatalf("empty registry unknown handle err=%v", err)
	}
	if providerRules.Timeout != 10*time.Second || exchangeRules.MaxCredentialBytes != 4096 || exchangeRules.Timeout != 30*time.Second || exchangeRules.MaxResponseBytes != 1<<20 || handlerRules.MaxBodyBytes != 64*1024 || serverRules.MaxConcurrent != 16 {
		t.Fatalf("fixed limits provider=%+v exchange=%+v handler=%+v server=%+v", providerRules, exchangeRules, handlerRules, serverRules)
	}
	if binderRules.ExpectedUID != input.identity.agentUID || binderRules.Subject != input.identity.subject || receiverRules.BrokerUID != input.identity.brokerUID || receiverRules.AgentGID != input.identity.agentGID || receiverRules.RuntimeDir != input.config.Paths.RuntimeDir {
		t.Fatalf("identity wiring binder=%+v receiver=%+v", binderRules, receiverRules)
	}
	if strings.Join(events, ",") != "policy,registry,control,transport,credentials,exchange,handler,session,binder,server,receiver" || controlRules.Registry != registryPtr || providerRules.Transport != transportPtr || exchangeRules.Policy != policyPtr || exchangeRules.Registry != registryPtr || exchangeRules.Resolver != credentialsPtr || exchangeRules.Transport != transportPtr || handlerRules.Exchange != exchangePtr || sessionRules.Handler != handlerPtr || sessionRules.Control != controlPtr || serverRules.Binder != binderPtr || serverRules.Session != sessionPtr {
		t.Fatalf("constructor order/pointer wiring events=%v provider=%+v exchange=%+v handler=%+v session=%+v server=%+v", events, providerRules, exchangeRules, handlerRules, sessionRules, serverRules)
	}
	if handlerRules.Resolver == nil || sessionRules.Handler == nil || sessionRules.Authority != input.credentials.authority {
		// The resolver is a concrete zero-value value; fake constructor rules
		// must still receive it, and the one authority snapshot is reused.
		t.Fatalf("unexpected handler/session dependency wiring resolver=%#v handler=%#v authority=%#v", handlerRules.Resolver, sessionRules.Handler, sessionRules.Authority)
	}
	if _, ok := controlRules.Resolver.(brokerlistener.Resolver); !ok {
		t.Fatalf("control resolver is not brokerlistener.Resolver: %#v", controlRules.Resolver)
	}
	var resolved, forwarded int
	transaction, err := egresstransaction.New(egresstransaction.Rules{
		Policy: policyPtr, Registry: registryPtr, MaxCredentialBytes: 64,
		Resolver: egresstransaction.CredentialResolverFunc(func(string, string) (string, error) {
			resolved++
			return "upstream-secret", nil
		}),
		Forwarder: egresstransaction.ForwarderFunc(func(egresstransaction.PreparedRequest) error {
			forwarded++
			return nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	handle, err := controlPtr.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
	if err != nil {
		t.Fatal(err)
	}
	request := egresstransaction.Request{Method: "GET", URL: "https://api.github.com/repos/octo/repo", Authorization: []string{"token " + handle}}
	const apiBudget = 16
	for use := 1; use <= apiBudget; use++ {
		if err := transaction.Execute(input.identity.subject, request); err != nil {
			t.Fatalf("shared registry transaction use %d: %v", use, err)
		}
	}
	if err := transaction.Execute(input.identity.subject, request); err != egresstransaction.ErrDenied || resolved != apiBudget || forwarded != apiBudget {
		t.Fatalf("seventeenth use err=%v resolved=%d forwarded=%d", err, resolved, forwarded)
	}
	revoked, err := controlPtr.Issue(context.Background(), capability.ProviderGitHub, "octo/repo")
	revokeErr := controlPtr.Revoke(context.Background(), revoked)
	if err != nil || revokeErr != nil {
		t.Fatalf("issue/revoke=(%v,%v)", err, revokeErr)
	}
	request.Authorization = []string{"token " + revoked}
	if err := transaction.Execute(input.identity.subject, request); err != egresstransaction.ErrDenied || resolved != apiBudget || forwarded != apiBudget {
		t.Fatalf("revoked use err=%v resolved=%d forwarded=%d", err, resolved, forwarded)
	}
}

func TestProductionGraphRejectsNilConstructorResults(t *testing.T) {
	input := graphInput{config: serviceTestConfig(), identity: serviceTestIdentity(), credentials: credentialSnapshot{bundle: &brokercredentials.Bundle{}, authority: fakeAuthority{}}}
	cases := []struct {
		name      string
		zero      func(*constructorSet)
		errorCase func(*constructorSet)
	}{
		{"policy", func(c *constructorSet) {
			c.policy = func(egresspolicy.Rules) (*egresspolicy.Policy, error) { return nil, nil }
		}, func(c *constructorSet) {
			base := c.policy
			c.policy = func(r egresspolicy.Rules) (*egresspolicy.Policy, error) {
				value, _ := base(r)
				return value, errors.New("policy")
			}
		}},
		{"registry", func(c *constructorSet) {
			c.registry = func(capability.Rules) (*capability.Registry, error) { return nil, nil }
		}, func(c *constructorSet) {
			base := c.registry
			c.registry = func(r capability.Rules) (*capability.Registry, error) {
				value, _ := base(r)
				return value, errors.New("registry")
			}
		}},
		{"control", func(c *constructorSet) {
			c.control = func(capabilitycontrol.Rules) (*capabilitycontrol.Controller, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.control = func(capabilitycontrol.Rules) (*capabilitycontrol.Controller, error) {
				return &capabilitycontrol.Controller{}, errors.New("control")
			}
		}},
		{"transport", func(c *constructorSet) { c.transport = func() *upstreamtransport.Transport { return nil } }, nil},
		{"credentials", func(c *constructorSet) {
			c.credentials = func(providercredentials.Rules) (*providercredentials.Resolver, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.credentials = func(r providercredentials.Rules) (*providercredentials.Resolver, error) {
				return &providercredentials.Resolver{}, errors.New("credentials")
			}
		}},
		{"exchange", func(c *constructorSet) {
			c.exchange = func(brokerexchange.Rules) (*brokerexchange.Exchange, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.exchange = func(r brokerexchange.Rules) (*brokerexchange.Exchange, error) {
				return &brokerexchange.Exchange{}, errors.New("exchange")
			}
		}},
		{"handler", func(c *constructorSet) {
			c.handler = func(brokerhttp.Rules) (*brokerhttp.Handler, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.handler = func(r brokerhttp.Rules) (*brokerhttp.Handler, error) {
				return &brokerhttp.Handler{}, errors.New("handler")
			}
		}},
		{"session", func(c *constructorSet) {
			c.session = func(connectsession.Rules) (*connectsession.Session, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.session = func(r connectsession.Rules) (*connectsession.Session, error) {
				return &connectsession.Session{}, errors.New("session")
			}
		}},
		{"binder", func(c *constructorSet) {
			c.binder = func(peerbinder.Rules) (*peerbinder.Binder, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.binder = func(r peerbinder.Rules) (*peerbinder.Binder, error) {
				return &peerbinder.Binder{}, errors.New("binder")
			}
		}},
		{"server", func(c *constructorSet) {
			c.server = func(brokerlistener.Rules) (*brokerlistener.Server, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.server = func(r brokerlistener.Rules) (*brokerlistener.Server, error) {
				return &brokerlistener.Server{}, errors.New("server")
			}
		}},
		{"receiver", func(c *constructorSet) {
			c.receiver = func(socketactivation.Rules) (*socketactivation.Receiver, error) { return nil, nil }
		}, func(c *constructorSet) {
			c.receiver = func(r socketactivation.Rules) (*socketactivation.Receiver, error) {
				return &socketactivation.Receiver{}, errors.New("receiver")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			constructors := validTestConstructors()
			tc.zero(&constructors)
			if _, err := buildProductionGraphWith(input, constructors); err != ErrStart {
				t.Fatalf("err=%v, want fixed ErrStart", err)
			}
			if tc.errorCase != nil {
				constructors = validTestConstructors()
				tc.errorCase(&constructors)
				if _, err := buildProductionGraphWith(input, constructors); err != ErrStart {
					t.Fatalf("non-nil constructor error=%v, want fixed ErrStart", err)
				}
			}
		})
	}
}

func validTestConstructors() constructorSet {
	return constructorSet{
		policy:   func(r egresspolicy.Rules) (*egresspolicy.Policy, error) { return egresspolicy.New(r) },
		registry: func(r capability.Rules) (*capability.Registry, error) { return capability.New(r) },
		control: func(r capabilitycontrol.Rules) (*capabilitycontrol.Controller, error) {
			return capabilitycontrol.New(r)
		},
		transport: func() *upstreamtransport.Transport { return upstreamtransport.New() },
		credentials: func(providercredentials.Rules) (*providercredentials.Resolver, error) {
			return &providercredentials.Resolver{}, nil
		},
		exchange: func(brokerexchange.Rules) (*brokerexchange.Exchange, error) { return &brokerexchange.Exchange{}, nil },
		handler:  func(brokerhttp.Rules) (*brokerhttp.Handler, error) { return &brokerhttp.Handler{}, nil },
		session:  func(connectsession.Rules) (*connectsession.Session, error) { return &connectsession.Session{}, nil },
		binder:   func(peerbinder.Rules) (*peerbinder.Binder, error) { return &peerbinder.Binder{}, nil },
		server:   func(brokerlistener.Rules) (*brokerlistener.Server, error) { return &brokerlistener.Server{}, nil },
		receiver: func(socketactivation.Rules) (*socketactivation.Receiver, error) {
			return &socketactivation.Receiver{}, nil
		},
	}
}

func containsEvent(events []string, name string) bool { return stageIndex(events, name) >= 0 }
func stageIndex(events []string, name string) int {
	for i, event := range events {
		if event == name {
			return i
		}
	}
	return -1
}
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
