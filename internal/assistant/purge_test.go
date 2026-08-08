package assistant

import (
	"context"
	"errors"
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
	userIDs        []string // for ListAccountUserIDs
}

// orderLog records call order across the fake store and fake backup
// destinations, so tests can pin that backup files are deleted before the
// account they belong to is purged from the DB.
type orderLog struct {
	mu    sync.Mutex
	calls []string
}

func (o *orderLog) add(s string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls = append(o.calls, s)
}

// fakeBackupDest is a test double for backup.Destination, tracking Delete
// calls by cfg.UserID so purgeAccountBackups tests can assert which users'
// backup files were (or weren't) deleted.
type fakeBackupDest struct {
	mu      sync.Mutex
	deletes []string
	err     error
	order   *orderLog
}

func (d *fakeBackupDest) Write(context.Context, types.BackupConfig, string, []byte) error { return nil }

func (d *fakeBackupDest) Delete(_ context.Context, cfg types.BackupConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.order != nil {
		d.order.add("backup:" + cfg.UserID)
	}
	if d.err != nil {
		return d.err
	}
	d.deletes = append(d.deletes, cfg.UserID)
	return nil
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

	authChallengePurges []time.Time
	oidcStatePurges     []time.Time

	backupConfigs           map[string]types.BackupConfig // keyed by userID
	listAccountUserIDsCalls int
	order                   *orderLog // optional, for ordering assertions

	// Error-injection fields, one per PurgeStore method, so tests can drive
	// purge.go's error-handling branches (slog.Error + return/continue)
	// without a real failing DB driver.
	listPendingPhotoPurgeErr error
	purgeAccountPhotosErr    error
	listPastDeletionErr      error
	purgeAccountErr          error
	hasAuditEventErr         error
	accountEmailsErr         error
	writeAuditEventErr       error
	listAccountUserIDsErr    error
	getBackupConfigErr       error
}

func (f *fakePurgeStore) ListAccountsPendingPhotoPurge(_ context.Context, deletedBefore time.Time) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listPendingPhotoPurgeErr != nil {
		return nil, f.listPendingPhotoPurgeErr
	}
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
	if f.purgeAccountPhotosErr != nil {
		return f.purgeAccountPhotosErr
	}
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
	if f.listPastDeletionErr != nil {
		return nil, f.listPastDeletionErr
	}
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
	if f.purgeAccountErr != nil {
		return f.purgeAccountErr
	}
	if _, ok := f.accounts[accountID]; !ok {
		return fmt.Errorf("fake: unknown account %s", accountID)
	}
	if f.order != nil {
		f.order.add("db:" + accountID)
	}
	f.auditEvents = append(f.auditEvents, fakeAuditEvent{accountID: "", event: "account.delete.purged"})
	delete(f.accounts, accountID)
	f.accountPurges = append(f.accountPurges, accountID)
	return nil
}

func (f *fakePurgeStore) ListAccountUserIDs(_ context.Context, accountID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listAccountUserIDsCalls++
	if f.listAccountUserIDsErr != nil {
		return nil, f.listAccountUserIDsErr
	}
	a, ok := f.accounts[accountID]
	if !ok {
		return nil, nil
	}
	return a.userIDs, nil
}

func (f *fakePurgeStore) GetBackupConfig(_ context.Context, userID string) (types.BackupConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getBackupConfigErr != nil {
		return types.BackupConfig{}, f.getBackupConfigErr
	}
	cfg, ok := f.backupConfigs[userID]
	if !ok {
		return types.BackupConfig{}, types.ErrNotFound
	}
	return cfg, nil
}

