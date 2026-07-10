# Graph Report - DietDaemon  (2026-07-10)

## Corpus Check
- 301 files · ~207,136 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2732 nodes · 4647 edges · 87 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 1244 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 79|Community 79]]
- [[_COMMUNITY_Community 80|Community 80]]
- [[_COMMUNITY_Community 91|Community 91]]
- [[_COMMUNITY_Community 92|Community 92]]
- [[_COMMUNITY_Community 121|Community 121]]
- [[_COMMUNITY_Community 122|Community 122]]
- [[_COMMUNITY_Community 123|Community 123]]
- [[_COMMUNITY_Community 124|Community 124]]
- [[_COMMUNITY_Community 126|Community 126]]
- [[_COMMUNITY_Community 171|Community 171]]
- [[_COMMUNITY_Community 172|Community 172]]
- [[_COMMUNITY_Community 173|Community 173]]
- [[_COMMUNITY_Community 174|Community 174]]
- [[_COMMUNITY_Community 175|Community 175]]
- [[_COMMUNITY_Community 176|Community 176]]
- [[_COMMUNITY_Community 177|Community 177]]
- [[_COMMUNITY_Community 178|Community 178]]
- [[_COMMUNITY_Community 179|Community 179]]
- [[_COMMUNITY_Community 180|Community 180]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 189 edges
2. `Store` - 175 edges
3. `Handler` - 147 edges
4. `now()` - 99 edges
5. `New()` - 90 edges
6. `doRequest()` - 82 edges
7. `newHandler()` - 81 edges
8. `fakeMealStore` - 78 edges
9. `newFakeMealStore()` - 78 edges
10. `contains()` - 76 edges

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

## Communities

### Community 0 - "Community 0"
Cohesion: 0.02
Nodes (107): AuthConfig, AuthStore, BackupRunner, ChatStore, emailToken, fakeMailer, Handler, clientIP() (+99 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (39): NewWebAuthnHandle(), parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, fastRow, foodDetailRow (+31 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (174): collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory(), TestRouterTextOnly(), TestRouterTextOnly_doneForwarded(), TestRouterToolCallMaxRounds() (+166 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (83): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal(), fakeCorrectResolver (+75 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (46): ProtectedRoute(), UtilityBar(), VerifyEmailBanner(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey() (+38 more)

### Community 5 - "Community 5"
Cohesion: 0.09
Nodes (83): fakeMealLogger, fakeSuggester, decodeJSON(), doRequest(), newFakeMealStore(), newHandler(), TestAddAlias(), TestAddAliasMissing() (+75 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (54): fakeChatAdapter, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming(), TestHandleChatMessageStreamError() (+46 more)

### Community 7 - "Community 7"
Cohesion: 0.05
Nodes (63): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, DigestRule, fakeChatRouteStore, fakeChatSender, fakeDigestStore, fakeFullStore (+55 more)

### Community 8 - "Community 8"
Cohesion: 0.03
Nodes (1): fakeMealStore

### Community 9 - "Community 9"
Cohesion: 0.03
Nodes (52): FS(), APIKey, AuditEvent, BackupConfig, BodyCompositionSummary, DailyRollup, DailyTargets, Fast (+44 more)

### Community 10 - "Community 10"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (1): fakeAuthStore

### Community 12 - "Community 12"
Cohesion: 0.05
Nodes (38): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), llmItem, llmResponse (+30 more)

### Community 13 - "Community 13"
Cohesion: 0.07
Nodes (48): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+40 more)

### Community 14 - "Community 14"
Cohesion: 0.04
Nodes (23): fakeStore, CorrectCommand, CorrectResolver, CorrectStore, MealStore, NewProfileCommand(), ProfileCommand, ProfileStore (+15 more)

### Community 15 - "Community 15"
Cohesion: 0.05
Nodes (47): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+39 more)

### Community 16 - "Community 16"
Cohesion: 0.06
Nodes (22): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, calcSleepHours(), computeSleepDuration(), formatDuration(), NewSleepCommand() (+14 more)

### Community 17 - "Community 17"
Cohesion: 0.05
Nodes (28): runHevyImport(), hevyClient, hevyKeyStatus, importResult, Client, NewClient(), listResponse, floatPtr() (+20 more)

### Community 18 - "Community 18"
Cohesion: 0.09
Nodes (21): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+13 more)

### Community 19 - "Community 19"
Cohesion: 0.09
Nodes (24): ChatRouteStore, ChatSender, DigestStore, HealthStore, Notifier, NudgeStore, Option, RuleConfigStore (+16 more)

### Community 20 - "Community 20"
Cohesion: 0.07
Nodes (28): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+20 more)

### Community 21 - "Community 21"
Cohesion: 0.1
Nodes (22): Bundle, NewBundle(), entry, Index, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB() (+14 more)

### Community 22 - "Community 22"
Cohesion: 0.12
Nodes (13): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), parseGrams(), plural() (+5 more)

### Community 23 - "Community 23"
Cohesion: 0.1
Nodes (17): Dialect, NewDialect(), SQLiteDialect(), TestColumnExists(), TestNewDialectInvalid(), TestNow(), TestPlaceholder(), TestPostgresRewritePlaceholders() (+9 more)

### Community 24 - "Community 24"
Cohesion: 0.15
Nodes (14): fakeCmd, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape(), TestHelpCommand_ListAll() (+6 more)

### Community 25 - "Community 25"
Cohesion: 0.12
Nodes (16): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+8 more)

