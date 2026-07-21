# Graph Report - DietDaemon  (2026-07-21)

## Corpus Check
- 388 files · ~296,606 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3627 nodes · 6663 edges · 85 communities detected
- Extraction: 71% EXTRACTED · 29% INFERRED · 0% AMBIGUOUS · INFERRED: 1961 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 75|Community 75]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 83|Community 83]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 107|Community 107]]
- [[_COMMUNITY_Community 108|Community 108]]
- [[_COMMUNITY_Community 109|Community 109]]
- [[_COMMUNITY_Community 110|Community 110]]
- [[_COMMUNITY_Community 112|Community 112]]
- [[_COMMUNITY_Community 113|Community 113]]
- [[_COMMUNITY_Community 114|Community 114]]
- [[_COMMUNITY_Community 115|Community 115]]
- [[_COMMUNITY_Community 173|Community 173]]
- [[_COMMUNITY_Community 174|Community 174]]
- [[_COMMUNITY_Community 175|Community 175]]
- [[_COMMUNITY_Community 176|Community 176]]
- [[_COMMUNITY_Community 177|Community 177]]
- [[_COMMUNITY_Community 178|Community 178]]
- [[_COMMUNITY_Community 179|Community 179]]
- [[_COMMUNITY_Community 180|Community 180]]
- [[_COMMUNITY_Community 181|Community 181]]
- [[_COMMUNITY_Community 182|Community 182]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 280 edges
2. `Store` - 216 edges
3. `Handler` - 179 edges
4. `doRequest()` - 126 edges
5. `newHandler()` - 119 edges
6. `newFakeMealStore()` - 112 edges
7. `contains()` - 99 edges
8. `New()` - 90 edges
9. `fakeMealStore` - 87 edges
10. `run()` - 72 edges

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
- **Backup/Restore CSV round-trip pattern** — backup_backup_runner, backup_exportfmt, restore_cmd_restore, restore_idempotency [EXTRACTED 0.85]
- **Backup control plane (config, immediate trigger, scheduled loop)** — backup_settings_backup_api, backup_backup_runner, backup_run_now [EXTRACTED 0.80]

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (109): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthStore, BackupRunner, ChatStore, credCreateConfig (+101 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (43): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, fastRow (+35 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (224): TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), TestExpensiveRequestRoutes(), collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory() (+216 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (175): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), emailToken, erroringCountAuthStore, fakeMailer, fakeMealLogger, fakeSuggester, fakeVisionAdapter (+167 more)

### Community 4 - "Community 4"
Cohesion: 0.03
Nodes (120): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestExtractLabel(), TestExtractLabelHTTPError(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement() (+112 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (103): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, countRows(), newTestStore(), seedAllEntities(), seedUser(), TestRestoreCLI_DryRun() (+95 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (25): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, mfaEmailTestStore, fakeSessionRepo, Session, CreateSession(), RotateSession() (+17 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 8 - "Community 8"
Cohesion: 0.03
Nodes (68): Adapter, contentBlock, message, messagesRequest, messagesResponse, Destination, Runner, Store (+60 more)

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (54): Registry, renderModal(), dayLabel(), download(), sourceLabel(), onSubmit(), onAdd(), relativeCaption() (+46 more)

### Community 10 - "Community 10"
Cohesion: 0.03
Nodes (59): fakeChatAdapter, sendOut(), blockingChatAdapter, fakeChatAdapter, Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray() (+51 more)

### Community 11 - "Community 11"
Cohesion: 0.03
Nodes (89): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+81 more)

### Community 12 - "Community 12"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 13 - "Community 13"
Cohesion: 0.1
Nodes (72): totpChallengeAuthStore, IPRateLimiter, TestPendingStoreContract(), postgresDB(), TestFoodImportFingerprintStore(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods() (+64 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (32): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+24 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (49): macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore, adminTempStore(), TestFoodImportAdmin_ImportSource_MaxRowsCap(), TestFoodImportAdmin_ImportSource_TACO() (+41 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (51): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+43 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (35): CorrectCommand, CorrectResolver, CorrectStore, MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand() (+27 more)

### Community 18 - "Community 18"
Cohesion: 0.03
Nodes (63): FS(), NotFound(), APIKey, AuditEvent, BackupConfig, BodyCompositionSummary, CorrectionFeedback, CustomFoodInput (+55 more)

### Community 19 - "Community 19"
Cohesion: 0.04
Nodes (42): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+34 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (31): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+23 more)

### Community 21 - "Community 21"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 22 - "Community 22"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 23 - "Community 23"
Cohesion: 0.07
Nodes (37): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+29 more)

### Community 24 - "Community 24"
Cohesion: 0.09
Nodes (21): newPasskeyHandler(), newPasskeyTestStore(), TestHandlePasskeyLoginBeginCreatesDiscoverableCeremony(), TestHandlePasskeyLoginFinishRejectsMissingOrExpiredCeremony(), TestHandlePasskeyRegisterBeginCreatesCeremony(), WithWebAuthn(), passkeyTestStore, CredentialIDBytes() (+13 more)

### Community 25 - "Community 25"
Cohesion: 0.09
Nodes (25): Embedder, fakeEmbedder, fakeSource, fingerprintStore, localFingerprint(), New(), NewWithLocalPaths(), replaceDataset() (+17 more)

### Community 26 - "Community 26"
Cohesion: 0.1
Nodes (18): food, foodCategory, foodNutrient, searchResponse, Source, bulkDataTypes(), extractMacros(), foodToMatch() (+10 more)

### Community 27 - "Community 27"
Cohesion: 0.13
Nodes (19): entry, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB(), requireNoErr(), TestCacheUpdatesOnUpsertAndInvalidatesOnDelete(), TestCosineSimilarity() (+11 more)

### Community 28 - "Community 28"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 29 - "Community 29"
Cohesion: 0.1
Nodes (24): AWS default credential chain (backup), Backup runner, BACKUP_CHECK_INTERVAL, Database-level backup (pg_dump / sqlite3 .backup), internal/exportfmt (shared CSV writer), BACKUP_LOCAL_DIR, local_subdir path-traversal validation, Nudge scheduler (existing background loop) (+16 more)

### Community 30 - "Community 30"
Cohesion: 0.12
Nodes (12): fakeFoodSearcher, fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_IngredientArgsResolveViaSearch(), TestSuggestCommand_IngredientArgsSkipUnresolvedNames() (+4 more)

### Community 31 - "Community 31"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 32 - "Community 32"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 33 - "Community 33"
Cohesion: 0.14
Nodes (8): Adapter, embedRequest, embedResponse, generateRequest, generateResponse, uniqueModels(), pullRequest, tagsResponse

### Community 34 - "Community 34"
Cohesion: 0.26
Nodes (12): fakeFoodImportRunner, doAdminRequest(), newAdminTestHandler(), TestAdminFoodImport_BackfillEmbeddings200(), TestAdminFoodImport_MissingToken401(), TestAdminFoodImport_Repair200(), TestAdminFoodImport_RepairMissingSource400(), TestAdminFoodImport_Run200() (+4 more)

### Community 35 - "Community 35"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 36 - "Community 36"
Cohesion: 0.13
Nodes (1): fakeStore

### Community 37 - "Community 37"
Cohesion: 0.18
Nodes (7): fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore

### Community 38 - "Community 38"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 39 - "Community 39"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 40 - "Community 40"
Cohesion: 0.18
Nodes (1): allEntitiesFakeStore

### Community 41 - "Community 41"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 42 - "Community 42"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 43 - "Community 43"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 44 - "Community 44"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 46 - "Community 46"
Cohesion: 0.36
Nodes (1): Store

### Community 47 - "Community 47"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 48 - "Community 48"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 49 - "Community 49"
Cohesion: 0.25
Nodes (3): NewLinkCommand(), LinkCodeStore, LinkCommand

### Community 50 - "Community 50"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 52 - "Community 52"
Cohesion: 0.52
Nodes (5): floatPtr(), intPtr(), TestToWorkout(), TestToWorkoutNilSafety(), ToWorkout()

### Community 53 - "Community 53"
Cohesion: 0.29
Nodes (2): NewStartCommand(), StartCommand

### Community 54 - "Community 54"
Cohesion: 0.29
Nodes (2): NewTimezoneCommand(), TimezoneCommand

### Community 55 - "Community 55"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 56 - "Community 56"
Cohesion: 0.29
Nodes (1): stubStore

### Community 57 - "Community 57"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 58 - "Community 58"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 59 - "Community 59"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 65 - "Community 65"
Cohesion: 0.4
Nodes (1): fakeCommand

### Community 66 - "Community 66"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 67 - "Community 67"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 72 - "Community 72"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 73 - "Community 73"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 75 - "Community 75"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 76 - "Community 76"
Cohesion: 0.5
Nodes (4): AI Compose Profile, docker compose (quick start), .env.example, PostgreSQL Compose Profile

### Community 83 - "Community 83"
Cohesion: 0.67
Nodes (2): deleteAccountRequest, UserDataExport

### Community 84 - "Community 84"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

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
Nodes (1): visionRequest

### Community 113 - "Community 113"
Cohesion: 1.0
Nodes (1): VisionAdapter

### Community 114 - "Community 114"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 115 - "Community 115"
Cohesion: 1.0
Nodes (2): DELETE /api/v1/account, GET /api/v1/export/all

### Community 173 - "Community 173"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 174 - "Community 174"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 175 - "Community 175"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 176 - "Community 176"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 177 - "Community 177"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 178 - "Community 178"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 179 - "Community 179"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 180 - "Community 180"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 181 - "Community 181"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 182 - "Community 182"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Ambiguous Edges - Review These
- `PARSER_TIER` → `ENABLE_STT`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to
- `DELETE /api/v1/account` → `Backup runner`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to

## Knowledge Gaps
- **390 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+385 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 12`** (86 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterDailyTotals()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (15 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (11 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (7 nodes): `NewStartCommand()`, `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `start.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (7 nodes): `NewTimezoneCommand()`, `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `timezone.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (6 nodes): `fakeStore`, `.FrequentFoods()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetRollup()`, `.GetTargets()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (5 nodes): `fakeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (3 nodes): `deleteAccountRequest`, `UserDataExport`, `handler_account.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 108`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 109`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 110`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 112`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 113`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 114`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 115`** (2 nodes): `DELETE /api/v1/account`, `GET /api/v1/export/all`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 173`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 174`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 175`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 176`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 177`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 178`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 179`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 180`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 181`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 182`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `PARSER_TIER` and `ENABLE_STT`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `DELETE /api/v1/account` and `Backup runner`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `Community 2` to `Community 0`, `Community 1`, `Community 3`, `Community 4`, `Community 5`, `Community 8`, `Community 9`, `Community 10`, `Community 13`, `Community 15`, `Community 17`, `Community 19`, `Community 23`, `Community 24`, `Community 25`, `Community 26`, `Community 27`, `Community 30`?**
  _High betweenness centrality (0.316) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 2` to `Community 0`, `Community 3`, `Community 4`, `Community 5`, `Community 9`, `Community 14`, `Community 15`, `Community 17`, `Community 18`, `Community 23`, `Community 24`, `Community 25`, `Community 30`, `Community 35`, `Community 37`, `Community 43`, `Community 47`, `Community 48`, `Community 49`, `Community 53`, `Community 54`?**
  _High betweenness centrality (0.133) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 4` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 33`, `Community 5`, `Community 8`, `Community 10`, `Community 15`, `Community 16`, `Community 18`, `Community 19`, `Community 20`, `Community 23`, `Community 24`, `Community 25`, `Community 30`?**
  _High betweenness centrality (0.084) - this node is a cross-community bridge._
- **Are the 275 inferred relationships involving `New()` (e.g. with `adminTempStore()` and `.BackfillEmbeddings()`) actually correct?**
  _`New()` has 275 INFERRED edges - model-reasoned connections that need verification._
- **Are the 4 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 4 INFERRED edges - model-reasoned connections that need verification._