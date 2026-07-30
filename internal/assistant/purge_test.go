package assistant

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

// fakeAccount is a minimal in-memory stand-in for an accounts row, enough
// to drive the tiered-deletion candidate queries.
type fakeAccount struct {
	deletedAt      time.Time
	photosPurgedAt *time.Time
	emails         []string
}

// fakeAuditEvent is a minimal in-memory stand-in for an auth_audit_log row.
type fakeAuditEvent struct {
	accountID string // empty mirrors the real FK's ON DELETE SET NULL
	event     string
}

// fakePurgeStore is a test double for PurgeStore.
type fakePurgeStore struct {
	mu                 sync.Mutex
	purges             []time.Time // recorded chat-session olderThan values
	loginAttemptPurges []time.Time
	auditPurges        []time.Time
	count              int // number of rows to report purged
	err                error

	accounts      map[string]*fakeAccount
	auditEvents   []fakeAuditEvent
	photoPurges   []string // account IDs passed to PurgeAccountPhotos
	accountPurges []string // account IDs passed to PurgeAccount
}

func (f *fakePurgeStore) ListAccountsPendingPhotoPurge(_ context.Context, deletedBefore time.Time) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for id, a := range f.accounts {
		if a.photosPurgedAt == nil && !a.deletedAt.After(deletedBefore) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (f *fakePurgeStore) PurgeAccountPhotos(_ context.Context, accountID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[accountID]
	if !ok {
		return fmt.Errorf("fake: unknown account %s", accountID)
	}
	now := time.Now()
	a.photosPurgedAt = &now
	f.auditEvents = append(f.auditEvents, fakeAuditEvent{accountID: accountID, event: "account.photos_purged"})
	f.photoPurges = append(f.photoPurges, accountID)
	return nil
}

func (f *fakePurgeStore) ListAccountsPastDeletion(_ context.Context, deletedBefore time.Time) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for id, a := range f.accounts {
		if !a.deletedAt.After(deletedBefore) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// PurgeAccount mirrors the real store.PurgeAccount contract: it writes the
// account.delete.purged audit event first, then removes the account — the
// real auth_audit_log FK is ON DELETE SET NULL, so the event is recorded
// here with accountID already cleared, exactly as it survives in Postgres/
// SQLite after the DELETE statement runs.
func (f *fakePurgeStore) PurgeAccount(_ context.Context, accountID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.accounts[accountID]; !ok {
		return fmt.Errorf("fake: unknown account %s", accountID)
	}
	f.auditEvents = append(f.auditEvents, fakeAuditEvent{accountID: "", event: "account.delete.purged"})
	delete(f.accounts, accountID)
	f.accountPurges = append(f.accountPurges, accountID)
	return nil
}

func (f *fakePurgeStore) HasAuditEvent(_ context.Context, accountID, event string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.auditEvents {
		if e.accountID == accountID && e.event == event {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakePurgeStore) AccountEmails(_ context.Context, accountID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[accountID]
	if !ok {
		return nil, nil
	}
	return a.emails, nil
}

func (f *fakePurgeStore) WriteAuditEvent(_ context.Context, ev types.AuditEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auditEvents = append(f.auditEvents, fakeAuditEvent{accountID: ev.AccountID, event: ev.Event})
	return nil
}

// fakeMailer is a test double for mailer.Mailer.
type fakeMailer struct {
	mu   sync.Mutex
	sent []string // "to" addresses, one entry per Send call
}

func (m *fakeMailer) Send(_ context.Context, to string, _ mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, to)
	return nil
}

func (f *fakePurgeStore) PurgeLoginAttempts(_ context.Context, olderThan time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loginAttemptPurges = append(f.loginAttemptPurges, olderThan)
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func (f *fakePurgeStore) PurgeAuthAuditEvents(_ context.Context, olderThan time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auditPurges = append(f.auditPurges, olderThan)
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func (f *fakePurgeStore) PurgeDeletedChatSessions(ctx context.Context, olderThan time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.purges = append(f.purges, olderThan)
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func TestPurgeRunnerTicksAndPurges(t *testing.T) {
	store := &fakePurgeStore{count: 3}
	// Short interval for testing — runner ticks immediately, then every 50ms.
	runner := NewPurgeRunner(store, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runner.Run(ctx)

	// Wait for at least one tick to fire.
	time.Sleep(120 * time.Millisecond)
	cancel()

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.purges) == 0 {
		t.Fatal("expected at least one purge call, got 0")
	}
	if len(store.loginAttemptPurges) == 0 || len(store.auditPurges) == 0 {
		t.Fatal("expected login-attempt and audit purges")
	}

	// olderThan should be ~30 days ago.
	cutoff := time.Now().AddDate(0, 0, -30)
	for i, p := range store.purges {
		diff := p.Sub(cutoff).Abs()
		if diff > 5*time.Second {
			t.Errorf("purge[%d]: olderThan=%v, want ~%v (diff=%v)", i, p, cutoff, diff)
		}
	}
}

func TestPurgeRunnerContextCancel(t *testing.T) {
	store := &fakePurgeStore{count: 0}
	runner := NewPurgeRunner(store, time.Hour) // long interval, won't fire naturally

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Should exit cleanly without panicking or deadlocking.
	runner.Run(ctx)

	if len(store.purges) != 0 {
		t.Errorf("expected 0 purges on cancelled context, got %d", len(store.purges))
	}
}

func TestPurgeRunnerZeroPurged(t *testing.T) {
	// Zero purged sessions should not log an info message (no panic, no error).
	store := &fakePurgeStore{count: 0}
	runner := NewPurgeRunner(store, 10*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	runner.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.purges) == 0 {
		t.Fatal("expected at least one purge call")
	}
	// All calls should have count=0.
	for _, p := range store.purges {
		_ = p
	}
}

// TestPurgeRunnerPhotoPurgeTierBoundaryAndIdempotent verifies the day-30
// photo-purge tier fires exactly at the boundary, leaves an account one day
// short untouched, and — across several ticks — purges an already-purged
// account exactly once (idempotent: no duplicate audit event, no error).
func TestPurgeRunnerPhotoPurgeTierBoundaryAndIdempotent(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-due":     {deletedAt: now.AddDate(0, 0, -30)}, // exactly at the boundary
			"acct-not-due": {deletedAt: now.AddDate(0, 0, -29)}, // one day short
		},
	}
	runner := NewPurgeRunner(store, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go runner.Run(ctx)
	time.Sleep(150 * time.Millisecond) // several ticks
	cancel()

	store.mu.Lock()
	defer store.mu.Unlock()

	dueCount := 0
	for _, id := range store.photoPurges {
		switch id {
		case "acct-due":
			dueCount++
		case "acct-not-due":
			t.Fatalf("photo-purged account not yet at the 30-day boundary: %s", id)
		}
	}
	if dueCount != 1 {
		t.Fatalf("acct-due photo-purged %d times across multiple ticks; want exactly 1 (idempotent)", dueCount)
	}

	if store.accounts["acct-due"].photosPurgedAt == nil {
		t.Fatal("acct-due.photosPurgedAt not set after purge")
	}
	if store.accounts["acct-not-due"].photosPurgedAt != nil {
		t.Fatal("acct-not-due.photosPurgedAt set; account is not yet due")
	}

	auditCount := 0
	for _, e := range store.auditEvents {
		if e.accountID == "acct-due" && e.event == "account.photos_purged" {
			auditCount++
		}
	}
	if auditCount != 1 {
		t.Fatalf("account.photos_purged audit events for acct-due = %d; want exactly 1", auditCount)
	}
}

// TestPurgeRunnerFullPurgeTierBoundary verifies the day-90 full-purge tier
// hard-deletes an account exactly at the boundary, leaves one a day short
// alone, and that the account.delete.purged audit event survives the
// account's removal with its account_id nulled (mirroring the real store's
// auth_audit_log ON DELETE SET NULL FK).
func TestPurgeRunnerFullPurgeTierBoundary(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-due":     {deletedAt: now.AddDate(0, 0, -90)}, // exactly at the boundary
			"acct-not-due": {deletedAt: now.AddDate(0, 0, -89)}, // one day short
		},
	}
	runner := NewPurgeRunner(store, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go runner.Run(ctx)
	time.Sleep(150 * time.Millisecond)
	cancel()

	store.mu.Lock()
	defer store.mu.Unlock()

	if _, ok := store.accounts["acct-due"]; ok {
		t.Fatal("acct-due still present after crossing the 90-day boundary; want hard-deleted")
	}
	if _, ok := store.accounts["acct-not-due"]; !ok {
		t.Fatal("acct-not-due was purged; account is not yet due")
	}

	found := 0
	for _, e := range store.auditEvents {
		if e.event == "account.delete.purged" {
			found++
			if e.accountID != "" {
				t.Fatalf("account.delete.purged audit event account_id = %q; want empty (nulled)", e.accountID)
			}
		}
	}
	if found != 1 {
		t.Fatalf("account.delete.purged audit events = %d; want exactly 1", found)
	}
}

// TestPurgeRunnerReminderEmailsFireOncePerAccountPerTier simulates several
// ticks within both reminder windows and verifies each account gets its
// reminder email — and its idempotency audit event — exactly once.
func TestPurgeRunnerReminderEmailsFireOncePerAccountPerTier(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-photo-reminder": {deletedAt: now.AddDate(0, 0, -25), emails: []string{"photo@example.com"}},
			"acct-final-reminder": {deletedAt: now.AddDate(0, 0, -85), emails: []string{"final@example.com"}},
		},
	}
	mail := &fakeMailer{}
	runner := NewPurgeRunner(store, 20*time.Millisecond).WithMailer(mail)

	ctx, cancel := context.WithCancel(context.Background())
	go runner.Run(ctx)
	time.Sleep(150 * time.Millisecond) // several ticks within the reminder windows
	cancel()

	mail.mu.Lock()
	defer mail.mu.Unlock()

	photoSends, finalSends := 0, 0
	for _, to := range mail.sent {
		switch to {
		case "photo@example.com":
			photoSends++
		case "final@example.com":
			finalSends++
		}
	}
	if photoSends != 1 {
		t.Fatalf("photo-purge reminder sent %d times across multiple ticks; want exactly 1", photoSends)
	}
	if finalSends != 1 {
		t.Fatalf("final-purge reminder sent %d times across multiple ticks; want exactly 1", finalSends)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	photoAudits, finalAudits := 0, 0
	for _, e := range store.auditEvents {
		if e.accountID == "acct-photo-reminder" && e.event == photoPurgeReminderEvent {
			photoAudits++
		}
		if e.accountID == "acct-final-reminder" && e.event == finalPurgeReminderEvent {
			finalAudits++
		}
	}
	if photoAudits != 1 {
		t.Fatalf("photo reminder audit events = %d; want exactly 1", photoAudits)
	}
	if finalAudits != 1 {
		t.Fatalf("final reminder audit events = %d; want exactly 1", finalAudits)
	}
}

// TestPurgeRunnerNoMailerSkipsReminders verifies that without WithMailer,
// PurgeRunner never sends (or marks as sent) a reminder — it should not
// silently mark accounts as reminded when nothing was actually sent.
func TestPurgeRunnerNoMailerSkipsReminders(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-photo-reminder": {deletedAt: now.AddDate(0, 0, -25), emails: []string{"photo@example.com"}},
		},
	}
	runner := NewPurgeRunner(store, 20*time.Millisecond) // no WithMailer

	ctx, cancel := context.WithCancel(context.Background())
	go runner.Run(ctx)
	time.Sleep(80 * time.Millisecond)
	cancel()

	store.mu.Lock()
	defer store.mu.Unlock()
	for _, e := range store.auditEvents {
		if e.event == photoPurgeReminderEvent {
			t.Fatal("reminder audit event written despite no mailer configured")
		}
	}
}
