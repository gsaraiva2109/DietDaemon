// Package api implements the REST API for the DietDaemon dashboard. It uses
// the Go standard library net/http and http.ServeMux for routing. All endpoints
// return JSON and are gated behind ENABLE_DASHBOARD=true.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"time"

	gowa "github.com/go-webauthn/webauthn/webauthn"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/i18n"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
	"github.com/gsaraiva2109/dietdaemon/internal/oidc"
)

// AccountStore covers core user account and credential lookups.
type AccountStore interface {
	GetUserByEmail(ctx context.Context, email string) (types.User, error)
	CreateUserWithPassword(ctx context.Context, accountID, userID, email, displayName, phcHash string) (types.User, error)
	GetPasswordHash(ctx context.Context, userID string) (string, error)
	SetPasswordHash(ctx context.Context, userID, phcHash string) error
	CountUsers(ctx context.Context) (int, error)
	DeleteAccount(ctx context.Context, userID string) error
}

// APIKeyStore covers long-lived API key issuance and revocation.
type APIKeyStore interface {
	GetUserByAPIKey(ctx context.Context, hashedKey string) (types.User, error)
	CreateAPIKey(ctx context.Context, id, userID, hashedKey, label string) error
	ListAPIKeys(ctx context.Context, userID string) ([]types.APIKey, error)
	RevokeAPIKey(ctx context.Context, userID, keyID string) error
}

// ShareTokenStore covers read-only share links.
type ShareTokenStore interface {
	GetUserByShareToken(ctx context.Context, hashedToken string) (types.User, error)
	CreateShareToken(ctx context.Context, id, userID, hashedToken, label string) error
	ListShareTokens(ctx context.Context, userID string) ([]types.ShareToken, error)
	RevokeShareToken(ctx context.Context, userID, tokenID string) error
}

// AuditStore covers audit logging and login-attempt bookkeeping.
type AuditStore interface {
	WriteAuditEvent(ctx context.Context, ev types.AuditEvent) error
	RecordLoginAttempt(ctx context.Context, identifier string, succeeded bool) error
}

// OIDCStore covers OIDC identity linking and login-flow state.
type OIDCStore interface {
	GetUserByOIDCIdentity(ctx context.Context, provider, subject string) (types.User, error)
	LinkOIDCIdentity(ctx context.Context, id, userID, provider, subject, email string) error
	ListOIDCIdentities(ctx context.Context, userID string) ([]types.OIDCIdentity, error)
	DeleteOIDCIdentity(ctx context.Context, userID, id string) error
	CreateUserWithOIDC(ctx context.Context, accountID, userID, email, displayName, identityID, provider, subject string) (types.User, error)
	CreateOIDCState(ctx context.Context, id, nonce, pkceVerifier, linkUserID, next, expiresAt string) error
	ConsumeOIDCState(ctx context.Context, id string) (nonce, pkceVerifier, linkUserID, next string, err error)
	DeleteOIDCState(ctx context.Context, id string) error
}

// EmailTokenStore covers single-use email verification / address-change tokens.
type EmailTokenStore interface {
	MarkEmailVerified(ctx context.Context, userID string) error
	UpdateUserEmail(ctx context.Context, userID, email string) error
	CreateEmailToken(ctx context.Context, id, userID, purpose, expiresAt string) error
	ConsumeEmailToken(ctx context.Context, id, purpose string) (userID string, err error)
	DeleteEmailTokensByUserAndPurpose(ctx context.Context, userID, purpose string) error
}

// MagicCodeStore covers passwordless magic-link sign-in codes.
type MagicCodeStore interface {
	UpsertMagicCode(ctx context.Context, userID, codeHash, expiresAt string) error
	GetMagicCode(ctx context.Context, userID string) (codeHash, expiresAt string, attempts int, err error)
	IncrementMagicCodeAttempts(ctx context.Context, userID string) error
	DeleteMagicCode(ctx context.Context, userID string) error
}

// WebAuthnStore covers passkey handles, credentials, and ceremony sessions.
type WebAuthnStore interface {
	GetOrCreateWebAuthnHandle(ctx context.Context, userID string) (string, error)
	GetUserByWebAuthnHandle(ctx context.Context, handle string) (types.User, error)
	CreateWebAuthnCredential(ctx context.Context, id, userID, label, credentialJSON string, signCount int, createdAt string) error
	ListWebAuthnCredentials(ctx context.Context, userID string) ([]types.Passkey, error)
	GetWebAuthnCredentialsRaw(ctx context.Context, userID string) ([]types.WebAuthnCredential, error)
	UpdateWebAuthnCredentialOnAuth(ctx context.Context, id, credentialJSON string, signCount int, lastUsedAt string) error
	RenameWebAuthnCredential(ctx context.Context, userID, id, label string) error
	DeleteWebAuthnCredential(ctx context.Context, userID, id string) error

	// CreateWebAuthnSession Ceremony sessions.
	CreateWebAuthnSession(ctx context.Context, id, userID, sessionDataJSON, expiresAt string) error
	ConsumeWebAuthnSession(ctx context.Context, id string) (userID, sessionDataJSON string, err error)
}

// MFAEmailCodeStore covers one-time MFA codes delivered by email.
type MFAEmailCodeStore interface {
	UpsertMFAEmailCode(ctx context.Context, userID, codeHash, expiresAt string) error
	GetMFAEmailCode(ctx context.Context, userID string) (codeHash, expiresAt string, attempts int, err error)
	IncrementMFAEmailCodeAttempts(ctx context.Context, userID string) error
	DeleteMFAEmailCode(ctx context.Context, userID string) error
}

// AuthStore is the subset of store methods the auth endpoints need. It is
// composed of focused sub-interfaces (one per auth concern) so handlers and
// test doubles that only care about one concern can depend on that narrower
// interface instead of this god interface.
type AuthStore interface {
	AccountStore
	APIKeyStore
	ShareTokenStore
	AuditStore
	OIDCStore
	EmailTokenStore
	MagicCodeStore
	WebAuthnStore
	MFAEmailCodeStore
}

// AuthConfig bundles auth-related configuration for the Handler.
type AuthConfig struct {
	SessionCfg       auth.SessionConfig
	LockoutCfg       auth.LockoutConfig
	RegistrationMode types.RegistrationMode
	MultiUser        bool
	CookieSecure     bool
	CookieDomain     string
}

