package discord

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestName(t *testing.T) {
	a := New("token")
	if a.Name() != "discord" {
		t.Errorf("Name = %q, want %q", a.Name(), "discord")
	}
}

func TestSendMissingChannel(t *testing.T) {
	a := New("token")
	err := a.Send(t.Context(), types.Reply{Text: "hello", ChannelMeta: nil})
	if err == nil {
		t.Error("expected error for missing channel_id, got nil")
	}
}

// ---------------------------------------------------------------------------
// fetchGatewayURL — plain REST GET, mocked via a custom RoundTripper so no
// real network call to discord.com ever happens.
// ---------------------------------------------------------------------------

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchGatewayURLSuccess(t *testing.T) {
	a := New("tok123")
	a.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if want := RESTBaseURL + "/gateway"; req.URL.String() != want {
			t.Errorf("url = %s, want %s", req.URL.String(), want)
		}
		if got := req.Header.Get("Authorization"); got != "Bot tok123" {
			t.Errorf("Authorization = %q, want %q", got, "Bot tok123")
		}
		body := io.NopCloser(strings.NewReader(`{"url":"wss://gateway.discord.gg"}`))
		return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
	})

	got, err := a.fetchGatewayURL(context.Background())
	if err != nil {
		t.Fatalf("fetchGatewayURL: %v", err)
	}
	if got != "wss://gateway.discord.gg" {
		t.Errorf("url = %q, want %q", got, "wss://gateway.discord.gg")
	}
}

func TestFetchGatewayURLTransportError(t *testing.T) {
	a := New("tok123")
	a.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})

	if _, err := a.fetchGatewayURL(context.Background()); err == nil {
		t.Error("expected error when the underlying transport fails, got nil")
	}
}

// TestFetchGatewayURLBadJSON characterizes existing (lenient) behavior:
// fetchGatewayURL neither checks the HTTP status code nor propagates a JSON
// decode failure — both are silently swallowed and it returns "", nil.
func TestFetchGatewayURLBadJSON(t *testing.T) {
	a := New("tok123")
	a.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := io.NopCloser(strings.NewReader(`not json`))
		return &http.Response{StatusCode: 500, Body: body, Header: make(http.Header)}, nil
	})

	got, err := a.fetchGatewayURL(context.Background())
	if err != nil {
		t.Fatalf("expected no error (decode failure is swallowed), got %v", err)
	}
	if got != "" {
		t.Errorf("url = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// dialWebSocket — only the pre-TLS-verification branches are reachable in a
// test: url.Parse failure and TCP dial failure. The handshake-write,
// handshake-read and non-101-status branches all happen only after a
// successful TLS certificate verification, and dialWebSocket's tls.Config has
// no RootCAs/InsecureSkipVerify override point, so no test-controlled
// self-signed server can ever reach them without a production code change.
// ---------------------------------------------------------------------------

func TestDialWebSocketInvalidURL(t *testing.T) {
	_, err := dialWebSocket(context.Background(), "http://example.com/%zz")
	if err == nil || !strings.Contains(err.Error(), "discord: parse gateway url") {
		t.Fatalf("err = %v, want wrapped parse error", err)
	}
}

func TestDialWebSocketConnRefused(t *testing.T) {
	// Bind then immediately close to obtain a port nothing is listening on.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = dialWebSocket(ctx, "wss://"+addr)
	if err == nil || !strings.Contains(err.Error(), "discord: dial") {
		t.Fatalf("err = %v, want wrapped dial error", err)
	}
}

// ---------------------------------------------------------------------------
// readWSFrame / readGatewayPayload — pure decoding, no socket needed at all.
// ---------------------------------------------------------------------------

// buildServerFrame encodes payload as an unmasked server->client WS frame,
// mirroring the wire format readWSFrame expects.
func buildServerFrame(payload []byte) []byte {
	var frame []byte
	frame = append(frame, 0x81) // FIN + text opcode
	switch {
	case len(payload) < 126:
		frame = append(frame, byte(len(payload)))
	case len(payload) < 65536:
		frame = append(frame, 126, byte(len(payload)>>8), byte(len(payload)))
	default:
		frame = append(frame, 127)
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte(len(payload)>>(8*i)))
		}
	}
	return append(frame, payload...)
}