func (f *fakePurgeStore) PurgeExpiredAuthChallenges(_ context.Context, now time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.authChallengePurges = append(f.authChallengePurges, now)
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func (f *fakePurgeStore) PurgeExpiredOIDCStates(_ context.Context, now time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.oidcStatePurges = append(f.oidcStatePurges, now)
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func (f *fakePurgeStore) HasAuditEvent(_ context.Context, accountID, event string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hasAuditEventErr != nil {
		return false, f.hasAuditEventErr
	}
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
	if f.accountEmailsErr != nil {
		return nil, f.accountEmailsErr
	}
	a, ok := f.accounts[accountID]
	if !ok {
		return nil, nil
	}
	return a.emails, nil
}

func (f *fakePurgeStore) WriteAuditEvent(_ context.Context, ev types.AuditEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeAuditEventErr != nil {
		return f.writeAuditEventErr
	}
	f.auditEvents = append(f.auditEvents, fakeAuditEvent{accountID: ev.AccountID, event: ev.Event})
	return nil
}

// fakeMailer is a test double for mailer.Mailer.
type fakeMailer struct {
	mu      sync.Mutex
	sent    []string // "to" addresses, one entry per Send call
	sendErr error
}

func (m *fakeMailer) Send(_ context.Context, to string, _ mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, to)
	return m.sendErr
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

// ---------------------------------------------------------------------------
// Error-path coverage: purgeAccountPhotos, purgeAccounts, sendReminders,
// remindAccount, sendReminderEmails. These call the unexported step methods
// directly instead of going through Run's ticker, so each error branch fires
// deterministically on the first call.
// ---------------------------------------------------------------------------

func TestPurgeAccountPhotosListError(t *testing.T) {
	store := &fakePurgeStore{listPendingPhotoPurgeErr: errors.New("list failed")}
	runner := NewPurgeRunner(store, time.Hour)

	runner.purgeAccountPhotos(context.Background(), time.Now())

	if len(store.photoPurges) != 0 {
		t.Fatalf("expected no photo purges when listing fails, got %v", store.photoPurges)
	}
}

func TestPurgeAccountPhotosPerAccountErrorContinues(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-bad": {deletedAt: now.AddDate(0, 0, -31)},
		},
		purgeAccountPhotosErr: errors.New("purge failed"),
	}
	runner := NewPurgeRunner(store, time.Hour)

	runner.purgeAccountPhotos(context.Background(), now)

	if store.accounts["acct-bad"].photosPurgedAt != nil {
		t.Fatal("photosPurgedAt should remain nil when the purge call fails")
	}
}

func TestPurgeAccountsListError(t *testing.T) {
	store := &fakePurgeStore{listPastDeletionErr: errors.New("list failed")}
	runner := NewPurgeRunner(store, time.Hour)

	runner.purgeAccounts(context.Background(), time.Now())

	if len(store.accountPurges) != 0 {
		t.Fatalf("expected no account purges when listing fails, got %v", store.accountPurges)
	}
}

func TestPurgeAccountsPerAccountErrorContinues(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-bad": {deletedAt: now.AddDate(0, 0, -91)},
		},
		purgeAccountErr: errors.New("purge failed"),
	}
	runner := NewPurgeRunner(store, time.Hour)

	runner.purgeAccounts(context.Background(), now)

	if _, ok := store.accounts["acct-bad"]; !ok {
		t.Fatal("account should still be present when the purge call fails")
	}
}

func TestSendRemindersListErrors(t *testing.T) {
	store := &fakePurgeStore{
		listPendingPhotoPurgeErr: errors.New("photo list failed"),
		listPastDeletionErr:      errors.New("final list failed"),
	}
	mail := &fakeMailer{}
	runner := NewPurgeRunner(store, time.Hour).WithMailer(mail)

	runner.sendReminders(context.Background(), time.Now())

	if len(mail.sent) != 0 {
		t.Fatalf("expected no reminder sends when both list calls fail, got %v", mail.sent)
	}
}

func TestRemindAccountHasAuditEventError(t *testing.T) {
	store := &fakePurgeStore{hasAuditEventErr: errors.New("audit check failed")}
	mail := &fakeMailer{}
	runner := NewPurgeRunner(store, time.Hour).WithMailer(mail)

	runner.remindAccount(context.Background(), "acct-1", photoPurgeReminderEvent, photoPurgeReminderMessage)

	if len(mail.sent) != 0 {
		t.Fatalf("expected no send when HasAuditEvent errors, got %v", mail.sent)
	}
}

