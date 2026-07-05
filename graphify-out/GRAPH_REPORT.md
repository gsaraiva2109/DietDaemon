# Graph Report - .  (2026-07-05)

## Corpus Check
- 0 files · ~99,999 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1978 nodes · 3536 edges · 71 communities detected
- Extraction: 78% EXTRACTED · 22% INFERRED · 0% AMBIGUOUS · INFERRED: 788 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Data Store Layer|Data Store Layer]]
- [[_COMMUNITY_Data Store Layer|Data Store Layer]]
- [[_COMMUNITY_Food Parser (Deterministic & LLM)|Food Parser (Deterministic & LLM)]]
- [[_COMMUNITY_API Handlers & HTTP Layer|API Handlers & HTTP Layer]]
- [[_COMMUNITY_Bot Commands|Bot Commands]]
- [[_COMMUNITY_React Components & Hooks|React Components & Hooks]]
- [[_COMMUNITY_Data Store Layer|Data Store Layer]]
- [[_COMMUNITY_Fasting Commands|Fasting Commands]]
- [[_COMMUNITY_Email & MFA Auth Flow|Email & MFA Auth Flow]]
- [[_COMMUNITY_Scheduler & Notifications|Scheduler & Notifications]]
- [[_COMMUNITY_API Test Infrastructure|API Test Infrastructure]]
- [[_COMMUNITY_Email & MFA Auth Flow|Email & MFA Auth Flow]]
- [[_COMMUNITY_Core Domain Types|Core Domain Types]]
- [[_COMMUNITY_UI Icons|UI Icons]]
- [[_COMMUNITY_Food Parser (Deterministic & LLM)|Food Parser (Deterministic & LLM)]]
- [[_COMMUNITY_Store Module|Store Module]]
- [[_COMMUNITY_Parse Pipeline Engine|Parse Pipeline Engine]]
- [[_COMMUNITY_i18n & Localization|i18n & Localization]]
- [[_COMMUNITY_Mailer Adapters|Mailer Adapters]]
- [[_COMMUNITY_Discord Messaging Adapter|Discord Messaging Adapter]]
- [[_COMMUNITY_React Components & Hooks|React Components & Hooks]]
- [[_COMMUNITY_React Components & Hooks|React Components & Hooks]]
- [[_COMMUNITY_Telegram Messaging Adapter|Telegram Messaging Adapter]]
- [[_COMMUNITY_Design System & Brand Docs|Design System & Brand Docs]]
- [[_COMMUNITY_Web Frontend Entry|Web Frontend Entry]]
- [[_COMMUNITY_Matrix Messaging Adapter|Matrix Messaging Adapter]]
- [[_COMMUNITY_React Route Pages|React Route Pages]]
- [[_COMMUNITY_OIDC Provider Integration|OIDC Provider Integration]]
- [[_COMMUNITY_Frontend Library Utilities|Frontend Library Utilities]]
- [[_COMMUNITY_Auth Primitives (TOTPWebAuthnRecovery)|Auth Primitives (TOTP/WebAuthn/Recovery)]]
- [[_COMMUNITY_Food Resolver|Food Resolver]]
- [[_COMMUNITY_Core Domain Types|Core Domain Types]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_Ollama Model Adapter|Ollama Model Adapter]]
- [[_COMMUNITY_Open Food Facts Adapter|Open Food Facts Adapter]]
- [[_COMMUNITY_USDA Nutrition Adapter|USDA Nutrition Adapter]]
- [[_COMMUNITY_React Route Pages|React Route Pages]]
- [[_COMMUNITY_Pending Store (Durable Queue)|Pending Store (Durable Queue)]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_Frontend Library Utilities|Frontend Library Utilities]]
- [[_COMMUNITY_Ntfy Notifier Adapter|Ntfy Notifier Adapter]]
- [[_COMMUNITY_Design System & Brand Docs|Design System & Brand Docs]]
- [[_COMMUNITY_Auth Primitives (TOTPWebAuthnRecovery)|Auth Primitives (TOTP/WebAuthn/Recovery)]]
- [[_COMMUNITY_Embedding Matcher|Embedding Matcher]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_Whisper STT Adapter|Whisper STT Adapter]]
- [[_COMMUNITY_USDA Nutrition Adapter|USDA Nutrition Adapter]]
- [[_COMMUNITY_React Route Pages|React Route Pages]]
- [[_COMMUNITY_In-Memory Message Queue|In-Memory Message Queue]]
- [[_COMMUNITY_In-Memory Message Queue|In-Memory Message Queue]]
- [[_COMMUNITY_UI Icons|UI Icons]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_React Route Pages|React Route Pages]]
- [[_COMMUNITY_React Route Pages|React Route Pages]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_React UI Components|React UI Components]]
- [[_COMMUNITY_Frontend Library Utilities|Frontend Library Utilities]]
- [[_COMMUNITY_Internal Module|Internal Module]]
- [[_COMMUNITY_Internal Module|Internal Module]]
- [[_COMMUNITY_Internal Module|Internal Module]]
- [[_COMMUNITY_Internal Module|Internal Module]]
- [[_COMMUNITY_Internal Module|Internal Module]]
- [[_COMMUNITY_Community 127|Community 127]]
- [[_COMMUNITY_Community 128|Community 128]]
- [[_COMMUNITY_Community 129|Community 129]]
- [[_COMMUNITY_Community 130|Community 130]]