func TestReadWSFrameSmall(t *testing.T) {
	br := bufio.NewReader(bytes.NewReader(buildServerFrame([]byte("hi"))))
	got, err := readWSFrame(br)
	if err != nil {
		t.Fatalf("readWSFrame: %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("got %q, want %q", got, "hi")
	}
}

func TestReadWSFrameExtended16(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 200) // forces the 126 extended-length branch
	br := bufio.NewReader(bytes.NewReader(buildServerFrame(payload)))
	got, err := readWSFrame(br)
	if err != nil {
		t.Fatalf("readWSFrame: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("length = %d, want %d", len(got), len(payload))
	}
}

func TestReadWSFrameExtended64(t *testing.T) {
	payload := bytes.Repeat([]byte("y"), 70000) // forces the 127 extended-length branch
	br := bufio.NewReader(bytes.NewReader(buildServerFrame(payload)))
	got, err := readWSFrame(br)
	if err != nil {
		t.Fatalf("readWSFrame: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("length = %d, want %d", len(got), len(payload))
	}
}

func TestReadWSFrameTruncated(t *testing.T) {
	full := buildServerFrame([]byte("truncate me"))
	br := bufio.NewReader(bytes.NewReader(full[:len(full)-3]))
	if _, err := readWSFrame(br); err == nil {
		t.Error("expected error for truncated frame, got nil")
	}
}

func TestReadWSFrameEmptyReader(t *testing.T) {
	br := bufio.NewReader(bytes.NewReader(nil))
	if _, err := readWSFrame(br); err == nil {
		t.Error("expected error for empty reader, got nil")
	}
}

func TestReadGatewayPayloadValid(t *testing.T) {
	seq := 7
	want := gatewayPayload{Op: gatewayOpDispatch, T: "MESSAGE_CREATE", S: &seq, D: json.RawMessage(`{"content":"hi"}`)}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	br := bufio.NewReader(bytes.NewReader(buildServerFrame(raw)))

	got, err := readGatewayPayload(br)
	if err != nil {
		t.Fatalf("readGatewayPayload: %v", err)
	}
	if got.Op != want.Op || got.T != want.T || got.S == nil || *got.S != *want.S {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestReadGatewayPayloadInvalidJSON(t *testing.T) {
	br := bufio.NewReader(bytes.NewReader(buildServerFrame([]byte("not json"))))
	_, err := readGatewayPayload(br)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "discord: unmarshal gateway") {
		t.Errorf("err = %v, want wrapped unmarshal error", err)
	}
}

func TestReadGatewayPayloadFrameError(t *testing.T) {
	br := bufio.NewReader(bytes.NewReader(nil))
	if _, err := readGatewayPayload(br); err == nil {
		t.Error("expected error when the underlying frame read fails")
	}
}

// ---------------------------------------------------------------------------
// writeWSFrame / writeGatewayFrame / sendHeartbeat / heartbeat — these take a
// concrete *tls.Conn, so a real (but local, in-memory) TLS connection is
// built over a net.Pipe with a throwaway self-signed cert. This is a genuine
// *tls.Conn, just not one obtained via the network dial in dialWebSocket.
// ---------------------------------------------------------------------------

func genSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv}
}

// tlsPipe returns a handshaked (client, server) *tls.Conn pair connected via
// an in-memory net.Pipe.
func tlsPipe(t *testing.T) (client, server *tls.Conn) {
	t.Helper()
	client, server, _, _ = tlsPipeRaw(t)
	return client, server
}

// tlsPipeRaw returns a handshaked TLS pair and its underlying pipes so a test
// can interrupt a blocked gateway read without waiting for TLS close_notify.
func tlsPipeRaw(t *testing.T) (client, server *tls.Conn, clientRaw, serverRaw net.Conn) {
	t.Helper()
	cert := genSelfSignedCert(t)
	clientRaw, serverRaw = net.Pipe()

	serverConn := tls.Server(serverRaw, &tls.Config{Certificates: []tls.Certificate{cert}})
	clientConn := tls.Client(clientRaw, &tls.Config{InsecureSkipVerify: true}) // #nosec G402 -- test-only pipe, not network-facing

	serverErrCh := make(chan error, 1)
	go func() { serverErrCh <- serverConn.Handshake() }()
	if err := clientConn.Handshake(); err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	if err := <-serverErrCh; err != nil {
		t.Fatalf("server handshake: %v", err)
	}

	// Close the underlying raw pipes rather than the *tls.Conn: tls.Conn.Close
	// sends a close_notify alert under a 5s write deadline, which stalls every
	// test for 5s since nothing reads it back over net.Pipe once the test body
	// is done checking the frame.
	t.Cleanup(func() {
		_ = clientRaw.Close()
		_ = serverRaw.Close()
	})
	return clientConn, serverConn, clientRaw, serverRaw
}

func writeServerGatewayPayload(t *testing.T, conn net.Conn, payload gatewayPayload) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal gateway payload: %v", err)
	}
	if _, err := conn.Write(buildServerFrame(raw)); err != nil {
		t.Fatalf("write server gateway payload: %v", err)
	}
}