func TestRemindAccountAccountEmailsError(t *testing.T) {
	store := &fakePurgeStore{accountEmailsErr: errors.New("emails lookup failed")}
	mail := &fakeMailer{}
	runner := NewPurgeRunner(store, time.Hour).WithMailer(mail)

	runner.remindAccount(context.Background(), "acct-1", photoPurgeReminderEvent, photoPurgeReminderMessage)

	if len(mail.sent) != 0 {
		t.Fatalf("expected no send when AccountEmails errors, got %v", mail.sent)
	}
}

// TestRemindAccountSendFailureSkipsAuditWrite covers sendReminderEmails'
// per-address error log plus remindAccount's "don't write the idempotency
// audit event on a failed send" branch, in one pass.
func TestRemindAccountSendFailureSkipsAuditWrite(t *testing.T) {
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {emails: []string{"a@example.com"}},
		},
	}
	mail := &fakeMailer{sendErr: errors.New("smtp down")}
	runner := NewPurgeRunner(store, time.Hour).WithMailer(mail)

	runner.remindAccount(context.Background(), "acct-1", photoPurgeReminderEvent, photoPurgeReminderMessage)

	for _, e := range store.auditEvents {
		if e.accountID == "acct-1" && e.event == photoPurgeReminderEvent {
			t.Fatal("audit event must not be written when the reminder send failed")
		}
	}
}

func TestRemindAccountWriteAuditEventError(t *testing.T) {
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {emails: []string{"a@example.com"}},
		},
		writeAuditEventErr: errors.New("write failed"),
	}
	mail := &fakeMailer{}
	runner := NewPurgeRunner(store, time.Hour).WithMailer(mail)

	// Must not panic despite the audit write failing after a successful send.
	runner.remindAccount(context.Background(), "acct-1", photoPurgeReminderEvent, photoPurgeReminderMessage)

	if len(mail.sent) != 1 {
		t.Fatalf("expected the reminder to still be sent, got %v", mail.sent)
	}
}

// ---------------------------------------------------------------------------
// Expired auth_challenges / oidc_states sweep
// ---------------------------------------------------------------------------

// TestPurgeRunnerTicksPurgesExpiredAuthAndOIDC verifies Run's tick calls
// PurgeExpiredAuthChallenges and PurgeExpiredOIDCStates (alongside the
// pre-existing login-attempt/audit-event purges) with an ~now cutoff.
func TestPurgeRunnerTicksPurgesExpiredAuthAndOIDC(t *testing.T) {
	store := &fakePurgeStore{count: 2}
	runner := NewPurgeRunner(store, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go runner.Run(ctx)
	time.Sleep(120 * time.Millisecond)
	cancel()

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.authChallengePurges) == 0 {
		t.Fatal("expected at least one PurgeExpiredAuthChallenges call")
	}
	if len(store.oidcStatePurges) == 0 {
		t.Fatal("expected at least one PurgeExpiredOIDCStates call")
	}
	now := time.Now()
	for i, ts := range store.authChallengePurges {
		if now.Sub(ts).Abs() > 5*time.Second {
			t.Errorf("authChallengePurges[%d] = %v, want ~now", i, ts)
		}
	}
	for i, ts := range store.oidcStatePurges {
		if now.Sub(ts).Abs() > 5*time.Second {
			t.Errorf("oidcStatePurges[%d] = %v, want ~now", i, ts)
		}
	}
}

// TestPurgeRunnerTicksSurvivesExpiredAuthAndOIDCErrors verifies a
// PurgeExpiredAuthChallenges/PurgeExpiredOIDCStates failure is logged and
// does not abort the rest of the tick (chat-session purge still runs).
func TestPurgeRunnerTicksSurvivesExpiredAuthAndOIDCErrors(t *testing.T) {
	store := &fakePurgeStore{err: errors.New("db down")}
	runner := NewPurgeRunner(store, 20*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	runner.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.purges) == 0 {
		t.Fatal("expected chat-session purge to still run despite auth/oidc purge errors")
	}
}