// MealStore is the subset of the store the API needs.
type MealStore interface {
	// GetMeal Meals & rollups.
	GetMeal(ctx context.Context, mealID string) (types.Meal, error)
	RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error)
	GetMealsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Meal, error)
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error)
	CorrectMealItem(ctx context.Context, userID string, mealID string, itemIndex int, corrected types.ResolvedItem) error
	AddMealItem(ctx context.Context, userID, mealID string, item types.ResolvedItem) error
	DeleteMealItem(ctx context.Context, userID, mealID string, itemIndex int) error
	SaveMeal(ctx context.Context, m types.Meal) error
	LatestMealTime(ctx context.Context, userID string) (string, error)

	// GetTargets Targets.
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	SetTargets(ctx context.Context, t types.DailyTargets) error
	UpdateRollupTargets(ctx context.Context, userID, localDate string, t types.Macros) error

	// GetNudgeRuleConfig Nudge rule config (per-user overrides of scheduler rules).
	GetNudgeRuleConfig(ctx context.Context, userID string) ([]types.NudgeRuleConfig, error)
	SetNudgeRuleConfig(ctx context.Context, userID, ruleID string, enabled bool, params json.RawMessage) error
	DeleteNudgeRuleConfig(ctx context.Context, userID, ruleID string) error

	// GetBackupConfig Scheduled backup settings.
	GetBackupConfig(ctx context.Context, userID string) (types.BackupConfig, error)
	SetBackupConfig(ctx context.Context, cfg types.BackupConfig) error

	// GetUserAIKey Per-user AI API keys (BYOK).
	GetUserAIKey(ctx context.Context, userID string) (provider string, encKey string, found bool, err error)
	SetUserAIKey(ctx context.Context, userID, provider, encKey string) error
	DeleteUserAIKey(ctx context.Context, userID string) error

	// GetUserHevyKey Per-user Hevy API keys (workout import).
	GetUserHevyKey(ctx context.Context, userID string) (encKey string, found bool, err error)
	SetUserHevyKey(ctx context.Context, userID, encKey string) error
	DeleteUserHevyKey(ctx context.Context, userID string) error

	// GetUser Users.
	GetUser(ctx context.Context, userID string) (types.User, error)
	UpsertUser(ctx context.Context, u types.User) error

	// ListFoods Food discovery.
	ListFoods(ctx context.Context, userID, source string, limit, offset int) ([]types.FoodDetail, error)
	SearchFoods(ctx context.Context, userID, query string) ([]types.FoodDetail, error)
	FrequentFoods(ctx context.Context, userID string, limit int) ([]types.FoodDetail, error)
	GetFoodDetail(ctx context.Context, userID, foodID string) (types.FoodDetail, error)
	GetFood(ctx context.Context, foodID string) (types.FoodMatch, error)
	GetFoodForUser(ctx context.Context, userID, foodID string) (types.FoodMatch, error)
	SearchCatalog(ctx context.Context, userID, query, source string, limit, offset int) ([]types.FoodDetail, error)
	RemoveFromLibrary(ctx context.Context, userID, foodID string) error
	AddToLibrary(ctx context.Context, userID, foodID string) error
	AddFoodAlias(ctx context.Context, userID, foodID, alias string) error
	DeleteFoodAlias(ctx context.Context, userID, foodID, alias string) error
	CreateCustomFood(ctx context.Context, userID string, input types.CustomFoodInput) (types.FoodDetail, error)
	UpdateCustomFood(ctx context.Context, userID, foodID string, input types.CustomFoodInput) (types.FoodDetail, error)
	DeleteCustomFood(ctx context.Context, userID, foodID string) error
	CreateFoodServingUnit(ctx context.Context, userID, foodID, label string, grams float64) (types.FoodServingUnit, error)
	DeleteFoodServingUnit(ctx context.Context, userID, unitID string) error

	// ListPendingAliases Pending aliases (embedding near-misses awaiting confirmation).
	ListPendingAliases(ctx context.Context, userID string) ([]types.PendingAlias, error)
	ConfirmPendingAlias(ctx context.Context, userID, id string) error
	RejectPendingAlias(ctx context.Context, userID, id string) error

	// GetSourcePrecedence Per-user nutrition source precedence.
	GetSourcePrecedence(ctx context.Context, userID string) ([]string, error)
	SetSourcePrecedence(ctx context.Context, userID string, order []string) error

	// GetFoodImportStatuses Bulk food-import status, keyed by source.
	GetFoodImportStatuses(ctx context.Context) ([]types.FoodImportStatus, error)

	// SaveTemplate Meal templates.
	SaveTemplate(ctx context.Context, t types.MealTemplate) error
	GetTemplates(ctx context.Context, userID string) ([]types.MealTemplate, error)
	GetTemplate(ctx context.Context, templateID string) (types.MealTemplate, error)
	DeleteTemplate(ctx context.Context, userID, templateID string) error
	LogTemplateUse(ctx context.Context, tl types.TemplateLog) error

	// ListWeight Body tracking — weight. LogWeight upserts by (user_id, date) — logging
	// twice in the same day overwrites the earlier entry — and returns the
	// persisted row's ID, which may differ from w.ID when an existing entry
	// was updated instead of a new one inserted.
	ListWeight(ctx context.Context, userID string, days int) ([]types.WeightEntry, error)
	LogWeight(ctx context.Context, w types.WeightEntry) (string, error)
	DeleteWeight(ctx context.Context, userID, entryID string) error
	WeightTrend(ctx context.Context, userID string, days int) ([]types.WeightTrend, error)

	// StartFast Fasting.
	StartFast(ctx context.Context, f types.Fast) error
	GetActiveFast(ctx context.Context, userID string) (types.Fast, error)
	EndFast(ctx context.Context, userID, fastID string, endAt time.Time, completed bool) (types.Fast, error)
	ListFasts(ctx context.Context, userID string, limit int) ([]types.Fast, error)

	// ListMeasurements Body tracking — measurements. LogMeasurement upserts by (user_id, date)
	// — same one-entry-per-day rule as LogWeight — and returns the persisted
	// row's ID.
	ListMeasurements(ctx context.Context, userID string, days int) ([]types.MeasurementEntry, error)
	LogMeasurement(ctx context.Context, m types.MeasurementEntry) (string, error)
	DeleteMeasurement(ctx context.Context, userID, entryID string) error

	// ListPhotoMetadata Body tracking — photos.
	ListPhotoMetadata(ctx context.Context, userID string) ([]types.ProgressPhoto, error)
	GetPhotoData(ctx context.Context, photoID string) (types.ProgressPhoto, error)
	UploadPhoto(ctx context.Context, p types.ProgressPhoto) error
	DeletePhoto(ctx context.Context, userID, photoID string) error

	// GetProfile Profile & goals.
	GetProfile(ctx context.Context, userID string) (types.UserProfile, error)
	UpsertProfile(ctx context.Context, p types.UserProfile) error

	// CreateLinkingCode Linking codes for bot account linking.
	CreateLinkingCode(ctx context.Context, userID, platform, code string) error
	LookupLinkingCode(ctx context.Context, code string) (types.LinkingCode, error)
	LookupLinkingCodeAny(ctx context.Context, code string) (types.LinkingCode, error)
	ConsumeLinkingCode(ctx context.Context, code string) error

	// LogWater Water tracking.
	LogWater(ctx context.Context, w types.WaterLog) error
	GetWaterToday(ctx context.Context, userID, localDate string) ([]types.WaterLog, int, error)
	DeleteWater(ctx context.Context, userID, id string) error
	GetWaterDailyTotals(ctx context.Context, userID, startDate, endDate string) ([]types.WaterDayTotal, error)

	// LogWorkout Workout tracking.
	LogWorkout(ctx context.Context, w types.Workout) error
	ImportWorkout(ctx context.Context, w types.Workout) error
	GetWorkout(ctx context.Context, id string) (types.Workout, error)
	ListWorkouts(ctx context.Context, userID string, limit int) ([]types.Workout, error)
	DeleteWorkout(ctx context.Context, userID, id string) error

	// LogSleep Sleep tracking.
	LogSleep(ctx context.Context, sl types.SleepLog) error
	GetActiveSleep(ctx context.Context, userID string) (*types.SleepLog, error)
	EndSleep(ctx context.Context, userID, id, wakeAt, quality string) error
	ListSleep(ctx context.Context, userID string, limit int) ([]types.SleepLog, error)
	DeleteSleep(ctx context.Context, userID, id string) error
}

