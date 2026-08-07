package mailer

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strconv"
	"testing"
	"time"
)

const (
	testFrom = "from@example.com"
	testTo   = "to@example.com"
	testHost = "127.0.0.1"
	testAddr = "127.0.0.1:0"

	errListen        = "listen: %v"
	errSplitHostPort = "split host port: %v"
	errParsePort     = "parse port: %v"
)

// listenTCP starts a plain (non-TLS) TCP listener on an ephemeral port and
// returns its host/port split for newSMTP.
func listenTCP(t *testing.T) (net.Listener, string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", testAddr)
	if err != nil {
		t.Fatalf(errListen, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	host, port := splitHostPort(t, ln.Addr().String())
	return ln, host, port
}

func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf(errSplitHostPort, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf(errParsePort, err)
	}
	return host, port
}

// selfSignedCert generates a throwaway cert for a local TLS test server.
// insecureSkipVerify on the client side means SANs don't need to be exact,
// but 127.0.0.1 is included anyway for a realistic handshake.
func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: testHost},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP(testHost)},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// smtpTestListener starts a TLS listener with a throwaway cert and returns
// its host/port split for newSMTP, plus the listener for the caller to
// Accept on.
func smtpTestListener(t *testing.T) (net.Listener, string, int) {
	t.Helper()
	cert := selfSignedCert(t)
	ln, err := tls.Listen("tcp", testAddr, &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf(errListen, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	host, port := splitHostPort(t, ln.Addr().String())
	return ln, host, port
}

// smtpInsecure builds a smtpMailer that skips cert verification, for
// pointing at smtpTestListener's self-signed server.
func smtpInsecure(from, host string, port int, username, password string) *smtpMailer {
	m := newSMTP(from, host, port, username, password, true)
	m.insecureSkipVerify = true
	return m
}

// smtpReadUntilDot drains an SMTP DATA body up to and including the
// terminating "." line.
func smtpReadUntilDot(r *bufio.Reader) error {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		if line == ".\r\n" || line == ".\n" {
			return nil
		}
	}
}

// serveFakeSMTP plays a minimal EHLO/AUTH/MAIL/RCPT/DATA conversation over
// an already-accepted (TLS-handshaken) connection. rejectAt, if non-empty,
// is one of "auth", "mail", "rcpt", "data" — the stage at which to reply
// with a 5xx instead of continuing.
func serveFakeSMTP(conn net.Conn, auth bool, rejectAt string) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	w := func(s string) bool { _, err := conn.Write([]byte(s)); return err == nil }
	readLine := func() bool { _, err := r.ReadString('\n'); return err == nil }

	if !w("220 test.local ESMTP\r\n") || !readLine() || !w("250 test.local\r\n") { // EHLO
		return
	}
	if auth && !serveFakeSMTPAuth(w, readLine, rejectAt) {
		return
	}
	if !serveFakeSMTPEnvelope(w, readLine, rejectAt) {
		return
	}
	serveFakeSMTPData(r, w)
}

func serveFakeSMTPAuth(w func(string) bool, readLine func() bool, rejectAt string) bool {
	if !readLine() { // AUTH PLAIN ...
		return false
	}
	if rejectAt == "auth" {
		w("535 5.7.8 authentication failed\r\n")
		return false
	}
	return w("235 2.7.0 Authentication successful\r\n")
}

func serveFakeSMTPEnvelope(w func(string) bool, readLine func() bool, rejectAt string) bool {
	if !readLine() { // MAIL FROM
		return false
	}
	if rejectAt == "mail" {
		w("550 5.1.0 mailbox unavailable\r\n")
		return false
	}
	if !w("250 2.1.0 OK\r\n") || !readLine() { // RCPT TO
		return false
	}
	if rejectAt == "rcpt" {
		w("550 5.1.1 no such user\r\n")
		return false
	}
	if !w("250 2.1.5 OK\r\n") || !readLine() { // DATA
		return false
	}
	if rejectAt == "data" {
		w("550 5.7.1 data not allowed\r\n")
		return false
	}
	return true
}

func serveFakeSMTPData(r *bufio.Reader, w func(string) bool) {
	if !w("354 Go ahead\r\n") {
		return
	}
	if err := smtpReadUntilDot(r); err != nil {
		return
	}
	w("250 2.0.0 OK: queued\r\n")
}

// TestSMTPSendRespectsContextDeadline is the #271 regression test: a server
// that accepts the TCP connection but never responds must not hang Send
// indefinitely — it should return once the context deadline passes.
func TestSMTPSendRespectsContextDeadline(t *testing.T) {
	ln, host, port := listenTCP(t)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Black hole: accept the connection and never respond or close.
		<-t.Context().Done()
		_ = conn.Close()
	}()

	m := newSMTP(testFrom, host, port, "", "", true)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- m.Send(ctx, testTo, Message{Subject: "s", TextBody: "b"})
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error from unresponsive server, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Send did not return within 2s of a 50ms context deadline — it hung")
	}
}

func TestSMTPSendSuccess(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   string
		noDeadline bool // exercises ctx.Deadline()'s !ok branch
	}{
		{name: "no auth"},
		{name: "with auth", username: "user", password: "pass"},
		{name: "no auth, no context deadline", noDeadline: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ln, host, port := smtpTestListener(t)

			go func() {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				serveFakeSMTP(conn, tt.username != "", "")
			}()

			m := smtpInsecure(testFrom, host, port, tt.username, tt.password)

			ctx := context.Background()
			if !tt.noDeadline {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
			}

			if err := m.Send(ctx, testTo, Message{Subject: "s", TextBody: "b"}); err != nil {
				t.Errorf("Send: %v", err)
			}
		})
	}
}

func TestSMTPSendRejected(t *testing.T) {
	tests := []struct {
		name     string
		auth     bool
		rejectAt string
	}{
		{name: "auth rejected", auth: true, rejectAt: "auth"},
		{name: "mail from rejected", rejectAt: "mail"},
		{name: "rcpt to rejected", rejectAt: "rcpt"},
		{name: "data rejected", rejectAt: "data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ln, host, port := smtpTestListener(t)

			go func() {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				serveFakeSMTP(conn, tt.auth, tt.rejectAt)
			}()

			username, password := "", ""
			if tt.auth {
				username, password = "user", "pass"
			}
			m := smtpInsecure(testFrom, host, port, username, password)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := m.Send(ctx, testTo, Message{Subject: "s", TextBody: "b"}); err == nil {
				t.Fatalf("expected error from rejected %s, got nil", tt.rejectAt)
			}
		})
	}
}

func TestSMTPSendHandshakeFailure(t *testing.T) {
	// A plain (non-TLS) listener that closes the connection immediately
	// after accept simulates a server that never completes the TLS
	// handshake, exercising the HandshakeContext error branch.
	ln, host, port := listenTCP(t)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
	}()

	m := smtpInsecure(testFrom, host, port, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := m.Send(ctx, testTo, Message{Subject: "s", TextBody: "b"}); err == nil {
		t.Fatal("expected TLS handshake error, got nil")
	}
}

func TestSMTPSendDialError(t *testing.T) {
	// Port 1 with no listener forces a quick dial failure (connection
	// refused) and confirms error wrapping.
	m := newSMTP(testFrom, testHost, 1, "", "", true)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := m.Send(ctx, testTo, Message{Subject: "s", TextBody: "b"}); err == nil {
		t.Error("expected error dialing unreachable port, got nil")
	}
}
