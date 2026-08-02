// Package egressservice owns the single production startup graph for the
// development-agent egress proxy.  It performs no retries or background
// startup work: every dependency is built once, then the inherited listener
// is consumed once and handed to the listener server.
package egressservice

import (
	"context"
	"net"
	"path/filepath"
	"reflect"
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
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/peerbinder"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/providercredentials"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/runtimeidentity"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/socketactivation"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/upstreamtransport"
)

// Error is the deliberately fixed service diagnostic.  Underlying config,
// identity, credential, socket, and constructor errors never cross this API.
type Error string

func (e Error) Error() string { return string(e) }

const ErrStart Error = "egress-service-start-failed"

type identitySnapshot struct {
	brokerUID     int
	agentUID      int
	agentGID      int
	agentInstance string
	workspace     string
	subject       brokerlistener.Subject
}

type credentialSnapshot struct {
	bundle    *brokercredentials.Bundle
	authority connectsession.Authority
}

type graphInput struct {
	config      *config.Config
	identity    identitySnapshot
	credentials credentialSnapshot
}

// runtimeGraph is intentionally made of closures.  The package-private
// factory seam below can count Take/Serve and inject deterministic failures,
// while production closures retain the concrete trusted constructors.
type runtimeGraph struct {
	take  func() (net.Listener, error)
	serve func(context.Context, net.Listener) error
}

type serviceFactory struct {
	loadConfig      func(string) (*config.Config, error)
	resolveIdentity func(*config.Config) (identitySnapshot, error)
	loadCredentials func(string) (credentialSnapshot, error)
	buildGraph      func(graphInput) (runtimeGraph, error)
}

// constructorSet is package-private: it lets tests capture every Rules value
// without publishing production factories or replacing any existing package
// constructor semantics.
type constructorSet struct {
	policy      func(egresspolicy.Rules) (*egresspolicy.Policy, error)
	registry    func(capability.Rules) (*capability.Registry, error)
	control     func(capabilitycontrol.Rules) (*capabilitycontrol.Controller, error)
	transport   func() *upstreamtransport.Transport
	credentials func(providercredentials.Rules) (*providercredentials.Resolver, error)
	exchange    func(brokerexchange.Rules) (*brokerexchange.Exchange, error)
	handler     func(brokerhttp.Rules) (*brokerhttp.Handler, error)
	session     func(connectsession.Rules) (*connectsession.Session, error)
	binder      func(peerbinder.Rules) (*peerbinder.Binder, error)
	server      func(brokerlistener.Rules) (*brokerlistener.Server, error)
	receiver    func(socketactivation.Rules) (*socketactivation.Receiver, error)
}

// Serve starts the only operational service surface.  It returns one fixed
// error for every startup or serving failure and never includes input values.
func Serve(ctx context.Context, configPath string) error {
	return serveWithFactory(ctx, configPath, productionFactory())
}

func serveWithFactory(ctx context.Context, configPath string, factory serviceFactory) error {
	if isNilContext(ctx) || configPath == "" || factory.loadConfig == nil || factory.resolveIdentity == nil || factory.loadCredentials == nil || factory.buildGraph == nil {
		return ErrStart
	}
	c, err := factory.loadConfig(configPath)
	if err != nil || c == nil {
		return ErrStart
	}
	identity, err := factory.resolveIdentity(c)
	if err != nil || !validIdentity(identity) {
		return ErrStart
	}
	credentials, err := factory.loadCredentials(filepath.Join(c.Paths.ConfigDir, "credentials"))
	if err != nil || credentials.bundle == nil || isNilDependency(credentials.authority) {
		return ErrStart
	}
	graph, err := factory.buildGraph(graphInput{config: c, identity: identity, credentials: credentials})
	if err != nil || graph.take == nil || graph.serve == nil {
		return ErrStart
	}
	listener, err := graph.take()
	if err != nil || isNilListener(listener) {
		return ErrStart
	}
	if err := graph.serve(ctx, listener); err != nil {
		return ErrStart
	}
	return nil
}

func productionFactory() serviceFactory {
	return serviceFactory{
		loadConfig:      config.Load,
		resolveIdentity: productionIdentity,
		loadCredentials: func(path string) (credentialSnapshot, error) {
			bundle, err := brokercredentials.Load(path)
			if err != nil || bundle == nil {
				return credentialSnapshot{}, ErrStart
			}
			authority := bundle.ProxyCAAuthority()
			if authority == nil {
				return credentialSnapshot{}, ErrStart
			}
			return credentialSnapshot{bundle: bundle, authority: authority}, nil
		},
		buildGraph: func(input graphInput) (runtimeGraph, error) {
			return buildProductionGraphWith(input, productionConstructors())
		},
	}
}

func productionIdentity(c *config.Config) (identitySnapshot, error) {
	if c == nil {
		return identitySnapshot{}, ErrStart
	}
	resolver, err := runtimeidentity.New(runtimeidentity.Rules{AgentUser: c.Users.Agent, BrokerUser: c.Users.Broker, WorkspaceID: c.Identity.WorkspaceID})
	if err != nil {
		return identitySnapshot{}, ErrStart
	}
	identity, err := resolver.Resolve()
	if err != nil || identity == nil {
		return identitySnapshot{}, ErrStart
	}
	snapshot := identitySnapshot{
		brokerUID: identity.BrokerUID(), agentUID: identity.AgentUID(), agentGID: identity.AgentGID(),
		agentInstance: identity.AgentInstanceID(), workspace: identity.WorkspaceID(), subject: identity.Subject(),
	}
	if !validIdentity(snapshot) {
		return identitySnapshot{}, ErrStart
	}
	return snapshot, nil
}