// MealLogger submits raw text through the parsing pipeline, and can also directly
// log a fully-resolved meal (used by template logging and meal duplication).
// Satisfied by the pipeline.Engine.
type MealLogger interface {
	Handle(ctx context.Context, msg types.InboundMessage) error
	LogMeal(ctx context.Context, meal types.Meal) error
}

// Suggester recommends a next meal from what's left of today's targets and
// foods the user already eats. Satisfied by *suggest.Engine.
type Suggester interface {
	Suggest(ctx context.Context, userID string) (types.MealSuggestion, error)
	SuggestFromIngredients(ctx context.Context, userID string, foodIDs []string) (types.MealSuggestion, error)
}

// ChatStore is the persistence interface the chat assistant endpoints need.
// Satisfied by *store.Store.
type ChatStore interface {
	CreateChatSession(ctx context.Context, id, userID, title string) error
	ListChatSessions(ctx context.Context, userID string) ([]assistant.Session, error)
	AppendChatMessage(ctx context.Context, id, userID, sessionID, role, content, toolName string) error
	GetChatMessages(ctx context.Context, userID, sessionID string) ([]assistant.Message, error)
	GetAssistantSettings(ctx context.Context, userID string) (customInstructions string, found bool, err error)
	SetAssistantSettings(ctx context.Context, userID, customInstructions string) error
	SoftDeleteChatSession(ctx context.Context, userID, sessionID string) error
	RestoreChatSession(ctx context.Context, userID, sessionID string) error
	ListDeletedChatSessions(ctx context.Context, userID string) ([]assistant.Session, error)
}

// Handler serves the DietDaemon REST API.
type Handler struct {
	store     MealStore
	authStore AuthStore
	logger    MealLogger
	suggester Suggester
	loc       *time.Location

	// Auth sub-components.
	sessions      auth.SessionRepo
	loginAttempts auth.LoginAttemptRepo
	totp          auth.TOTPRepo
	mfaChallenges auth.MFAChallengeRepo
	recoveryCodes auth.RecoveryCodeRepo
	totpEncKey    []byte
	totpIssuer    string

	// OIDC.
	providers map[string]*oidc.Provider

	// Mailer.
	mailer        mailer.Mailer
	emailProvider string
	publicBaseURL string

	// WebAuthn.
	webauthn *gowa.WebAuthn

	// Auth config.
	sessionCfg       auth.SessionConfig
	lockoutCfg       auth.LockoutConfig
	registrationMode types.RegistrationMode
	multiUser        bool
	cookieSecure     bool
	cookieDomain     string

	// Rate limiters use client IPs for public auth and user IDs after auth.
	ipLimiter        *auth.IPRateLimiter
	readLimiter      *auth.IPRateLimiter
	writeLimiter     *auth.IPRateLimiter
	expensiveLimiter *auth.IPRateLimiter

	// Peers whose X-Forwarded-For / X-Real-IP headers are trusted when
	// resolving the client IP. See clientIP in handler_auth.go.
	trustedProxies []netip.Prefix

	// Scheduled backup manual trigger. Nil when backups aren't wired up.
	backupRunner BackupRunner

	// Admin-only food-import/repair/backfill trigger. Nil when unwired.
	foodImportRunner FoodImportRunner

	// Full config (needed by BYOK adapter construction).
	cfg *config.Config

	// Chat adapter for the conversational assistant (nil when unsupported).
	chatAdapter ports.ChatAdapter

	// Vision adapter for OCR nutrition-label capture (nil when OCR_ADAPTER unset).
	visionAdapter ports.VisionAdapter

	// Assistant router for the chat endpoint (nil when unsupported).
	assistantRouter *assistant.Router

	// Tool descriptions and commands needed for per-user BYOK router construction.
	chatCommands []ports.Command
	toolDescs    map[string]string

	// Chat persistence (sessions, messages, settings).
	chatStore ChatStore

	// I18n bundle for localized system prompts.
	i18nBundle *i18n.Bundle
}

// BackupRunner triggers an immediate backup for one user, sharing the same
// export logic the scheduled ticker uses. Satisfied by *backup.Runner.
type BackupRunner interface {
	RunOnce(ctx context.Context, userID string) error
}

// FoodImportRunner triggers the global food-catalog bulk import, macro repair,
// and embedding backfill operations that cmd/import-foods otherwise requires
// direct DB/filesystem access to run, so an operator can trigger them against
// a live daemon over HTTP instead (issue #136). Satisfied by
// cmd/dietdaemon's foodImportAdmin.
type FoodImportRunner interface {
	ImportSource(ctx context.Context, source string, maxRows int) (rows int, err error)
	RepairSource(ctx context.Context, source string) (checked, fixed int, err error)
	BackfillEmbeddings(ctx context.Context) (embedded, failed int, err error)
}

// Option configures a Handler. Used with the variadic New constructor. This
// mirrors internal/scheduler.Option: it exists because *store.Store satisfies
// most of these params (AuthStore, auth.SessionRepo, auth.LoginAttemptRepo,
// auth.TOTPRepo, auth.MFAChallengeRepo, auth.RecoveryCodeRepo, ChatStore) at
// once, so a plain positional New() would let a future param reorder compile
// silently while wiring the wrong value into the wrong field. Grouping the
// related values behind named options removes that footgun.
type Option func(*Handler)

