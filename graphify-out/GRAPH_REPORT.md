# Graph Report - DietDaemon  (2026-07-05)

## Corpus Check
- 255 files · ~165,239 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2126 nodes · 3826 edges · 74 communities detected
- Extraction: 78% EXTRACTED · 22% INFERRED · 0% AMBIGUOUS · INFERRED: 858 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 81|Community 81]]
- [[_COMMUNITY_Community 82|Community 82]]
- [[_COMMUNITY_Community 107|Community 107]]
- [[_COMMUNITY_Community 129|Community 129]]
- [[_COMMUNITY_Community 130|Community 130]]
- [[_COMMUNITY_Community 131|Community 131]]
- [[_COMMUNITY_Community 132|Community 132]]
- [[_COMMUNITY_Community 133|Community 133]]
- [[_COMMUNITY_Community 134|Community 134]]
- [[_COMMUNITY_Community 135|Community 135]]
- [[_COMMUNITY_Community 136|Community 136]]
- [[_COMMUNITY_Community 137|Community 137]]
- [[_COMMUNITY_Community 138|Community 138]]

## God Nodes (most connected - your core abstractions)
1. `Store` - 149 edges
2. `New()` - 146 edges
3. `Handler` - 125 edges
4. `now()` - 102 edges
5. `doRequest()` - 80 edges
6. `newHandler()` - 78 edges
7. `newFakeMealStore()` - 76 edges
8. `fakeMealStore` - 71 edges
9. `fakeAuthStore` - 59 edges
10. `run()` - 41 edges

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

### Community 0 - "Community 0"
Cohesion: 0.02
Nodes (54): AuthConfig, AuthStore, BackupRunner, Handler, clientIP(), isSixDigit(), readSessionCookie(), bearerToken() (+46 more)

### Community 1 - "Community 1"
Cohesion: 0.02
Nodes (34): NewWebAuthnHandle(), parseTier(), FS(), Normalize(), TestNormalize(), unaccent(), Memory[T], scanRow (+26 more)

### Community 2 - "Community 2"
Cohesion: 0.03
Nodes (135): Stat(), open(), pendingInMemory(), pendingSQLite(), pendingStoreFactory, eq(), TestConjunctionSeparators(), TestParsePortugueseAndEnglishMatch() (+127 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (43): ProtectedRoute(), UtilityBar(), VerifyEmailBanner(), AuthProvider(), useAuth(), demoRange(), fd(), hoursAgo() (+35 more)

### Community 4 - "Community 4"
Cohesion: 0.05
Nodes (59): buildNudgeRuleView(), DigestRule, DigestStore, fakeDigestStore, fakeFullStore, fakeHealthStore, fakeNotifier, fakeNudges (+51 more)

### Community 5 - "Community 5"
Cohesion: 0.1
Nodes (80): fakeMealLogger, decodeJSON(), doRequest(), newFakeMealStore(), newHandler(), TestAddAlias(), TestAddAliasMissing(), TestAddMealItem() (+72 more)

### Community 6 - "Community 6"
Cohesion: 0.06
Nodes (51): NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal(), fakeCmd, fakeCorrectResolver, fakeCorrectStore, NewHelpCommand() (+43 more)

### Community 7 - "Community 7"
Cohesion: 0.04
Nodes (35): CorrectCommand, CorrectResolver, CorrectStore, MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand() (+27 more)

### Community 8 - "Community 8"
Cohesion: 0.03
Nodes (1): fakeMealStore

### Community 9 - "Community 9"
Cohesion: 0.04
Nodes (1): fakeAuthStore

### Community 10 - "Community 10"
Cohesion: 0.08
Nodes (44): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), CheckIcon() (+36 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (45): APIKey, AuditEvent, BackupConfig, BodyCompositionSummary, DailyRollup, DailyTargets, Fast, FoodAlias (+37 more)

### Community 12 - "Community 12"
Cohesion: 0.05
Nodes (47): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+39 more)

### Community 13 - "Community 13"
Cohesion: 0.06
Nodes (21): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, calcSleepHours(), computeSleepDuration(), formatDuration(), NewSleepCommand() (+13 more)

### Community 14 - "Community 14"
Cohesion: 0.15
Nodes (36): postgresDB(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods(), TestPostgresUserRoundTrip(), TestGetUserByOIDCIdentity(), TestLinkOIDCIdentityUniqueness(), TestListDeleteOIDCIdentities() (+28 more)

### Community 15 - "Community 15"
Cohesion: 0.09
Nodes (21): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+13 more)

