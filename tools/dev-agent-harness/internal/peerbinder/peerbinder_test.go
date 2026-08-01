package peerbinder

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerlistener"
)

func TestNewValidationAndFixedFormat(t *testing.T) {
	valid := Rules{ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "agent-1", UID: 1000, WorkspaceID: "workspace-1"}}
	cases := map[string]Rules{
		"zero":           {},
		"uid-zero":       {ExpectedUID: 0, Subject: valid.Subject},
		"uid-negative":   {ExpectedUID: -1, Subject: valid.Subject},
		"uid-mismatch":   {ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: 1001, WorkspaceID: "workspace"}},
		"agent-first":    {ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "-agent", UID: 1000, WorkspaceID: "workspace"}},
		"workspace-byte": {ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: 1000, WorkspaceID: "workspace/"}},
		"agent-long":     {ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: strings.Repeat("a", maxIdentifier+1), UID: 1000, WorkspaceID: "workspace"}},
	}
	if uint64(^uint32(0)) <= uint64(^uint(0)>>1) {
		max := int(uint64(^uint32(0)))
		maxRules := Rules{ExpectedUID: max, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: max, WorkspaceID: "workspace"}}
		if binder, err := newWithReader(maxRules, func(*net.UnixConn) (int, error) { return max, nil }); err != nil || binder == nil {
			t.Fatalf("maximum representable UID rejected: %v", err)
		}
		tooHigh := int(uint64(max) + 1)
		if binder, err := newWithReader(Rules{ExpectedUID: tooHigh, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: tooHigh, WorkspaceID: "workspace"}}, func(*net.UnixConn) (int, error) { return tooHigh, nil }); binder != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("UID above uint32 accepted: %#v %v", binder, err)
		}
	}
	for name, rules := range cases {
		if binder, err := newWithReader(rules, func(*net.UnixConn) (int, error) { return 1000, nil }); binder != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("%s: binder=%#v err=%v", name, binder, err)
		}
	}
	binder, err := newWithReader(valid, func(*net.UnixConn) (int, error) { return 1000, nil })
	if err != nil || binder == nil {
		t.Fatalf("valid rules rejected: %v", err)
	}
	if got := fmt.Sprintf("%v", binder); got != "peerbinder.Binder" {
		t.Fatalf("format=%q", got)
	}
	var zero Binder
	if _, err := zero.Bind(context.Background(), nil); !errors.Is(err, ErrDenied) {
		t.Fatalf("zero binder error=%v", err)
	}
	corrupt := []*Binder{
		{expectedUID: 1000, subject: valid.Subject},
		{expectedUID: 1000, subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: 1001, WorkspaceID: "workspace"}, reader: func(*net.UnixConn) (int, error) { return 1000, nil }},
	}
	for index, binder := range corrupt {
		if subject, err := binder.Bind(context.Background(), new(net.UnixConn)); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) {
			t.Fatalf("corrupt binder %d subject=%#v err=%v", index, subject, err)
		}
	}
}

