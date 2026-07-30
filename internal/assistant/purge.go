package assistant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
)

// PurgeStore is the subset of store methods the purge job needs.
type PurgeStore interface {
	PurgeDeletedChatSessions(ctx context.Context, olderThan time.Time) (int, error)
	PurgeLoginAttempts(ctx context.Context, olderThan time.Time) (int, error)
	PurgeAuthAuditEvents(ctx context.Context, olderThan time.Time) (int, error)

	// ListAccountsPendingPhotoPurge Tiered account-deletion retention (progress-photo purge at day 30,
	// full account purge at day 90, plus reminder emails ~5 days before
	// each). See the tiered deletion model this extends.
	ListAccountsPendingPhotoPurge(ctx context.Context, deletedBefore time.Time) ([]string, error)
	PurgeAccountPhotos(ctx context.Context, accountID string) error
	ListAccountsPastDeletion(ctx context.Context, deletedBefore time.Time) ([]string, error)
	PurgeAccount(ctx context.Context, accountID string) error
	HasAuditEvent(ctx context.Context, accountID, event string) (bool, error)
	AccountEmails(ctx context.Context, accountID string) ([]string, error)
	WriteAuditEvent(ctx context.Context, ev types.AuditEvent) error
}

const (
	loginAttemptRetention = 24 * time.Hour
	authAuditRetention    = 90 * 24 * time.Hour

	photoPurgeRetention   = 30 * 24 * time.Hour // day 30: progress photos hard-deleted
	accountPurgeRetention = 90 * 24 * time.Hour // day 90: account hard-deleted
	photoReminderLead     = 5 * 24 * time.Hour  // ~day 25: 5 days before the photo purge
	finalReminderLead     = 5 * 24 * time.Hour  // ~day 85: 5 days before the final purge

	photoPurgeReminderEvent = "account.delete.reminder.photo"
	finalPurgeReminderEvent = "account.delete.reminder.final"
)

// photoPurgeReminderMessage and finalPurgeReminderMessage are the two
// retention reminder emails. Plain and link-free by design — unlike
// verification/reset emails these are informational, not action flows.
var (
	photoPurgeReminderMessage = mailer.Message{
		Subject:  "Your progress photos will be deleted in 5 days — DietDaemon",
		HTMLBody: `<p>Your DietDaemon account is scheduled for deletion. In about 5 days, your progress photos will be permanently deleted per our retention policy.</p><p>Log back in before then to reactivate your account and keep everything, including your photos.</p>`,
		TextBody: "Your DietDaemon account is scheduled for deletion. In about 5 days, your progress photos will be permanently deleted per our retention policy.\n\nLog back in before then to reactivate your account and keep everything, including your photos.",
	}
	finalPurgeReminderMessage = mailer.Message{
		Subject:  "Your account will be permanently deleted in 5 days — DietDaemon",
		HTMLBody: `<p>Your DietDaemon account is scheduled for permanent deletion in about 5 days. This is irreversible — all remaining data will be erased.</p><p>Log back in before then to reactivate your account.</p>`,
		TextBody: "Your DietDaemon account is scheduled for permanent deletion in about 5 days. This is irreversible — all remaining data will be erased.\n\nLog back in before then to reactivate your account.",
	}
)

// PurgeRunner periodically hard-deletes chat sessions that have been
// soft-deleted for longer than the 30-day retention window, plus the
// tiered account-deletion purges (photos at day 30, account at day 90) and
// their reminder emails.
type PurgeRunner struct {
	store    PurgeStore
	interval time.Duration
	mailer   mailer.Mailer
}

// NewPurgeRunner creates a PurgeRunner with the given store and tick interval.
func NewPurgeRunner(s PurgeStore, interval time.Duration) *PurgeRunner {
	return &PurgeRunner{store: s, interval: interval}
}

// WithMailer enables the day-25/day-85 retention reminder emails. Without
// it, PurgeRunner still purges photos at day 30 and accounts at day 90, but
// skips sending — and therefore never marks as sent — the reminder emails.
func (r *PurgeRunner) WithMailer(m mailer.Mailer) *PurgeRunner {
	r.mailer = m
	return r
}

// Run ticks until ctx is cancelled, purging expired soft-deleted sessions
// and running the tiered account-deletion purges and reminders.
func (r *PurgeRunner) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			now := time.Now()
			n, err := r.store.PurgeDeletedChatSessions(ctx, now.AddDate(0, 0, -30))
			if err != nil {
				slog.Error("purge deleted chat sessions", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("purged deleted chat sessions", "count", n)
			}
			if n, err := r.store.PurgeLoginAttempts(ctx, now.Add(-loginAttemptRetention)); err != nil {
				slog.Error("purge login attempts", "err", err)
			} else if n > 0 {
				slog.Info("purged login attempts", "count", n)
			}
			if n, err := r.store.PurgeAuthAuditEvents(ctx, now.Add(-authAuditRetention)); err != nil {
				slog.Error("purge auth audit events", "err", err)
			} else if n > 0 {
				slog.Info("purged auth audit events", "count", n)
			}

			r.purgeAccountPhotos(ctx, now)
			r.purgeAccounts(ctx, now)
			r.sendReminders(ctx, now)
		case <-ctx.Done():
			return
		}
	}
}

