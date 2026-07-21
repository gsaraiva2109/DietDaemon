# Graph Report - DietDaemon  (2026-07-21)

## Corpus Check
- 382 files · ~293,167 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3580 nodes · 6543 edges · 81 communities detected
- Extraction: 70% EXTRACTED · 30% INFERRED · 0% AMBIGUOUS · INFERRED: 1935 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 80|Community 80]]
- [[_COMMUNITY_Community 81|Community 81]]
- [[_COMMUNITY_Community 104|Community 104]]
- [[_COMMUNITY_Community 105|Community 105]]
- [[_COMMUNITY_Community 106|Community 106]]
- [[_COMMUNITY_Community 108|Community 108]]
- [[_COMMUNITY_Community 109|Community 109]]
- [[_COMMUNITY_Community 110|Community 110]]
- [[_COMMUNITY_Community 111|Community 111]]
- [[_COMMUNITY_Community 169|Community 169]]
- [[_COMMUNITY_Community 170|Community 170]]
- [[_COMMUNITY_Community 171|Community 171]]
- [[_COMMUNITY_Community 172|Community 172]]
- [[_COMMUNITY_Community 173|Community 173]]
- [[_COMMUNITY_Community 174|Community 174]]
- [[_COMMUNITY_Community 175|Community 175]]
- [[_COMMUNITY_Community 176|Community 176]]
- [[_COMMUNITY_Community 177|Community 177]]
- [[_COMMUNITY_Community 178|Community 178]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 278 edges
2. `Store` - 216 edges
3. `Handler` - 175 edges
4. `doRequest()` - 121 edges
5. `newHandler()` - 111 edges
6. `newFakeMealStore()` - 107 edges
7. `contains()` - 98 edges
8. `New()` - 90 edges
9. `fakeMealStore` - 87 edges
10. `run()` - 71 edges

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
Cohesion: 0.02
Nodes (234): TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory(), TestRouterTextOnly() (+226 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (43): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, fastRow (+35 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (70): credCreateConfig, credRevokeConfig, customFoodRequest, Handler, hostOnly(), isSixDigit(), writeJSONList(), deriveSessionTitle() (+62 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (170): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), emailToken, erroringCountAuthStore, fakeMailer, fakeMealLogger, fakeSuggester, fakeVisionAdapter (+162 more)

### Community 4 - "Community 4"
Cohesion: 0.03
Nodes (112): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestExtractLabel(), TestExtractLabelHTTPError(), ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray() (+104 more)

### Community 5 - "Community 5"
Cohesion: 0.02
Nodes (66): confirmReplace(), scaledMacros(), sourceLabel(), renderModal(), dayLabel(), download(), sourceLabel(), MacroBar() (+58 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (97): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, blockingStore, ChatRouteStore, ChatSender, DigestRule, DigestStore (+89 more)

### Community 7 - "Community 7"
Cohesion: 0.03
Nodes (74): Adapter, contentBlock, message, messagesRequest, messagesResponse, Destination, Runner, Store (+66 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (7): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, mfaEmailTestStore, fakeSessionRepo, Store, fakePending

### Community 10 - "Community 10"
Cohesion: 0.07
Nodes (84): totpChallengeAuthStore, fakeLoginAttemptRepo, ipBucket, IPRateLimiter, CheckLockout(), NewIPRateLimiter(), TestCheckLockoutLocked(), TestCheckLockoutNotLocked() (+76 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (67): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthStore, BackupRunner, ChatStore, EmailTokenStore (+59 more)

### Community 12 - "Community 12"
Cohesion: 0.03
Nodes (89): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+81 more)

### Community 13 - "Community 13"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 14 - "Community 14"
Cohesion: 0.04
Nodes (42): MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand(), parseTargetArgs(), TargetCommand, close() (+34 more)

### Community 15 - "Community 15"
Cohesion: 0.03
Nodes (47): fakeChatAdapter, sendOut(), blockingChatAdapter, fakeChatAdapter, Router, Adapter, joinedRoom, callbackDataByIndex() (+39 more)

### Community 16 - "Community 16"
Cohesion: 0.03
Nodes (38): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+30 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (51): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+43 more)

### Community 18 - "Community 18"
Cohesion: 0.03
Nodes (63): FS(), NotFound(), APIKey, AuditEvent, BackupConfig, BodyCompositionSummary, CorrectionFeedback, CustomFoodInput (+55 more)

### Community 19 - "Community 19"
Cohesion: 0.04
Nodes (42): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+34 more)

### Community 20 - "Community 20"
Cohesion: 0.05
Nodes (40): groupIntoMeals(), importMeals(), main(), mealTime(), newImportID(), run(), tempStore(), TestGroupIntoMeals() (+32 more)

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
Nodes (25): macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore, entry, Index, cosineSimilarity() (+17 more)

### Community 25 - "Community 25"
Cohesion: 0.09
Nodes (25): Embedder, fakeEmbedder, fakeSource, fingerprintStore, localFingerprint(), New(), NewWithLocalPaths(), replaceDataset() (+17 more)

### Community 26 - "Community 26"
Cohesion: 0.1
Nodes (18): food, foodCategory, foodNutrient, searchResponse, Source, bulkDataTypes(), extractMacros(), foodToMatch() (+10 more)

### Community 27 - "Community 27"
Cohesion: 0.13
Nodes (15): fakeCmd, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape(), TestHelpCommand_ListAll() (+7 more)

### Community 28 - "Community 28"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 29 - "Community 29"
Cohesion: 0.1
Nodes (24): AWS default credential chain (backup), Backup runner, BACKUP_CHECK_INTERVAL, Database-level backup (pg_dump / sqlite3 .backup), internal/exportfmt (shared CSV writer), BACKUP_LOCAL_DIR, local_subdir path-traversal validation, Nudge scheduler (existing background loop) (+16 more)

### Community 30 - "Community 30"
Cohesion: 0.11
Nodes (10): NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal(), CorrectCommand, CorrectResolver, CorrectStore (+2 more)

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
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 35 - "Community 35"
Cohesion: 0.13
Nodes (1): fakeStore

### Community 36 - "Community 36"
Cohesion: 0.18
Nodes (7): fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore

### Community 37 - "Community 37"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 38 - "Community 38"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 39 - "Community 39"
Cohesion: 0.18
Nodes (1): allEntitiesFakeStore

### Community 40 - "Community 40"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 41 - "Community 41"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 42 - "Community 42"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 44 - "Community 44"
Cohesion: 0.36
Nodes (1): Store

### Community 45 - "Community 45"
Cohesion: 0.25
Nodes (3): NewFoodCommand(), FoodCommand, FoodStore

### Community 46 - "Community 46"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 47 - "Community 47"
Cohesion: 0.25
Nodes (3): NewLinkCommand(), LinkCodeStore, LinkCommand

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 50 - "Community 50"
Cohesion: 0.52
Nodes (5): floatPtr(), intPtr(), TestToWorkout(), TestToWorkoutNilSafety(), ToWorkout()

### Community 51 - "Community 51"
Cohesion: 0.29
Nodes (2): NewStartCommand(), StartCommand

### Community 52 - "Community 52"
Cohesion: 0.29
Nodes (2): NewTimezoneCommand(), TimezoneCommand

### Community 53 - "Community 53"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 54 - "Community 54"
Cohesion: 0.29
Nodes (1): stubStore

### Community 55 - "Community 55"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 56 - "Community 56"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 62 - "Community 62"
Cohesion: 0.4
Nodes (1): fakeCommand

### Community 63 - "Community 63"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 64 - "Community 64"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 69 - "Community 69"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 70 - "Community 70"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 72 - "Community 72"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 73 - "Community 73"
Cohesion: 0.5
Nodes (4): AI Compose Profile, docker compose (quick start), .env.example, PostgreSQL Compose Profile

### Community 80 - "Community 80"
Cohesion: 0.67
Nodes (2): deleteAccountRequest, UserDataExport

### Community 81 - "Community 81"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 104 - "Community 104"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 106 - "Community 106"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 108 - "Community 108"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 109 - "Community 109"
Cohesion: 1.0
Nodes (1): VisionAdapter

### Community 110 - "Community 110"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 111 - "Community 111"
Cohesion: 1.0
Nodes (2): DELETE /api/v1/account, GET /api/v1/export/all

### Community 169 - "Community 169"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 170 - "Community 170"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 171 - "Community 171"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 172 - "Community 172"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 173 - "Community 173"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 174 - "Community 174"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 175 - "Community 175"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 176 - "Community 176"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 177 - "Community 177"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 178 - "Community 178"
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
- **Thin community `Community 13`** (86 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterDailyTotals()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (15 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (11 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (7 nodes): `NewStartCommand()`, `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `start.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (7 nodes): `NewTimezoneCommand()`, `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `timezone.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 62`** (5 nodes): `fakeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 80`** (3 nodes): `deleteAccountRequest`, `UserDataExport`, `handler_account.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 81`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 104`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 106`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 108`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 109`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 110`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 111`** (2 nodes): `DELETE /api/v1/account`, `GET /api/v1/export/all`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 169`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 170`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 171`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 172`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 173`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 174`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 175`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 176`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 177`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 178`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `PARSER_TIER` and `ENABLE_STT`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `DELETE /api/v1/account` and `Backup runner`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `Community 0` to `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 10`, `Community 11`, `Community 14`, `Community 19`, `Community 20`, `Community 23`, `Community 24`, `Community 25`, `Community 26`?**
  _High betweenness centrality (0.296) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 0` to `Community 2`, `Community 3`, `Community 4`, `Community 6`, `Community 11`, `Community 14`, `Community 16`, `Community 18`, `Community 20`, `Community 23`, `Community 25`, `Community 27`, `Community 30`, `Community 34`, `Community 36`, `Community 41`, `Community 45`, `Community 46`, `Community 47`, `Community 51`, `Community 52`?**
  _High betweenness centrality (0.109) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 2` to `Community 0`, `Community 4`, `Community 6`, `Community 7`, `Community 11`, `Community 19`, `Community 31`?**
  _High betweenness centrality (0.091) - this node is a cross-community bridge._
- **Are the 273 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 273 INFERRED edges - model-reasoned connections that need verification._
- **Are the 4 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 4 INFERRED edges - model-reasoned connections that need verification._