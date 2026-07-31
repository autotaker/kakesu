package command

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run("example", []string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run returned %d", code)
	}
	if !strings.HasPrefix(stdout.String(), "example ") || stderr.Len() != 0 {
		t.Fatalf("unexpected output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestOperationalInvocationFailsClosed(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run("example", nil, &stdout, &stderr); code == 0 {
		t.Fatal("operational invocation unexpectedly succeeded")
	}
	if !strings.Contains(stderr.String(), "refusing to start") {
		t.Fatalf("missing refusal: %q", stderr.String())
	}
}
