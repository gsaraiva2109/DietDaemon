package matrix

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestName(t *testing.T) {
	a := New("https://matrix.example.com", "@bot:example.com", "token")
	if a.Name() != "matrix" {
		t.Errorf("Name = %q, want %q", a.Name(), "matrix")
	}
}

func TestSendMissingRoom(t *testing.T) {
	a := New("https://matrix.example.com", "@bot:example.com", "token")
	err := a.Send(t.Context(), types.Reply{Text: "hello", ChannelMeta: nil})
	if err == nil {
		t.Error("expected error for missing room_id, got nil")
	}
}

// ---------------------------------------------------------------------------
// CHARACTERIZATION tests for syncLoop (issue #158).
//
// These pin down syncLoop's current, observed behavior against a real
// httptest.NewServer (syncLoop uses plain net/http, not websockets, so this
// works directly) rather than asserting a spec. No production code changes.
// ---------------------------------------------------------------------------

// newTestAdapter builds an Adapter wired to an httptest server URL.
func newTestAdapter(serverURL, userID string) *Adapter {
	return New(serverURL, userID, "test-token")
}

// readOne waits up to d for a single value on ch, failing the test on timeout.
func readOne(t *testing.T, ch <-chan types.InboundMessage, d time.Duration) types.InboundMessage {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatal("channel closed while waiting for a message")
		}
		return msg
	case <-time.After(d):
		t.Fatal("timed out waiting for message on ch")
	}
	return types.InboundMessage{}
}

// assertNoMessage fails if a message arrives on ch within d.
func assertNoMessage(t *testing.T, ch <-chan types.InboundMessage, d time.Duration) {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if ok {
			t.Fatalf("unexpected message on ch: %+v", msg)
		}
	case <-time.After(d):
	}
}

// TestSyncLoop_SinceProgression covers branch 1: the first /sync request
// carries no "since", and the second request carries the since value returned
// by the first response's next_batch.
func TestSyncLoop_SinceProgression(t *testing.T) {
	t.Parallel()

	var reqCount int32
	var sinceOnSecondReq string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&reqCount, 1)
		since := r.URL.Query().Get("since")
		switch n {
		case 1:
			if since != "" {
				t.Errorf("first request: got since=%q, want empty", since)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"next_batch":"batch1","rooms":{"join":{}}}`)
		case 2:
			sinceOnSecondReq = since
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"next_batch":"batch2","rooms":{"join":{}}}`)
		default:
			// Keep any later polls parked so the loop doesn't spin.
			<-r.Context().Done()
		}
	}))
	defer srv.Close()

	a := newTestAdapter(srv.URL, "@bot:example.com")
	ch := make(chan types.InboundMessage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.syncLoop(ctx, ch)

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&reqCount) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	assertNoMessage(t, ch, 200*time.Millisecond)

	if sinceOnSecondReq != "batch1" {
		t.Errorf("second request: got since=%q, want %q", sinceOnSecondReq, "batch1")
	}
}

// TestSyncLoop_MalformedBodyDoesNotAdvanceSince covers branch 2: a response
// with an unparseable (non-JSON) body makes the loop retry without treating
// the response as having advanced `since`. json.Decode failure is handled
// with an immediate `continue` (no sleep in that path), so this test is fast;
// the real ~2s backoff only fires when a.client.Do itself errors, which is
// covered by TestSyncLoop_NetworkErrorBackoffPreservesSince below.
func TestSyncLoop_MalformedBodyDoesNotAdvanceSince(t *testing.T) {
	t.Parallel()

	var reqCount int32
	var sinceOnThirdReq string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&reqCount, 1)
		since := r.URL.Query().Get("since")
		switch n {
		case 1:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"next_batch":"batch1","rooms":{"join":{}}}`)
		case 2:
			// Non-200, unparseable body: decode fails, since must stay "batch1".
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "not json")
		case 3:
			sinceOnThirdReq = since
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"next_batch":"batch2","rooms":{"join":{}}}`)
		default:
			<-r.Context().Done()
		}
	}))
	defer srv.Close()

	a := newTestAdapter(srv.URL, "@bot:example.com")
	ch := make(chan types.InboundMessage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.syncLoop(ctx, ch)

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&reqCount) < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	assertNoMessage(t, ch, 200*time.Millisecond)

	if sinceOnThirdReq != "batch1" {
		t.Errorf("third request: got since=%q, want %q (unchanged from before the malformed response)", sinceOnThirdReq, "batch1")
	}
}

