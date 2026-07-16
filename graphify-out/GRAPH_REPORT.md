# Graph Report - DietDaemon  (2026-07-16)

## Corpus Check
- 339 files · ~257,387 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3141 nodes · 5444 edges · 84 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 1457 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 41|Community 41]]
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
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 74|Community 74]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 107|Community 107]]
- [[_COMMUNITY_Community 108|Community 108]]
- [[_COMMUNITY_Community 109|Community 109]]
- [[_COMMUNITY_Community 110|Community 110]]
- [[_COMMUNITY_Community 112|Community 112]]
- [[_COMMUNITY_Community 165|Community 165]]
- [[_COMMUNITY_Community 166|Community 166]]
- [[_COMMUNITY_Community 167|Community 167]]
- [[_COMMUNITY_Community 168|Community 168]]
- [[_COMMUNITY_Community 169|Community 169]]
- [[_COMMUNITY_Community 170|Community 170]]
- [[_COMMUNITY_Community 171|Community 171]]
- [[_COMMUNITY_Community 172|Community 172]]
- [[_COMMUNITY_Community 173|Community 173]]
- [[_COMMUNITY_Community 174|Community 174]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 242 edges
2. `Store` - 199 edges
3. `Handler` - 163 edges
4. `doRequest()` - 100 edges
5. `New()` - 90 edges
6. `newFakeMealStore()` - 89 edges
7. `newHandler()` - 88 edges
8. `fakeMealStore` - 82 edges
9. `contains()` - 82 edges
10. `run()` - 67 edges

## Surprising Connections (you probably didn't know these)
- `NumberField()` --calls--> `parseFloat()`  [INFERRED]
  web/src/components/OnboardingWizard.tsx → adapters/nutrition/taco/taco.go
- `run()` --calls--> `NewCancelCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/cancel.go
- `run()` --calls--> `NewTimezoneCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/timezone.go
- `run()` --calls--> `NewStartCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/start.go
- `run()` --calls--> `NewLinkCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/link.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.02
Nodes (208): collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory(), TestRouterTextOnly(), TestRouterTextOnly_doneForwarded(), TestRouterToolCallMaxRounds() (+200 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (40): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, fastRow (+32 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (65): credCreateConfig, credRevokeConfig, Handler, hostOnly(), isSixDigit(), readSessionCookie(), writeJSONList(), calculateTDEE() (+57 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (96): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal() (+88 more)

### Community 4 - "Community 4"
Cohesion: 0.05
Nodes (130): emailToken, fakeMailer, fakeMealLogger, fakeSuggester, buildAuthSecurityHandler(), TestHandleLoginUnknownEmailStillHashes(), TestHandleRegisterDuplicateEmailSkipsHash(), TestHandleTOTPChallengeLockout() (+122 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (95): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, ChatRouteStore, ChatSender, DigestRule, DigestStore, fakeChatRouteStore (+87 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (6): authHandlerTestStore, emailTestAuthStore, fakeMealStore, mfaEmailTestStore, Store, fakePending

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (51): dayLabel(), download(), sourceLabel(), onSubmit(), onAdd(), relativeCaption(), copy(), copyPng() (+43 more)

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (55): fakeChatAdapter, fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming() (+47 more)

### Community 10 - "Community 10"
Cohesion: 0.03
Nodes (35): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+27 more)

### Community 11 - "Community 11"
Cohesion: 0.05
Nodes (46): bulkUpserter, main(), run(), runBackfill(), runImport(), tempStore(), TestRunImport_DryRunWritesNothing(), TestRunImport_TACO() (+38 more)

### Community 12 - "Community 12"
Cohesion: 0.04
Nodes (49): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+41 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (57): FS(), APIKey, AuditEvent, BackupConfig, BodyCompositionSummary, CorrectionFeedback, DailyRollup, DailyTargets (+49 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (1): fakeAuthStore

### Community 15 - "Community 15"
Cohesion: 0.1
Nodes (53): postgresDB(), TestFoodImportFingerprintStore(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods(), TestPostgresUserRoundTrip(), TestGetUserByOIDCIdentity(), TestLinkOIDCIdentityUniqueness() (+45 more)

### Community 16 - "Community 16"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 17 - "Community 17"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 18 - "Community 18"
Cohesion: 0.06
Nodes (32): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+24 more)

### Community 19 - "Community 19"
Cohesion: 0.05
Nodes (47): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+39 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (32): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthStore, BackupRunner, ChatStore, EmailTokenStore (+24 more)

### Community 21 - "Community 21"
Cohesion: 0.09
Nodes (25): Embedder, fakeEmbedder, fakeSource, fingerprintStore, localFingerprint(), New(), NewWithLocalPaths(), replaceDataset() (+17 more)

### Community 22 - "Community 22"
Cohesion: 0.08
Nodes (22): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+14 more)

### Community 23 - "Community 23"
Cohesion: 0.1
Nodes (18): food, foodCategory, foodNutrient, searchResponse, Source, bulkDataTypes(), extractMacros(), foodToMatch() (+10 more)

### Community 24 - "Community 24"
Cohesion: 0.13
Nodes (12): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), plural(), questionText() (+4 more)

### Community 25 - "Community 25"
Cohesion: 0.07
Nodes (18): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), TestNew() (+10 more)

### Community 26 - "Community 26"
Cohesion: 0.15
Nodes (14): fakeCmd, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape(), TestHelpCommand_ListAll() (+6 more)

### Community 27 - "Community 27"
Cohesion: 0.14
Nodes (19): entry, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB(), requireNoErr(), TestCacheInvalidation(), TestCosineSimilarity() (+11 more)

### Community 28 - "Community 28"
Cohesion: 0.15
Nodes (19): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+11 more)

### Community 29 - "Community 29"
Cohesion: 0.12
Nodes (16): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+8 more)

### Community 30 - "Community 30"
Cohesion: 0.12
Nodes (12): fakeFoodSearcher, fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_IngredientArgsResolveViaSearch(), TestSuggestCommand_IngredientArgsSkipUnresolvedNames() (+4 more)

### Community 31 - "Community 31"
Cohesion: 0.09
Nodes (12): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), Adapter (+4 more)