### Community 16 - "Community 16"
Cohesion: 0.08
Nodes (12): emailTestAuthStore, emailToken, fakeMailer, buildEmailHandler(), newEmailTestAuthStore(), TestEmailVerifyExpiredToken(), TestEmailVerifyInvalidToken(), TestEmailVerifyPurposeMismatch() (+4 more)

### Community 17 - "Community 17"
Cohesion: 0.09
Nodes (20): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), llmItem, llmResponse (+12 more)

### Community 18 - "Community 18"
Cohesion: 0.1
Nodes (17): Dialect, NewDialect(), SQLiteDialect(), TestColumnExists(), TestNewDialectInvalid(), TestNow(), TestPlaceholder(), TestPostgresRewritePlaceholders() (+9 more)

### Community 19 - "Community 19"
Cohesion: 0.14
Nodes (19): entry, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB(), requireNoErr(), TestCacheInvalidation(), TestCosineSimilarity() (+11 more)

### Community 20 - "Community 20"
Cohesion: 0.08
Nodes (16): Config, Mailer, New(), smtpPortOrDefault(), TestNew(), TestNoneMailerSend(), TestTemplatesNotEmpty(), Message (+8 more)

### Community 21 - "Community 21"
Cohesion: 0.12
Nodes (16): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+8 more)

### Community 22 - "Community 22"
Cohesion: 0.12
Nodes (14): download(), copyPng(), dataUrlToBlob(), downloadPng(), render(), ApiError, blobRequest(), handleUnauthorized() (+6 more)

### Community 23 - "Community 23"
Cohesion: 0.12
Nodes (8): onSubmit(), onAdd(), isMfaChallenge(), isWebAuthnCancel(), loginWithPasskey(), registerPasskey(), signInWithPasskey(), usePasskey()

### Community 24 - "Community 24"
Cohesion: 0.12
Nodes (11): Adapter, callbackQuery, getUpdatesResponse, sendMessageRequest, sendMessageResponse, tgChat, tgInlineKeyboardButton, tgInlineKeyboardMarkup (+3 more)

### Community 25 - "Community 25"
Cohesion: 0.15
Nodes (10): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), parseCookies(), recordFailure(), seed(), sessionFor() (+2 more)

### Community 26 - "Community 26"
Cohesion: 0.18
Nodes (13): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+5 more)

### Community 27 - "Community 27"
Cohesion: 0.16
Nodes (9): Adapter, joinedRoom, callbackDataByIndex(), New(), newPendingMarkupStore(), matrixMessageContent, pendingMarkupStore, syncResponse (+1 more)

### Community 28 - "Community 28"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 29 - "Community 29"
Cohesion: 0.18
Nodes (8): newFakeStore(), TestRunFor_MissingDestinationErrors(), TestRunOnce_IgnoresIntervalGate(), TestTick_RunsWhenIntervalElapsed(), TestTick_SkipsDisabledOrUnconfigured(), TestTick_SkipsWhenNotYetDue(), fakeDest, fakeStore

### Community 30 - "Community 30"
Cohesion: 0.21
Nodes (5): Destination, Runner, Store, WriteMealsCSV(), WriteRollupsCSV()

### Community 31 - "Community 31"
Cohesion: 0.27
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 32 - "Community 32"
Cohesion: 0.22
Nodes (7): Embedder, FoodStore, Matcher, PrecedenceStore, Resolver, finalize(), Source

### Community 33 - "Community 33"
Cohesion: 0.2
Nodes (1): fakeStore

### Community 34 - "Community 34"
Cohesion: 0.2
Nodes (9): Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser, PendingStore, Store (+1 more)

### Community 35 - "Community 35"
Cohesion: 0.22
Nodes (5): macrosSum(), NewTemplateCommand(), TemplateCommand, TemplateMealLogger, TemplateStore

### Community 36 - "Community 36"
Cohesion: 0.22
Nodes (5): Adapter, embedRequest, embedResponse, generateRequest, generateResponse

### Community 37 - "Community 37"
Cohesion: 0.25
Nodes (4): nutriments, product, searchResponse, Source

### Community 38 - "Community 38"
Cohesion: 0.25
Nodes (5): food, foodNutrient, searchResponse, Source, extractMacros()

### Community 39 - "Community 39"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 42 - "Community 42"
Cohesion: 0.36
Nodes (1): Store