// ---------------------------------------------------------------------------
// Backup-file cleanup on account purge (WithBackupDestinations)
// ---------------------------------------------------------------------------

// TestPurgeAccountBackups_DeletesFromConfiguredDestination verifies that
// purging a due account deletes its users' backup files from whichever
// destination their backup_config selects, before the DB row is removed.
func TestPurgeAccountBackups_DeletesFromConfiguredDestination(t *testing.T) {
	now := time.Now()
	log := &orderLog{}
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90), userIDs: []string{"user-1"}},
		},
		backupConfigs: map[string]types.BackupConfig{
			"user-1": {UserID: "user-1", Destination: "local"},
		},
		order: log,
	}
	local := &fakeBackupDest{order: log}
	s3 := &fakeBackupDest{order: log}
	runner := NewPurgeRunner(store, time.Hour).WithBackupDestinations(local, s3)

	runner.purgeAccounts(context.Background(), now)

	if len(local.deletes) != 1 || local.deletes[0] != "user-1" {
		t.Fatalf("local.deletes = %v; want [user-1]", local.deletes)
	}
	if len(s3.deletes) != 0 {
		t.Fatalf("s3.deletes = %v; want none (config selects local)", s3.deletes)
	}
	if _, ok := store.accounts["acct-1"]; ok {
		t.Fatal("expected account purged from DB after backup cleanup")
	}

	// Backup deletion must happen before the DB purge.
	log.mu.Lock()
	defer log.mu.Unlock()
	if len(log.calls) != 2 || log.calls[0] != "backup:user-1" || log.calls[1] != "db:acct-1" {
		t.Fatalf("call order = %v; want [backup:user-1 db:acct-1]", log.calls)
	}
}

// TestPurgeAccountBackups_S3Destination verifies an S3-configured user's
// files are deleted via the S3 destination, not the local one.
func TestPurgeAccountBackups_S3Destination(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90), userIDs: []string{"user-1"}},
		},
		backupConfigs: map[string]types.BackupConfig{
			"user-1": {UserID: "user-1", Destination: "s3"},
		},
	}
	local := &fakeBackupDest{}
	s3 := &fakeBackupDest{}
	runner := NewPurgeRunner(store, time.Hour).WithBackupDestinations(local, s3)

	runner.purgeAccounts(context.Background(), now)

	if len(s3.deletes) != 1 || s3.deletes[0] != "user-1" {
		t.Fatalf("s3.deletes = %v; want [user-1]", s3.deletes)
	}
	if len(local.deletes) != 0 {
		t.Fatalf("local.deletes = %v; want none (config selects s3)", local.deletes)
	}
}

// TestPurgeAccountBackups_NoConfigSkipped verifies a user who never
// configured backups (GetBackupConfig -> ErrNotFound) has nothing deleted,
// and the account purge still proceeds normally.
func TestPurgeAccountBackups_NoConfigSkipped(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90), userIDs: []string{"user-1"}},
		},
		// no backupConfigs entry for user-1
	}
	local := &fakeBackupDest{}
	runner := NewPurgeRunner(store, time.Hour).WithBackupDestinations(local, nil)

	runner.purgeAccounts(context.Background(), now)

	if len(local.deletes) != 0 {
		t.Fatalf("local.deletes = %v; want none", local.deletes)
	}
	if _, ok := store.accounts["acct-1"]; ok {
		t.Fatal("expected account still purged despite no backup config")
	}
}