### Community 32 - "Community 32"
Cohesion: 0.15
Nodes (14): sendOut(), Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray(), TestExtractSuggestions_IntArray(), TestExtractSuggestions_MalformedJSON(), TestExtractSuggestions_NoBlock() (+6 more)

### Community 33 - "Community 33"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 34 - "Community 34"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 35 - "Community 35"
Cohesion: 0.2
Nodes (8): nutriments, meetsPopularity(), New(), NewBulk(), parseQuantity(), product, searchResponse, Source

### Community 36 - "Community 36"
Cohesion: 0.17
Nodes (5): Destination, Runner, Store, WriteMealsCSV(), WriteRollupsCSV()

### Community 37 - "Community 37"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 38 - "Community 38"
Cohesion: 0.16
Nodes (8): Adapter, embedRequest, embedResponse, generateRequest, generateResponse, uniqueModels(), pullRequest, tagsResponse

### Community 39 - "Community 39"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 40 - "Community 40"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 41 - "Community 41"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 42 - "Community 42"
Cohesion: 0.31
Nodes (5): close(), NumberField(), profilePayload(), save(), skipOrCancel()

### Community 43 - "Community 43"
Cohesion: 0.28
Nodes (6): fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeStore

### Community 44 - "Community 44"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 45 - "Community 45"
Cohesion: 0.25
Nodes (4): MealStore, NewTargetCommand(), parseTargetArgs(), TargetCommand

### Community 46 - "Community 46"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 48 - "Community 48"
Cohesion: 0.36
Nodes (1): Store

### Community 49 - "Community 49"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 50 - "Community 50"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 51 - "Community 51"
Cohesion: 0.25
Nodes (3): NewLinkCommand(), LinkCodeStore, LinkCommand

### Community 52 - "Community 52"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 54 - "Community 54"
Cohesion: 0.52
Nodes (5): floatPtr(), intPtr(), TestToWorkout(), TestToWorkoutNilSafety(), ToWorkout()

### Community 55 - "Community 55"
Cohesion: 0.29
Nodes (2): NewStartCommand(), StartCommand

### Community 56 - "Community 56"
Cohesion: 0.29
Nodes (2): NewTimezoneCommand(), TimezoneCommand

### Community 57 - "Community 57"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 58 - "Community 58"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 59 - "Community 59"
Cohesion: 0.29
Nodes (1): stubStore

### Community 60 - "Community 60"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 61 - "Community 61"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 62 - "Community 62"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 66 - "Community 66"
Cohesion: 0.4
Nodes (1): fakeCommand

### Community 67 - "Community 67"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 68 - "Community 68"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 73 - "Community 73"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 74 - "Community 74"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 76 - "Community 76"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 77 - "Community 77"
Cohesion: 0.5
Nodes (1): Dest

### Community 84 - "Community 84"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (1): pendingAliasView

### Community 108 - "Community 108"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 109 - "Community 109"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 110 - "Community 110"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 112 - "Community 112"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 165 - "Community 165"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 166 - "Community 166"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 167 - "Community 167"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 168 - "Community 168"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 169 - "Community 169"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 170 - "Community 170"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 171 - "Community 171"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 172 - "Community 172"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 173 - "Community 173"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 174 - "Community 174"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **338 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+333 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 14`** (61 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateAPIKey()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateShareToken()`, `.CreateUserWithOIDC()`, `.CreateUserWithPassword()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByShareToken()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListShareTokens()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RecordLoginAttempt()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (7 nodes): `NewStartCommand()`, `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `start.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (7 nodes): `NewTimezoneCommand()`, `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `timezone.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (7 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetRollups()`, `.ListUsers()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 62`** (6 nodes): `fakeStore`, `.FrequentFoods()`, `.GetFood()`, `.GetFoodDetail()`, `.GetRollup()`, `.GetTargets()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (5 nodes): `fakeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 67`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (4 nodes): `s3dest.go`, `Dest`, `.Write()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (2 nodes): `pendingAliasView`, `handler_food.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 108`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 109`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 110`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 112`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 165`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 166`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 167`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 168`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 169`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 170`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 171`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 172`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 173`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 174`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 0` to `Community 32`, `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 8`, `Community 9`, `Community 11`, `Community 12`, `Community 15`, `Community 20`, `Community 21`, `Community 22`, `Community 23`, `Community 25`, `Community 27`, `Community 30`?**
  _High betweenness centrality (0.331) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 0` to `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 8`, `Community 10`, `Community 11`, `Community 13`, `Community 20`, `Community 21`, `Community 26`, `Community 30`, `Community 37`, `Community 43`, `Community 44`, `Community 45`, `Community 49`, `Community 50`, `Community 51`, `Community 55`, `Community 56`?**
  _High betweenness centrality (0.127) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 2` to `Community 0`, `Community 33`, `Community 3`, `Community 36`, `Community 5`, `Community 12`, `Community 20`, `Community 22`?**
  _High betweenness centrality (0.100) - this node is a cross-community bridge._
- **Are the 237 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 237 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 19 inferred relationships involving `doRequest()` (e.g. with `TestEmailVerifySuccess()` and `TestEmailVerifyInvalidToken()`) actually correct?**
  _`doRequest()` has 19 INFERRED edges - model-reasoned connections that need verification._
- **Are the 88 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 88 INFERRED edges - model-reasoned connections that need verification._