// Package socketactivation receives the one listener handed to the broker by
// systemd socket activation. It does not create, chmod, chown, or unlink a
// socket; the unit and tmpfiles/provision boundaries own those operations.
package socketactivation

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

const (
	listenPID     = "LISTEN_PID"
	listenFDS     = "LISTEN_FDS"
	listenFDNames = "LISTEN_FDNAMES"
	fdNumber      = 3
	fdName        = "egress"
	socketName    = "egress.sock"
	maxUnixPath   = 107
)

// Error is the fixed diagnostic surface of this package. Values never
// include a path, identity, environment value, descriptor number, or cause.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrDenied       Error = "socket-activation-denied"
)

// Rules fixes the runtime directory and the two numeric identities expected
// for the systemd-created directory and socket.
type Rules struct {
	RuntimeDir string
	BrokerUID  int
	AgentGID   int
}

// Receiver is immutable after New. It consumes the inherited descriptor once
// and returns only a listener that has passed the platform metadata boundary.
type Receiver struct {
	runtimeDir string
	socketPath string
	brokerUID  int
	agentGID   int
}

// New validates and copies one fixed socket activation contract.
func New(r Rules) (*Receiver, error) {
	if !validRules(r) {
		return nil, ErrInvalidRules
	}
	return &Receiver{
		runtimeDir: r.RuntimeDir,
		socketPath: r.RuntimeDir + string(os.PathSeparator) + socketName,
		brokerUID:  r.BrokerUID,
		agentGID:   r.AgentGID,
	}, nil
}

// Format keeps Receiver diagnostics fixed for every value and formatting
// verb. In particular, it does not expose the configured path or identities.
func (r *Receiver) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "socketactivation.Receiver")
}

// Take consumes the recognized activation environment and inherited FD 3
// exactly once. The original descriptor is always closed after conversion;
// the returned Unix listener owns only the duplicated descriptor.
func (r *Receiver) Take() (*net.UnixListener, error) {
	if !validReceiver(r) {
		return nil, ErrDenied
	}
	if rawEnvironment == nil {
		return nil, ErrDenied
	}

	pid, pidSet := lookupEnv(listenPID)
	fds, fdsSet := lookupEnv(listenFDS)
	fdnames, namesSet := lookupEnv(listenFDNames)
	raw, rawOK := rawEnvironment()
	duplicate := duplicateActivationEntry(raw)
	// Clearing all activation keys before checking their values makes malformed
	// but recognized activation one-shot as well.
	if pidSet || fdsSet || namesSet || hasActivationEntry(raw) {
		if !clearActivationEnvironment() {
			return nil, ErrDenied
		}
	}
	if !rawOK || duplicate || !pidSet || !fdsSet || !namesSet || pid != strconv.Itoa(currentPID()) || fds != "1" || fdnames != fdName {
		return nil, ErrDenied
	}

	original := newActivationFile(uintptr(fdNumber))
	if original == nil {
		return nil, ErrDenied
	}
	listener, convertErr := convertFileListener(original)
	closeErr := original.Close()
	if convertErr != nil || closeErr != nil || listener == nil {
		if listener != nil {
			_ = listener.Close()
		}
		return nil, ErrDenied
	}
	unixListener, ok := listener.(*net.UnixListener)
	if !ok || unixListener == nil {
		_ = listener.Close()
		return nil, ErrDenied
	}
	addr, ok := unixListenerAddress(unixListener)
	if !ok || addr == nil || addr.Network() != "unix" || addr.Name != r.socketPath {
		_ = unixListener.Close()
		return nil, ErrDenied
	}
	// The core above has already checked the concrete Unix type and fixed
	// pathname. Linux validates the separately opened node metadata below.
	if err := validatePlatform(r.runtimeDir, r.brokerUID, r.agentGID); err != nil {
		_ = unixListener.Close()
		return nil, ErrDenied
	}
	return unixListener, nil
}

func unixListenerAddress(listener *net.UnixListener) (addr *net.UnixAddr, ok bool) {
	defer func() {
		if recover() != nil {
			addr, ok = nil, false
		}
	}()
	if listener == nil {
		return nil, false
	}
	addr, ok = listener.Addr().(*net.UnixAddr)
	return addr, ok
}

var (
	lookupEnv           = os.LookupEnv
	unsetEnv            = os.Unsetenv
	currentPID          = os.Getpid
	rawEnvironment      = activationRawEnvironment
	newActivationFile   = func(fd uintptr) *os.File { return os.NewFile(fd, "dev-agent-egress") }
	convertFileListener = net.FileListener
)

func clearActivationEnvironment() bool {
	for _, key := range []string{listenPID, listenFDS, listenFDNames} {
		if unsetEnv(key) != nil {
			return false
		}
	}
	return true
}

func hasActivationEntry(entries []string) bool {
	for _, entry := range entries {
		name, _, ok := splitEnvironmentEntry(entry)
		if ok && (name == listenPID || name == listenFDS || name == listenFDNames) {
			return true
		}
	}
	return false
}

func duplicateActivationEntry(entries []string) bool {
	counts := map[string]int{}
	for _, entry := range entries {
		name, _, ok := splitEnvironmentEntry(entry)
		if !ok || (name != listenPID && name != listenFDS && name != listenFDNames) {
			continue
		}
		counts[name]++
		if counts[name] > 1 {
			return true
		}
	}
	return false
}

func splitEnvironmentEntry(entry string) (name, value string, ok bool) {
	for index := 0; index < len(entry); index++ {
		if entry[index] == '=' {
			return entry[:index], entry[index+1:], true
		}
	}
	return "", "", false
}

func validRules(r Rules) bool {
	if !validIdentifier(r.RuntimeDir) || r.RuntimeDir == string(os.PathSeparator) {
		return false
	}
	if r.BrokerUID <= 0 || r.AgentGID <= 0 || uint64(r.BrokerUID) > uint64(^uint32(0)) || uint64(r.AgentGID) > uint64(^uint32(0)) {
		return false
	}
	if r.BrokerUID == 0 || len(r.RuntimeDir)+1+len(socketName) > maxUnixPath {
		return false
	}
	return true
}

func validReceiver(r *Receiver) bool {
	return r != nil && validRules(Rules{RuntimeDir: r.runtimeDir, BrokerUID: r.brokerUID, AgentGID: r.agentGID}) && r.socketPath == r.runtimeDir+string(os.PathSeparator)+socketName
}

func validIdentifier(path string) bool {
	return path != "" && path[0] == os.PathSeparator && path == filepath.Clean(path) && path[len(path)-1] != os.PathSeparator && !containsNUL(path)
}

func containsNUL(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] == 0 {
			return true
		}
	}
	return false
}
