# Graph Report - .  (2026-07-07)

## Corpus Check
- 88 files · ~0 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2445 nodes · 4496 edges · 71 communities detected
- Extraction: 74% EXTRACTED · 26% INFERRED · 0% AMBIGUOUS · INFERRED: 1156 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Store Layer|Store Layer]]
- [[_COMMUNITY_Auth & Backup System|Auth & Backup System]]
- [[_COMMUNITY_Integration Tests|Integration Tests]]
- [[_COMMUNITY_Bot Commands & Nudges|Bot Commands & Nudges]]
- [[_COMMUNITY_LLM & Command Tests|LLM & Command Tests]]
- [[_COMMUNITY_React Frontend Components|React Frontend Components]]
- [[_COMMUNITY_Meal Logging Pipeline|Meal Logging Pipeline]]
- [[_COMMUNITY_Suggest Command|Suggest Command]]
- [[_COMMUNITY_Fake Meal Store (Test Fixtures)|Fake Meal Store (Test Fixtures)]]
- [[_COMMUNITY_Messaging Adapters|Messaging Adapters]]
- [[_COMMUNITY_Documentation & Design Principles|Documentation & Design Principles]]
- [[_COMMUNITY_Fake Auth Store (Test Fixtures)|Fake Auth Store (Test Fixtures)]]
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
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 78|Community 78]]
- [[_COMMUNITY_Community 79|Community 79]]
- [[_COMMUNITY_Community 105|Community 105]]
- [[_COMMUNITY_Community 127|Community 127]]
- [[_COMMUNITY_Community 128|Community 128]]
- [[_COMMUNITY_Community 129|Community 129]]
- [[_COMMUNITY_Community 130|Community 130]]
- [[_COMMUNITY_Community 131|Community 131]]
- [[_COMMUNITY_Community 132|Community 132]]
- [[_COMMUNITY_Community 133|Community 133]]
- [[_COMMUNITY_Community 134|Community 134]]
- [[_COMMUNITY_Community 135|Community 135]]
- [[_COMMUNITY_Community 136|Community 136]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 167 edges
2. `Store` - 157 edges
3. `Handler` - 128 edges
4. `now()` - 102 edges
5. `New()` - 90 edges
6. `doRequest()` - 82 edges
7. `newHandler()` - 81 edges
8. `newFakeMealStore()` - 78 edges
9. `fakeMealStore` - 71 edges
10. `fakeAuthStore` - 59 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `NewStatusCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/status.go
- `run()` --calls--> `NewProfileCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/profile.go
- `run()` --calls--> `NewFoodCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/food.go
- `run()` --calls--> `NewSleepCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/sleep.go
- `run()` --calls--> `NewFastCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/fast.go

## Hyperedges (group relationships)
- **** — parser_deterministic, parser_embeddings, parser_llm [INFERRED]
- **** — feature_fasting, feature_weight, feature_water, feature_workouts, feature_sleep, feature_progress_photos [INFERRED]
- **** — roadmap_import_old_logs, roadmap_hevy_import [INFERRED]

## Communities

### Community 0 - "Store Layer"
Cohesion: 0.02
Nodes (51): NewWebAuthnHandle(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, fastRow, foodDetailRow, foodMatchRow (+43 more)

### Community 1 - "Auth & Backup System"
Cohesion: 0.02
Nodes (71): AuthConfig, AuthStore, BackupRunner, emailToken, fakeMailer, Handler, clientIP(), isSixDigit() (+63 more)

### Community 2 - "Integration Tests"
Cohesion: 0.02
Nodes (160): newFakeStore(), TestRunFor_MissingDestinationErrors(), TestRunOnce_IgnoresIntervalGate(), TestTick_RunsWhenIntervalElapsed(), TestTick_SkipsDisabledOrUnconfigured(), TestTick_SkipsWhenNotYetDue(), fakeDest, Stat() (+152 more)

### Community 3 - "Bot Commands & Nudges"
Cohesion: 0.02
Nodes (98): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), NewCancelCommand(), CancelCommand, NewLinkCommand(), LinkCodeStore, LinkCommand, PendingStore (+90 more)

### Community 4 - "LLM & Command Tests"
Cohesion: 0.03
Nodes (82): TestComplete(), TestEmbedNotSupported(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal(), CorrectCommand, CorrectResolver (+74 more)

### Community 5 - "React Frontend Components"
Cohesion: 0.02
Nodes (45): ProtectedRoute(), UtilityBar(), VerifyEmailBanner(), AuthProvider(), useAuth(), demoRange(), fd(), hoursAgo() (+37 more)

### Community 6 - "Meal Logging Pipeline"
Cohesion: 0.09
Nodes (83): fakeMealLogger, fakeSuggester, decodeJSON(), doRequest(), newFakeMealStore(), newHandler(), TestAddAlias(), TestAddAliasMissing() (+75 more)

### Community 7 - "Suggest Command"
Cohesion: 0.03
Nodes (61): fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_Metadata(), SuggestCommand, SuggestEngine (+53 more)

### Community 8 - "Fake Meal Store (Test Fixtures)"
Cohesion: 0.03
Nodes (1): fakeMealStore

### Community 9 - "Messaging Adapters"
Cohesion: 0.04
Nodes (37): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+29 more)

### Community 10 - "Documentation & Design Principles"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 11 - "Fake Auth Store (Test Fixtures)"
Cohesion: 0.04
Nodes (1): fakeAuthStore

### Community 12 - "Community 12"
Cohesion: 0.08
Nodes (44): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), CheckIcon() (+36 more)

### Community 13 - "Community 13"
Cohesion: 0.05
Nodes (47): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+39 more)

### Community 14 - "Community 14"
Cohesion: 0.06
Nodes (21): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, calcSleepHours(), computeSleepDuration(), formatDuration(), NewSleepCommand() (+13 more)

### Community 15 - "Community 15"
Cohesion: 0.15
Nodes (38): postgresDB(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods(), TestPostgresUserRoundTrip(), TestGetUserByOIDCIdentity(), TestLinkOIDCIdentityUniqueness(), TestListDeleteOIDCIdentities() (+30 more)

### Community 16 - "Community 16"
Cohesion: 0.09
Nodes (21): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+13 more)

### Community 17 - "Community 17"
Cohesion: 0.1
Nodes (22): Bundle, NewBundle(), entry, Index, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB() (+14 more)

### Community 18 - "Community 18"
Cohesion: 0.09
Nodes (20): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), llmItem, llmResponse (+12 more)

### Community 19 - "Community 19"
Cohesion: 0.1
Nodes (17): Dialect, NewDialect(), SQLiteDialect(), TestColumnExists(), TestNewDialectInvalid(), TestNow(), TestPlaceholder(), TestPostgresRewritePlaceholders() (+9 more)

### Community 20 - "Community 20"
Cohesion: 0.14
Nodes (12): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), plural(), questionText() (+4 more)

### Community 21 - "Community 21"
Cohesion: 0.12
Nodes (14): download(), copyPng(), dataUrlToBlob(), downloadPng(), render(), ApiError, blobRequest(), handleUnauthorized() (+6 more)

### Community 22 - "Community 22"
Cohesion: 0.09
Nodes (1): emailTestAuthStore

### Community 23 - "Community 23"
Cohesion: 0.13
Nodes (17): Candidate, CandidateItem, Engine, describeCombo(), rankPrompt(), toSuggestedCombos(), comboVariants(), FindCombos() (+9 more)

### Community 24 - "Community 24"
Cohesion: 0.19
Nodes (12): macroValue(), Scheduler, EffectiveWeeklyTarget(), parseLoggedAt(), resolveRule(), TestEffectiveWeeklyTarget_CeilingClamp(), TestEffectiveWeeklyTarget_FloorClamp(), TestEffectiveWeeklyTarget_LastDay() (+4 more)

### Community 25 - "Community 25"
Cohesion: 0.09
Nodes (12): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), Adapter (+4 more)

### Community 26 - "Community 26"
Cohesion: 0.12
Nodes (8): onSubmit(), onAdd(), isMfaChallenge(), isWebAuthnCancel(), loginWithPasskey(), registerPasskey(), signInWithPasskey(), usePasskey()

### Community 27 - "Community 27"
Cohesion: 0.14
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 28 - "Community 28"
Cohesion: 0.15
Nodes (10): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), parseCookies(), recordFailure(), seed(), sessionFor() (+2 more)

### Community 29 - "Community 29"
Cohesion: 0.12
Nodes (11): Config, Mailer, New(), smtpPortOrDefault(), Message, newResend(), resendMailer, newSES() (+3 more)

### Community 30 - "Community 30"
Cohesion: 0.18
Nodes (13): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+5 more)

### Community 31 - "Community 31"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 32 - "Community 32"
Cohesion: 0.21
Nodes (5): Destination, Runner, Store, WriteMealsCSV(), WriteRollupsCSV()

### Community 33 - "Community 33"
Cohesion: 0.27
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 34 - "Community 34"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 35 - "Community 35"
Cohesion: 0.22
Nodes (7): Embedder, FoodStore, Matcher, PrecedenceStore, Resolver, finalize(), Source

### Community 36 - "Community 36"
Cohesion: 0.2
Nodes (9): Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser, PendingStore, Store (+1 more)

### Community 37 - "Community 37"
Cohesion: 0.25
Nodes (4): MealStore, NewTargetCommand(), parseTargetArgs(), TargetCommand

### Community 38 - "Community 38"
Cohesion: 0.22
Nodes (5): macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore

### Community 39 - "Community 39"
Cohesion: 0.22
Nodes (5): Adapter, embedRequest, embedResponse, generateRequest, generateResponse

### Community 40 - "Community 40"
Cohesion: 0.25
Nodes (4): nutriments, product, searchResponse, Source

### Community 41 - "Community 41"
Cohesion: 0.25
Nodes (5): food, foodNutrient, searchResponse, Source, extractMacros()

### Community 42 - "Community 42"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 45 - "Community 45"
Cohesion: 0.36
Nodes (1): Store

### Community 46 - "Community 46"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 47 - "Community 47"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 50 - "Community 50"
Cohesion: 0.38
Nodes (4): dayFraction(), insights(), trend(), weeklyStats()

### Community 51 - "Community 51"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 52 - "Community 52"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 54 - "Community 54"
Cohesion: 0.33
Nodes (1): WebAuthnUser

### Community 55 - "Community 55"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 56 - "Community 56"
Cohesion: 0.33
Nodes (1): Matcher

### Community 57 - "Community 57"
Cohesion: 0.33
Nodes (1): stubStore

### Community 60 - "Community 60"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 62 - "Community 62"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 65 - "Community 65"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 67 - "Community 67"
Cohesion: 0.5
Nodes (1): Dest

### Community 78 - "Community 78"
Cohesion: 1.0
Nodes (2): dayKey(), relativeDayLabel()

### Community 79 - "Community 79"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 127 - "Community 127"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 128 - "Community 128"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 129 - "Community 129"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 130 - "Community 130"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 131 - "Community 131"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 132 - "Community 132"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 133 - "Community 133"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 134 - "Community 134"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 135 - "Community 135"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 136 - "Community 136"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **255 isolated node(s):** `phraseEntry`, `RecoveryCodeRepo`, `TOTPRepo`, `MFAChallengeRepo`, `LoginAttemptRepo` (+250 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Fake Meal Store (Test Fixtures)`** (70 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateLinkingCode()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetWaterToday()`, `.GetWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.StartFast()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Fake Auth Store (Test Fixtures)`** (54 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 22`** (23 nodes): `emailTestAuthStore`, `.ConsumeWebAuthnSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAEmailCode()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RenameWebAuthnCredential()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (6 nodes): `WebAuthnUser`, `.WebAuthnCredentials()`, `.WebAuthnDisplayName()`, `.WebAuthnIcon()`, `.WebAuthnID()`, `.WebAuthnName()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (6 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetRollups()`, `.ListUsers()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (6 nodes): `New()`, `Matcher`, `.EmbedFood()`, `.Match()`, `.SetThreshold()`, `embedding.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (6 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 67`** (4 nodes): `s3dest.go`, `Dest`, `.Write()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (3 nodes): `dayKey()`, `relativeDayLabel()`, `History.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 79`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 127`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 128`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 129`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 130`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 131`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 132`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 133`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 134`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 135`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 136`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Integration Tests` to `Store Layer`, `Auth & Backup System`, `Bot Commands & Nudges`, `Meal Logging Pipeline`, `Suggest Command`, `Community 15`, `Community 17`, `Community 19`?**
  _High betweenness centrality (0.166) - this node is a cross-community bridge._
- **Why does `run()` connect `Bot Commands & Nudges` to `Auth & Backup System`, `Integration Tests`, `LLM & Command Tests`, `Community 37`, `Suggest Command`, `Community 46`, `Community 47`, `Community 14`, `Community 17`, `Community 19`, `Community 31`?**
  _High betweenness centrality (0.138) - this node is a cross-community bridge._
- **Why does `New()` connect `Integration Tests` to `Store Layer`, `Auth & Backup System`, `Bot Commands & Nudges`, `Meal Logging Pipeline`, `Suggest Command`, `Messaging Adapters`, `Community 15`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **Are the 162 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 162 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 98 inferred relationships involving `now()` (e.g. with `TestCreateSession()` and `TestValidateSessionExpiredAbsolute()`) actually correct?**
  _`now()` has 98 INFERRED edges - model-reasoned connections that need verification._
- **Are the 88 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 88 INFERRED edges - model-reasoned connections that need verification._