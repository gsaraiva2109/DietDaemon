# Graph Report - DietDaemon  (2026-07-16)

## Corpus Check
- 345 files · ~262,818 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3189 nodes · 5547 edges · 76 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 1481 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]
- [[_COMMUNITY_Community 78|Community 78]]
- [[_COMMUNITY_Community 101|Community 101]]
- [[_COMMUNITY_Community 102|Community 102]]
- [[_COMMUNITY_Community 103|Community 103]]
- [[_COMMUNITY_Community 105|Community 105]]
- [[_COMMUNITY_Community 158|Community 158]]
- [[_COMMUNITY_Community 159|Community 159]]
- [[_COMMUNITY_Community 160|Community 160]]
- [[_COMMUNITY_Community 161|Community 161]]
- [[_COMMUNITY_Community 162|Community 162]]
- [[_COMMUNITY_Community 163|Community 163]]
- [[_COMMUNITY_Community 164|Community 164]]
- [[_COMMUNITY_Community 165|Community 165]]
- [[_COMMUNITY_Community 166|Community 166]]
- [[_COMMUNITY_Community 167|Community 167]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 244 edges
2. `Store` - 208 edges
3. `Handler` - 166 edges
4. `doRequest()` - 103 edges
5. `newFakeMealStore()` - 92 edges
6. `newHandler()` - 91 edges
7. `New()` - 90 edges
8. `fakeMealStore` - 86 edges
9. `contains()` - 82 edges
10. `run()` - 67 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `NewLinkCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/link.go
- `run()` --calls--> `NewStatusCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/status.go
- `run()` --calls--> `NewWeightCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/weight.go
- `run()` --calls--> `NewProfileCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/profile.go
- `run()` --calls--> `NewWaterCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/water.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (44): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, fastRow (+36 more)

### Community 1 - "Community 1"
Cohesion: 0.02
Nodes (194): collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory(), TestRouterTextOnly(), TestRouterTextOnly_doneForwarded(), TestRouterToolCallMaxRounds() (+186 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (57): credCreateConfig, credRevokeConfig, customFoodRequest, Handler, hostOnly(), isSixDigit(), readSessionCookie(), writeJSONList() (+49 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (114): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, NewCancelCommand(), CancelCommand, NewFoodCommand(), FoodCommand, FoodStore (+106 more)

### Community 4 - "Community 4"
Cohesion: 0.05
Nodes (134): emailToken, fakeMailer, fakeMealLogger, fakeSuggester, buildAuthSecurityHandler(), TestHandleLoginUnknownEmailStillHashes(), TestHandleRegisterDuplicateEmailSkipsHash(), TestHandleTOTPChallengeLockout() (+126 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (94): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), fakeCmd, fakeFoodSearcher, fakeLogMealEngine, fakeSuggestEngine, fakeTemplateComposer (+86 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (53): Registry, dayLabel(), download(), sourceLabel(), onSubmit(), onAdd(), relativeCaption(), copy() (+45 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (6): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, mfaEmailTestStore, Store, fakePending

### Community 9 - "Community 9"
Cohesion: 0.05
Nodes (77): totpChallengeAuthStore, fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore (+69 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 11 - "Community 11"
Cohesion: 0.03
Nodes (35): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+27 more)

### Community 12 - "Community 12"
Cohesion: 0.03
Nodes (45): fakeChatAdapter, blockingChatAdapter, fakeChatAdapter, Adapter, joinedRoom, callbackDataByIndex(), New(), newPendingMarkupStore() (+37 more)

### Community 13 - "Community 13"
Cohesion: 0.05
Nodes (46): bulkUpserter, main(), run(), runBackfill(), runImport(), tempStore(), TestRunImport_DryRunWritesNothing(), TestRunImport_TACO() (+38 more)

### Community 14 - "Community 14"
Cohesion: 0.05
Nodes (31): MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand(), parseTargetArgs(), TargetCommand, close() (+23 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (49): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+41 more)

### Community 16 - "Community 16"
Cohesion: 0.03
Nodes (56): FS(), AuditEvent, BackupConfig, BodyCompositionSummary, CorrectionFeedback, CustomFoodInput, DailyRollup, DailyTargets (+48 more)

### Community 17 - "Community 17"
Cohesion: 0.05
Nodes (42): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+34 more)

### Community 18 - "Community 18"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 19 - "Community 19"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (32): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+24 more)

### Community 21 - "Community 21"
Cohesion: 0.05
Nodes (47): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+39 more)

### Community 22 - "Community 22"
Cohesion: 0.05
Nodes (38): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthStore, BackupRunner, ChatStore, EmailTokenStore (+30 more)

### Community 23 - "Community 23"
Cohesion: 0.1
Nodes (18): food, foodCategory, foodNutrient, searchResponse, Source, bulkDataTypes(), extractMacros(), foodToMatch() (+10 more)

### Community 24 - "Community 24"
Cohesion: 0.1
Nodes (10): fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming(), TestHandleChatMessageStreamError() (+2 more)

### Community 25 - "Community 25"
Cohesion: 0.13
Nodes (19): entry, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB(), requireNoErr(), TestCacheUpdatesOnUpsertAndInvalidatesOnDelete(), TestCosineSimilarity() (+11 more)

### Community 26 - "Community 26"
Cohesion: 0.15
Nodes (19): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+11 more)

### Community 27 - "Community 27"
Cohesion: 0.12
Nodes (16): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+8 more)

### Community 28 - "Community 28"
Cohesion: 0.09
Nodes (14): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+6 more)

### Community 29 - "Community 29"
Cohesion: 0.11
Nodes (10): NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal(), CorrectCommand, CorrectResolver, CorrectStore (+2 more)

### Community 30 - "Community 30"
Cohesion: 0.09
Nodes (12): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), Adapter (+4 more)

### Community 31 - "Community 31"
Cohesion: 0.15
Nodes (14): sendOut(), Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray(), TestExtractSuggestions_IntArray(), TestExtractSuggestions_MalformedJSON(), TestExtractSuggestions_NoBlock() (+6 more)

### Community 32 - "Community 32"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 33 - "Community 33"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 34 - "Community 34"
Cohesion: 0.2
Nodes (8): nutriments, meetsPopularity(), New(), NewBulk(), parseQuantity(), product, searchResponse, Source

### Community 35 - "Community 35"
Cohesion: 0.17
Nodes (5): Destination, Runner, Store, WriteMealsCSV(), WriteRollupsCSV()

### Community 36 - "Community 36"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 37 - "Community 37"
Cohesion: 0.16
Nodes (8): Adapter, embedRequest, embedResponse, generateRequest, generateResponse, uniqueModels(), pullRequest, tagsResponse

### Community 38 - "Community 38"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 39 - "Community 39"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 40 - "Community 40"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 41 - "Community 41"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 42 - "Community 42"
Cohesion: 0.33
Nodes (8): EffectiveWeeklyTarget(), TestEffectiveWeeklyTarget_CeilingClamp(), TestEffectiveWeeklyTarget_FloorClamp(), TestEffectiveWeeklyTarget_LastDay(), TestEffectiveWeeklyTarget_MidweekOvereating_LowersTarget(), TestEffectiveWeeklyTarget_MidweekUndereating_RaisesTarget(), TestEffectiveWeeklyTarget_Monday_PlainTarget(), TestEffectiveWeeklyTarget_OverrideTarget()

### Community 43 - "Community 43"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 44 - "Community 44"
Cohesion: 0.25
Nodes (3): SuggestCommand, SuggestEngine, SuggestFoodSearcher

### Community 45 - "Community 45"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 47 - "Community 47"
Cohesion: 0.36
Nodes (1): Store

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 50 - "Community 50"
Cohesion: 0.52
Nodes (5): floatPtr(), intPtr(), TestToWorkout(), TestToWorkoutNilSafety(), ToWorkout()

### Community 51 - "Community 51"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 52 - "Community 52"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 53 - "Community 53"
Cohesion: 0.29
Nodes (1): stubStore

### Community 54 - "Community 54"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 55 - "Community 55"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 56 - "Community 56"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 60 - "Community 60"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 61 - "Community 61"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 67 - "Community 67"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 68 - "Community 68"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 70 - "Community 70"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 71 - "Community 71"
Cohesion: 0.5
Nodes (1): Dest

### Community 78 - "Community 78"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 101 - "Community 101"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 102 - "Community 102"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 103 - "Community 103"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 158 - "Community 158"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 159 - "Community 159"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 160 - "Community 160"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 161 - "Community 161"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 162 - "Community 162"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 163 - "Community 163"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 164 - "Community 164"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 165 - "Community 165"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 166 - "Community 166"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 167 - "Community 167"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **339 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+334 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 10`** (85 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (7 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetRollups()`, `.ListUsers()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (6 nodes): `fakeStore`, `.FrequentFoods()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetRollup()`, `.GetTargets()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 68`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (4 nodes): `s3dest.go`, `Dest`, `.Write()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 101`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 102`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 103`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 158`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 159`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 160`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 161`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 162`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 163`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 164`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 165`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 166`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 167`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 7`, `Community 8`, `Community 9`, `Community 13`, `Community 14`, `Community 15`, `Community 17`, `Community 22`, `Community 23`, `Community 24`, `Community 25`, `Community 31`?**
  _High betweenness centrality (0.308) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 3` to `Community 1`, `Community 2`, `Community 4`, `Community 5`, `Community 36`, `Community 7`, `Community 9`, `Community 43`, `Community 11`, `Community 13`, `Community 14`, `Community 16`, `Community 17`, `Community 22`, `Community 29`?**
  _High betweenness centrality (0.107) - this node is a cross-community bridge._
- **Why does `Store` connect `Community 0` to `Community 2`, `Community 12`?**
  _High betweenness centrality (0.089) - this node is a cross-community bridge._
- **Are the 239 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 239 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 19 inferred relationships involving `doRequest()` (e.g. with `TestEmailVerifySuccess()` and `TestEmailVerifyInvalidToken()`) actually correct?**
  _`doRequest()` has 19 INFERRED edges - model-reasoned connections that need verification._
- **Are the 7 inferred relationships involving `newFakeMealStore()` (e.g. with `buildEmailHandler()` and `newAuthHandlerForTest()`) actually correct?**
  _`newFakeMealStore()` has 7 INFERRED edges - model-reasoned connections that need verification._