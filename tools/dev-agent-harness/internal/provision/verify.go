package provision

import (
	"bytes"
	"io"
	"os"
	"syscall"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/config"
)

// MaxManifestSize is the largest manifest accepted by Verify.  The manifest
// is read as opaque bytes; it is not parsed or normalized by this package.
const MaxManifestSize = 128 * 1024

const (
	ClassManifestFilePolicy ErrorClass = "manifest-file-policy"
	ClassManifestRead       ErrorClass = "manifest-read"
	ClassManifestMismatch   ErrorClass = "manifest-mismatch"
)

// manifestReadBeforeHook is a test-only seam for exercising the read-time
// metadata check. Production code leaves it nil. It is deliberately invoked
// after the descriptor has been opened and checked, but before the first read.
var manifestReadBeforeHook func(*os.File)

// Verify reads manifestPath once, using one checked descriptor, and accepts
// it only when its raw bytes exactly equal Build(c, targetRoot). No parser or
// filesystem lookup is involved in the comparison.
func Verify(c *config.Config, manifestPath, targetRoot string) error {
	data, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	want, err := Build(c, targetRoot)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, want) {
		return fail(ClassManifestMismatch)
	}
	return nil
}

func readManifest(path string) ([]byte, error) {
	if path == "" {
		return nil, fail(ClassManifestFilePolicy)
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fail(ClassManifestFilePolicy)
	}
	f := os.NewFile(uintptr(fd), "harness-manifest")
	if f == nil {
		_ = syscall.Close(fd)
		return nil, fail(ClassManifestRead)
	}
	defer f.Close()

	before, err := f.Stat()
	if err != nil {
		return nil, fail(ClassManifestFilePolicy)
	}
	if !safeManifestInfo(before) || before.Size() > MaxManifestSize {
		return nil, fail(ClassManifestFilePolicy)
	}
	if manifestReadBeforeHook != nil {
		manifestReadBeforeHook(f)
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxManifestSize+1))
	if err != nil {
		return nil, fail(ClassManifestRead)
	}
	after, err := f.Stat()
	if err != nil {
		return nil, fail(ClassManifestFilePolicy)
	}
	if !safeManifestInfo(after) || after.Size() > MaxManifestSize ||
		before.Mode() != after.Mode() || before.Size() != after.Size() ||
		int64(len(data)) != before.Size() || int64(len(data)) != after.Size() {
		return nil, fail(ClassManifestFilePolicy)
	}
	return data, nil
}

func safeManifestInfo(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode().Perm()&0o022 == 0
}