### Community 26 - "Community 26"
Cohesion: 0.12
Nodes (14): download(), copyPng(), dataUrlToBlob(), downloadPng(), render(), ApiError, blobRequest(), handleUnauthorized() (+6 more)

### Community 27 - "Community 27"
Cohesion: 0.09
Nodes (1): emailTestAuthStore

### Community 28 - "Community 28"
Cohesion: 0.09
Nodes (12): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), Adapter (+4 more)

### Community 29 - "Community 29"
Cohesion: 0.15
Nodes (15): New(), sendOut(), Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray(), TestExtractSuggestions_IntArray(), TestExtractSuggestions_MalformedJSON() (+7 more)

### Community 30 - "Community 30"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 31 - "Community 31"
Cohesion: 0.12
Nodes (8): onSubmit(), onAdd(), isMfaChallenge(), isWebAuthnCancel(), loginWithPasskey(), registerPasskey(), signInWithPasskey(), usePasskey()

### Community 32 - "Community 32"
Cohesion: 0.15
Nodes (10): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), parseCookies(), recordFailure(), seed(), sessionFor() (+2 more)

### Community 33 - "Community 33"
Cohesion: 0.18
Nodes (13): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+5 more)

### Community 34 - "Community 34"
Cohesion: 0.17
Nodes (5): Destination, Runner, Store, WriteMealsCSV(), WriteRollupsCSV()

### Community 35 - "Community 35"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 36 - "Community 36"
Cohesion: 0.27
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 37 - "Community 37"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 38 - "Community 38"
Cohesion: 0.22
Nodes (7): Embedder, FoodStore, Matcher, PrecedenceStore, Resolver, finalize(), Source

### Community 39 - "Community 39"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 40 - "Community 40"
Cohesion: 0.2
Nodes (9): Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser, PendingStore, Store (+1 more)

### Community 41 - "Community 41"
Cohesion: 0.22
Nodes (5): macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore

### Community 42 - "Community 42"
Cohesion: 0.22
Nodes (5): Adapter, embedRequest, embedResponse, generateRequest, generateResponse

### Community 43 - "Community 43"
Cohesion: 0.25
Nodes (4): nutriments, product, searchResponse, Source

### Community 44 - "Community 44"
Cohesion: 0.25
Nodes (5): food, foodNutrient, searchResponse, Source, extractMacros()

### Community 45 - "Community 45"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 48 - "Community 48"
Cohesion: 0.36
Nodes (1): Store

### Community 49 - "Community 49"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 50 - "Community 50"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 51 - "Community 51"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 52 - "Community 52"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 54 - "Community 54"
Cohesion: 0.38
Nodes (4): dayFraction(), insights(), trend(), weeklyStats()

### Community 55 - "Community 55"
Cohesion: 0.29
Nodes (2): NewStartCommand(), StartCommand

### Community 56 - "Community 56"
Cohesion: 0.29
Nodes (2): NewTimezoneCommand(), TimezoneCommand

### Community 57 - "Community 57"
Cohesion: 0.29
Nodes (3): NewWorkoutCommand(), WorkoutCommand, WorkoutStore

