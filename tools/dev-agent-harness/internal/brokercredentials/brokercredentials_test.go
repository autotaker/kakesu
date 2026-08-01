package brokercredentials

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadAndJWT(t *testing.T) {
	requireNonRoot(t)
	key, dir := validFixture(t, "PRIVATE KEY")
	bundle, err := Load(dir)
	if err != nil {
		t.Fatalf("valid fixture rejected: %v", err)
	}
	if bundle.ClientID() != "Iv1.test-client" || bundle.InstallationID() != 123456 || bundle.OpenAIAPIKey() != "test-openai-key" {
		t.Fatal("validated broker values were not returned")
	}
	fixed := time.Unix(1_800_000_000, 987654321).UTC()
	previousNow := nowUTC
	nowUTC = func() time.Time { return fixed }
	defer func() { nowUTC = previousNow }()
	token, err := bundle.GitHubAppJWT()
	if err != nil {
		t.Fatalf("JWT generation failed: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT parts=%d", len(parts))
	}
	decode := func(part string, out any) {
		t.Helper()
		data, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			t.Fatalf("base64: %v", err)
		}
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("json: %v", err)
		}
	}
	var header map[string]any
	decode(parts[0], &header)
	if len(header) != 2 || header["alg"] != "RS256" || header["typ"] != "JWT" {
		t.Fatalf("unexpected header: %#v", header)
	}
	var payload map[string]any
	decode(parts[1], &payload)
	if len(payload) != 3 || payload["iss"] != "Iv1.test-client" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if got := int64(payload["iat"].(float64)); got != fixed.Unix()-60 {
		t.Fatalf("iat=%d", got)
	}
	if got := int64(payload["exp"].(float64)); got != fixed.Unix()+540 {
		t.Fatalf("exp=%d", got)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature); err != nil {
		t.Fatalf("signature did not verify: %v", err)
	}
	second, err := bundle.GitHubAppJWT()
	if err != nil {
		t.Fatal(err)
	}
	if token != second {
		t.Fatal("same-second JWT was not deterministic")
	}
}

func TestLoadPolicyAndFixedErrors(t *testing.T) {
	requireNonRoot(t)
	cases := []struct {
		name string
		edit func(t *testing.T, dir string)
	}{
		{"directory-permission", func(t *testing.T, dir string) { chmod(t, dir, 0o750) }},
		{"group-permission", func(t *testing.T, dir string) { chmod(t, filepath.Join(dir, githubClientID), 0o620) }},
		{"read-permission", func(t *testing.T, dir string) { chmod(t, filepath.Join(dir, githubClientID), 0o200) }},
		{"execute-permission", func(t *testing.T, dir string) { chmod(t, filepath.Join(dir, githubClientID), 0o700) }},
		{"oversize", func(t *testing.T, dir string) {
			path := filepath.Join(dir, openAIAPIKey)
			if err := os.WriteFile(path, make([]byte, MaxFileSize+1), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"missing", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, githubClientID)); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink", func(t *testing.T, dir string) {
			path := filepath.Join(dir, githubClientID)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(dir, "other"), path); err != nil {
				t.Skipf("symlink unsupported: %v", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, fresh := validFixture(t, "RSA PRIVATE KEY")
			tc.edit(t, fresh)
			_, err := Load(fresh)
			if !errors.Is(err, ErrLoad) || err.Error() != ErrLoad.Error() {
				t.Fatalf("err=%v", err)
			}
			if strings.Contains(err.Error(), fresh) || strings.Contains(err.Error(), "test-client") {
				t.Fatal("load error leaked input")
			}
		})
	}

}

func TestReadMetadataChangeIsRejected(t *testing.T) {
	requireNonRoot(t)
	_, dir := validFixture(t, "RSA PRIVATE KEY")
	path := filepath.Join(dir, githubClientID)
	previousHook := readCompleteHook
	readCompleteHook = func() { _ = os.Chmod(path, 0o640) }
	defer func() { readCompleteHook = previousHook }()
	if _, err := Load(dir); !errors.Is(err, ErrLoad) {
		t.Fatalf("metadata mutation accepted: %v", err)
	}
}

func TestTextAndKeyBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name     string
		data     []byte
		min, max int
		ok       bool
	}{
		{"one-lf", []byte("visible\n"), 1, 128, true},
		{"two-lf", []byte("visible\n\n"), 1, 128, false},
		{"space", []byte("visible value"), 1, 128, false},
		{"control", []byte("visible\r"), 1, 128, false},
		{"non-ascii", []byte("visiblé"), 1, 128, false},
		{"empty", nil, 1, 128, false},
		{"boundary", []byte(strings.Repeat("x", 4)), 1, 4, true},
		{"overlong", []byte(strings.Repeat("x", 5)), 1, 4, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := visibleText(tc.data, tc.min, tc.max)
			if ok != tc.ok {
				t.Fatalf("ok=%v want %v", ok, tc.ok)
			}
		})
	}
	for _, tc := range []struct {
		data string
		ok   bool
	}{
		{"1", true}, {"9223372036854775807", true}, {"0", false}, {"01", false}, {"-1", false},
		{"9223372036854775808", false}, {"42\n", true}, {"42\n\n", false}, {"42x", false},
	} {
		t.Run("installation-"+tc.data, func(t *testing.T) {
			_, ok := parseInstallationID([]byte(tc.data))
			if ok != tc.ok {
				t.Fatalf("ok=%v want %v", ok, tc.ok)
			}
		})
	}
}

func TestPrivateKeyFormsAndRejections(t *testing.T) {
	key, _ := validFixture(t, "RSA PRIVATE KEY")
	pkcs1 := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8 := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8Bytes})
	for _, data := range [][]byte{pkcs1, pkcs8} {
		if parsed, ok := parsePrivateKey(data); !ok || parsed.N.BitLen() != 2048 {
			t.Fatal("valid RSA form rejected")
		}
	}
	bad := [][]byte{
		[]byte("not-pem"), append(append([]byte{}, pkcs1...), []byte("garbage")...),
		append(append([]byte{}, pkcs1...), '\n'),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: map[string]string{"Proc-Type": "4,ENCRYPTED"}, Bytes: x509.MarshalPKCS1PrivateKey(key)}),
	}
	for _, data := range bad {
		if _, ok := parsePrivateKey(data); ok {
			t.Fatal("invalid PEM accepted")
		}
	}
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecdsaDER, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := parsePrivateKey(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecdsaDER})); ok {
		t.Fatal("non-RSA key accepted")
	}
	weak, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := parsePrivateKey(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(weak)})); ok {
		t.Fatal("short RSA key accepted")
	}
}

func TestCallerInputAndNilJWT(t *testing.T) {
	if _, err := Load("/tmp/credential-sentinel"); !errors.Is(err, ErrLoad) || strings.Contains(err.Error(), "sentinel") {
		t.Fatalf("bad load error=%v", err)
	}
	var bundle *Bundle
	if _, err := bundle.GitHubAppJWT(); !errors.Is(err, ErrJWT) || err.Error() != ErrJWT.Error() {
		t.Fatalf("nil JWT error=%v", err)
	}
	if got := bundle.ClientID(); got != "" {
		t.Fatalf("nil ClientID=%q", got)
	}
}

func TestBundleFormattingDoesNotExposeSecrets(t *testing.T) {
	key, dir := validFixture(t, "RSA PRIVATE KEY")
	bundle, err := Load(dir)
	if err != nil {
		t.Skipf("credential policy unavailable: %v", err)
	}
	for _, value := range []any{bundle, *bundle} {
		for _, format := range []string{"%v", "%+v", "%#v", "%q", "%s"} {
			formatted := fmt.Sprintf(format, value)
			if strings.Contains(formatted, "test-openai-key") || strings.Contains(formatted, "Iv1.test-client") || strings.Contains(formatted, key.N.String()) {
				t.Fatalf("format %s exposed bundle secret/material", format)
			}
		}
	}
}

func validFixture(t *testing.T, pemType string) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	var pemBytes []byte
	if pemType == "PRIVATE KEY" {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		pemBytes = pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: der})
	} else {
		pemBytes = pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: x509.MarshalPKCS1PrivateKey(key)})
	}
	contents := map[string][]byte{
		githubClientID: []byte("Iv1.test-client\n"), githubInstallID: []byte("123456\n"),
		githubPrivateKey: pemBytes, openAIAPIKey: []byte("test-openai-key\n"),
	}
	for name, data := range contents {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return key, dir
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(fmt.Errorf("chmod: %w", err))
	}
}

func requireNonRoot(t *testing.T) {
	t.Helper()
	if testIsRoot() {
		t.Skip("credential policy intentionally rejects root")
	}
}
