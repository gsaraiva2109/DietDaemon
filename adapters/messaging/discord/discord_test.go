package discord

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"golang.org/x/net/websocket"
)

func TestName(t *testing.T) {
	if got := New("token").Name(); got != "discord" {
		t.Errorf("Name = %q, want discord", got)
	}
}

func TestSendMissingChannel(t *testing.T) {
	if err := New("token").Send(t.Context(), types.Reply{Text: "hello"}); err == nil {
		t.Error("Send without channel_id returned nil error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestFetchGatewayURLSuccess(t *testing.T) {
	a := New("tok123")
	a.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if want := RESTBaseURL + "/gateway"; req.URL.String() != want {
			t.Errorf("URL = %s, want %s", req.URL, want)
		}
		if got := req.Header.Get("Authorization"); got != "Bot tok123" {
			t.Errorf("Authorization = %q, want Bot tok123", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"url":"wss://gateway.discord.gg"}`)), Header: make(http.Header)}, nil
	})

	got, err := a.fetchGatewayURL(t.Context())
	if err != nil || got != "wss://gateway.discord.gg" {
		t.Fatalf("fetchGatewayURL = %q, %v", got, err)
	}
}

func TestFetchGatewayURLTransportError(t *testing.T) {
	a := New("tok123")
	a.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("boom") })
	if _, err := a.fetchGatewayURL(t.Context()); err == nil {
		t.Error("fetchGatewayURL returned nil error")
	}
}

func TestFetchGatewayURLBadJSONPreservesLenientBehavior(t *testing.T) {
	a := New("tok123")
	a.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("not json")), Header: make(http.Header)}, nil
	})

	got, err := a.fetchGatewayURL(t.Context())
	if err != nil || got != "" {
		t.Fatalf("fetchGatewayURL = %q, %v; want empty URL and nil error", got, err)
	}
}

func TestDialWebSocketRejectsInvalidURL(t *testing.T) {
	if _, err := dialWebSocket(t.Context(), "http://example.com/%zz"); err == nil || !strings.Contains(err.Error(), "discord: configure gateway") {
		t.Fatalf("dialWebSocket error = %v", err)
	}
}

func TestDialWebSocketConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, err := dialWebSocket(ctx, "wss://"+address); err == nil || !strings.Contains(err.Error(), "discord: dial") {
		t.Fatalf("dialWebSocket error = %v", err)
	}
}

func TestDialWebSocketWithTLSConfigSucceeds(t *testing.T) {
	server, connections := newGatewayServer(t)
	endpoint := "wss" + strings.TrimPrefix(server.URL, "https")
	result := make(chan error, 1)
	go func() {
		conn, err := dialWebSocketWithTLSConfig(t.Context(), endpoint, &tls.Config{RootCAs: server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs, MinVersion: tls.VersionTLS12})
		if err == nil {
			err = conn.Close()
		}
		result <- err
	}()
	_ = receiveConnection(t, connections)
	if err := <-result; err != nil {
		t.Fatalf("dialWebSocketWithTLSConfig: %v", err)
	}
}

func TestGatewayLoopProtocolAndEvents(t *testing.T) {
	server, connections := newGatewayServer(t)
	a := newGatewayAdapter(t, server)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ch, err := a.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	conn := receiveConnection(t, connections)

	sendGatewayPayload(t, conn, gatewayPayload{Op: 10, D: json.RawMessage(`{"heartbeat_interval":60000}`)})
	identify := receiveGatewayPayload(t, conn)
	if identify.Op != gatewayOpIdentify {
		t.Fatalf("IDENTIFY opcode = %d, want %d", identify.Op, gatewayOpIdentify)
	}
	var identity identifyData
	if err := json.Unmarshal(identify.D, &identity); err != nil {
		t.Fatalf("unmarshal IDENTIFY: %v", err)
	}
	if identity.Token != "tok123" || identity.Intents != gatewayIntentMessageContent || identity.Properties.Browser != "dietdaemon" {
		t.Fatalf("IDENTIFY payload = %+v", identity)
	}

	sendGatewayPayload(t, conn, gatewayPayload{Op: gatewayOpDispatch, T: "MESSAGE_CREATE", D: json.RawMessage(`{"id":"ignored","channel_id":"channel","author":{"id":"bot","bot":true},"content":"ignored"}`)})
	sendGatewayPayload(t, conn, gatewayPayload{Op: gatewayOpDispatch, T: "INTERACTION_CREATE", D: json.RawMessage(`{"id":"ignored","channel_id":"channel","data":{"custom_id":"ignored"},"member":{"user":{"id":"bot","bot":true}}}`)})
	sendGatewayPayload(t, conn, gatewayPayload{Op: gatewayOpDispatch, T: "MESSAGE_CREATE", S: intPtr(7), D: json.RawMessage(`{"id":"message","channel_id":"channel","author":{"id":"user"},"content":"hello"}`)})
	message := receiveInbound(t, ch)
	if message.UserID != "user" || message.Text != "hello" || message.Kind != types.MessageText || message.ChannelMeta["message_id"] != "message" {
		t.Fatalf("message = %+v", message)
	}

	sendGatewayPayload(t, conn, gatewayPayload{Op: gatewayOpDispatch, T: "INTERACTION_CREATE", D: json.RawMessage(`{"id":"interaction","channel_id":"channel","token":"token","data":{"custom_id":"action"},"member":{"user":{"id":"user"}}}`)})
	interaction := receiveInbound(t, ch)
	if interaction.UserID != "user" || interaction.Text != "action" || interaction.ChannelMeta["is_callback"] != "true" || interaction.ChannelMeta["interaction_id"] != "interaction" || interaction.ChannelMeta["interaction_token"] != "token" {
		t.Fatalf("interaction = %+v", interaction)
	}

	sendGatewayPayload(t, conn, gatewayPayload{Op: gatewayOpHeartbeat})
	heartbeat := receiveGatewayPayload(t, conn)
	if heartbeat.Op != gatewayOpHeartbeat || string(heartbeat.D) != "7" {
		t.Fatalf("heartbeat = %+v, want sequence 7", heartbeat)
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("close gateway: %v", err)
	}
	awaitClosed(t, ch)
}

func newGatewayServer(t *testing.T) (*httptest.Server, <-chan *websocket.Conn) {
	t.Helper()
	connections := make(chan *websocket.Conn, 1)
	done := make(chan struct{})
	server := httptest.NewTLSServer(websocket.Handler(func(conn *websocket.Conn) {
		connections <- conn
		<-done
	}))
	t.Cleanup(func() {
		close(done)
		server.Close()
	})
	return server, connections
}

func newGatewayAdapter(t *testing.T, server *httptest.Server) *Adapter {
	t.Helper()
	a := New("tok123")
	endpoint := "wss" + strings.TrimPrefix(server.URL, "https")
	a.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"url":"` + endpoint + `"}`)), Header: make(http.Header)}, nil
	})
	a.dialWebSocket = func(ctx context.Context, rawURL string) (*websocket.Conn, error) {
		return testWebSocketConfig(t, rawURL, server).DialContext(ctx)
	}
	return a
}