func buildProductionGraph(input graphInput) (runtimeGraph, error) {
	return buildProductionGraphWith(input, productionConstructors())
}

func productionConstructors() constructorSet {
	return constructorSet{
		policy: egresspolicy.New, registry: capability.New, control: capabilitycontrol.New, transport: upstreamtransport.New,
		credentials: providercredentials.New, exchange: brokerexchange.New, handler: brokerhttp.New,
		session: connectsession.New, binder: peerbinder.New, server: brokerlistener.New, receiver: socketactivation.New,
	}
}

func buildProductionGraphWith(input graphInput, constructors constructorSet) (runtimeGraph, error) {
	if input.config == nil || input.credentials.bundle == nil || isNilDependency(input.credentials.authority) || !validIdentity(input.identity) {
		return runtimeGraph{}, ErrStart
	}
	if constructors.policy == nil || constructors.registry == nil || constructors.control == nil || constructors.transport == nil || constructors.credentials == nil || constructors.exchange == nil || constructors.handler == nil || constructors.session == nil || constructors.binder == nil || constructors.server == nil || constructors.receiver == nil {
		return runtimeGraph{}, ErrStart
	}
	policy, err := constructors.policy(egresspolicy.Rules{
		GitHubRepositories: append([]string(nil), input.config.Egress.GitHubRepositories...),
		OpenAIModels:       append([]string(nil), input.config.Egress.OpenAIModels...),
		MaxBodyBytes:       64 * 1024,
		MaxOutputTokens:    4096,
	})
	if err != nil || policy == nil {
		return runtimeGraph{}, ErrStart
	}
	registry, err := constructors.registry(capability.Rules{PolicyVersion: "egress-v1", MaxTTL: 10 * time.Minute, MaxUses: 16, InitialRevocationEpoch: 1})
	if err != nil || registry == nil {
		return runtimeGraph{}, ErrStart
	}
	control, err := constructors.control(capabilitycontrol.Rules{
		Registry: registry, Resolver: brokerlistener.Resolver{},
		GitHubRepositories: append([]string(nil), input.config.Egress.GitHubRepositories...),
		OpenAIModels:       append([]string(nil), input.config.Egress.OpenAIModels...),
	})
	if err != nil || control == nil {
		return runtimeGraph{}, ErrStart
	}
	transport := constructors.transport()
	if transport == nil {
		return runtimeGraph{}, ErrStart
	}
	credentials, err := constructors.credentials(providercredentials.Rules{Bundle: input.credentials.bundle, Transport: transport, Timeout: 10 * time.Second})
	if err != nil || credentials == nil {
		return runtimeGraph{}, ErrStart
	}
	exchange, err := constructors.exchange(brokerexchange.Rules{Policy: policy, Registry: registry, Resolver: credentials, Transport: transport, MaxCredentialBytes: 4096, Timeout: 30 * time.Second, MaxResponseBytes: 1 << 20})
	if err != nil || exchange == nil {
		return runtimeGraph{}, ErrStart
	}
	handler, err := constructors.handler(brokerhttp.Rules{Exchange: exchange, Resolver: brokerlistener.Resolver{}, MaxBodyBytes: 64 * 1024})
	if err != nil || handler == nil {
		return runtimeGraph{}, ErrStart
	}
	session, err := constructors.session(connectsession.Rules{Authority: input.credentials.authority, Handler: handler, Control: control})
	if err != nil || session == nil {
		return runtimeGraph{}, ErrStart
	}
	binder, err := constructors.binder(peerbinder.Rules{ExpectedUID: input.identity.agentUID, Subject: input.identity.subject})
	if err != nil || binder == nil {
		return runtimeGraph{}, ErrStart
	}
	server, err := constructors.server(brokerlistener.Rules{Binder: binder, Session: session, MaxConcurrent: 16})
	if err != nil || server == nil {
		return runtimeGraph{}, ErrStart
	}
	receiver, err := constructors.receiver(socketactivation.Rules{RuntimeDir: input.config.Paths.RuntimeDir, BrokerUID: input.identity.brokerUID, AgentGID: input.identity.agentGID})
	if err != nil || receiver == nil {
		return runtimeGraph{}, ErrStart
	}
	return runtimeGraph{
		take:  func() (net.Listener, error) { return receiver.Take() },
		serve: func(ctx context.Context, listener net.Listener) error { return server.Serve(ctx, listener) },
	}, nil
}

func validIdentity(identity identitySnapshot) bool {
	return identity.brokerUID > 0 && identity.agentUID > 0 && identity.agentGID > 0 && identity.brokerUID != identity.agentUID && identity.agentInstance != "" && identity.workspace != "" && identity.subject.UID == identity.agentUID && identity.subject.AgentInstanceID == identity.agentInstance && identity.subject.WorkspaceID == identity.workspace
}

func isNilContext(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	rv := reflect.ValueOf(ctx)
	return (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface || rv.Kind() == reflect.Func || rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Chan) && rv.IsNil()
}

func isNilListener(listener net.Listener) bool {
	if listener == nil {
		return true
	}
	rv := reflect.ValueOf(listener)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return rv.IsNil()
	default:
		return false
	}
}

func isNilDependency(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return rv.IsNil()
	default:
		return false
	}
}