// purgeAccountPhotos runs the day-30 photo-purge tier. One account's
// failure is logged and skipped rather than aborting the rest.
func (r *PurgeRunner) purgeAccountPhotos(ctx context.Context, now time.Time) {
	ids, err := r.store.ListAccountsPendingPhotoPurge(ctx, now.Add(-photoPurgeRetention))
	if err != nil {
		slog.Error("list accounts pending photo purge", "err", err)
		return
	}
	for _, id := range ids {
		if err := r.store.PurgeAccountPhotos(ctx, id); err != nil {
			slog.Error("purge account photos", "account_id", id, "err", err)
			continue
		}
		slog.Info("purged account photos", "account_id", id)
	}
}

// purgeAccounts runs the day-90 full-purge tier. One account's failure is
// logged and skipped rather than aborting the rest.
func (r *PurgeRunner) purgeAccounts(ctx context.Context, now time.Time) {
	ids, err := r.store.ListAccountsPastDeletion(ctx, now.Add(-accountPurgeRetention))
	if err != nil {
		slog.Error("list accounts past retention", "err", err)
		return
	}
	for _, id := range ids {
		if err := r.store.PurgeAccount(ctx, id); err != nil {
			slog.Error("purge account", "account_id", id, "err", err)
			continue
		}
		slog.Info("purged account", "account_id", id)
	}
}

// sendReminders runs the two ~5-day-ahead reminder checks. No-op if no
// mailer was configured via WithMailer.
func (r *PurgeRunner) sendReminders(ctx context.Context, now time.Time) {
	if r.mailer == nil {
		return
	}

	if ids, err := r.store.ListAccountsPendingPhotoPurge(ctx, now.Add(-(photoPurgeRetention - photoReminderLead))); err != nil {
		slog.Error("list accounts for photo-purge reminder", "err", err)
	} else {
		r.remind(ctx, ids, photoPurgeReminderEvent, photoPurgeReminderMessage)
	}

	if ids, err := r.store.ListAccountsPastDeletion(ctx, now.Add(-(accountPurgeRetention - finalReminderLead))); err != nil {
		slog.Error("list accounts for final-purge reminder", "err", err)
	} else {
		r.remind(ctx, ids, finalPurgeReminderEvent, finalPurgeReminderMessage)
	}
}

// remind sends msg to every email under each account in ids that has not
// already received event, then records event in auth_audit_log so it is
// never sent twice. A send failure skips the audit write so the account is
// retried on the next tick rather than silently marked as reminded.
func (r *PurgeRunner) remind(ctx context.Context, ids []string, event string, msg mailer.Message) {
	for _, id := range ids {
		r.remindAccount(ctx, id, event, msg)
	}
}

// remindAccount sends msg to every email under account id if it hasn't
// already received event, then records event in auth_audit_log. Split out
// of remind to keep both under the cognitive-complexity limit.
func (r *PurgeRunner) remindAccount(ctx context.Context, id, event string, msg mailer.Message) {
	sent, err := r.store.HasAuditEvent(ctx, id, event)
	if err != nil {
		slog.Error("check reminder audit event", "account_id", id, "event", event, "err", err)
		return
	}
	if sent {
		return
	}

	emails, err := r.store.AccountEmails(ctx, id)
	if err != nil {
		slog.Error("list account emails", "account_id", id, "err", err)
		return
	}

	if !r.sendReminderEmails(ctx, id, event, msg, emails) {
		return
	}

	ev := types.AuditEvent{ID: newAuditID(), AccountID: id, Event: event, CreatedAt: time.Now()}
	if err := r.store.WriteAuditEvent(ctx, ev); err != nil {
		slog.Error("write reminder audit event", "account_id", id, "event", event, "err", err)
	}
}

// sendReminderEmails sends msg to every address in emails, logging any
// per-address failure. Returns false if any send failed, so the caller
// skips the audit write and retries the whole account on the next tick.
func (r *PurgeRunner) sendReminderEmails(ctx context.Context, id, event string, msg mailer.Message, emails []string) bool {
	allSent := true
	for _, email := range emails {
		if err := r.mailer.Send(ctx, email, msg); err != nil {
			slog.Error("send retention reminder email", "account_id", id, "event", event, "err", err)
			allSent = false
		}
	}
	return allSent
}

// newAuditID returns a short random hex ID for audit rows written from this
// package (mirrors store.newID's role, without exporting that helper).
func newAuditID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
