package proxyca

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type mutableClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *mutableClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mutableClock) Advance(now time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

var fixtureNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func fixtureCA(t *testing.T, now time.Time, remaining time.Duration) ([]byte, []byte, *x509.Certificate) {
	return fixtureCAWith(t, now, remaining, nil)
}

func fixtureCAWith(t *testing.T, now time.Time, remaining time.Duration, modify func(*x509.Certificate)) ([]byte, []byte, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "fixture-ca"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(remaining), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign}
	if modify != nil {
		modify(template)
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certPEM, keyPEM, cert
}

func newAuthority(t *testing.T) (*Authority, *x509.Certificate, []byte, []byte) {
	t.Helper()
	certPEM, keyPEM, cert := fixtureCA(t, fixtureNow, time.Hour)
	authority, err := New(Rules{CACertificatePEM: certPEM, CAPrivateKeyPEM: keyPEM, Clock: fixedClock{fixtureNow}})
	if err != nil {
		t.Fatal(err)
	}
	return authority, cert, certPEM, keyPEM
}

func TestNewValidationCopyAndNonLeak(t *testing.T) {
	certPEM, keyPEM, _ := fixtureCA(t, fixtureNow, time.Hour)
	base := Rules{CACertificatePEM: certPEM, CAPrivateKeyPEM: keyPEM, Clock: fixedClock{fixtureNow}}
	_, mismatchKeyPEM, _ := fixtureCA(t, fixtureNow, time.Hour)
	parsedKeyDER, _ := pem.Decode(keyPEM)
	parsedKey, err := x509.ParseECPrivateKey(parsedKeyDER.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(parsedKey)
	if err != nil {
		t.Fatal(err)
	}
	encryptedMarker := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Headers: map[string]string{"Proc-Type": "4,ENCRYPTED"}, Bytes: parsedKeyDER.Bytes})
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p384DER, err := x509.MarshalECPrivateKey(p384)
	if err != nil {
		t.Fatal(err)
	}
	p384PEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: p384DER})
	mutatedSignature := func() []byte {
		block, _ := pem.Decode(certPEM)
		block.Bytes[len(block.Bytes)-1] ^= 0xff
		return pem.EncodeToMemory(block)
	}()
	for name, rules := range map[string]Rules{
		"zero":    {},
		"no cert": {CAPrivateKeyPEM: keyPEM, Clock: fixedClock{fixtureNow}},
		"no key":  {CACertificatePEM: certPEM, Clock: fixedClock{fixtureNow}},
		"short ca": func() Rules {
			c, k, _ := fixtureCA(t, fixtureNow, leafLifetime-time.Second)
			return Rules{CACertificatePEM: c, CAPrivateKeyPEM: k, Clock: fixedClock{fixtureNow}}
		}(),
		"trailing cert": func() Rules { r := base; r.CACertificatePEM = append(append([]byte(nil), certPEM...), '\n'); return r }(),
		"trailing key":  func() Rules { r := base; r.CAPrivateKeyPEM = append(append([]byte(nil), keyPEM...), '\n'); return r }(),
		"multiple cert": func() Rules {
			r := base
			r.CACertificatePEM = append(append([]byte(nil), certPEM...), certPEM...)
			return r
		}(),
		"encrypted marker": {CACertificatePEM: certPEM, CAPrivateKeyPEM: encryptedMarker, Clock: fixedClock{fixtureNow}},
		"not yet valid": func() Rules {
			c, k, _ := fixtureCAWith(t, fixtureNow, time.Hour, func(v *x509.Certificate) {
				v.NotBefore = fixtureNow.Add(time.Hour)
				v.NotAfter = fixtureNow.Add(2 * time.Hour)
			})
			return Rules{CACertificatePEM: c, CAPrivateKeyPEM: k, Clock: fixedClock{fixtureNow}}
		}(),
		"expired": func() Rules {
			c, k, _ := fixtureCAWith(t, fixtureNow, -time.Second, func(v *x509.Certificate) {
				v.NotBefore = fixtureNow.Add(-time.Hour)
				v.NotAfter = fixtureNow.Add(-time.Second)
			})
			return Rules{CACertificatePEM: c, CAPrivateKeyPEM: k, Clock: fixedClock{fixtureNow}}
		}(),
		"not ca": func() Rules {
			c, k, _ := fixtureCAWith(t, fixtureNow, time.Hour, func(v *x509.Certificate) { v.IsCA = false })
			return Rules{CACertificatePEM: c, CAPrivateKeyPEM: k, Clock: fixedClock{fixtureNow}}
		}(),
		"basic constraints": func() Rules {
			c, k, _ := fixtureCAWith(t, fixtureNow, time.Hour, func(v *x509.Certificate) { v.BasicConstraintsValid = false })
			return Rules{CACertificatePEM: c, CAPrivateKeyPEM: k, Clock: fixedClock{fixtureNow}}
		}(),
		"cert sign missing": func() Rules {
			c, k, _ := fixtureCAWith(t, fixtureNow, time.Hour, func(v *x509.Certificate) { v.KeyUsage = 0 })
			return Rules{CACertificatePEM: c, CAPrivateKeyPEM: k, Clock: fixedClock{fixtureNow}}
		}(),
		"bad signature":  {CACertificatePEM: mutatedSignature, CAPrivateKeyPEM: keyPEM, Clock: fixedClock{fixtureNow}},
		"mismatched key": {CACertificatePEM: certPEM, CAPrivateKeyPEM: mismatchKeyPEM, Clock: fixedClock{fixtureNow}},
		"pkcs8 encoding": {CACertificatePEM: certPEM, CAPrivateKeyPEM: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), Clock: fixedClock{fixtureNow}},
		"p384 key":       {CACertificatePEM: certPEM, CAPrivateKeyPEM: p384PEM, Clock: fixedClock{fixtureNow}},
	} {
		t.Run(name, func(t *testing.T) {
			if authority, err := New(rules); authority != nil || !errors.Is(err, ErrInvalidRules) || strings.Contains(fmt.Sprintf("%+v", err), "CERTIFICATE") {
				t.Fatalf("authority=%p error=%v", authority, err)
			}
		})
	}
	var nilClock *fixedClock
	if authority, err := New(Rules{CACertificatePEM: certPEM, CAPrivateKeyPEM: keyPEM, Clock: nilClock}); authority != nil || !errors.Is(err, ErrInvalidRules) {
		t.Fatalf("typed nil clock authority=%p error=%v", authority, err)
	}
	authority, _, inputCert, inputKey := newAuthority(t)
	inputCert[0] ^= 0xff
	inputKey[0] ^= 0xff
	public := authority.PublicCertificatePEM()
	if len(public) == 0 || !bytes.Equal(public, authority.PublicCertificatePEM()) {
		t.Fatal("public certificate missing or unstable")
	}
	public[0] ^= 0xff
	if bytes.Equal(public, authority.PublicCertificatePEM()) {
		t.Fatal("public certificate aliases authority state")
	}
	block, rest := pem.Decode(authority.PublicCertificatePEM())
	if block == nil || block.Type != "CERTIFICATE" || len(rest) != 0 {
		t.Fatalf("public PEM blocks type=%v rest=%d", block, len(rest))
	}
	if got := fmt.Sprintf("%+v", authority); got != "proxyca.Authority" || strings.Contains(got, "fixture-ca") {
		t.Fatalf("authority format=%q", got)
	}
	var zero Authority
	if zero.PublicCertificatePEM() != nil {
		t.Fatal("zero authority returned public certificate")
	}
	if _, err := zero.Issue("api.github.com"); !errors.Is(err, ErrDenied) {
		t.Fatalf("zero Issue error=%v", err)
	}
}

