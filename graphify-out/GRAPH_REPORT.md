# Graph Report - .  (2026-07-04)

## Corpus Check
- Large corpus: 218 files · ~141,636 words. Semantic extraction will be expensive (many Claude tokens). Consider running on a subfolder, or use --no-semantic to run AST-only.

## Summary
- 1879 nodes · 3173 edges · 49 communities detected
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 609 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Pending Store (Durable Queue)|Pending Store (Durable Queue)]]
- [[_COMMUNITY_Gotify Notifier Adapter|Gotify Notifier Adapter]]
- [[_COMMUNITY_Frontend Library Utilities|Frontend Library Utilities]]
- [[_COMMUNITY_Ntfy Notifier Adapter|Ntfy Notifier Adapter]]
- [[_COMMUNITY_Design System & Brand Docs|Design System & Brand Docs]]
- [[_COMMUNITY_Parser Tuning CLI|Parser Tuning CLI]]
- [[_COMMUNITY_Auth Primitives (TOTPWebAuthnRecovery)|Auth Primitives (TOTP/WebAuthn/Recovery)]]
- [[_COMMUNITY_Embedding Matcher|Embedding Matcher]]
- [[_COMMUNITY_Whisper STT Adapter|Whisper STT Adapter]]
- [[_COMMUNITY_In-Memory Message Queue|In-Memory Message Queue]]
- [[_COMMUNITY_React Route Pages|React Route Pages]]
- [[_COMMUNITY_Contract Tests|Contract Tests]]
- [[_COMMUNITY_Design System & Brand Docs|Design System & Brand Docs]]

## God Nodes (most connected - your core abstractions)
1. `Store` - 136 edges
2. `New()` - 121 edges
3. `Handler` - 116 edges
4. `now()` - 102 edges
5. `doRequest()` - 77 edges
6. `newHandler()` - 75 edges
7. `newFakeMealStore()` - 73 edges
8. `fakeMealStore` - 60 edges
9. `fakeAuthStore` - 59 edges
10. `run()` - 33 edges

## Surprising Connections (you probably didn't know these)
- `TestPendingStoreContract()` --calls--> `now()`  [INFERRED]
  tests/contract/pendingstore_test.go → web/dev-mock-api.mjs
- `NumberField()` --calls--> `parseFloat()`  [INFERRED]
  web/src/components/OnboardingWizard.tsx → adapters/nutrition/taco/taco.go
