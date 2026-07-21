# Graph Report - .  (2026-07-21)

## Corpus Check
- 108 files · ~99,999 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3724 nodes · 7283 edges · 82 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 1944 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Assistant Test.Go|Assistant Test.Go]]
- [[_COMMUNITY_Handler.Go Vision.Go|Handler.Go Vision.Go]]
- [[_COMMUNITY_Store Meals.Go|Store Meals.Go]]
- [[_COMMUNITY_Handler Auth.Go|Handler Auth.Go]]
- [[_COMMUNITY_Handler Auth Test.Go|Handler Auth Test.Go]]
- [[_COMMUNITY_Correct Test.Go|Correct Test.Go]]
- [[_COMMUNITY_Lib .Constructor()|Lib .Constructor()]]
- [[_COMMUNITY_Components Sourcelabel()|Components Sourcelabel()]]
- [[_COMMUNITY_Handler Email Test.Go|Handler Email Test.Go]]
- [[_COMMUNITY_Purge Test.Go|Purge Test.Go]]
- [[_COMMUNITY_Anthropic Chat.Go|Anthropic Chat.Go]]
- [[_COMMUNITY_Product.Md Dietdaemon|Product.Md Dietdaemon]]
- [[_COMMUNITY_Handler Test.Go|Handler Test.Go]]
- [[_COMMUNITY_Components Set()|Components Set()]]
- [[_COMMUNITY_Normalize New()|Normalize New()]]
- [[_COMMUNITY_Fast.Go|Fast.Go]]
- [[_COMMUNITY_Types.Go|Types.Go]]
- [[_COMMUNITY_Foodimport Test.Go|Foodimport Test.Go]]
- [[_COMMUNITY_Mfp Main.Go|Mfp Main.Go]]
- [[_COMMUNITY_Pipeline.Go|Pipeline.Go]]
- [[_COMMUNITY_Roadmap.Md Backup|Roadmap.Md Backup]]
- [[_COMMUNITY_Components|Components]]
- [[_COMMUNITY_Scheduler.Go|Scheduler.Go]]
- [[_COMMUNITY_Anthropic|Anthropic]]
- [[_COMMUNITY_Usda|Usda]]
- [[_COMMUNITY_Session Test.Go|Session Test.Go]]
- [[_COMMUNITY_Handler Chat Test.Go|Handler Chat Test.Go]]
- [[_COMMUNITY_Index.Go|Index.Go]]
- [[_COMMUNITY_Mailer.Go|Mailer.Go]]
- [[_COMMUNITY_Backup Test.Go .Getmealsinrange()|Backup Test.Go .Getmealsinrange()]]
- [[_COMMUNITY_Backup.Md Backup|Backup.Md Backup]]
- [[_COMMUNITY_Suggestions Test.Go|Suggestions Test.Go]]
- [[_COMMUNITY_Adherence Test.Go|Adherence Test.Go]]
- [[_COMMUNITY_Dev-Mock-Api.Mjs|Dev-Mock-Api.Mjs]]
- [[_COMMUNITY_Openfoodfacts|Openfoodfacts]]
- [[_COMMUNITY_Provider.Go|Provider.Go]]
- [[_COMMUNITY_Ports.Go|Ports.Go]]
- [[_COMMUNITY_Handler Passkeys Test.Go|Handler Passkeys Test.Go]]
- [[_COMMUNITY_Pipeline Test.Go|Pipeline Test.Go]]
- [[_COMMUNITY_Lib|Lib]]
- [[_COMMUNITY_Restore Test.Go|Restore Test.Go]]
- [[_COMMUNITY_Status.Go|Status.Go]]
- [[_COMMUNITY_Target.Go|Target.Go]]
- [[_COMMUNITY_Suggest.Go|Suggest.Go]]
- [[_COMMUNITY_Correct.Go|Correct.Go]]
- [[_COMMUNITY_Design.Md Color|Design.Md Color]]
- [[_COMMUNITY_Pendingstore.Go|Pendingstore.Go]]
- [[_COMMUNITY_Food.Go|Food.Go]]
- [[_COMMUNITY_Profile.Go|Profile.Go]]
- [[_COMMUNITY_Cancel.Go|Cancel.Go]]
- [[_COMMUNITY_Gotify|Gotify]]
- [[_COMMUNITY_Hevy|Hevy]]
- [[_COMMUNITY_Start.Go|Start.Go]]
- [[_COMMUNITY_Timezone.Go|Timezone.Go]]
- [[_COMMUNITY_Embedding|Embedding]]
- [[_COMMUNITY_Ntfy|Ntfy]]
- [[_COMMUNITY_Chat.Go|Chat.Go]]
- [[_COMMUNITY_Engine Test.Go|Engine Test.Go]]
- [[_COMMUNITY_Assistant Test.Go|Assistant Test.Go]]
- [[_COMMUNITY_Help.Go|Help.Go]]
- [[_COMMUNITY_Whisper|Whisper]]
- [[_COMMUNITY_Hevy|Hevy]]
- [[_COMMUNITY_Queue.Go|Queue.Go]]
- [[_COMMUNITY_Store.Go|Store.Go]]
- [[_COMMUNITY_Compose Profile|Compose Profile]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Notifier Test.Go|Notifier Test.Go]]
- [[_COMMUNITY_Handler Settings.Go|Handler Settings.Go]]
- [[_COMMUNITY_Store Nudges.Go|Store Nudges.Go]]
- [[_COMMUNITY_Store Provider Keys.Go|Store Provider Keys.Go]]
- [[_COMMUNITY_Stt|Stt.Md]]
- [[_COMMUNITY_Community 106|Community 106]]
- [[_COMMUNITY_Community 146|Community 146]]
- [[_COMMUNITY_Community 147|Community 147]]
- [[_COMMUNITY_Community 148|Community 148]]
- [[_COMMUNITY_Community 149|Community 149]]
- [[_COMMUNITY_Community 150|Community 150]]
- [[_COMMUNITY_Community 151|Community 151]]
- [[_COMMUNITY_Community 152|Community 152]]
- [[_COMMUNITY_Community 153|Community 153]]
- [[_COMMUNITY_Community 154|Community 154]]
- [[_COMMUNITY_Community 155|Community 155]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 278 edges
2. `Store` - 216 edges
3. `Handler` - 175 edges
4. `doRequest()` - 126 edges
5. `newHandler()` - 116 edges
6. `newFakeMealStore()` - 112 edges
7. `contains()` - 98 edges
8. `New()` - 90 edges
9. `fakeMealStore` - 87 edges
10. `run()` - 71 edges

## Surprising Connections (you probably didn't know these)
- `NumberField()` --calls--> `parseFloat()`  [INFERRED]
  web/src/components/OnboardingWizard.tsx → adapters/nutrition/taco/taco.go
- `NotFound()` --calls--> `Handler()`  [INFERRED]
  web/src/routes/NotFound.tsx → internal/web/web.go
- `run()` --calls--> `NewCancelCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/cancel.go
- `run()` --calls--> `NewTimezoneCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/timezone.go
- `run()` --calls--> `NewStartCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/start.go

## Communities

### Community 0 - "Assistant Test.Go"
Cohesion: 0.01
Nodes (282): TestAPIRouteFallbackUsesErrorEnvelope(), buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), TestAuthenticatedRateLimitCategories(), collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError() (+274 more)

### Community 1 - "Handler.Go Vision.Go"
Cohesion: 0.02
Nodes (231): imageSource, visionContentBlock, visionMessage, visionRequest, accountRepos, AccountStore, APIKeyStore, AuditStore (+223 more)

### Community 2 - "Store Meals.Go"
Cohesion: 0.01
Nodes (42): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, credRow, fastRow, foodDetailRow (+34 more)

### Community 3 - "Handler Auth.Go"
Cohesion: 0.01
Nodes (76): TestAPIErrorEnvelope(), TestAPIErrorEnvelopePreservesStreaming(), withAPIErrorEnvelope(), writeAPIError(), WriteError(), Handler, hostOnly(), isSixDigit() (+68 more)

### Community 4 - "Handler Auth Test.Go"
Cohesion: 0.03
Nodes (177): assertBYOKFailure(), TestBYOKFailuresDoNotFallBackToSharedAdapters(), TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), emailToken, erroringCountAuthStore, fakeMailer, fakeMealLogger, fakeSuggester (+169 more)

### Community 5 - "Correct Test.Go"
Cohesion: 0.02
Nodes (123): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestExtractLabel(), TestExtractLabelHTTPError(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement() (+115 more)

### Community 6 - "Lib .Constructor()"
Cohesion: 0.02
Nodes (58): browser, ProtectedRoute(), home_gsaraiva_projects_dietdaemon_web_src_lib_demo, home_gsaraiva_projects_dietdaemon_web_src_lib_demodata, home_gsaraiva_projects_dietdaemon_web_src_lib_types, ApiError, blobRequest(), handleUnauthorized() (+50 more)

### Community 7 - "Components Sourcelabel()"
Cohesion: 0.02
Nodes (58): Registry, confirmReplace(), scaledMacros(), sourceLabel(), dayLabel(), download(), sourceLabel(), MacroBar() (+50 more)

### Community 8 - "Handler Email Test.Go"
Cohesion: 0.02
Nodes (6): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, mfaEmailTestStore, Store, fakePending

### Community 9 - "Purge Test.Go"
Cohesion: 0.07
Nodes (82): totpChallengeAuthStore, fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore (+74 more)

### Community 10 - "Anthropic Chat.Go"
Cohesion: 0.02
Nodes (66): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+58 more)

### Community 11 - "Product.Md Dietdaemon"
Cohesion: 0.03
Nodes (89): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+81 more)

### Community 12 - "Handler Test.Go"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 13 - "Components Set()"
Cohesion: 0.04
Nodes (47): appshell, auth, authcallback, authlayout, commandpalette, renderModal(), close(), NumberField() (+39 more)

### Community 14 - "Normalize New()"
Cohesion: 0.04
Nodes (54): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+46 more)

### Community 15 - "Fast.Go"
Cohesion: 0.03
Nodes (36): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+28 more)

### Community 16 - "Types.Go"
Cohesion: 0.03
Nodes (65): go_pkg_embed, go_pkg_io_fs, go_pkg_path, FS(), APIKey, AuditEvent, BackupConfig, BodyCompositionSummary (+57 more)

### Community 17 - "Foodimport Test.Go"
Cohesion: 0.04
Nodes (46): errorEnvelopeWriter, errorForStatus(), publicErrorMessage(), MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount() (+38 more)

### Community 18 - "Mfp Main.Go"
Cohesion: 0.05
Nodes (44): BuildSource(), LocalPaths(), main(), run(), runBackfill(), runRepair(), groupIntoMeals(), importMeals() (+36 more)

### Community 19 - "Pipeline.Go"
Cohesion: 0.06
Nodes (38): cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer(), newRequestID(), observeRequests(), recoverPanics() (+30 more)

### Community 20 - "Roadmap.Md Backup"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 21 - "Components"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 22 - "Scheduler.Go"
Cohesion: 0.07
Nodes (28): ChatRouteStore, ChatSender, DigestStore, fakeHealthStore, HealthStore, MealHistoryStore, Notifier, NudgeStore (+20 more)

### Community 23 - "Anthropic"
Cohesion: 0.06
Nodes (18): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), ParseResponse() (+10 more)

### Community 24 - "Usda"
Cohesion: 0.1
Nodes (18): food, foodCategory, foodNutrient, searchResponse, Source, bulkDataTypes(), extractMacros(), foodToMatch() (+10 more)

### Community 25 - "Session Test.Go"
Cohesion: 0.12
Nodes (23): readSessionCookie(), bearerToken(), isMutating(), TestBearerTokenEdgeCases(), fakeSessionRepo, Session, CreateSession(), RotateSession() (+15 more)

### Community 26 - "Handler Chat Test.Go"
Cohesion: 0.1
Nodes (10): fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming(), TestHandleChatMessageStreamError() (+2 more)

### Community 27 - "Index.Go"
Cohesion: 0.13
Nodes (19): entry, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB(), requireNoErr(), TestCacheUpdatesOnUpsertAndInvalidatesOnDelete(), TestCosineSimilarity() (+11 more)

### Community 28 - "Mailer.Go"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 29 - "Backup Test.Go .Getmealsinrange()"
Cohesion: 0.07
Nodes (2): allEntitiesFakeStore, fakeStore

### Community 30 - "Backup.Md Backup"
Cohesion: 0.1
Nodes (24): AWS default credential chain (backup), Backup runner, BACKUP_CHECK_INTERVAL, Database-level backup (pg_dump / sqlite3 .backup), internal/exportfmt (shared CSV writer), BACKUP_LOCAL_DIR, local_subdir path-traversal validation, Nudge scheduler (existing background loop) (+16 more)

### Community 31 - "Suggestions Test.Go"
Cohesion: 0.15
Nodes (14): sendOut(), Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray(), TestExtractSuggestions_IntArray(), TestExtractSuggestions_MalformedJSON(), TestExtractSuggestions_NoBlock() (+6 more)

### Community 32 - "Adherence Test.Go"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 33 - "Dev-Mock-Api.Mjs"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 34 - "Openfoodfacts"
Cohesion: 0.2
Nodes (8): nutriments, meetsPopularity(), New(), NewBulk(), parseQuantity(), product, searchResponse, Source

### Community 35 - "Provider.Go"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 36 - "Ports.Go"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 37 - "Handler Passkeys Test.Go"
Cohesion: 0.29
Nodes (7): newPasskeyHandler(), newPasskeyTestStore(), TestHandlePasskeyLoginBeginCreatesDiscoverableCeremony(), TestHandlePasskeyLoginFinishRejectsMissingOrExpiredCeremony(), TestHandlePasskeyRegisterBeginCreatesCeremony(), WithWebAuthn(), passkeyTestStore

### Community 38 - "Pipeline Test.Go"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 39 - "Lib"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 40 - "Restore Test.Go"
Cohesion: 0.2
Nodes (1): fakeStore

### Community 41 - "Status.Go"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 42 - "Target.Go"
Cohesion: 0.25
Nodes (4): MealStore, NewTargetCommand(), parseTargetArgs(), TargetCommand

### Community 43 - "Suggest.Go"
Cohesion: 0.25
Nodes (3): SuggestCommand, SuggestEngine, SuggestFoodSearcher

### Community 44 - "Correct.Go"
Cohesion: 0.25
Nodes (3): CorrectCommand, CorrectResolver, CorrectStore

### Community 45 - "Design.Md Color"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 47 - "Pendingstore.Go"
Cohesion: 0.36
Nodes (1): Store

### Community 48 - "Food.Go"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 49 - "Profile.Go"
Cohesion: 0.25
Nodes (3): NewProfileCommand(), ProfileCommand, ProfileStore

### Community 50 - "Cancel.Go"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 51 - "Gotify"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 53 - "Hevy"
Cohesion: 0.52
Nodes (5): floatPtr(), intPtr(), TestToWorkout(), TestToWorkoutNilSafety(), ToWorkout()

### Community 54 - "Start.Go"
Cohesion: 0.29
Nodes (2): NewStartCommand(), StartCommand

### Community 55 - "Timezone.Go"
Cohesion: 0.29
Nodes (2): NewTimezoneCommand(), TimezoneCommand

### Community 56 - "Embedding"
Cohesion: 0.29
Nodes (1): stubStore

### Community 57 - "Ntfy"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 58 - "Chat.Go"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 59 - "Engine Test.Go"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 62 - "Assistant Test.Go"
Cohesion: 0.4
Nodes (1): fakeCommand

### Community 63 - "Help.Go"
Cohesion: 0.7
Nodes (1): HelpCommand

### Community 64 - "Whisper"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 68 - "Hevy"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 69 - "Queue.Go"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 71 - "Store.Go"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 72 - "Compose Profile"
Cohesion: 0.5
Nodes (4): AI Compose Profile, docker compose (quick start), .env.example, PostgreSQL Compose Profile

### Community 73 - "Community 73"
Cohesion: 0.67
Nodes (2): config, home_gsaraiva_projects_dietdaemon_web_vite_config

### Community 80 - "Notifier Test.Go"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 101 - "Handler Settings.Go"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 102 - "Store Nudges.Go"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 103 - "Store Provider Keys.Go"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 105 - "Stt.Md"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 106 - "Community 106"
Cohesion: 1.0
Nodes (2): DELETE /api/v1/account, GET /api/v1/export/all

### Community 146 - "Community 146"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 147 - "Community 147"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 148 - "Community 148"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 149 - "Community 149"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 150 - "Community 150"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 151 - "Community 151"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 152 - "Community 152"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 153 - "Community 153"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 154 - "Community 154"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 155 - "Community 155"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Ambiguous Edges - Review These
- `PARSER_TIER` → `ENABLE_STT`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to
- `DELETE /api/v1/account` → `Backup runner`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to

## Knowledge Gaps
- **388 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+383 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Handler Test.Go`** (86 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterDailyTotals()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Backup Test.Go .Getmealsinrange()`** (26 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Pipeline Test.Go`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Restore Test.Go`** (10 nodes): `fakeStore`, `.ImportWorkout()`, `.LogMeasurement()`, `.LogWeight()`, `.RestoreFast()`, `.RestorePhoto()`, `.RestoreSleep()`, `.RestoreWater()`, `.SaveMeal()`, `.UpsertRollup()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Pendingstore.Go`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Start.Go`** (7 nodes): `NewStartCommand()`, `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `start.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Timezone.Go`** (7 nodes): `NewTimezoneCommand()`, `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `timezone.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Embedding`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Engine Test.Go`** (6 nodes): `fakeStore`, `.FrequentFoods()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetRollup()`, `.GetTargets()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Assistant Test.Go`** (5 nodes): `fakeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Help.Go`** (5 nodes): `HelpCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Whisper`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Queue.Go`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (3 nodes): `config`, `home_gsaraiva_projects_dietdaemon_web_vite_config`, `vitest.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Notifier Test.Go`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Handler Settings.Go`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Store Nudges.Go`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Store Provider Keys.Go`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Stt.Md`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 106`** (2 nodes): `DELETE /api/v1/account`, `GET /api/v1/export/all`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 146`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 147`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 148`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 149`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 150`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 151`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 152`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 153`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 154`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 155`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `PARSER_TIER` and `ENABLE_STT`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `DELETE /api/v1/account` and `Backup runner`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `Assistant Test.Go` to `Handler.Go Vision.Go`, `Store Meals.Go`, `Handler Auth.Go`, `Handler Auth Test.Go`, `Handler Passkeys Test.Go`, `Correct Test.Go`, `Components Sourcelabel()`, `Purge Test.Go`, `Foodimport Test.Go`, `Mfp Main.Go`, `Usda`, `Handler Chat Test.Go`, `Index.Go`, `Suggestions Test.Go`?**
  _High betweenness centrality (0.278) - this node is a cross-community bridge._
- **Why does `run()` connect `Assistant Test.Go` to `Handler.Go Vision.Go`, `Handler Auth.Go`, `Handler Auth Test.Go`, `Correct Test.Go`, `Components Sourcelabel()`, `Purge Test.Go`, `Anthropic Chat.Go`, `Fast.Go`, `Types.Go`, `Foodimport Test.Go`, `Mfp Main.Go`, `Pipeline.Go`, `Scheduler.Go`, `Provider.Go`, `Handler Passkeys Test.Go`, `Status.Go`, `Target.Go`, `Food.Go`, `Profile.Go`, `Cancel.Go`, `Start.Go`, `Timezone.Go`?**
  _High betweenness centrality (0.105) - this node is a cross-community bridge._
- **Why does `Handler` connect `Handler Auth.Go` to `Assistant Test.Go`, `Handler.Go Vision.Go`, `Adherence Test.Go`, `Scheduler.Go`, `Session Test.Go`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Are the 273 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 273 INFERRED edges - model-reasoned connections that need verification._
- **Are the 4 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 4 INFERRED edges - model-reasoned connections that need verification._