// WithAuth attaches the auth subsystem: the auth-specific store view plus its
// session/lockout/TOTP/MFA/recovery-code repos and config. authStore and the
// five repo params are typically all the same concrete *store.Store, cast to
// different narrow interfaces.
func WithAuth(authStore AuthStore, sessions auth.SessionRepo, loginAttempts auth.LoginAttemptRepo, totpRepo auth.TOTPRepo, mfaChallenges auth.MFAChallengeRepo, recoveryCodes auth.RecoveryCodeRepo, totpEncKey []byte, totpIssuer string, cfg AuthConfig) Option {
	return func(h *Handler) {
		h.authStore = authStore
		h.sessions = sessions
		h.loginAttempts = loginAttempts
		h.totp = totpRepo
		h.mfaChallenges = mfaChallenges
		h.recoveryCodes = recoveryCodes
		h.totpEncKey = totpEncKey
		h.totpIssuer = totpIssuer
		h.sessionCfg = cfg.SessionCfg
		h.lockoutCfg = cfg.LockoutCfg
		h.registrationMode = cfg.RegistrationMode
		h.multiUser = cfg.MultiUser
		h.cookieSecure = cfg.CookieSecure
		h.cookieDomain = cfg.CookieDomain
	}
}

// WithOIDC attaches the registry of configured OIDC providers. A nil map is
// normalized to an empty one, same as the pre-options constructor did.
func WithOIDC(providers map[string]*oidc.Provider) Option {
	return func(h *Handler) {
		if providers == nil {
			providers = map[string]*oidc.Provider{}
		}
		h.providers = providers
	}
}

// WithMailer attaches the mailer and the provider name used for its
// human-readable error messages.
func WithMailer(m mailer.Mailer, emailProvider string) Option {
	return func(h *Handler) {
		h.mailer = m
		h.emailProvider = emailProvider
	}
}

// WithPublicBaseURL sets the externally reachable base URL used to build
// links in outgoing emails (verification, password reset, etc.).
func WithPublicBaseURL(publicBaseURL string) Option {
	return func(h *Handler) { h.publicBaseURL = publicBaseURL }
}

// WithWebAuthn attaches the WebAuthn (passkey) relying party.
func WithWebAuthn(wa *gowa.WebAuthn) Option {
	return func(h *Handler) { h.webauthn = wa }
}

// WithBackupRunner attaches the manual "run now" backup trigger. When not
// passed, the endpoint returns 503, same as passing nil did before options.
func WithBackupRunner(r BackupRunner) Option {
	return func(h *Handler) { h.backupRunner = r }
}

// WithFoodImportRunner attaches the admin food-import/repair/backfill
// trigger. When not passed, the admin/food-import/* endpoints return 503.
func WithFoodImportRunner(r FoodImportRunner) Option {
	return func(h *Handler) { h.foodImportRunner = r }
}

// WithChat attaches the conversational assistant subsystem: its model
// adapter, tool-calling router, available commands/descriptions, and
// persistence. When not passed, the chat endpoints stay unsupported (nil
// adapter/router), same as before options.
func WithChat(adapter ports.ChatAdapter, router *assistant.Router, commands []ports.Command, toolDescs map[string]string, store ChatStore) Option {
	return func(h *Handler) {
		h.chatAdapter = adapter
		h.assistantRouter = router
		h.chatCommands = commands
		h.toolDescs = toolDescs
		h.chatStore = store
	}
}

// WithI18n attaches the i18n bundle used for localized system prompts.
func WithI18n(bundle *i18n.Bundle) Option {
	return func(h *Handler) { h.i18nBundle = bundle }
}

// WithOCR attaches the vision adapter used for OCR nutrition-label capture
// (issue #87). When not passed, the scan endpoint returns 501.
func WithOCR(adapter ports.VisionAdapter) Option {
	return func(h *Handler) { h.visionAdapter = adapter }
}

// New returns a ready API Handler. store, logger, loc, suggester, and c are
// the params every caller needs regardless of configuration; everything else
// is attached via Option (see WithAuth, WithMailer, WithChat, etc.) so that
// values sharing a concrete type — most notably *store.Store, which satisfies
// AuthStore and several auth.*Repo interfaces at once — are wired through
// named, self-documenting calls instead of a long positional list.
func New(store MealStore, logger MealLogger, loc *time.Location, suggester Suggester, c *config.Config, opts ...Option) *Handler {
	if loc == nil {
		loc = time.UTC
	}
	publicLimit, readLimit, writeLimit, expensiveLimit := rateLimits(c)
	h := &Handler{
		store:            store,
		logger:           logger,
		loc:              loc,
		suggester:        suggester,
		cfg:              c,
		providers:        map[string]*oidc.Provider{},
		ipLimiter:        auth.NewIPRateLimiter(publicLimit, time.Minute),
		readLimiter:      auth.NewIPRateLimiter(readLimit, time.Minute),
		writeLimiter:     auth.NewIPRateLimiter(writeLimit, time.Minute),
		expensiveLimiter: auth.NewIPRateLimiter(expensiveLimit, time.Minute),
		trustedProxies:   trustedProxyPrefixes(c),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func rateLimits(c *config.Config) (public, read, write, expensive int) {
	public, read, write, expensive = 10, 120, 30, 10
	if c == nil {
		return
	}
	if c.PublicRateLimitPerMinute > 0 {
		public = c.PublicRateLimitPerMinute
	}
	if c.AuthenticatedReadRateLimitPerMinute > 0 {
		read = c.AuthenticatedReadRateLimitPerMinute
	}
	if c.AuthenticatedWriteRateLimitPerMinute > 0 {
		write = c.AuthenticatedWriteRateLimitPerMinute
	}
	if c.AuthenticatedExpensiveRateLimitPerMinute > 0 {
		expensive = c.AuthenticatedExpensiveRateLimitPerMinute
	}
	return
}

// StartRateLimiterCleanup releases inactive per-IP and per-user limiter
// buckets until ctx is canceled by application shutdown.
func (h *Handler) StartRateLimiterCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.ipLimiter.Cleanup()
				h.readLimiter.Cleanup()
				h.writeLimiter.Cleanup()
				h.expensiveLimiter.Cleanup()
			}
		}
	}()
}

// trustedProxyPrefixes returns the configured trusted-proxy allowlist, or
// nil when no config was supplied (e.g. in tests that pass a nil *config.Config).
func trustedProxyPrefixes(c *config.Config) []netip.Prefix {
	if c == nil {
		return nil
	}
	return c.TrustedProxyPrefixes()
}