func TestIssueExtensionsChainValidityAndHostGate(t *testing.T) {
	authority, ca, _, _ := newAuthority(t)
	seenSerials := make(map[string]struct{})
	seenKeys := make(map[string]struct{})
	for _, host := range []string{"api.github.com", "api.openai.com"} {
		certificate, err := authority.Issue(host)
		if err != nil {
			t.Fatal(err)
		}
		if certificate.Leaf == nil || len(certificate.Certificate) != 2 || !bytes.Equal(certificate.Certificate[1], ca.Raw) {
			t.Fatalf("chain=%d leaf=%v", len(certificate.Certificate), certificate.Leaf)
		}
		leaf := certificate.Leaf
		privateKey, privateOK := certificate.PrivateKey.(*ecdsa.PrivateKey)
		leafPublic, publicOK := leaf.PublicKey.(*ecdsa.PublicKey)
		if leaf.Subject.CommonName != "" || len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != host || leaf.IsCA || !leaf.BasicConstraintsValid ||
			leaf.KeyUsage != x509.KeyUsageDigitalSignature || len(leaf.ExtKeyUsage) != 1 || leaf.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth ||
			len(leaf.IPAddresses) != 0 || len(leaf.EmailAddresses) != 0 || len(leaf.URIs) != 0 || leaf.SerialNumber.Sign() == 0 ||
			leaf.SerialNumber.BitLen() > 128 || !privateOK || !publicOK || privateKey.Curve != elliptic.P256() || privateKey.PublicKey.X.Cmp(leafPublic.X) != 0 || privateKey.PublicKey.Y.Cmp(leafPublic.Y) != 0 ||
			leaf.NotBefore.Before(fixtureNow.Add(-5*time.Minute)) || !leaf.NotBefore.Before(fixtureNow) || leaf.NotAfter.After(fixtureNow.Add(leafLifetime)) || leaf.NotAfter.After(ca.NotAfter) {
			t.Fatalf("leaf=%#v", leaf)
		}
		serial := leaf.SerialNumber.String()
		key := privateKey.PublicKey.X.String() + ":" + privateKey.PublicKey.Y.String()
		if _, ok := seenSerials[serial]; ok {
			t.Fatal("duplicate serial")
		}
		if _, ok := seenKeys[key]; ok {
			t.Fatal("duplicate leaf key")
		}
		seenSerials[serial], seenKeys[key] = struct{}{}, struct{}{}
	}
	for _, host := range []string{"", "API.GITHUB.COM", "api.github.com.", "api.github.com:443", "*.github.com", "127.0.0.1", "api.example.com", "api.github.com\n"} {
		if certificate, err := authority.Issue(host); certificate.Certificate != nil || !errors.Is(err, ErrDenied) || (host != "" && strings.Contains(err.Error(), host)) {
			t.Fatalf("host=%q certificate=%#v error=%v", host, certificate, err)
		}
	}
}

