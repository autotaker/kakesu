// Package proxyca creates the broker's short-lived in-memory TLS leaf
// certificates for the two fixed upstream hostnames.
package proxyca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"time"
)

const (
	leafLifetime = 15 * time.Minute
	backdate     = time.Minute
)

// Error values are fixed and contain no PEM, parser, key, serial, or host
// detail.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrDenied       Error = "proxy-ca-denied"
)

// Clock is the only runtime dependency retained by Authority.
type Clock interface {
	Now() time.Time
}

// ClockFunc adapts a function to Clock.
type ClockFunc func() time.Time

func (f ClockFunc) Now() time.Time { return f() }

// Rules supplies one in-memory CA certificate, its matching ECDSA P-256
// private key, and a clock. Input slices are parsed and then discarded.
type Rules struct {
	CACertificatePEM []byte
	CAPrivateKeyPEM  []byte
	Clock            Clock
}

// Authority validates and retains only parsed CA state and a public PEM copy.
type Authority struct {
	certificate *x509.Certificate
	signer      *ecdsa.PrivateKey
	publicPEM   []byte
	clock       Clock
}

// Format deliberately avoids all certificate and key details.
func (a Authority) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "proxyca.Authority")
}

// New validates a single self-signed ECDSA P-256 CA and retains no input PEM.
func New(r Rules) (*Authority, error) {
	if isNil(r.Clock) || len(r.CACertificatePEM) == 0 || len(r.CAPrivateKeyPEM) == 0 {
		return nil, ErrInvalidRules
	}
	certDER, ok := singlePEM(r.CACertificatePEM, "CERTIFICATE")
	if !ok {
		return nil, ErrInvalidRules
	}
	keyDER, ok := singlePEM(r.CAPrivateKeyPEM, "EC PRIVATE KEY")
	if !ok {
		return nil, ErrInvalidRules
	}
	certificate, err := x509.ParseCertificate(certDER)
	if err != nil || !validCA(certificate) {
		return nil, ErrInvalidRules
	}
	signer, err := x509.ParseECPrivateKey(keyDER)
	if err != nil || !validP256(signer) || !matchingPublicKey(certificate, &signer.PublicKey) {
		return nil, ErrInvalidRules
	}
	now := r.Clock.Now().UTC()
	if now.IsZero() || now.Before(certificate.NotBefore) || !now.Before(certificate.NotAfter) || certificate.NotAfter.Sub(now) < leafLifetime {
		return nil, ErrInvalidRules
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: append([]byte(nil), certDER...)})
	if len(publicPEM) == 0 {
		return nil, ErrInvalidRules
	}
	return &Authority{certificate: certificate, signer: signer, publicPEM: publicPEM, clock: r.Clock}, nil
}

// PublicCertificatePEM returns a fresh certificate-only public CA PEM copy.
func (a *Authority) PublicCertificatePEM() []byte {
	if a == nil || len(a.publicPEM) == 0 {
		return nil
	}
	return append([]byte(nil), a.publicPEM...)
}

// Issue creates a fresh short-lived server certificate for one exact host.
func (a *Authority) Issue(host string) (tls.Certificate, error) {
	if a == nil || a.certificate == nil || !validP256(a.signer) || isNil(a.clock) || !exactHost(host) {
		return tls.Certificate{}, ErrDenied
	}
	now := a.clock.Now().UTC()
	if now.IsZero() || now.Before(a.certificate.NotBefore) || !now.Before(a.certificate.NotAfter) {
		return tls.Certificate{}, ErrDenied
	}
	serial, err := randomSerial()
	if err != nil {
		return tls.Certificate{}, ErrDenied
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, ErrDenied
	}
	notBefore := now.Add(-backdate)
	if notBefore.Before(a.certificate.NotBefore) {
		notBefore = a.certificate.NotBefore
	}
	notAfter := now.Add(leafLifetime)
	if notAfter.After(a.certificate.NotAfter) {
		notAfter = a.certificate.NotAfter
	}
	if !notAfter.After(notBefore) || !notAfter.After(now) {
		return tls.Certificate{}, ErrDenied
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{},
		DNSNames:              []string{host},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, template, a.certificate, &leafKey.PublicKey, a.signer)
	if err != nil {
		return tls.Certificate{}, ErrDenied
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return tls.Certificate{}, ErrDenied
	}
	return tls.Certificate{Certificate: [][]byte{append([]byte(nil), leafDER...), append([]byte(nil), a.certificate.Raw...)}, PrivateKey: leafKey, Leaf: leaf}, nil
}

func singlePEM(input []byte, blockType string) ([]byte, bool) {
	block, rest := pem.Decode(input)
	if block == nil || block.Type != blockType || len(block.Headers) != 0 || len(rest) != 0 || len(block.Bytes) == 0 {
		return nil, false
	}
	return append([]byte(nil), block.Bytes...), true
}

func validCA(certificate *x509.Certificate) bool {
	return certificate != nil && certificate.BasicConstraintsValid && certificate.IsCA &&
		certificate.KeyUsage&x509.KeyUsageCertSign != 0 && validP256Public(certificate.PublicKey) &&
		certificate.CheckSignatureFrom(certificate) == nil
}

func validP256(key *ecdsa.PrivateKey) bool {
	return key != nil && key.Curve == elliptic.P256() && key.D != nil && key.D.Sign() > 0 && validP256Public(&key.PublicKey)
}

func validP256Public(value any) bool {
	key, ok := value.(*ecdsa.PublicKey)
	return ok && key != nil && key.Curve == elliptic.P256() && key.X != nil && key.Y != nil && elliptic.P256().IsOnCurve(key.X, key.Y)
}

func matchingPublicKey(certificate *x509.Certificate, key *ecdsa.PublicKey) bool {
	public, ok := certificate.PublicKey.(*ecdsa.PublicKey)
	return ok && key != nil && public.X.Cmp(key.X) == 0 && public.Y.Cmp(key.Y) == 0
}

func randomSerial() (*big.Int, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	serial := new(big.Int).SetBytes(bytes)
	if serial.Sign() == 0 {
		return nil, ErrDenied
	}
	return serial, nil
}

func exactHost(host string) bool {
	return host == "api.github.com" || host == "api.openai.com"
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