// RegisterRoutes mounts all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Health — no auth, no rate limit. Used by orchestration probes.
	mux.HandleFunc("GET /api/v1/healthz", withAPIErrorEnvelope(http.HandlerFunc(h.handleHealthz)))

	// Existing.
	mux.HandleFunc("GET /api/v1/rollups/today", h.wrap(h.handleRollupsToday))
	mux.HandleFunc("GET /api/v1/rollups/range", h.wrap(h.handleRollupsRange))
	mux.HandleFunc("GET /api/v1/meals", h.wrap(h.handleMealsList))
	mux.HandleFunc("GET /api/v1/meals/{mealID}", h.wrap(h.handleMealDetail))
	mux.HandleFunc("POST /api/v1/meals/{mealID}/items/{itemID}/correct", h.wrap(h.handleCorrectItem))
	mux.HandleFunc("POST /api/v1/meals/{mealID}/items", h.wrap(h.handleAddItem))
	mux.HandleFunc("DELETE /api/v1/meals/{mealID}/items/{itemID}", h.wrap(h.handleDeleteItem))
	mux.HandleFunc("POST /api/v1/meals/log", h.wrap(h.handleLogMeal))
	mux.HandleFunc("POST /api/v1/meals", h.wrap(h.handleCreateStructuredMeal))
	mux.HandleFunc("GET /api/v1/targets", h.wrap(h.handleGetTargets))
	mux.HandleFunc("PUT /api/v1/targets", h.wrap(h.handleSetTargets))
	mux.HandleFunc("GET /api/v1/budget/weekly", h.wrap(h.handleGetBudgetWeekly))
	mux.HandleFunc("GET /api/v1/settings/nudges", h.wrap(h.handleGetNudgeSettings))
	mux.HandleFunc("PUT /api/v1/settings/nudges", h.wrap(h.handleSetNudgeSettings))

	// Meals — latest.
	mux.HandleFunc("GET /api/v1/meals/latest", h.wrap(h.handleMealsLatest))

	// Food discovery.
	mux.HandleFunc("GET /api/v1/foods", h.wrap(h.handleListFoods))
	mux.HandleFunc("GET /api/v1/foods/search", h.wrap(h.handleSearchFoods))
	mux.HandleFunc("GET /api/v1/foods/frequent", h.wrap(h.handleFrequentFoods))
	mux.HandleFunc("POST /api/v1/foods/custom", h.wrap(h.handleCreateCustomFood))
	mux.HandleFunc("POST /api/v1/foods/custom/ocr", h.wrap(h.handleOCRExtractCustomFood))
	mux.HandleFunc("GET /api/v1/foods/{foodID}", h.wrap(h.handleGetFood))
	mux.HandleFunc("PUT /api/v1/foods/{foodID}/custom", h.wrap(h.handleUpdateCustomFood))
	mux.HandleFunc("DELETE /api/v1/foods/{foodID}/custom", h.wrap(h.handleDeleteCustomFood))
	mux.HandleFunc("POST /api/v1/foods/{foodID}/units", h.wrap(h.handleCreateFoodServingUnit))
	mux.HandleFunc("DELETE /api/v1/foods/{foodID}/units/{unitID}", h.wrap(h.handleDeleteFoodServingUnit))
	mux.HandleFunc("GET /api/v1/suggest", h.wrap(h.handleSuggest))
	mux.HandleFunc("POST /api/v1/suggest/ingredients", h.wrap(h.handleSuggestFromIngredients))
	mux.HandleFunc("POST /api/v1/foods/{foodID}/aliases", h.wrap(h.handleAddAlias))
	mux.HandleFunc("DELETE /api/v1/foods/{foodID}/aliases/{alias}", h.wrap(h.handleDeleteAlias))
	mux.HandleFunc("GET /api/v1/catalog/search", h.wrap(h.handleSearchCatalog))
	mux.HandleFunc("DELETE /api/v1/foods/{foodID}/library", h.wrap(h.handleRemoveFromLibrary))
	mux.HandleFunc("POST /api/v1/foods/{foodID}/library", h.wrap(h.handleAddToLibrary))

	// Pending aliases.
	mux.HandleFunc("GET /api/v1/aliases/pending", h.wrap(h.handleListPendingAliases))
	mux.HandleFunc("POST /api/v1/aliases/pending/{id}/confirm", h.wrap(h.handleConfirmPendingAlias))
	mux.HandleFunc("DELETE /api/v1/aliases/pending/{id}", h.wrap(h.handleRejectPendingAlias))

	// Nutrition source precedence.
	mux.HandleFunc("GET /api/v1/settings/precedence", h.wrap(h.handleGetPrecedence))
	mux.HandleFunc("PUT /api/v1/settings/precedence", h.wrap(h.handleSetPrecedence))

	// Bulk food-import status.
	mux.HandleFunc("GET /api/v1/food-import/status", h.wrap(h.handleFoodImportStatus))

	// Meal templates.
	mux.HandleFunc("GET /api/v1/templates", h.wrap(h.handleListTemplates))
	mux.HandleFunc("POST /api/v1/templates", h.wrap(h.handleCreateTemplate))
	mux.HandleFunc("POST /api/v1/templates/compose", h.wrap(h.handleComposeTemplate))
	mux.HandleFunc("GET /api/v1/templates/{id}", h.wrap(h.handleGetTemplate))
	mux.HandleFunc("DELETE /api/v1/templates/{id}", h.wrap(h.handleDeleteTemplate))
	mux.HandleFunc("POST /api/v1/templates/{id}/log", h.wrap(h.handleLogTemplate))
	mux.HandleFunc("POST /api/v1/meals/{mealID}/duplicate", h.wrap(h.handleDuplicateMeal))

	// Body tracking — weight.
	mux.HandleFunc("GET /api/v1/body/weight", h.wrap(h.handleListWeight))
	mux.HandleFunc("POST /api/v1/body/weight", h.wrap(h.handleLogWeight))
	mux.HandleFunc("GET /api/v1/body/weight/trend", h.wrap(h.handleWeightTrend))
	mux.HandleFunc("DELETE /api/v1/body/weight/{id}", h.wrap(h.handleDeleteWeight))

	// Fasting.
	mux.HandleFunc("POST /api/v1/fasting/start", h.wrap(h.handleStartFast))
	mux.HandleFunc("POST /api/v1/fasting/end", h.wrap(h.handleEndFast))
	mux.HandleFunc("GET /api/v1/fasting/active", h.wrap(h.handleGetActiveFast))
	mux.HandleFunc("GET /api/v1/fasting/history", h.wrap(h.handleListFasts))

	// Water tracking.
	mux.HandleFunc("POST /api/v1/body/water", h.wrap(h.handleLogWater))
	mux.HandleFunc("GET /api/v1/body/water", h.wrap(h.handleGetWaterToday))
	mux.HandleFunc("DELETE /api/v1/body/water/{id}", h.wrap(h.handleDeleteWater))

	// Workout tracking.
	mux.HandleFunc("POST /api/v1/body/workouts", h.wrap(h.handleLogWorkout))
	mux.HandleFunc("GET /api/v1/body/workouts", h.wrap(h.handleListWorkouts))
	mux.HandleFunc("GET /api/v1/body/workouts/{id}", h.wrap(h.handleGetWorkout))
	mux.HandleFunc("DELETE /api/v1/body/workouts/{id}", h.wrap(h.handleDeleteWorkout))

	// Sleep tracking.
	mux.HandleFunc("POST /api/v1/body/sleep", h.wrap(h.handleLogSleep))
	mux.HandleFunc("GET /api/v1/body/sleep", h.wrap(h.handleListSleep))
	mux.HandleFunc("GET /api/v1/body/sleep/active", h.wrap(h.handleGetActiveSleep))
	mux.HandleFunc("PATCH /api/v1/body/sleep/{id}/end", h.wrap(h.handleEndSleep))
	mux.HandleFunc("DELETE /api/v1/body/sleep/{id}", h.wrap(h.handleDeleteSleep))

	// Body tracking — measurements.
	mux.HandleFunc("GET /api/v1/body/measurements", h.wrap(h.handleListMeasurements))
	mux.HandleFunc("POST /api/v1/body/measurements", h.wrap(h.handleLogMeasurements))
	mux.HandleFunc("DELETE /api/v1/body/measurements/{id}", h.wrap(h.handleDeleteMeasurement))

	// Body tracking — photos.
	mux.HandleFunc("GET /api/v1/body/photos", h.wrap(h.handleListPhotos))
	mux.HandleFunc("GET /api/v1/body/photos/{id}/data", h.wrap(h.handlePhotoData))
	mux.HandleFunc("POST /api/v1/body/photos", h.wrap(h.handleUploadPhoto))
	mux.HandleFunc("DELETE /api/v1/body/photos/{id}", h.wrap(h.handleDeletePhoto))

	// Body tracking — summary.
	mux.HandleFunc("GET /api/v1/body/summary", h.wrap(h.handleBodySummary))

	// Goals & profile.
	mux.HandleFunc("GET /api/v1/profile", h.wrap(h.handleGetProfile))
	mux.HandleFunc("PUT /api/v1/profile", h.wrap(h.handleUpsertProfile))
	mux.HandleFunc("GET /api/v1/tdee", h.wrap(h.handleCalculateTDEE))
	mux.HandleFunc("GET /api/v1/goals/suggestions", h.wrap(h.handleGoalSuggestions))

	// Data export.
	mux.HandleFunc("GET /api/v1/export/meals", h.wrap(h.handleExportMeals))
	mux.HandleFunc("GET /api/v1/export/rollups", h.wrap(h.handleExportRollups))
	mux.HandleFunc("GET /api/v1/export/all", h.wrap(h.handleExportAll))

	// Account deletion.
	mux.HandleFunc("DELETE /api/v1/account", h.wrap(h.handleDeleteAccount))

	// Scheduled backup settings.
	mux.HandleFunc("GET /api/v1/settings/backup", h.wrap(h.handleGetBackupConfig))
	mux.HandleFunc("PUT /api/v1/settings/backup", h.wrap(h.handleSetBackupConfig))
	mux.HandleFunc("POST /api/v1/settings/backup/run", h.wrap(h.handleRunBackupNow))

	// Admin: operator-only food-catalog import/repair/backfill triggers.
	// Gated by API_AUTH_TOKEN via wrapAdmin, not the per-user session/API-key
	// auth the rest of this file uses — these mutate the global food catalog,
	// not user-scoped data. See issue #136.
	mux.HandleFunc("POST /api/v1/admin/food-import/run", h.wrapAdmin(h.handleAdminFoodImportRun))
	mux.HandleFunc("POST /api/v1/admin/food-import/repair", h.wrapAdmin(h.handleAdminFoodImportRepair))
	mux.HandleFunc("POST /api/v1/admin/food-import/backfill-embeddings", h.wrapAdmin(h.handleAdminFoodImportBackfillEmbeddings))

	// BYOK: per-user AI API keys.
	mux.HandleFunc("GET /api/v1/settings/ai-key", h.wrap(h.handleGetAIKey))
	mux.HandleFunc("POST /api/v1/settings/ai-key", h.wrap(h.handleSetAIKey))
	mux.HandleFunc("DELETE /api/v1/settings/ai-key", h.wrap(h.handleDeleteAIKey))

	// Hevy: per-user API key + workout import.
	mux.HandleFunc("GET /api/v1/settings/hevy-key", h.wrap(h.handleGetHevyKey))
	mux.HandleFunc("POST /api/v1/settings/hevy-key", h.wrap(h.handleSetHevyKey))
	mux.HandleFunc("DELETE /api/v1/settings/hevy-key", h.wrap(h.handleDeleteHevyKey))
	mux.HandleFunc("POST /api/v1/import/hevy", h.wrap(h.handleImportHevy))

	// Auth endpoints.
	mux.HandleFunc("POST /api/v1/auth/register", h.wrapPublicLimited(h.handleRegister))
	mux.HandleFunc("POST /api/v1/auth/login", h.wrapPublicLimited(h.handleLogin))
	mux.HandleFunc("POST /api/v1/auth/logout", h.wrap(h.handleLogout))
	mux.HandleFunc("GET /api/v1/auth/session", h.wrap(h.handleSession))
	mux.HandleFunc("GET /api/v1/auth/providers", h.wrapPublic(h.handleProviders))
	mux.HandleFunc("POST /api/v1/auth/change-password", h.wrap(h.handleChangePassword))
	mux.HandleFunc("GET /api/v1/auth/api-keys", h.wrap(h.handleListAPIKeys))
	mux.HandleFunc("POST /api/v1/auth/api-keys", h.wrap(h.handleCreateAPIKey))
	mux.HandleFunc("DELETE /api/v1/auth/api-keys/{id}", h.wrap(h.handleRevokeAPIKey))
	mux.HandleFunc("GET /api/v1/auth/share-tokens", h.wrap(h.handleListShareTokens))
	mux.HandleFunc("POST /api/v1/auth/share-tokens", h.wrap(h.handleCreateShareToken))
	mux.HandleFunc("DELETE /api/v1/auth/share-tokens/{id}", h.wrap(h.handleRevokeShareToken))

	// TOTP two-factor authentication.
	mux.HandleFunc("POST /api/v1/auth/totp/enroll", h.wrap(h.handleTOTPEnroll))
	mux.HandleFunc("POST /api/v1/auth/totp/verify", h.wrap(h.handleTOTPVerify))
	mux.HandleFunc("POST /api/v1/auth/totp/challenge", h.wrapPublicLimited(h.handleTOTPChallenge))
	mux.HandleFunc("DELETE /api/v1/auth/totp", h.wrap(h.handleTOTPDisable))
	mux.HandleFunc("POST /api/v1/auth/totp/recovery-codes/regenerate", h.wrap(h.handleRegenerateRecovery))

	// OIDC client login + account linking.
	mux.HandleFunc("GET /api/v1/auth/oidc/{id}/start", h.wrapPublicLimited(h.handleOIDCStart))
	mux.HandleFunc("GET /api/v1/auth/oidc/{id}/callback", h.wrapPublicLimited(h.handleOIDCCallback))
	mux.HandleFunc("GET /api/v1/auth/identities", h.wrap(h.handleListIdentities))
	mux.HandleFunc("DELETE /api/v1/auth/identities/{id}", h.wrap(h.handleUnlinkIdentity))

	// Email verification + password reset.
	mux.HandleFunc("POST /api/v1/auth/email/verify", h.wrapPublicLimited(h.handleEmailVerify))
	mux.HandleFunc("POST /api/v1/auth/email/verify/resend", h.wrap(h.handleResendVerify))
	mux.HandleFunc("POST /api/v1/auth/email/change", h.wrap(h.handleEmailChange))
	mux.HandleFunc("POST /api/v1/auth/password/forgot", h.wrapPublicLimited(h.handleForgotPassword))
	mux.HandleFunc("POST /api/v1/auth/password/reset", h.wrapPublicLimited(h.handleResetPassword))

	// Passwordless email sign-in.
	mux.HandleFunc("POST /api/v1/auth/magic/request", h.wrapPublicLimited(h.handleMagicRequest))
	mux.HandleFunc("POST /api/v1/auth/magic/verify", h.wrapPublicLimited(h.handleMagicVerify))

	// Passkeys (WebAuthn) — management and login.
	mux.HandleFunc("GET /api/v1/auth/passkeys", h.wrap(h.handleListPasskeys))
	mux.HandleFunc("POST /api/v1/auth/passkeys/register/begin", h.wrap(h.handlePasskeyRegisterBegin))
	mux.HandleFunc("POST /api/v1/auth/passkeys/register/finish", h.wrap(h.handlePasskeyRegisterFinish))
	mux.HandleFunc("PATCH /api/v1/auth/passkeys/{id}", h.wrap(h.handleRenamePasskey))
	mux.HandleFunc("DELETE /api/v1/auth/passkeys/{id}", h.wrap(h.handleDeletePasskey))
	mux.HandleFunc("POST /api/v1/auth/passkeys/login/begin", h.wrapPublicLimited(h.handlePasskeyLoginBegin))
	mux.HandleFunc("POST /api/v1/auth/passkeys/login/finish", h.wrapPublicLimited(h.handlePasskeyLoginFinish))
	// Passkey-as-2FA + email-OTP fallback.
	mux.HandleFunc("POST /api/v1/auth/mfa/passkey/begin", h.wrapPublicLimited(h.handleMFAPasskeyBegin))
	mux.HandleFunc("POST /api/v1/auth/mfa/passkey/finish", h.wrapPublicLimited(h.handleMFAPasskeyFinish))
	mux.HandleFunc("POST /api/v1/auth/mfa/email/send", h.wrapPublicLimited(h.handleMFAEmailSend))
	mux.HandleFunc("POST /api/v1/auth/mfa/email/verify", h.wrapPublicLimited(h.handleMFAEmailVerify))

	// Adherence streak.
	mux.HandleFunc("GET /api/v1/streak", h.wrap(h.handleStreak))

	// AI chat assistant.
	mux.HandleFunc("POST /api/v1/chat/sessions/{id}/messages", h.wrap(h.handleChatMessage))
	mux.HandleFunc("GET /api/v1/chat/sessions/deleted", h.wrap(h.handleListDeletedChatSessions))
	mux.HandleFunc("GET /api/v1/chat/sessions", h.wrap(h.handleListChatSessions))
	mux.HandleFunc("GET /api/v1/chat/sessions/{id}/messages", h.wrap(h.handleGetChatMessages))
	mux.HandleFunc("DELETE /api/v1/chat/sessions/{id}", h.wrap(h.handleDeleteChatSession))
	mux.HandleFunc("POST /api/v1/chat/sessions/{id}/restore", h.wrap(h.handleRestoreChatSession))
	mux.HandleFunc("GET /api/v1/chat/settings", h.wrap(h.handleGetChatSettings))
	mux.HandleFunc("PUT /api/v1/chat/settings", h.wrap(h.handleSetChatSettings))

	// Bot account linking.
	mux.HandleFunc("POST /api/v1/bot/link-code", h.wrap(h.handleCreateLinkCode))
	mux.HandleFunc("POST /api/v1/bot/link", h.wrap(h.handleCompleteLink))
	mux.HandleFunc("GET /api/v1/bot/link-code/{code}/stream", h.wrap(h.handleStreamLinkCode))

	// Shared read-only dashboard — same handlers as above, mounted under a
	// distinct prefix and authenticated by the token in the URL instead of a
	// cookie/API key. No new handler code: the share token just resolves to
	// a userID and every one of these already scopes its query by userID.
	mux.HandleFunc("GET /api/v1/shared/{token}/rollups/today", h.wrapReadOnly(h.handleRollupsToday))
	mux.HandleFunc("GET /api/v1/shared/{token}/meals", h.wrapReadOnly(h.handleMealsList))
	mux.HandleFunc("GET /api/v1/shared/{token}/targets", h.wrapReadOnly(h.handleGetTargets))
	mux.HandleFunc("GET /api/v1/shared/{token}/budget/weekly", h.wrapReadOnly(h.handleGetBudgetWeekly))
	mux.HandleFunc("GET /api/v1/shared/{token}/body/summary", h.wrapReadOnly(h.handleBodySummary))
	mux.HandleFunc("GET /api/v1/shared/{token}/streak", h.wrapReadOnly(h.handleStreak))
	mux.HandleFunc("/api/v1/shared/{token}/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, ErrorMethodNotAllowed, "Method not allowed.")
			return
		}
		WriteError(w, http.StatusNotFound, ErrorNotFound, "Not found.")
	})

	// Keep API misses inside the JSON contract instead of falling through to
	// the dashboard handler mounted at "/".
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, _ *http.Request) {
		WriteError(w, http.StatusNotFound, ErrorNotFound, "Not found.")
	})
}