func TestBindExactUIDCopiesAndReadsOnce(t *testing.T) {
	serverConn, cleanup := unixPair(t)
	defer cleanup()
	agentBytes := []byte("agent-1")
	workspaceBytes := []byte("workspace-1")
	input := brokerlistener.Subject{
		AgentInstanceID: unsafe.String(&agentBytes[0], len(agentBytes)),
		UID:             1000,
		WorkspaceID:     unsafe.String(&workspaceBytes[0], len(workspaceBytes)),
	}
	reads := 0
	binder, err := newWithReader(Rules{ExpectedUID: 1000, Subject: input}, func(*net.UnixConn) (int, error) {
		reads++
		return 1000, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	agentBytes[0], workspaceBytes[0] = 'X', 'Y'
	got, err := binder.Bind(context.Background(), serverConn)
	if err != nil || got.AgentInstanceID != "agent-1" || got.WorkspaceID != "workspace-1" || got.UID != 1000 {
		t.Fatalf("Bind()=%#v err=%v", got, err)
	}
	if reads != 1 {
		t.Fatalf("reader calls=%d", reads)
	}
	if unsafe.StringData(got.AgentInstanceID) == unsafe.StringData(binder.subject.AgentInstanceID) || unsafe.StringData(got.WorkspaceID) == unsafe.StringData(binder.subject.WorkspaceID) {
		t.Fatal("Bind did not return an independent subject copy")
	}
}

func TestBindRejectsTypeMismatchAndFixedFailures(t *testing.T) {
	serverConn := new(net.UnixConn)
	reads := 0
	binder, err := newWithReader(Rules{ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: 1000, WorkspaceID: "workspace"}}, func(*net.UnixConn) (int, error) {
		reads++
		return 1001, errors.New("socket secret")
	})
	if err != nil {
		t.Fatal(err)
	}
	pipeA, pipeB := net.Pipe()
	defer pipeA.Close()
	defer pipeB.Close()
	var nilUnix *net.UnixConn
	wrapped := unixWrapper{UnixConn: serverConn}
	cases := map[string]net.Conn{"pipe": pipeA, "nil": nilUnix, "wrapped": wrapped}
	for name, conn := range cases {
		t.Run(name, func(t *testing.T) {
			if subject, err := binder.Bind(context.Background(), conn); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) {
				t.Fatalf("Bind()=%#v err=%v", subject, err)
			}
		})
	}
	if reads != 0 {
		t.Fatalf("reader called for rejected connection: %d", reads)
	}
	if subject, err := binder.Bind(context.Background(), serverConn); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) || fmt.Sprint(err) != "peer-bind-denied" {
		t.Fatalf("reader failure leaked or returned subject=%#v err=%v", subject, err)
	}
	reads = 0
	mismatch, err := newWithReader(Rules{ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: 1000, WorkspaceID: "workspace"}}, func(*net.UnixConn) (int, error) { reads++; return 1001, nil })
	if err != nil {
		t.Fatal(err)
	}
	if subject, err := mismatch.Bind(context.Background(), serverConn); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) || reads != 1 {
		t.Fatalf("UID mismatch subject=%#v err=%v calls=%d", subject, err, reads)
	}
}

type unixWrapper struct{ *net.UnixConn }

func TestBindContextBeforeAndAfterReader(t *testing.T) {
	serverConn := new(net.UnixConn)
	makeBinder := func(read peerReader) *Binder {
		binder, err := newWithReader(Rules{ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: 1000, WorkspaceID: "workspace"}}, read)
		if err != nil {
			t.Fatal(err)
		}
		return binder
	}
	called := 0
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if subject, err := makeBinder(func(*net.UnixConn) (int, error) { called++; return 1000, nil }).Bind(cancelled, serverConn); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) || called != 0 {
		t.Fatalf("cancelled context subject=%#v err=%v calls=%d", subject, err, called)
	}
	afterCancel, cancelAfter := context.WithCancel(context.Background())
	binder := makeBinder(func(*net.UnixConn) (int, error) { cancelAfter(); return 1000, nil })
	if subject, err := binder.Bind(afterCancel, serverConn); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) {
		t.Fatalf("reader cancellation subject=%#v err=%v", subject, err)
	}
	var typedNil *testContext
	if subject, err := binder.Bind(typedNil, serverConn); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) {
		t.Fatalf("typed nil context subject=%#v err=%v", subject, err)
	}
}

func TestPlatformReaderFailsClosedOutsideLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("Linux adapter is covered by the Linux socket test")
	}
	binder, err := New(Rules{ExpectedUID: 1000, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: 1000, WorkspaceID: "workspace"}})
	if err != nil {
		t.Fatal(err)
	}
	if subject, err := binder.Bind(context.Background(), new(net.UnixConn)); subject != (brokerlistener.Subject{}) || !errors.Is(err, ErrDenied) {
		t.Fatalf("non-Linux reader accepted subject=%#v err=%v", subject, err)
	}
}

type testContext struct{}

func (*testContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*testContext) Done() <-chan struct{}       { return nil }
func (*testContext) Err() error                  { return nil }
func (*testContext) Value(any) any               { return nil }

func unixPair(t *testing.T) (*net.UnixConn, func()) {
	t.Helper()
	return new(net.UnixConn), func() {}
}