// TestSyncLoop_NetworkErrorBackoffPreservesSince covers branch 2's real
// backoff path: when a.client.Do itself returns an error (here, a client-side
// timeout because the server stalls past the client's timeout), the loop
// sleeps ~2s before retrying, and `since` is unaffected by the failed
// attempt. This test carries a real ~2s sleep by design.
func TestSyncLoop_NetworkErrorBackoffPreservesSince(t *testing.T) {
	var reqCount int32
	var sinceOnRetry string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&reqCount, 1)
		since := r.URL.Query().Get("since")
		switch n {
		case 1:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"next_batch":"batch1","rooms":{"join":{}}}`)
		case 2:
			// Stall past the client timeout so a.client.Do returns an error.
			time.Sleep(300 * time.Millisecond)
		case 3:
			sinceOnRetry = since
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"next_batch":"batch2","rooms":{"join":{}}}`)
		default:
			<-r.Context().Done()
		}
	}))
	defer srv.Close()

	a := newTestAdapter(srv.URL, "@bot:example.com")
	a.client = &http.Client{Timeout: 100 * time.Millisecond} // short so req 2 fails fast

	ch := make(chan types.InboundMessage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.syncLoop(ctx, ch)

	deadline := time.Now().Add(5 * time.Second) // tolerates the real 2s backoff sleep
	for atomic.LoadInt32(&reqCount) < 3 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	assertNoMessage(t, ch, 200*time.Millisecond)

	if got := atomic.LoadInt32(&reqCount); got < 3 {
		t.Fatalf("expected at least 3 requests (incl. the failed one and its retry), got %d", got)
	}
	if sinceOnRetry != "batch1" {
		t.Errorf("retry request: got since=%q, want %q (unchanged by the failed attempt)", sinceOnRetry, "batch1")
	}
}

// TestSyncLoop_EventFiltering covers branch 3: own-sender messages, non-
// m.room.message event types, and non-m.text msgtypes are all skipped; a
// real text message from another user is emitted onto ch.
func TestSyncLoop_EventFiltering(t *testing.T) {
	t.Parallel()

	const botID = "@bot:example.com"
	const roomID = "!room1:example.com"

	var reqCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&reqCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			fmt.Fprintf(w, `{"next_batch":"batch1","rooms":{"join":{%q:{"timeline":{"events":[
				{"type":"m.room.message","sender":%q,"event_id":"$own","content":{"body":"ignore me","msgtype":"m.text"}},
				{"type":"m.room.member","sender":"@other:example.com","event_id":"$member","content":{"body":"","msgtype":""}},
				{"type":"m.room.message","sender":"@other:example.com","event_id":"$notice","content":{"body":"a notice","msgtype":"m.notice"}},
				{"type":"m.room.message","sender":"@other:example.com","event_id":"$real","content":{"body":"hello there","msgtype":"m.text"}}
			]}}}}}`, roomID, botID)
			return
		}
		fmt.Fprint(w, `{"next_batch":"batch2","rooms":{"join":{}}}`)
	}))
	defer srv.Close()

	a := newTestAdapter(srv.URL, botID)
	ch := make(chan types.InboundMessage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.syncLoop(ctx, ch)

	msg := readOne(t, ch, 2*time.Second)
	if msg.Text != "hello there" {
		t.Errorf("Text = %q, want %q", msg.Text, "hello there")
	}
	if msg.UserID != "@other:example.com" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "@other:example.com")
	}
	if msg.ChannelMeta["room_id"] != roomID {
		t.Errorf("room_id = %q, want %q", msg.ChannelMeta["room_id"], roomID)
	}
	if msg.ChannelMeta["event_id"] != "$real" {
		t.Errorf("event_id = %q, want %q", msg.ChannelMeta["event_id"], "$real")
	}

	// The own-sender, non-message-type, and non-text-msgtype events must not
	// also produce messages.
	assertNoMessage(t, ch, 200*time.Millisecond)
}