### Community 58 - "Community 58"
Cohesion: 0.29
Nodes (3): NewWeightCommand(), WeightCommand, WeightStore

### Community 59 - "Community 59"
Cohesion: 0.29
Nodes (3): NewWaterCommand(), WaterCommand, WaterStore

### Community 60 - "Community 60"
Cohesion: 0.29
Nodes (3): NewLinkCommand(), LinkCodeStore, LinkCommand

### Community 61 - "Community 61"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 62 - "Community 62"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 63 - "Community 63"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 65 - "Community 65"
Cohesion: 0.33
Nodes (1): WebAuthnUser

### Community 66 - "Community 66"
Cohesion: 0.33
Nodes (1): Matcher

### Community 69 - "Community 69"
Cohesion: 0.4
Nodes (1): fakeCommand

### Community 70 - "Community 70"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 72 - "Community 72"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 76 - "Community 76"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 77 - "Community 77"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 79 - "Community 79"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 80 - "Community 80"
Cohesion: 0.5
Nodes (1): Dest

### Community 91 - "Community 91"
Cohesion: 1.0
Nodes (2): dayKey(), relativeDayLabel()

### Community 92 - "Community 92"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 121 - "Community 121"
Cohesion: 1.0
Nodes (1): pendingAliasView

### Community 122 - "Community 122"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 123 - "Community 123"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 124 - "Community 124"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 126 - "Community 126"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 171 - "Community 171"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 172 - "Community 172"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 173 - "Community 173"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 174 - "Community 174"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 175 - "Community 175"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 176 - "Community 176"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 177 - "Community 177"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 178 - "Community 178"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 179 - "Community 179"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 180 - "Community 180"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **304 isolated node(s):** `phraseEntry`, `HevyWorkout`, `HevyExercise`, `HevySet`, `listResponse` (+299 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 8`** (77 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateLinkingCode()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 11`** (55 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetSession()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 27`** (23 nodes): `emailTestAuthStore`, `.ConsumeWebAuthnSession()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAEmailCode()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RenameWebAuthnCredential()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (7 nodes): `NewStartCommand()`, `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `start.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (7 nodes): `NewTimezoneCommand()`, `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `timezone.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 61`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (6 nodes): `WebAuthnUser`, `.WebAuthnCredentials()`, `.WebAuthnDisplayName()`, `.WebAuthnIcon()`, `.WebAuthnID()`, `.WebAuthnName()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (6 nodes): `New()`, `Matcher`, `.EmbedFood()`, `.Match()`, `.SetThreshold()`, `embedding.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (5 nodes): `fakeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 80`** (4 nodes): `s3dest.go`, `Dest`, `.Write()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 91`** (3 nodes): `dayKey()`, `relativeDayLabel()`, `History.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 92`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 121`** (2 nodes): `pendingAliasView`, `handler_food.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 122`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 123`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 124`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 126`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 171`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 172`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 173`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 174`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 175`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 176`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 177`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 178`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 179`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 180`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 2` to `Community 0`, `Community 1`, `Community 3`, `Community 5`, `Community 6`, `Community 7`, `Community 11`, `Community 14`, `Community 17`, `Community 20`, `Community 21`, `Community 23`, `Community 29`?**
  _High betweenness centrality (0.171) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 2` to `Community 0`, `Community 3`, `Community 7`, `Community 14`, `Community 16`, `Community 19`, `Community 21`, `Community 23`, `Community 24`, `Community 35`, `Community 49`, `Community 50`, `Community 51`, `Community 55`, `Community 56`, `Community 57`, `Community 58`, `Community 59`, `Community 60`?**
  _High betweenness centrality (0.125) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 0` to `Community 33`, `Community 34`, `Community 2`, `Community 36`, `Community 3`, `Community 7`, `Community 17`, `Community 19`, `Community 20`, `Community 30`?**
  _High betweenness centrality (0.082) - this node is a cross-community bridge._
- **Are the 184 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 184 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 95 inferred relationships involving `now()` (e.g. with `TestCreateSession()` and `TestValidateSessionExpiredAbsolute()`) actually correct?**
  _`now()` has 95 INFERRED edges - model-reasoned connections that need verification._
- **Are the 88 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 88 INFERRED edges - model-reasoned connections that need verification._