# Graph Report - DietDaemon  (2026-07-07)

## Corpus Check
- 263 files · ~180,708 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2454 nodes · 4225 edges · 72 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 1133 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 78|Community 78]]
- [[_COMMUNITY_Community 103|Community 103]]
- [[_COMMUNITY_Community 104|Community 104]]
- [[_COMMUNITY_Community 106|Community 106]]
- [[_COMMUNITY_Community 143|Community 143]]
- [[_COMMUNITY_Community 144|Community 144]]
- [[_COMMUNITY_Community 145|Community 145]]
- [[_COMMUNITY_Community 146|Community 146]]
- [[_COMMUNITY_Community 147|Community 147]]
- [[_COMMUNITY_Community 148|Community 148]]
- [[_COMMUNITY_Community 149|Community 149]]
- [[_COMMUNITY_Community 150|Community 150]]
- [[_COMMUNITY_Community 151|Community 151]]
- [[_COMMUNITY_Community 152|Community 152]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 173 edges
2. `Store` - 158 edges
3. `Handler` - 129 edges
4. `now()` - 99 edges
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
- `run()` --calls--> `NewWaterCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/water.go
- `run()` --calls--> `NewSleepCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/sleep.go

## Hyperedges (group relationships)
- **** — parser_deterministic, parser_embeddings, parser_llm [INFERRED]
- **** — feature_fasting, feature_weight, feature_water, feature_workouts, feature_sleep, feature_progress_photos [INFERRED]
- **** — roadmap_import_old_logs, roadmap_hevy_import [INFERRED]

## Communities

### Community 0 - "Community 0"
Cohesion: 0.02
Nodes (83): AuthConfig, AuthStore, BackupRunner, emailToken, fakeMailer, Handler, clientIP(), isSixDigit() (+75 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (37): NewWebAuthnHandle(), parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, fastRow, foodDetailRow (+29 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (159): newFakeStore(), TestRunFor_MissingDestinationErrors(), TestRunFor_SetsBackupCounts(), TestRunFor_WarnsOnCountDrop(), TestRunOnce_IgnoresIntervalGate(), TestTick_RunsWhenIntervalElapsed(), TestTick_SkipsDisabledOrUnconfigured(), TestTick_SkipsWhenNotYetDue() (+151 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (96): generateLinkCode(), buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, NewWebAuthn(), NewCancelCommand(), CancelCommand, NewLinkCommand() (+88 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (45): ProtectedRoute(), UtilityBar(), VerifyEmailBanner(), AuthProvider(), useAuth(), demoRange(), fd(), hoursAgo() (+37 more)

### Community 5 - "Community 5"
Cohesion: 0.04
Nodes (67): TestComplete(), TestEmbedNotSupported(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal(), fakeCmd, fakeCorrectResolver (+59 more)

### Community 6 - "Community 6"
Cohesion: 0.09
Nodes (83): fakeMealLogger, fakeSuggester, decodeJSON(), doRequest(), newFakeMealStore(), newHandler(), TestAddAlias(), TestAddAliasMissing() (+75 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (4): fakeMealStore, fakeSessionRepo, Store, fakePending

### Community 8 - "Community 8"
Cohesion: 0.03
Nodes (62): fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_Metadata(), SuggestCommand, SuggestEngine (+54 more)

### Community 9 - "Community 9"
Cohesion: 0.04
Nodes (36): CorrectCommand, CorrectResolver, CorrectStore, MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand() (+28 more)

### Community 10 - "Community 10"
Cohesion: 0.04
Nodes (37): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+29 more)

### Community 11 - "Community 11"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 12 - "Community 12"
Cohesion: 0.04
Nodes (1): fakeAuthStore

### Community 13 - "Community 13"
Cohesion: 0.08
Nodes (44): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), CheckIcon() (+36 more)

### Community 14 - "Community 14"
Cohesion: 0.05
Nodes (47): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+39 more)

### Community 15 - "Community 15"
Cohesion: 0.06
Nodes (21): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, calcSleepHours(), computeSleepDuration(), formatDuration(), NewSleepCommand() (+13 more)

### Community 16 - "Community 16"
Cohesion: 0.14
Nodes (39): postgresDB(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods(), TestPostgresUserRoundTrip(), TestGetUserByOIDCIdentity(), TestLinkOIDCIdentityUniqueness(), TestListDeleteOIDCIdentities() (+31 more)

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
Cohesion: 0.12
Nodes (14): download(), copyPng(), dataUrlToBlob(), downloadPng(), render(), ApiError, blobRequest(), handleUnauthorized() (+6 more)

### Community 21 - "Community 21"
Cohesion: 0.09
Nodes (1): emailTestAuthStore

### Community 22 - "Community 22"
Cohesion: 0.13
Nodes (17): Candidate, CandidateItem, Engine, describeCombo(), rankPrompt(), toSuggestedCombos(), comboVariants(), FindCombos() (+9 more)

### Community 23 - "Community 23"
Cohesion: 0.1
Nodes (13): Config, Mailer, New(), smtpPortOrDefault(), Message, newNone(), noneMailer, newResend() (+5 more)

### Community 24 - "Community 24"
Cohesion: 0.19
Nodes (11): macroValue(), Scheduler, EffectiveWeeklyTarget(), resolveRule(), TestEffectiveWeeklyTarget_CeilingClamp(), TestEffectiveWeeklyTarget_FloorClamp(), TestEffectiveWeeklyTarget_LastDay(), TestEffectiveWeeklyTarget_MidweekOvereating_LowersTarget() (+3 more)

### Community 25 - "Community 25"
Cohesion: 0.09
Nodes (12): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), Adapter (+4 more)

### Community 26 - "Community 26"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 27 - "Community 27"
Cohesion: 0.12
Nodes (8): onSubmit(), onAdd(), isMfaChallenge(), isWebAuthnCancel(), loginWithPasskey(), registerPasskey(), signInWithPasskey(), usePasskey()

### Community 28 - "Community 28"
Cohesion: 0.15
Nodes (10): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), parseCookies(), recordFailure(), seed(), sessionFor() (+2 more)

### Community 29 - "Community 29"
Cohesion: 0.18
Nodes (13): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+5 more)

### Community 30 - "Community 30"
Cohesion: 0.17
Nodes (5): Destination, Runner, Store, WriteMealsCSV(), WriteRollupsCSV()

### Community 31 - "Community 31"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 32 - "Community 32"
Cohesion: 0.27
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 33 - "Community 33"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 34 - "Community 34"
Cohesion: 0.22
Nodes (7): Embedder, FoodStore, Matcher, PrecedenceStore, Resolver, finalize(), Source

### Community 35 - "Community 35"
Cohesion: 0.2
Nodes (9): Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser, PendingStore, Store (+1 more)

### Community 36 - "Community 36"
Cohesion: 0.22
Nodes (5): macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore

### Community 37 - "Community 37"
Cohesion: 0.22
Nodes (5): Adapter, embedRequest, embedResponse, generateRequest, generateResponse

### Community 38 - "Community 38"
Cohesion: 0.25
Nodes (4): nutriments, product, searchResponse, Source

### Community 39 - "Community 39"
Cohesion: 0.25
Nodes (5): food, foodNutrient, searchResponse, Source, extractMacros()

### Community 40 - "Community 40"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 43 - "Community 43"
Cohesion: 0.36
Nodes (1): Store

### Community 44 - "Community 44"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 45 - "Community 45"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 46 - "Community 46"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 48 - "Community 48"
Cohesion: 0.38
Nodes (4): dayFraction(), insights(), trend(), weeklyStats()

### Community 49 - "Community 49"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 50 - "Community 50"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 51 - "Community 51"
Cohesion: 0.29
Nodes (3): NewWaterCommand(), WaterCommand, WaterStore

### Community 52 - "Community 52"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 54 - "Community 54"
Cohesion: 0.33
Nodes (1): WebAuthnUser

### Community 55 - "Community 55"
Cohesion: 0.33
Nodes (1): Matcher

### Community 56 - "Community 56"
Cohesion: 0.33
Nodes (1): stubStore

### Community 59 - "Community 59"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 61 - "Community 61"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 64 - "Community 64"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 66 - "Community 66"
Cohesion: 0.5
Nodes (1): Dest

### Community 77 - "Community 77"
Cohesion: 1.0
Nodes (2): dayKey(), relativeDayLabel()

### Community 78 - "Community 78"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 103 - "Community 103"
Cohesion: 1.0
Nodes (1): pendingAliasView

### Community 104 - "Community 104"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 106 - "Community 106"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 143 - "Community 143"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 144 - "Community 144"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 145 - "Community 145"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 146 - "Community 146"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 147 - "Community 147"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 148 - "Community 148"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 149 - "Community 149"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 150 - "Community 150"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 151 - "Community 151"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 152 - "Community 152"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **258 isolated node(s):** `phraseEntry`, `RecoveryCodeRepo`, `TOTPRepo`, `MFAChallengeRepo`, `LoginAttemptRepo` (+253 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 12`** (54 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 21`** (23 nodes): `emailTestAuthStore`, `.ConsumeWebAuthnSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAEmailCode()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RenameWebAuthnCredential()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (7 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetRollups()`, `.ListUsers()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (6 nodes): `WebAuthnUser`, `.WebAuthnCredentials()`, `.WebAuthnDisplayName()`, `.WebAuthnIcon()`, `.WebAuthnID()`, `.WebAuthnName()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (6 nodes): `New()`, `Matcher`, `.EmbedFood()`, `.Match()`, `.SetThreshold()`, `embedding.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (6 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 64`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (4 nodes): `s3dest.go`, `Dest`, `.Write()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (3 nodes): `dayKey()`, `relativeDayLabel()`, `History.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 103`** (2 nodes): `pendingAliasView`, `handler_food.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 104`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 106`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 143`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 144`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 145`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 146`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 147`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 148`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 149`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 150`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 151`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 152`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 2` to `Community 0`, `Community 1`, `Community 3`, `Community 6`, `Community 8`, `Community 9`, `Community 16`, `Community 17`, `Community 19`?**
  _High betweenness centrality (0.151) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 3` to `Community 0`, `Community 2`, `Community 5`, `Community 8`, `Community 9`, `Community 44`, `Community 45`, `Community 15`, `Community 17`, `Community 19`, `Community 51`, `Community 31`?**
  _High betweenness centrality (0.142) - this node is a cross-community bridge._
- **Why does `Store` connect `Community 1` to `Community 0`, `Community 2`, `Community 10`?**
  _High betweenness centrality (0.086) - this node is a cross-community bridge._
- **Are the 168 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 168 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 95 inferred relationships involving `now()` (e.g. with `TestCreateSession()` and `TestValidateSessionExpiredAbsolute()`) actually correct?**
  _`now()` has 95 INFERRED edges - model-reasoned connections that need verification._
- **Are the 88 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 88 INFERRED edges - model-reasoned connections that need verification._