// TestSyncLoop_NumberedReplyResolvesCallbackData covers branch 4: after a.Send
// registers an inline keyboard for a room, a numbered reply ("2") through
// /sync resolves to that button's callback data.
func TestSyncLoop_NumberedReplyResolvesCallbackData(t *testing.T) {
	t.Parallel()

	const botID = "@bot:example.com"
	const roomID = "!markuproom:example.com"

	var reqCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/send/m.room.message/") {
			// Send() request.
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"event_id":"$sent"}`)
			return
		}

		n := atomic.AddInt32(&reqCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			fmt.Fprintf(w, `{"next_batch":"batch1","rooms":{"join":{%q:{"timeline":{"events":[
				{"type":"m.room.message","sender":"@other:example.com","event_id":"$reply","content":{"body":"2","msgtype":"m.text"}}
			]}}}}}`, roomID)
			return
		}
		fmt.Fprint(w, `{"next_batch":"batch2","rooms":{"join":{}}}`)
	}))
	defer srv.Close()

	a := newTestAdapter(srv.URL, botID)

	// Populate the pendingMarkupStore for roomID via Send, as the issue asks.
	err := a.Send(context.Background(), types.Reply{
		Text:        "pick one",
		ChannelMeta: map[string]string{"room_id": roomID},
		Markup: &types.ReplyMarkup{InlineKeyboard: [][]types.InlineButton{
			{{Text: "Yes", CallbackData: "opt_yes"}},
			{{Text: "No", CallbackData: "opt_no"}},
		}},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	ch := make(chan types.InboundMessage)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.syncLoop(ctx, ch)

	msg := readOne(t, ch, 2*time.Second)
	if msg.Text != "opt_no" {
		t.Errorf("Text = %q, want %q (2nd button's callback data)", msg.Text, "opt_no")
	}
}

// TestSyncLoop_ContextCancellation covers branch 5: cancelling ctx exits the
// loop cleanly — no panic, no send on a closed channel, no goroutine leak —
// both while a poll is in flight and while the loop is blocked trying to
// send a decoded message onto ch.
func TestSyncLoop_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("MidPoll", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Stall so the request is still in flight when we cancel.
			<-r.Context().Done()
		}))
		defer srv.Close()

		a := newTestAdapter(srv.URL, "@bot:example.com")
		ch := make(chan types.InboundMessage)
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			a.syncLoop(ctx, ch)
			close(done)
		}()

		time.Sleep(50 * time.Millisecond) // let the poll start
		cancel()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("syncLoop did not return after context cancellation")
		}

		// ch must have been closed by syncLoop's defer, and reading from a
		// closed channel must not panic or block.
		if _, ok := <-ch; ok {
			t.Error("expected ch to be closed after cancellation")
		}
	})

	t.Run("MidChannelSend", func(t *testing.T) {
		t.Parallel()

		const botID = "@bot:example.com"
		const roomID = "!room1:example.com"

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"next_batch":"batch1","rooms":{"join":{%q:{"timeline":{"events":[
				{"type":"m.room.message","sender":"@other:example.com","event_id":"$e1","content":{"body":"hi","msgtype":"m.text"}}
			]}}}}}`, roomID)
		}))
		defer srv.Close()

		a := newTestAdapter(srv.URL, botID)
		ch := make(chan types.InboundMessage) // unbuffered, nobody reads it below
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			a.syncLoop(ctx, ch)
			close(done)
		}()

		// Give the loop time to decode the response and block on `ch <- msg`.
		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("syncLoop did not return while blocked sending to ch")
		}

		if _, ok := <-ch; ok {
			t.Error("expected ch to be closed after cancellation")
		}
	})
}