func readClientGatewayPayload(t *testing.T, conn net.Conn) gatewayPayload {
	t.Helper()
	var payload gatewayPayload
	if err := json.Unmarshal(readRawMaskedFrame(t, conn), &payload); err != nil {
		t.Fatalf("unmarshal client gateway payload: %v", err)
	}
	return payload
}

func TestGatewayLoopDispatchesAndFiltersEvents(t *testing.T) {
	client, server, _, serverRaw := tlsPipeRaw(t)
	a := New("tok123")
	a.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"url":"wss://gateway.test"}`)),
			Header:     make(http.Header),
		}, nil
	})
	a.dialWebSocket = func(context.Context, string) (*tls.Conn, error) { return client, nil }

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ch, err := a.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	writeServerGatewayPayload(t, server, gatewayPayload{
		Op: 10,
		D:  json.RawMessage(`{"heartbeat_interval":60000}`),
	})
	identify := readClientGatewayPayload(t, server)
	if identify.Op != gatewayOpIdentify {
		t.Fatalf("identify opcode = %d, want %d", identify.Op, gatewayOpIdentify)
	}

	writeServerGatewayPayload(t, server, gatewayPayload{
		Op: gatewayOpDispatch,
		T:  "MESSAGE_CREATE",
		D:  json.RawMessage(`{"id":"ignored","channel_id":"channel","author":{"id":"bot","bot":true},"content":"ignored"}`),
	})
	writeServerGatewayPayload(t, server, gatewayPayload{
		Op: gatewayOpDispatch,
		T:  "INTERACTION_CREATE",
		D:  json.RawMessage(`{"id":"ignored","channel_id":"channel","data":{"custom_id":"ignored"},"member":{"user":{"id":"bot","bot":true}}}`),
	})
	writeServerGatewayPayload(t, server, gatewayPayload{
		Op: gatewayOpDispatch,
		T:  "INTERACTION_CREATE",
		D:  json.RawMessage(`{"id":"ignored","channel_id":"channel","data":{},"member":{"user":{"id":"user"}}}`),
	})
	writeServerGatewayPayload(t, server, gatewayPayload{
		Op: gatewayOpDispatch,
		T:  "MESSAGE_CREATE",
		S:  intPtr(7),
		D:  json.RawMessage(`{"id":"message","channel_id":"channel","author":{"id":"user"},"content":"hello"}`),
	})

	select {
	case message := <-ch:
		if message.UserID != "user" || message.Text != "hello" || message.Kind != types.MessageText {
			t.Fatalf("message = %+v, want user text message", message)
		}
		if got := message.ChannelMeta; got["channel_id"] != "channel" || got["message_id"] != "message" {
			t.Fatalf("ChannelMeta = %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message event")
	}

	writeServerGatewayPayload(t, server, gatewayPayload{
		Op: gatewayOpDispatch,
		T:  "INTERACTION_CREATE",
		D:  json.RawMessage(`{"id":"interaction","channel_id":"channel","token":"token","data":{"custom_id":"action"},"member":{"user":{"id":"user"}}}`),
	})
	select {
	case message := <-ch:
		if message.UserID != "user" || message.Text != "action" || message.ChannelMeta["is_callback"] != "true" || message.ChannelMeta["interaction_id"] != "interaction" || message.ChannelMeta["interaction_token"] != "token" {
			t.Fatalf("interaction = %+v", message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for interaction event")
	}

	writeServerGatewayPayload(t, server, gatewayPayload{Op: gatewayOpHeartbeat})
	heartbeat := readClientGatewayPayload(t, server)
	if heartbeat.Op != gatewayOpHeartbeat || string(heartbeat.D) != "7" {
		t.Fatalf("heartbeat = %+v, want opcode %d with sequence 7", heartbeat, gatewayOpHeartbeat)
	}

	_ = serverRaw.Close()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel remains open after the gateway connection closes")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway loop to stop")
	}
}

func intPtr(value int) *int { return &value }

// readRawMaskedFrame parses a single client->server masked WS frame off r,
// verifying the framing bits writeWSFrame is responsible for, and returns the
// unmasked payload.
func readRawMaskedFrame(t *testing.T, r io.Reader) []byte {
	t.Helper()
	br := bufio.NewReader(r)

	b0, err := br.ReadByte()
	if err != nil {
		t.Fatalf("read b0: %v", err)
	}
	if b0 != 0x81 {
		t.Fatalf("b0 = %#x, want 0x81 (FIN + text opcode)", b0)
	}

	b1, err := br.ReadByte()
	if err != nil {
		t.Fatalf("read b1: %v", err)
	}
	if b1&0x80 == 0 {
		t.Fatalf("mask bit not set in b1 = %#x", b1)
	}

	length := int64(b1 & 0x7f)
	switch length {
	case 126:
		var buf [2]byte
		if _, err := io.ReadFull(br, buf[:]); err != nil {
			t.Fatalf("read extended length: %v", err)
		}
		length = int64(buf[0])<<8 | int64(buf[1])
	case 127:
		var buf [8]byte
		if _, err := io.ReadFull(br, buf[:]); err != nil {
			t.Fatalf("read extended length: %v", err)
		}
		length = 0
		for _, b := range buf {
			length = length<<8 | int64(b)
		}
	}

	var maskKey [4]byte
	if _, err := io.ReadFull(br, maskKey[:]); err != nil {
		t.Fatalf("read mask key: %v", err)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(br, data); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	for i := range data {
		data[i] ^= maskKey[i%4]
	}
	return data
}

func TestWriteWSFrameSmallPayload(t *testing.T) {
	client, server := tlsPipe(t)
	want := []byte("hello gateway")

	writeErrCh := make(chan error, 1)
	go func() { writeErrCh <- writeWSFrame(client, want) }()

	got := readRawMaskedFrame(t, server)
	if err := <-writeErrCh; err != nil {
		t.Fatalf("writeWSFrame: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("payload = %q, want %q", got, want)
	}
}

func TestWriteWSFrameExtended16Payload(t *testing.T) {
	client, server := tlsPipe(t)
	want := bytes.Repeat([]byte("a"), 200) // forces the 126 extended-length branch

	writeErrCh := make(chan error, 1)
	go func() { writeErrCh <- writeWSFrame(client, want) }()

	got := readRawMaskedFrame(t, server)
	if err := <-writeErrCh; err != nil {
		t.Fatalf("writeWSFrame: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("payload length = %d, want %d", len(got), len(want))
	}
}

func TestWriteWSFrameExtended64Payload(t *testing.T) {
	client, server := tlsPipe(t)
	want := bytes.Repeat([]byte("b"), 70000) // forces the 127 extended-length branch

	writeErrCh := make(chan error, 1)
	go func() { writeErrCh <- writeWSFrame(client, want) }()

	got := readRawMaskedFrame(t, server)
	if err := <-writeErrCh; err != nil {
		t.Fatalf("writeWSFrame: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("payload length = %d, want %d", len(got), len(want))
	}
}

func TestWriteGatewayFrame(t *testing.T) {
	client, server := tlsPipe(t)
	pl := gatewayPayload{Op: gatewayOpIdentify, D: json.RawMessage(`{"foo":"bar"}`)}

	writeErrCh := make(chan error, 1)
	go func() { writeErrCh <- writeGatewayFrame(client, pl) }()

	raw := readRawMaskedFrame(t, server)
	if err := <-writeErrCh; err != nil {
		t.Fatalf("writeGatewayFrame: %v", err)
	}

	var got gatewayPayload
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Op != gatewayOpIdentify || string(got.D) != `{"foo":"bar"}` {
		t.Errorf("got %+v", got)
	}
}

func TestSendHeartbeatWithSeq(t *testing.T) {
	client, server := tlsPipe(t)
	a := New("tok")
	seq := 42

	go a.sendHeartbeat(client, &seq)

	raw := readRawMaskedFrame(t, server)
	var got gatewayPayload
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Op != gatewayOpHeartbeat {
		t.Errorf("op = %d, want %d", got.Op, gatewayOpHeartbeat)
	}
	if string(got.D) != "42" {
		t.Errorf("d = %s, want 42", got.D)
	}
}

func TestSendHeartbeatNilSeq(t *testing.T) {
	client, server := tlsPipe(t)
	a := New("tok")

	go a.sendHeartbeat(client, nil)

	raw := readRawMaskedFrame(t, server)
	var got gatewayPayload
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Op != gatewayOpHeartbeat {
		t.Errorf("op = %d, want %d", got.Op, gatewayOpHeartbeat)
	}
	if string(got.D) != "null" {
		t.Errorf("d = %s, want null (no seq marshaled)", got.D)
	}
}

func TestHeartbeatSendsOnInterval(t *testing.T) {
	client, server := tlsPipe(t)
	a := New("tok")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.heartbeat(ctx, client, 10) // 10ms interval

	raw := readRawMaskedFrame(t, server)
	var got gatewayPayload
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Op != gatewayOpHeartbeat {
		t.Errorf("op = %d, want %d", got.Op, gatewayOpHeartbeat)
	}
}

func TestHeartbeatZeroIntervalReturnsImmediately(t *testing.T) {
	a := New("tok")
	done := make(chan struct{})
	go func() {
		a.heartbeat(context.Background(), nil, 0)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat with intervalMs<=0 did not return promptly")
	}
}
