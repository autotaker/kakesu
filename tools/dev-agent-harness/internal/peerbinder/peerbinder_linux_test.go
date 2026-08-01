//go:build linux

package peerbinder

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerlistener"
)

func TestLinuxSOPEERCREDWithAcceptedUnixSocket(t *testing.T) {
	uid := os.Geteuid()
	rules := Rules{ExpectedUID: uid, Subject: brokerlistener.Subject{AgentInstanceID: "agent", UID: uid, WorkspaceID: "workspace"}}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(t.TempDir(), "peer.sock"), Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	client, err := net.DialUnix("unix", nil, listener.Addr().(*net.UnixAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	accepted, err := listener.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	defer accepted.Close()
	if peerUID, err := readPeerUID(accepted); err != nil || peerUID != uid {
		t.Fatalf("SO_PEERCRED uid=%d err=%v", peerUID, err)
	}
	binder, err := New(rules)
	if uid == 0 {
		if binder != nil || !errors.Is(err, ErrInvalidRules) {
			t.Fatalf("root UID binding was accepted: %#v %v", binder, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	got, err := binder.Bind(context.Background(), accepted)
	if err != nil || got != rules.Subject {
		t.Fatalf("accepted peer subject=%#v err=%v", got, err)
	}
}