// wrap applies auth middleware and JSON content-type headers to a handler.
// The handler receives the authenticated userID.
func (h *Handler) wrap(next func(w http.ResponseWriter, r *http.Request, userID string)) http.HandlerFunc {
	return withAPIErrorEnvelope(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := h.authenticate(r)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, ErrorUnauthorized, "Unauthorized.")
			return
		}
		if !h.authLimiter(r).Allow(userID) {
			w.Header().Set("Retry-After", "60")
			WriteError(w, http.StatusTooManyRequests, ErrorRateLimited, "Too many requests.")
			return
		}
		next(w, r, userID)
	}))
}

func (h *Handler) authLimiter(r *http.Request) *auth.IPRateLimiter {
	if isExpensiveRequest(r) {
		return h.expensiveLimiter
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return h.readLimiter
	}
	return h.writeLimiter
}

func isExpensiveRequest(r *http.Request) bool {
	path := r.URL.Path
	return strings.HasPrefix(path, "/api/v1/chat/") ||
		path == "/api/v1/suggest" ||
		path == "/api/v1/suggest/ingredients" ||
		path == "/api/v1/goals/suggestions" ||
		path == "/api/v1/foods/custom/ocr" ||
		path == "/api/v1/settings/backup/run"
}

// wrapReadOnly authenticates via a share token embedded in the URL path
// (not a cookie or Bearer header — a share link opens standalone in a
// browser with no DietDaemon session) and rejects anything but GET. It
// deliberately skips the cookie/CSRF machinery in authenticate(): a share
// link has no session to forge a CSRF token against, and the GET-only
// restriction is what keeps it from being a mutation vector.
func (h *Handler) wrapReadOnly(next func(w http.ResponseWriter, r *http.Request, userID string)) http.HandlerFunc {
	return withAPIErrorEnvelope(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, ErrorMethodNotAllowed, "Method not allowed.")
			return
		}

		token := r.PathValue("token")
		hashed := auth.HashToken(token)
		u, err := h.authStore.GetUserByShareToken(r.Context(), hashed)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, ErrorUnauthorized, "Unauthorized.")
			return
		}
		if !h.authLimiter(r).Allow(u.ID) {
			w.Header().Set("Retry-After", "60")
			WriteError(w, http.StatusTooManyRequests, ErrorRateLimited, "Too many requests.")
			return
		}
		next(w, r, u.ID)
	}))
}