## God Nodes (most connected - your core abstractions)
1. `Store` - 137 edges
2. `New()` - 128 edges
3. `Handler` - 116 edges
4. `now()` - 102 edges
5. `doRequest()` - 77 edges
6. `newHandler()` - 75 edges
7. `newFakeMealStore()` - 73 edges
8. `fakeMealStore` - 60 edges
9. `fakeAuthStore` - 59 edges
10. `contains()` - 37 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `NewCancelCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/cancel.go
- `run()` --calls--> `NewTimezoneCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/timezone.go
- `run()` --calls--> `NewStartCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/start.go
- `run()` --calls--> `NewLinkCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/link.go
- `run()` --calls--> `NewStatusCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/status.go

## Hyperedges (group relationships)
- **Group 2 — Food Logging & Resolution batch** — roadmap_alias_review_ui, roadmap_precedence_ui, roadmap_recipe_composition, roadmap_correct_meal_item_bot [EXTRACTED 0.95]
- **Group 3 — Scheduler & Data Ops batch** — roadmap_weekly_monthly_digest, roadmap_configurable_nudge_rules, roadmap_health_platform_import_export, roadmap_scheduled_data_export_backup [EXTRACTED 0.95]
- **Parser Tier / STT Independence Concept** — readme_parser_pipeline, readme_stt, stt_speech_to_text, stt_parser_tier_independence [INFERRED 0.80]

## Communities

### Community 0 - "Data Store Layer"
Cohesion: 0.02
Nodes (57): AuthConfig, AuthStore, Handler, clientIP(), isSixDigit(), readSessionCookie(), bearerToken(), calculateTDEE() (+49 more)

### Community 1 - "Data Store Layer"
Cohesion: 0.03
Nodes (32): NewWebAuthnHandle(), parseTier(), Normalize(), TestNormalize(), unaccent(), Memory[T], scanRow, Store (+24 more)

### Community 2 - "Food Parser (Deterministic & LLM)"
Cohesion: 0.03
Nodes (122): contains(), pendingInMemory(), pendingSQLite(), pendingStoreFactory, eq(), TestConjunctionSeparators(), TestParsePortugueseAndEnglishMatch(), TestQuantitylessAndEmpty() (+114 more)

### Community 3 - "API Handlers & HTTP Layer"
Cohesion: 0.02
Nodes (41): ProtectedRoute(), UtilityBar(), VerifyEmailBanner(), AuthProvider(), useAuth(), demoRange(), fd(), hoursAgo() (+33 more)

### Community 4 - "Bot Commands"
Cohesion: 0.1
Nodes (77): fakeMealLogger, decodeJSON(), doRequest(), newFakeMealStore(), newHandler(), TestAddAlias(), TestAddAliasMissing(), TestAddMealItem() (+69 more)

### Community 5 - "React Components & Hooks"
Cohesion: 0.05
Nodes (34): MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand(), parseTargetArgs(), TargetCommand, close() (+26 more)

### Community 6 - "Data Store Layer"
Cohesion: 0.04
Nodes (62): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), DietDaemon Container Service (+54 more)

### Community 7 - "Fasting Commands"
Cohesion: 0.07
Nodes (41): fakeHealthStore, fakeNotifier, fakeNudges, fakeStore, HealthRule, HealthStore, Macro, Notifier (+33 more)

### Community 8 - "Email & MFA Auth Flow"
Cohesion: 0.03
Nodes (1): fakeMealStore

### Community 9 - "Scheduler & Notifications"
Cohesion: 0.04
Nodes (1): fakeAuthStore

### Community 10 - "API Test Infrastructure"
Cohesion: 0.04
Nodes (42): APIKey, AuditEvent, BodyCompositionSummary, DailyRollup, DailyTargets, Fast, FoodAlias, FoodDetail (+34 more)

### Community 11 - "Email & MFA Auth Flow"
Cohesion: 0.09
Nodes (43): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), CheckIcon() (+35 more)

### Community 12 - "Core Domain Types"
Cohesion: 0.06
Nodes (21): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, calcSleepHours(), computeSleepDuration(), formatDuration(), NewSleepCommand() (+13 more)

### Community 13 - "UI Icons"
Cohesion: 0.12
Nodes (30): Stat(), Config, decodeKey(), getBool(), getDuration(), getFloat(), getInt(), getStr() (+22 more)

### Community 14 - "Food Parser (Deterministic & LLM)"
Cohesion: 0.08
Nodes (12): emailTestAuthStore, emailToken, fakeMailer, buildEmailHandler(), newEmailTestAuthStore(), TestEmailVerifyExpiredToken(), TestEmailVerifyInvalidToken(), TestEmailVerifyPurposeMismatch() (+4 more)

### Community 15 - "Store Module"
Cohesion: 0.09
Nodes (21): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+13 more)

### Community 16 - "Parse Pipeline Engine"
Cohesion: 0.11
Nodes (16): fakeCmd, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape(), TestHelpCommand_ListAll() (+8 more)

### Community 17 - "i18n & Localization"
Cohesion: 0.16
Nodes (32): postgresDB(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods(), TestPostgresUserRoundTrip(), TestGetUserByOIDCIdentity(), TestLinkOIDCIdentityUniqueness(), TestListDeleteOIDCIdentities() (+24 more)

### Community 18 - "Mailer Adapters"
Cohesion: 0.09
Nodes (20): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), llmItem, llmResponse (+12 more)

### Community 19 - "Discord Messaging Adapter"
Cohesion: 0.1
Nodes (17): Dialect, NewDialect(), SQLiteDialect(), TestColumnExists(), TestNewDialectInvalid(), TestNow(), TestPlaceholder(), TestPostgresRewritePlaceholders() (+9 more)

### Community 20 - "React Components & Hooks"
Cohesion: 0.14
Nodes (19): entry, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB(), requireNoErr(), TestCacheInvalidation(), TestCosineSimilarity() (+11 more)

### Community 21 - "React Components & Hooks"
Cohesion: 0.08
Nodes (16): Config, Mailer, New(), smtpPortOrDefault(), TestNew(), TestNoneMailerSend(), TestTemplatesNotEmpty(), Message (+8 more)

### Community 22 - "Telegram Messaging Adapter"
Cohesion: 0.12
Nodes (16): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+8 more)

### Community 23 - "Design System & Brand Docs"
Cohesion: 0.12
Nodes (14): download(), copyPng(), dataUrlToBlob(), downloadPng(), render(), ApiError, blobRequest(), handleUnauthorized() (+6 more)

### Community 24 - "Web Frontend Entry"
Cohesion: 0.12
Nodes (8): onSubmit(), onAdd(), isMfaChallenge(), isWebAuthnCancel(), loginWithPasskey(), registerPasskey(), signInWithPasskey(), usePasskey()

### Community 25 - "Matrix Messaging Adapter"
Cohesion: 0.12
Nodes (11): Adapter, callbackQuery, getUpdatesResponse, sendMessageRequest, sendMessageResponse, tgChat, tgInlineKeyboardButton, tgInlineKeyboardMarkup (+3 more)

### Community 26 - "React Route Pages"
Cohesion: 0.15
Nodes (10): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), parseCookies(), recordFailure(), seed(), sessionFor() (+2 more)

### Community 27 - "OIDC Provider Integration"
Cohesion: 0.18
Nodes (13): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+5 more)

### Community 28 - "Frontend Library Utilities"
Cohesion: 0.16
Nodes (9): Adapter, joinedRoom, callbackDataByIndex(), New(), newPendingMarkupStore(), matrixMessageContent, pendingMarkupStore, syncResponse (+1 more)

### Community 29 - "Auth Primitives (TOTP/WebAuthn/Recovery)"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 30 - "Food Resolver"
Cohesion: 0.27
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 31 - "Core Domain Types"
Cohesion: 0.2
Nodes (1): fakeStore

### Community 32 - "React UI Components"
Cohesion: 0.24
Nodes (6): Embedder, FoodStore, Matcher, Resolver, finalize(), Source

### Community 33 - "Ollama Model Adapter"
Cohesion: 0.2
Nodes (9): Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser, PendingStore, Store (+1 more)

### Community 34 - "Open Food Facts Adapter"
Cohesion: 0.22
Nodes (5): macrosSum(), NewTemplateCommand(), TemplateCommand, TemplateMealLogger, TemplateStore

### Community 35 - "USDA Nutrition Adapter"
Cohesion: 0.22
Nodes (5): Adapter, embedRequest, embedResponse, generateRequest, generateResponse

### Community 36 - "React Route Pages"
Cohesion: 0.25
Nodes (4): nutriments, product, searchResponse, Source

### Community 37 - "Pending Store (Durable Queue)"
Cohesion: 0.25
Nodes (5): food, foodNutrient, searchResponse, Source, extractMacros()

### Community 39 - "React UI Components"
Cohesion: 0.36
Nodes (1): Store

### Community 40 - "React UI Components"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 41 - "Frontend Library Utilities"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 42 - "Ntfy Notifier Adapter"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 43 - "Design System & Brand Docs"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 46 - "Auth Primitives (TOTP/WebAuthn/Recovery)"
Cohesion: 0.38
Nodes (4): dayFraction(), insights(), trend(), weeklyStats()

### Community 47 - "Embedding Matcher"
Cohesion: 0.29
Nodes (2): NewStartCommand(), StartCommand

### Community 48 - "React UI Components"
Cohesion: 0.29
Nodes (2): NewTimezoneCommand(), TimezoneCommand

### Community 49 - "Whisper STT Adapter"
Cohesion: 0.29
Nodes (3): NewWorkoutCommand(), WorkoutCommand, WorkoutStore

### Community 50 - "USDA Nutrition Adapter"
Cohesion: 0.29
Nodes (3): NewWeightCommand(), WeightCommand, WeightStore

### Community 51 - "React Route Pages"
Cohesion: 0.29
Nodes (3): NewWaterCommand(), WaterCommand, WaterStore

### Community 52 - "In-Memory Message Queue"
Cohesion: 0.29
Nodes (3): NewLinkCommand(), LinkCodeStore, LinkCommand

### Community 53 - "In-Memory Message Queue"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 55 - "UI Icons"
Cohesion: 0.33
Nodes (1): WebAuthnUser

### Community 56 - "React UI Components"
Cohesion: 0.33
Nodes (1): Matcher

### Community 58 - "React UI Components"
Cohesion: 0.4
Nodes (1): fakeStore

### Community 59 - "React UI Components"
Cohesion: 0.4
Nodes (1): stubStore

### Community 60 - "React UI Components"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 62 - "React Route Pages"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 64 - "React Route Pages"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 76 - "React UI Components"
Cohesion: 1.0
Nodes (2): dayKey(), relativeDayLabel()

### Community 77 - "React UI Components"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 78 - "React UI Components"
Cohesion: 0.67
Nodes (3): Health Platform Import/Export, weight.go, workout.go

### Community 79 - "Frontend Library Utilities"
Cohesion: 0.67
Nodes (1): Configurable Nudge Rules

### Community 102 - "Internal Module"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 103 - "Internal Module"
Cohesion: 1.0
Nodes (2): Recipe / Multi-ingredient Composition, internal/commands/template.go

### Community 104 - "Internal Module"
Cohesion: 1.0
Nodes (2): internal/scheduler/rules.go, Weekly/Monthly Digest Notification

### Community 105 - "Internal Module"
Cohesion: 1.0
Nodes (2): ExportModal.tsx, Scheduled Data Export/Backup

### Community 106 - "Internal Module"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 127 - "Community 127"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 128 - "Community 128"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 129 - "Community 129"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 130 - "Community 130"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **196 isolated node(s):** `phraseEntry`, `RecoveryCodeRepo`, `TOTPRepo`, `MFAChallengeRepo`, `LoginAttemptRepo` (+191 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Email & MFA Auth Flow`** (60 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateLinkingCode()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetFoodDetail()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetWaterToday()`, `.GetWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchFoods()`, `.SetTargets()`, `.StartFast()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Scheduler & Notifications`** (55 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetSession()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Core Domain Types`** (10 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Embedding Matcher`** (7 nodes): `NewStartCommand()`, `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `start.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (7 nodes): `NewTimezoneCommand()`, `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `timezone.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `UI Icons`** (6 nodes): `WebAuthnUser`, `.WebAuthnCredentials()`, `.WebAuthnDisplayName()`, `.WebAuthnIcon()`, `.WebAuthnID()`, `.WebAuthnName()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (6 nodes): `New()`, `Matcher`, `.EmbedFood()`, `.Match()`, `.SetThreshold()`, `embedding.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (5 nodes): `fakeStore`, `.GetFood()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (5 nodes): `stubStore`, `.GetFood()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React Route Pages`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (3 nodes): `dayKey()`, `relativeDayLabel()`, `History.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React UI Components`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Frontend Library Utilities`** (3 nodes): `Configurable Nudge Rules`, `scheduler.DefaultHealthRules()`, `scheduler.DefaultRules()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Internal Module`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Internal Module`** (2 nodes): `Recipe / Multi-ingredient Composition`, `internal/commands/template.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Internal Module`** (2 nodes): `internal/scheduler/rules.go`, `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Internal Module`** (2 nodes): `ExportModal.tsx`, `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Internal Module`** (2 nodes): `Precedence UI`, `resolver.New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 127`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 128`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 129`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 130`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Food Parser (Deterministic & LLM)` to `Data Store Layer`, `Bot Commands`, `React Components & Hooks`, `Fasting Commands`, `Scheduler & Notifications`, `Food Parser (Deterministic & LLM)`, `Parse Pipeline Engine`, `i18n & Localization`, `Discord Messaging Adapter`, `React Components & Hooks`, `React Components & Hooks`?**
  _High betweenness centrality (0.190) - this node is a cross-community bridge._
- **Why does `now()` connect `Data Store Layer` to `Data Store Layer`, `Food Parser (Deterministic & LLM)`, `Store Module`, `i18n & Localization`, `Telegram Messaging Adapter`, `Matrix Messaging Adapter`, `React Route Pages`, `Frontend Library Utilities`?**
  _High betweenness centrality (0.107) - this node is a cross-community bridge._
- **Why does `run()` connect `Food Parser (Deterministic & LLM)` to `Data Store Layer`, `Open Food Facts Adapter`, `React Components & Hooks`, `Fasting Commands`, `React UI Components`, `Frontend Library Utilities`, `Ntfy Notifier Adapter`, `Core Domain Types`, `UI Icons`, `Embedding Matcher`, `Parse Pipeline Engine`, `React UI Components`, `USDA Nutrition Adapter`, `Discord Messaging Adapter`, `In-Memory Message Queue`, `React Route Pages`, `Whisper STT Adapter`, `Auth Primitives (TOTP/WebAuthn/Recovery)`?**
  _High betweenness centrality (0.103) - this node is a cross-community bridge._
- **Are the 123 inferred relationships involving `New()` (e.g. with `run()` and `buildModelAndIndex()`) actually correct?**
  _`New()` has 123 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 98 inferred relationships involving `now()` (e.g. with `TestCreateSession()` and `TestValidateSessionExpiredAbsolute()`) actually correct?**
  _`now()` has 98 INFERRED edges - model-reasoned connections that need verification._
- **Are the 6 inferred relationships involving `doRequest()` (e.g. with `TestEmailVerifySuccess()` and `TestEmailVerifyInvalidToken()`) actually correct?**
  _`doRequest()` has 6 INFERRED edges - model-reasoned connections that need verification._