func receiveConnection(t *testing.T, connections <-chan *websocket.Conn) *websocket.Conn {
	t.Helper()
	select {
	case conn := <-connections:
		return conn
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway connection")
		return nil
	}
}

func sendGatewayPayload(t *testing.T, conn *websocket.Conn, payload gatewayPayload) {
	t.Helper()
	if err := websocket.JSON.Send(conn, payload); err != nil {
		t.Fatalf("send gateway payload: %v", err)
	}
}

func receiveGatewayPayload(t *testing.T, conn *websocket.Conn) gatewayPayload {
	t.Helper()
	var payload gatewayPayload
	if err := websocket.JSON.Receive(conn, &payload); err != nil {
		t.Fatalf("receive gateway payload: %v", err)
	}
	return payload
}

func receiveInbound(t *testing.T, ch <-chan types.InboundMessage) types.InboundMessage {
	t.Helper()
	select {
	case message := <-ch:
		return message
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inbound message")
		return types.InboundMessage{}
	}
}

func awaitClosed(t *testing.T, ch <-chan types.InboundMessage) {
	t.Helper()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("gateway channel remains open")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway channel to close")
	}
}

func TestWebSocketDialContextCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(requestStarted)
		<-release
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	config := testWebSocketConfig(t, "wss"+strings.TrimPrefix(server.URL, "https"), server)
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() {
		_, err := config.DialContext(ctx)
		result <- err
	}()
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("websocket handshake request did not reach server")
	}
	cancel()
	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
			t.Fatalf("DialContext error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("DialContext did not stop after context cancellation")
	}
}

func TestWebSocketDialRejectsNonUpgradeResponse(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not a websocket", http.StatusBadGateway)
	}))
	defer server.Close()

	config := testWebSocketConfig(t, "wss"+strings.TrimPrefix(server.URL, "https"), server)
	if _, err := config.DialContext(t.Context()); err == nil || !strings.Contains(err.Error(), "bad status") {
		t.Fatalf("DialContext error = %v, want rejected upgrade", err)
	}
}

func testWebSocketConfig(t *testing.T, endpoint string, server *httptest.Server) *websocket.Config {
	t.Helper()
	config, err := websocket.NewConfig(endpoint, "https://discord.com")
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	config.TlsConfig = &tls.Config{RootCAs: server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs, MinVersion: tls.VersionTLS12}
	return config
}

func intPtr(value int) *int { return &value }