// wrapPublic sets JSON headers but performs no authentication.
func (h *Handler) wrapPublic(next http.HandlerFunc) http.HandlerFunc {
	return withAPIErrorEnvelope(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}))
}

// handleHealthz is a liveness probe for orchestration health checks.
// No auth, no rate limit — it only confirms the HTTP server is alive.
func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// wrapPublicLimited adds per-IP rate limiting on top of wrapPublic.
func (h *Handler) wrapPublicLimited(next http.HandlerFunc) http.HandlerFunc {
	return h.wrapPublic(func(w http.ResponseWriter, r *http.Request) {
		if !h.ipLimiter.Allow(h.clientIP(r)) {
			w.Header().Set("Retry-After", "60")
			WriteError(w, http.StatusTooManyRequests, ErrorRateLimited, "Too many requests.")
			return
		}
		next(w, r)
	})
}

// wrapAdmin gates operator-only admin endpoints behind the static
// API_AUTH_TOKEN bearer token (cfg.APIAuthToken) instead of the normal
// per-user authenticate() path: these endpoints trigger a global food-catalog
// import/repair/embedding pass, not a user-scoped action, so there is no
// userID to authenticate as. When no token is configured, admin endpoints are
// disabled entirely (503) rather than left open. See issue #136.
func (h *Handler) wrapAdmin(next http.HandlerFunc) http.HandlerFunc {
	return withAPIErrorEnvelope(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if h.cfg == nil || h.cfg.APIAuthToken == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "admin endpoints are not enabled on this server"})
			return
		}
		token := bearerToken(r)
		if subtle.ConstantTimeCompare([]byte(token), []byte(h.cfg.APIAuthToken)) != 1 {
			WriteError(w, http.StatusUnauthorized, ErrorUnauthorized, "Unauthorized.")
			return
		}
		if !h.expensiveLimiter.Allow("admin") {
			w.Header().Set("Retry-After", "60")
			WriteError(w, http.StatusTooManyRequests, ErrorRateLimited, "Too many requests.")
			return
		}
		next(w, r)
	}))
}