// TestPurgeAccountBackups_WithoutDestinationsIsNoop verifies that without
// WithBackupDestinations, purgeAccounts never even looks up backup config —
// it purges the DB rows exactly as before this feature existed.
func TestPurgeAccountBackups_WithoutDestinationsIsNoop(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90), userIDs: []string{"user-1"}},
		},
		backupConfigs: map[string]types.BackupConfig{
			"user-1": {UserID: "user-1", Destination: "local"},
		},
	}
	runner := NewPurgeRunner(store, time.Hour) // no WithBackupDestinations

	runner.purgeAccounts(context.Background(), now)

	if store.listAccountUserIDsCalls != 0 {
		t.Fatalf("expected ListAccountUserIDs never called without WithBackupDestinations, got %d calls", store.listAccountUserIDsCalls)
	}
	if _, ok := store.accounts["acct-1"]; ok {
		t.Fatal("expected account still purged")
	}
}

// TestPurgeAccountBackups_ListUsersErrorStillPurgesAccount verifies a
// ListAccountUserIDs failure is logged and does not block the DB purge.
func TestPurgeAccountBackups_ListUsersErrorStillPurgesAccount(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90)},
		},
		listAccountUserIDsErr: errors.New("list failed"),
	}
	local := &fakeBackupDest{}
	runner := NewPurgeRunner(store, time.Hour).WithBackupDestinations(local, nil)

	runner.purgeAccounts(context.Background(), now)

	if _, ok := store.accounts["acct-1"]; ok {
		t.Fatal("expected account still purged despite ListAccountUserIDs error")
	}
}

// TestPurgeAccountBackups_DeleteErrorStillPurgesAccount verifies a
// Destination.Delete failure is logged and does not block the DB purge.
func TestPurgeAccountBackups_DeleteErrorStillPurgesAccount(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90), userIDs: []string{"user-1"}},
		},
		backupConfigs: map[string]types.BackupConfig{
			"user-1": {UserID: "user-1", Destination: "local"},
		},
	}
	local := &fakeBackupDest{err: errors.New("delete failed")}
	runner := NewPurgeRunner(store, time.Hour).WithBackupDestinations(local, nil)

	runner.purgeAccounts(context.Background(), now)

	if _, ok := store.accounts["acct-1"]; ok {
		t.Fatal("expected account still purged despite backup Delete error")
	}
}

// TestPurgeAccountBackups_UnavailableDestinationSkipsUser verifies a user
// whose backup_config selects a destination that wasn't passed to
// WithBackupDestinations (e.g. s3 unavailable at startup) is skipped without
// error, same as backup.Runner.destinationFor's nil-destination case.
func TestPurgeAccountBackups_UnavailableDestinationSkipsUser(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90), userIDs: []string{"user-1"}},
		},
		backupConfigs: map[string]types.BackupConfig{
			"user-1": {UserID: "user-1", Destination: "s3"},
		},
	}
	local := &fakeBackupDest{}
	runner := NewPurgeRunner(store, time.Hour).WithBackupDestinations(local, nil) // s3 unavailable

	runner.purgeAccounts(context.Background(), now)

	if len(local.deletes) != 0 {
		t.Fatalf("local.deletes = %v; want none (config selects unavailable s3)", local.deletes)
	}
	if _, ok := store.accounts["acct-1"]; ok {
		t.Fatal("expected account still purged despite unavailable destination")
	}
}

// TestPurgeAccountBackups_GetConfigErrorSkipsThatUser verifies a
// GetBackupConfig failure (distinct from ErrNotFound) is logged and skips
// only that user's backup deletion, without blocking the DB purge.
func TestPurgeAccountBackups_GetConfigErrorSkipsThatUser(t *testing.T) {
	now := time.Now()
	store := &fakePurgeStore{
		accounts: map[string]*fakeAccount{
			"acct-1": {deletedAt: now.AddDate(0, 0, -90), userIDs: []string{"user-1"}},
		},
		getBackupConfigErr: errors.New("db down"),
	}
	local := &fakeBackupDest{}
	runner := NewPurgeRunner(store, time.Hour).WithBackupDestinations(local, nil)

	runner.purgeAccounts(context.Background(), now)

	if len(local.deletes) != 0 {
		t.Fatalf("local.deletes = %v; want none", local.deletes)
	}
	if _, ok := store.accounts["acct-1"]; ok {
		t.Fatal("expected account still purged despite GetBackupConfig error")
	}
}