- `run()` --calls--> `NewWeightCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/weight.go
- `run()` --calls--> `NewWaterCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/water.go
- `run()` --calls--> `NewWorkoutCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/workout.go

## Hyperedges (group relationships)
- **Architecture Data Flow** — readme_messaging_adapters, readme_parsertier0, readme_parsertier1, readme_parsertier2, readme_architecture, readme_notifier, readme_food_library [EXTRACTED 1.00]
- **Design System** — design_colorsystem, design_macro_hues, design_typography, design_macro_ring, design_motion_system, product_brand_personality, product_design_principles [INFERRED 0.85]

## Communities

### Community 0 - "Data Store Layer"
Cohesion: 0.02
Nodes (32): NewWebAuthnHandle(), Normalize(), TestNormalize(), unaccent(), Memory[T], scanRow, Store, scanSession() (+24 more)

### Community 1 - "Data Store Layer"
Cohesion: 0.03
Nodes (45): AuthConfig, AuthStore, Handler, clientIP(), isSixDigit(), readSessionCookie(), bearerToken(), calculateTDEE() (+37 more)

### Community 2 - "Food Parser (Deterministic & LLM)"
Cohesion: 0.02
Nodes (118): open(), pendingInMemory(), pendingSQLite(), TestPendingStoreContract(), pendingStoreFactory, eq(), TestConjunctionSeparators(), TestParsePortugueseAndEnglishMatch() (+110 more)

### Community 3 - "API Handlers & HTTP Layer"
Cohesion: 0.08
Nodes (91): emailToken, fakeMailer, fakeMealLogger, buildEmailHandler(), newEmailTestAuthStore(), TestEmailVerifyExpiredToken(), TestEmailVerifyInvalidToken(), TestEmailVerifyPurposeMismatch() (+83 more)

### Community 4 - "Bot Commands"
Cohesion: 0.05
Nodes (57): fakeCmd, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape(), TestHelpCommand_ListAll() (+49 more)

### Community 5 - "React Components & Hooks"
Cohesion: 0.03
Nodes (37): ProtectedRoute(), UtilityBar(), VerifyEmailBanner(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useApiKeys() (+29 more)

### Community 6 - "Data Store Layer"
Cohesion: 0.02
Nodes (40): NewCancelCommand(), CancelCommand, NewFoodCommand(), FoodCommand, FoodStore, NewLinkCommand(), LinkCodeStore, LinkCommand (+32 more)

### Community 7 - "Fasting Commands"
Cohesion: 0.04
Nodes (31): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+23 more)

### Community 8 - "Email & MFA Auth Flow"
Cohesion: 0.05
Nodes (22): emailTestAuthStore, fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession() (+14 more)

### Community 9 - "Scheduler & Notifications"
Cohesion: 0.07
Nodes (41): fakeHealthStore, fakeNotifier, fakeNudges, fakeStore, HealthRule, HealthStore, Macro, Notifier (+33 more)

### Community 10 - "API Test Infrastructure"
Cohesion: 0.03
Nodes (1): fakeMealStore

### Community 11 - "Email & MFA Auth Flow"
Cohesion: 0.03
Nodes (1): fakeAuthStore

### Community 12 - "Core Domain Types"
Cohesion: 0.04
Nodes (42): APIKey, AuditEvent, BodyCompositionSummary, DailyRollup, DailyTargets, Fast, FoodAlias, FoodDetail (+34 more)

### Community 13 - "UI Icons"
Cohesion: 0.09
Nodes (43): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), CheckIcon() (+35 more)

### Community 14 - "Food Parser (Deterministic & LLM)"
Cohesion: 0.09
Nodes (20): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), llmItem, llmResponse (+12 more)

### Community 15 - "Store Module"
Cohesion: 0.21
Nodes (27): TestGetUserByOIDCIdentity(), TestLinkOIDCIdentityUniqueness(), TestListDeleteOIDCIdentities(), TestMagicCodeDelete(), TestMagicCodeIncrementAttempts(), TestMagicCodeNotFound(), TestMagicCodeUpsertGet(), TestMagicCodeUpsertOverwrite() (+19 more)

### Community 16 - "Parse Pipeline Engine"
Cohesion: 0.15
Nodes (12): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), plural(), questionText() (+4 more)

### Community 17 - "i18n & Localization"
Cohesion: 0.11
Nodes (13): Bundle, NewBundle(), entry, Index, cosineSimilarity(), packF32LE(), sortByScore(), TestCosineSimilarity() (+5 more)

### Community 18 - "Mailer Adapters"
Cohesion: 0.08
Nodes (15): Config, Mailer, New(), smtpPortOrDefault(), TestNew(), TestNoneMailerSend(), Message, newNone() (+7 more)

### Community 19 - "Discord Messaging Adapter"
Cohesion: 0.12
Nodes (16): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+8 more)

### Community 20 - "React Components & Hooks"
Cohesion: 0.12
Nodes (14): download(), copyPng(), dataUrlToBlob(), downloadPng(), render(), ApiError, blobRequest(), handleUnauthorized() (+6 more)

### Community 21 - "React Components & Hooks"
Cohesion: 0.12
Nodes (8): onSubmit(), onAdd(), isMfaChallenge(), isWebAuthnCancel(), loginWithPasskey(), registerPasskey(), signInWithPasskey(), usePasskey()

### Community 22 - "Telegram Messaging Adapter"
Cohesion: 0.12
Nodes (11): Adapter, callbackQuery, getUpdatesResponse, sendMessageRequest, sendMessageResponse, tgChat, tgInlineKeyboardButton, tgInlineKeyboardMarkup (+3 more)

### Community 23 - "Design System & Brand Docs"
Cohesion: 0.14
Nodes (19): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+11 more)

### Community 24 - "Web Frontend Entry"
Cohesion: 0.15
Nodes (10): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), parseCookies(), recordFailure(), seed(), sessionFor() (+2 more)

### Community 25 - "Matrix Messaging Adapter"
Cohesion: 0.15
Nodes (9): Adapter, joinedRoom, callbackDataByIndex(), New(), newPendingMarkupStore(), matrixMessageContent, pendingMarkupStore, syncResponse (+1 more)

### Community 26 - "React Route Pages"
Cohesion: 0.19
Nodes (13): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+5 more)

### Community 27 - "OIDC Provider Integration"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 28 - "Frontend Library Utilities"
Cohesion: 0.2
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 29 - "Auth Primitives (TOTP/WebAuthn/Recovery)"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 30 - "Food Resolver"
Cohesion: 0.24
Nodes (6): Embedder, FoodStore, Matcher, Resolver, finalize(), Source

### Community 31 - "Core Domain Types"
Cohesion: 0.2
Nodes (9): Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser, PendingStore, Store (+1 more)

### Community 32 - "React UI Components"
Cohesion: 0.31
Nodes (5): close(), NumberField(), profilePayload(), save(), skipOrCancel()

### Community 33 - "Ollama Model Adapter"
Cohesion: 0.22
Nodes (5): Adapter, embedRequest, embedResponse, generateRequest, generateResponse

### Community 34 - "Open Food Facts Adapter"
Cohesion: 0.25
Nodes (4): nutriments, product, searchResponse, Source

### Community 35 - "USDA Nutrition Adapter"
Cohesion: 0.25
Nodes (5): food, foodNutrient, searchResponse, Source, extractMacros()

### Community 37 - "Pending Store (Durable Queue)"
Cohesion: 0.36
Nodes (1): Store

### Community 38 - "Gotify Notifier Adapter"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 41 - "Frontend Library Utilities"
Cohesion: 0.38
Nodes (4): dayFraction(), insights(), trend(), weeklyStats()

### Community 42 - "Ntfy Notifier Adapter"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 43 - "Design System & Brand Docs"
Cohesion: 0.38
Nodes (7): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), WCAG AA Accessibility, Brand Personality (Restful, Precise, Glanceable), Design Principles

### Community 45 - "Parser Tuning CLI"
Cohesion: 0.53
Nodes (5): evaluate(), loadPhrases(), main(), run(), phraseEntry

### Community 46 - "Auth Primitives (TOTP/WebAuthn/Recovery)"
Cohesion: 0.33
Nodes (1): WebAuthnUser

### Community 47 - "Embedding Matcher"
Cohesion: 0.33
Nodes (1): Matcher

### Community 49 - "Whisper STT Adapter"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 52 - "In-Memory Message Queue"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 64 - "React Route Pages"
Cohesion: 1.0
Nodes (2): dayKey(), relativeDayLabel()

### Community 65 - "Contract Tests"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 109 - "Design System & Brand Docs"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

## Knowledge Gaps
- **168 isolated node(s):** `phraseEntry`, `RecoveryCodeRepo`, `TOTPRepo`, `MFAChallengeRepo`, `LoginAttemptRepo` (+163 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `API Test Infrastructure`** (60 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateLinkingCode()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetFoodDetail()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetWaterToday()`, `.GetWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchFoods()`, `.SetTargets()`, `.StartFast()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Email & MFA Auth Flow`** (59 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateAPIKey()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateUserWithOIDC()`, `.CreateUserWithPassword()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetSession()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RecordLoginAttempt()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Pending Store (Durable Queue)`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Auth Primitives (TOTP/WebAuthn/Recovery)`** (6 nodes): `WebAuthnUser`, `.WebAuthnCredentials()`, `.WebAuthnDisplayName()`, `.WebAuthnIcon()`, `.WebAuthnID()`, `.WebAuthnName()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Embedding Matcher`** (6 nodes): `New()`, `Matcher`, `.EmbedFood()`, `.Match()`, `.SetThreshold()`, `embedding.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Whisper STT Adapter`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `In-Memory Message Queue`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React Route Pages`** (3 nodes): `dayKey()`, `relativeDayLabel()`, `History.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Contract Tests`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Design System & Brand Docs`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Food Parser (Deterministic & LLM)` to `Data Store Layer`, `API Handlers & HTTP Layer`, `Bot Commands`, `Data Store Layer`, `Scheduler & Notifications`, `Email & MFA Auth Flow`, `Parser Tuning CLI`, `Store Module`, `i18n & Localization`, `Mailer Adapters`?**
  _High betweenness centrality (0.235) - this node is a cross-community bridge._
- **Why does `now()` connect `Data Store Layer` to `Data Store Layer`, `Food Parser (Deterministic & LLM)`, `API Handlers & HTTP Layer`, `Data Store Layer`, `Fasting Commands`, `Email & MFA Auth Flow`, `Email & MFA Auth Flow`, `Store Module`, `Discord Messaging Adapter`, `Telegram Messaging Adapter`, `Web Frontend Entry`, `Matrix Messaging Adapter`?**
  _High betweenness centrality (0.223) - this node is a cross-community bridge._
- **Why does `run()` connect `Data Store Layer` to `Data Store Layer`, `Food Parser (Deterministic & LLM)`, `API Handlers & HTTP Layer`, `Bot Commands`, `Fasting Commands`, `Scheduler & Notifications`, `i18n & Localization`, `OIDC Provider Integration`?**
  _High betweenness centrality (0.140) - this node is a cross-community bridge._
- **Are the 116 inferred relationships involving `New()` (e.g. with `run()` and `buildModelAndIndex()`) actually correct?**
  _`New()` has 116 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 98 inferred relationships involving `now()` (e.g. with `TestCreateSession()` and `TestValidateSessionExpiredAbsolute()`) actually correct?**
  _`now()` has 98 INFERRED edges - model-reasoned connections that need verification._
- **Are the 6 inferred relationships involving `doRequest()` (e.g. with `TestEmailVerifySuccess()` and `TestEmailVerifyInvalidToken()`) actually correct?**
  _`doRequest()` has 6 INFERRED edges - model-reasoned connections that need verification._