func TestIssueRejectsClockAfterCAExpiry(t *testing.T) {
	certPEM, keyPEM, _ := fixtureCA(t, fixtureNow, time.Hour)
	clock := &mutableClock{now: fixtureNow}
	authority, err := New(Rules{CACertificatePEM: certPEM, CAPrivateKeyPEM: keyPEM, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(fixtureNow.Add(2 * time.Hour))
	certificate, err := authority.Issue("api.github.com")
	if certificate.Certificate != nil || certificate.PrivateKey != nil || certificate.Leaf != nil || !errors.Is(err, ErrDenied) {
		t.Fatalf("expired CA issue certificate=%#v error=%v", certificate, err)
	}
}

func TestTLSNetPipeHostnameVerification(t *testing.T) {
	authority, _, caPEM, _ := newAuthority(t)
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("failed to append CA")
	}
	for _, host := range []string{"api.github.com", "api.openai.com"} {
		t.Run(host, func(t *testing.T) {
			certificate, err := authority.Issue(host)
			if err != nil {
				t.Fatal(err)
			}
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			deadline := time.Now().Add(3 * time.Second)
			_ = clientConn.SetDeadline(deadline)
			_ = serverConn.SetDeadline(deadline)
			server := tls.Server(serverConn, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}, Time: func() time.Time { return fixtureNow }})
			client := tls.Client(clientConn, &tls.Config{RootCAs: pool, ServerName: host, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}, Time: func() time.Time { return fixtureNow }})
			serverErr := make(chan error, 1)
			go func() { serverErr <- server.Handshake() }()
			if err := client.Handshake(); err != nil {
				t.Fatal(err)
			}
			if err := <-serverErr; err != nil {
				t.Fatal(err)
			}
			if got := client.ConnectionState().NegotiatedProtocol; got != "http/1.1" {
				t.Fatalf("negotiated protocol=%q", got)
			}
			client.Close()
			server.Close()
		})
	}
	certificate, err := authority.Issue("api.github.com")
	if err != nil {
		t.Fatal(err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	deadline := time.Now().Add(3 * time.Second)
	_ = clientConn.SetDeadline(deadline)
	_ = serverConn.SetDeadline(deadline)
	server := tls.Server(serverConn, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12, Time: func() time.Time { return fixtureNow }})
	client := tls.Client(clientConn, &tls.Config{RootCAs: pool, ServerName: "api.openai.com", MinVersion: tls.VersionTLS12, Time: func() time.Time { return fixtureNow }})
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Handshake() }()
	if err := client.Handshake(); err == nil {
		t.Fatal("wrong hostname unexpectedly verified")
	}
	client.Close()
	server.Close()
	<-serverErr
}

func TestConcurrentIssueUniqueness(t *testing.T) {
	authority, _, _, _ := newAuthority(t)
	const count = 16
	serials := make(chan string, count)
	keys := make(chan string, count)
	hostSANs := make(chan string, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			host := "api.github.com"
			if i%2 != 0 {
				host = "api.openai.com"
			}
			certificate, err := authority.Issue(host)
			if err != nil {
				t.Errorf("Issue error=%v", err)
				return
			}
			key := certificate.PrivateKey.(*ecdsa.PrivateKey)
			serials <- certificate.Leaf.SerialNumber.String()
			keys <- key.PublicKey.X.String() + ":" + key.PublicKey.Y.String()
			hostSANs <- host + "|" + certificate.Leaf.DNSNames[0]
		}(i)
	}
	wg.Wait()
	close(serials)
	close(keys)
	close(hostSANs)
	seenSerials, seenKeys := map[string]struct{}{}, map[string]struct{}{}
	for serial := range serials {
		if _, ok := seenSerials[serial]; ok {
			t.Fatal("concurrent duplicate serial")
		}
		seenSerials[serial] = struct{}{}
	}
	for key := range keys {
		if _, ok := seenKeys[key]; ok {
			t.Fatal("concurrent duplicate key")
		}
		seenKeys[key] = struct{}{}
	}
	if len(seenSerials) != count || len(seenKeys) != count {
		t.Fatalf("unique serials=%d keys=%d", len(seenSerials), len(seenKeys))
	}
	seenHosts := map[string]struct{}{}
	for pair := range hostSANs {
		parts := strings.Split(pair, "|")
		if len(parts) != 2 || parts[0] != parts[1] {
			t.Fatalf("cross-host SAN=%q", pair)
		}
		seenHosts[pair] = struct{}{}
	}
	if len(seenHosts) != 2 {
		t.Fatalf("host/SAN coverage=%d", len(seenHosts))
	}
}