// authenticate tries cookie-session first, then Bearer API key. Returns the
// authenticated userID or an error. For cookie sessions on mutating methods,
// CSRF is verified via the double-submit cookie pattern.
func (h *Handler) authenticate(r *http.Request) (string, error) {
	// 1. Cookie session.
	if cookie := readSessionCookie(r); cookie != "" {
		sess, result, err := auth.ValidateSession(r.Context(), h.sessions, cookie, h.sessionCfg)
		if err == nil && result == auth.ValidateOK {
			// CSRF on mutating methods.
			if isMutating(r.Method) {
				csrfHeader := r.Header.Get("X-CSRF-Token")
				if !auth.VerifyCSRF(csrfHeader, sess.CSRFToken) {
					return "", fmt.Errorf("csrf mismatch")
				}
			}
			// Slide the idle expiry forward.
			now := time.Now().UTC()
			idleExpires := now.Add(h.sessionCfg.IdleTTL)
			if idleExpires.After(sess.AbsoluteExpiresAt) {
				idleExpires = sess.AbsoluteExpiresAt
			}
			_ = h.sessions.TouchSession(r.Context(), sess.ID, now, idleExpires)
			return sess.UserID, nil
		}
	}

	// 2. Bearer API key.
	if token := bearerToken(r); token != "" {
		hashed := auth.HashToken(token)
		u, err := h.authStore.GetUserByAPIKey(r.Context(), hashed)
		if err == nil {
			return u.ID, nil
		}
	}

	return "", fmt.Errorf("unauthorized")
}

func isMutating(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete || method == http.MethodPatch
}

func bearerToken(r *http.Request) string {
	hdr := r.Header.Get("Authorization")
	if len(hdr) < 7 || hdr[:7] != "Bearer " {
		return ""
	}
	return hdr[7:]
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newHandlerID returns a CSPRNG-derived ID for API-created entities. It
// reuses auth.NewToken (32 crypto/rand bytes, base64url) rather than a wall
// clock timestamp, since these IDs are also used for account/user/session/
// API-key identifiers where predictability would enable guessing attacks.
func newHandlerID() string {
	return auth.NewToken()
}

// calculateTDEE computes BMR, TDEE, and macro splits using Mifflin-St Jeor.
func calculateTDEE(p types.TDEEParams) types.TDEEResult {
	var bmr float64
	switch p.Gender {
	case "male":
		bmr = 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) + 5
	case "female":
		bmr = 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) - 161
	default:
		// Average male/female for "other".
		bmrMale := 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) + 5
		bmrFemale := 10*p.WeightKg + 6.25*p.HeightCm - 5*float64(p.Age) - 161
		bmr = (bmrMale + bmrFemale) / 2
	}

	multipliers := map[string]float64{
		"sedentary": 1.2, "light": 1.375, "moderate": 1.55,
		"active": 1.725, "very_active": 1.9,
	}
	actMult, ok := multipliers[p.ActivityLevel]
	if !ok {
		actMult = 1.2
	}
	tdee := bmr * actMult

	return types.TDEEResult{
		BMR:         bmr,
		TDEE:        tdee,
		CutCal:      tdee - 500,
		MaintainCal: tdee,
		BulkCal:     tdee + 500,
		Protein:     p.WeightKg * 2.2,
		Fat:         tdee * 0.25 / 9,
		Carbs:       (tdee - (p.WeightKg*2.2*4 + tdee*0.25)) / 4,
	}
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, types.ErrNotFound) || errors.Is(err, types.ErrNoMatch):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	case errors.Is(err, types.ErrConflict):
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "conflict"})
	default:
		w.WriteHeader(http.StatusInternalServerError)
		// Log the real error server-side; return a generic message to avoid
		// leaking internal details (DB paths, SQL errors, etc.) to clients.
		slog.Error("api error", "err", err)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
	}
}
