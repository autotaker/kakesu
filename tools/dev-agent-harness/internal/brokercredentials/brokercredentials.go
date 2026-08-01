// Package brokercredentials is the broker-only credential boundary.
//
// It reads one fixed directory layout and returns only values needed by a
// trusted broker. It deliberately has no environment, process, network, or
// persistent-write integration.
package brokercredentials

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"time"
)

// MaxFileSize is the upper bound applied before and during every secret file
// read. Individual text fields have smaller semantic limits.
const MaxFileSize = 64 * 1024

const (
	maxClientID      = 128
	maxOpenAIKey     = 4096
	minRSAKeyBits    = 2048
	maxRSAKeyBits    = 8192
	githubClientID   = "github-client-id"
	githubInstallID  = "github-installation-id"
	githubPrivateKey = "github-private-key.pem"
	openAIAPIKey     = "openai-api-key"
)

// ErrLoad is the only load failure exposed by this package. It intentionally
// carries no path, input, parser, OS, ownership, or key details.
var ErrLoad = errors.New("credential load failed")

// ErrJWT is the only JWT generation failure exposed by this package.
var ErrJWT = errors.New("credential jwt failed")

var basenames = [...]string{githubClientID, githubInstallID, githubPrivateKey, openAIAPIKey}

// These seams are package-private and exist only to make boundary tests
// deterministic. Production callers cannot replace either behavior.
var nowUTC = func() time.Time { return time.Now().UTC() }
var readCompleteHook func()

func validDirectoryPath(dir string) bool {
	return dir != "" && filepath.IsAbs(dir) && filepath.Clean(dir) == dir && filepath.Base(dir) != ".."
}

// Bundle contains validated broker credentials. Its fields are private so a
// private key or source bytes cannot be returned, marshalled, or formatted by
// an accidental exported field. Callers should retain this only in trusted
// broker memory.
type Bundle struct {
	clientID       string
	installationID int64
	openAIKey      string
	privateKey     *rsa.PrivateKey
}

// Format prevents accidental diagnostic formatting from exposing the private
// key or API key held in unexported fields. It emits the same fixed label for
// every verb and flag; callers must use the narrow accessors instead.
func (b Bundle) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "brokercredentials.Bundle")
}

// Load reads the fixed four-file broker directory and validates every value.
// A complete bundle is returned only when all files satisfy the policy.
func Load(dir string) (*Bundle, error) {
	files, err := readSecretFiles(dir)
	if err != nil || len(files) != len(basenames) {
		return nil, ErrLoad
	}
	clientID, ok := visibleText(files[0], 1, maxClientID)
	if !ok {
		return nil, ErrLoad
	}
	installationID, ok := parseInstallationID(files[1])
	if !ok {
		return nil, ErrLoad
	}
	key, ok := parsePrivateKey(files[2])
	if !ok {
		return nil, ErrLoad
	}
	openAIKey, ok := visibleText(files[3], 1, maxOpenAIKey)
	if !ok {
		return nil, ErrLoad
	}
	return &Bundle{clientID: clientID, installationID: installationID, openAIKey: openAIKey, privateKey: key}, nil
}

// ClientID returns the validated GitHub App client ID for trusted broker use.
func (b *Bundle) ClientID() string {
	if b == nil {
		return ""
	}
	return b.clientID
}

// InstallationID returns the validated GitHub installation ID.
func (b *Bundle) InstallationID() int64 {
	if b == nil {
		return 0
	}
	return b.installationID
}

// OpenAIAPIKey returns the validated OpenAI key for trusted broker use.
func (b *Bundle) OpenAIAPIKey() string {
	if b == nil {
		return ""
	}
	return b.openAIKey
}

// GitHubAppJWT creates a short-lived RS256 JWT. Claims use integer Unix
// seconds from the UTC clock; PKCS#1 v1.5 signing is deterministic for equal
// claims, so two calls in one second may return the same token.
func (b *Bundle) GitHubAppJWT() (string, error) {
	if b == nil || b.privateKey == nil || b.clientID == "" {
		return "", ErrJWT
	}
	now := nowUTC().UTC().Unix()
	header, err := json.Marshal(struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{Alg: "RS256", Typ: "JWT"})
	if err != nil {
		return "", ErrJWT
	}
	payload, err := json.Marshal(struct {
		IAT int64  `json:"iat"`
		EXP int64  `json:"exp"`
		ISS string `json:"iss"`
	}{IAT: now - 60, EXP: now + 540, ISS: b.clientID})
	if err != nil {
		return "", ErrJWT
	}
	enc := base64.RawURLEncoding
	encodedHeader := enc.EncodeToString(header)
	encodedPayload := enc.EncodeToString(payload)
	message := encodedHeader + "." + encodedPayload
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, b.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", ErrJWT
	}
	return message + "." + enc.EncodeToString(signature), nil
}

func visibleText(data []byte, min, max int) (string, bool) {
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	if len(data) < min || len(data) > max {
		return "", false
	}
	for _, c := range data {
		if c < 0x21 || c > 0x7e {
			return "", false
		}
	}
	return string(data), true
}

func parseInstallationID(data []byte) (int64, bool) {
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	if len(data) == 0 || (len(data) > 1 && data[0] == '0') {
		return 0, false
	}
	var value int64
	for _, c := range data {
		if c < '0' || c > '9' {
			return 0, false
		}
		digit := int64(c - '0')
		if value > (math.MaxInt64-digit)/10 {
			return 0, false
		}
		value = value*10 + digit
	}
	if value <= 0 {
		return 0, false
	}
	return value, true
}

func parsePrivateKey(data []byte) (*rsa.PrivateKey, bool) {
	block, rest := pem.Decode(data)
	if block == nil || len(block.Headers) != 0 || len(rest) != 0 {
		return nil, false
	}
	var key *rsa.PrivateKey
	var err error
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		var parsed any
		parsed, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			key, _ = parsed.(*rsa.PrivateKey)
		}
	default:
		return nil, false
	}
	if err != nil || key == nil || key.N == nil {
		return nil, false
	}
	bits := key.N.BitLen()
	if bits < minRSAKeyBits || bits > maxRSAKeyBits || key.Validate() != nil {
		return nil, false
	}
	return key, true
}