### Community 43 - "Community 43"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 44 - "Community 44"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 45 - "Community 45"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 46 - "Community 46"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 48 - "Community 48"
Cohesion: 0.38
Nodes (4): dayFraction(), insights(), trend(), weeklyStats()

### Community 49 - "Community 49"
Cohesion: 0.29
Nodes (2): NewStartCommand(), StartCommand

### Community 50 - "Community 50"
Cohesion: 0.29
Nodes (2): NewTimezoneCommand(), TimezoneCommand

### Community 51 - "Community 51"
Cohesion: 0.29
Nodes (3): NewWorkoutCommand(), WorkoutCommand, WorkoutStore

### Community 52 - "Community 52"
Cohesion: 0.29
Nodes (3): NewWeightCommand(), WeightCommand, WeightStore

### Community 53 - "Community 53"
Cohesion: 0.29
Nodes (3): NewWaterCommand(), WaterCommand, WaterStore

### Community 54 - "Community 54"
Cohesion: 0.29
Nodes (3): NewLinkCommand(), LinkCodeStore, LinkCommand

### Community 55 - "Community 55"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 56 - "Community 56"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 58 - "Community 58"
Cohesion: 0.33
Nodes (1): WebAuthnUser

### Community 59 - "Community 59"
Cohesion: 0.33
Nodes (1): Matcher

### Community 60 - "Community 60"
Cohesion: 0.33
Nodes (1): stubStore

### Community 63 - "Community 63"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 65 - "Community 65"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 68 - "Community 68"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 70 - "Community 70"
Cohesion: 0.5
Nodes (1): Dest

### Community 81 - "Community 81"
Cohesion: 1.0
Nodes (2): dayKey(), relativeDayLabel()

### Community 82 - "Community 82"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 129 - "Community 129"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 130 - "Community 130"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 131 - "Community 131"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 132 - "Community 132"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 133 - "Community 133"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 134 - "Community 134"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 135 - "Community 135"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 136 - "Community 136"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 137 - "Community 137"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 138 - "Community 138"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **208 isolated node(s):** `phraseEntry`, `RecoveryCodeRepo`, `TOTPRepo`, `MFAChallengeRepo`, `LoginAttemptRepo` (+203 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 8`** (70 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateLinkingCode()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetWaterToday()`, `.GetWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.StartFast()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 9`** (55 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetSession()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (10 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (7 nodes): `NewStartCommand()`, `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `start.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (7 nodes): `NewTimezoneCommand()`, `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `timezone.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (6 nodes): `WebAuthnUser`, `.WebAuthnCredentials()`, `.WebAuthnDisplayName()`, `.WebAuthnIcon()`, `.WebAuthnID()`, `.WebAuthnName()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (6 nodes): `New()`, `Matcher`, `.EmbedFood()`, `.Match()`, `.SetThreshold()`, `embedding.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (6 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 68`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (4 nodes): `s3dest.go`, `Dest`, `.Write()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 81`** (3 nodes): `dayKey()`, `relativeDayLabel()`, `History.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 82`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 129`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 130`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 131`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 132`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 133`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 134`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 135`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 136`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 137`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 138`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 2` to `Community 0`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 9`, `Community 14`, `Community 16`, `Community 18`, `Community 19`, `Community 20`, `Community 29`?**
  _High betweenness centrality (0.230) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 2` to `Community 0`, `Community 35`, `Community 4`, `Community 6`, `Community 7`, `Community 43`, `Community 44`, `Community 45`, `Community 13`, `Community 49`, `Community 18`, `Community 50`, `Community 52`, `Community 53`, `Community 54`, `Community 51`, `Community 28`?**
  _High betweenness centrality (0.127) - this node is a cross-community bridge._
- **Why does `now()` connect `Community 0` to `Community 1`, `Community 2`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 21`, `Community 24`, `Community 25`, `Community 27`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Are the 141 inferred relationships involving `New()` (e.g. with `run()` and `buildModelAndIndex()`) actually correct?**
  _`New()` has 141 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 98 inferred relationships involving `now()` (e.g. with `TestCreateSession()` and `TestValidateSessionExpiredAbsolute()`) actually correct?**
  _`now()` has 98 INFERRED edges - model-reasoned connections that need verification._
- **Are the 6 inferred relationships involving `doRequest()` (e.g. with `TestEmailVerifySuccess()` and `TestEmailVerifyInvalidToken()`) actually correct?**
  _`doRequest()` has 6 INFERRED edges - model-reasoned connections that need verification._