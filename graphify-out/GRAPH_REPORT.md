# Graph Report - DietDaemon  (2026-07-21)

## Corpus Check
- 394 files · ~300,458 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3892 nodes · 9274 edges · 70 communities detected
- Extraction: 76% EXTRACTED · 24% INFERRED · 0% AMBIGUOUS · INFERRED: 2209 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 280 edges
2. `Store` - 221 edges
3. `New()` - 211 edges
4. `Handler` - 181 edges
5. `doRequest()` - 127 edges
6. `newHandler()` - 121 edges
7. `newFakeMealStore()` - 113 edges
8. `contains()` - 100 edges
9. `New()` - 90 edges
10. `fakeMealStore` - 89 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `NewCancelCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/cancel.go
- `run()` --calls--> `WithWeeklyBudgetRules()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/scheduler/scheduler.go
- `run()` --calls--> `LocalPaths()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/foodimport/sources.go
- `run()` --calls--> `WithOIDC()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/api/handler.go
- `run()` --calls--> `WithBackupRunner()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/api/handler.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (304): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter, chatContentBlock (+296 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (341): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), TestAuthenticatedRateLimitCategories(), collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory() (+333 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (161): animatednumber, api, appshell, assistant_stream, auth, authcallback, authlayout, browser (+153 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (79): customFoodRequest, ErrorCode, errorEnvelope, errorEnvelopeWriter, errorForStatus(), publicErrorMessage(), TestAPIErrorEnvelope(), TestAPIErrorEnvelopePreservesStreaming() (+71 more)

### Community 4 - "Community 4"
Cohesion: 0.01
Nodes (38): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, fastRow, foodDetailRow, foodMatchRow (+30 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (197): accountRepos, assertBYOKFailure(), TestBYOKFailuresDoNotFallBackToSharedAdapters(), TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), emailToken, erroringCountAuthStore, fakeFoodImportRunner, fakeMailer (+189 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (132): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestStreamChatHTTPError(), TestExtractLabel(), TestExtractLabelHTTPError(), ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd() (+124 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (29): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, readSessionCookie(), bearerToken(), isMutating(), TestBearerTokenEdgeCases(), mfaEmailTestStore (+21 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (49): home_gsaraiva_projects_dietdaemon_web_src_lib_demodata, ApiError, blobRequest(), handleUnauthorized(), multipart(), RateLimitError, readCookie(), request() (+41 more)

### Community 9 - "Community 9"
Cohesion: 0.06
Nodes (97): totpChallengeAuthStore, PurgeRunner, fakeLoginAttemptRepo, ipBucket, IPRateLimiter, CheckLockout(), NewIPRateLimiter(), TestCheckLockoutLocked() (+89 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (60): CorrectCommand, formatDurationShort(), FastCommand, ProfileCommand, parseTargetArgs(), TargetCommand, WeightCommand, NumberField() (+52 more)

### Community 11 - "Community 11"
Cohesion: 0.03
Nodes (60): passkeyTestStore, cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness() (+52 more)

### Community 12 - "Community 12"
Cohesion: 0.04
Nodes (56): Adapter, Runner, csvEscape(), PhotoFilename(), parseFloat(), readAll(), ReadFastsCSV(), ReadMealsCSV() (+48 more)

### Community 13 - "Community 13"
Cohesion: 0.04
Nodes (71): adminTempStore(), TestFoodImportAdmin_ImportSource_MaxRowsCap(), TestFoodImportAdmin_ImportSource_TACO(), TestFoodImportAdmin_ImportSource_UnknownSource(), TestFoodImportAdmin_RepairSource(), go_pkg_encoding_csv, go_pkg_flag, go_pkg_github_com_gsaraiva2109_dietdaemon_adapters_model_ollama (+63 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (89): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+81 more)

### Community 15 - "Community 15"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 16 - "Community 16"
Cohesion: 0.03
Nodes (32): fakeChatAdapter, fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming() (+24 more)

### Community 17 - "Community 17"
Cohesion: 0.03
Nodes (66): go_pkg_embed, go_pkg_io_fs, go_pkg_path, FS(), NotFound(), APIKey, AuditEvent, BackupConfig (+58 more)

### Community 18 - "Community 18"
Cohesion: 0.05
Nodes (45): go_pkg_sort, ModelOverrideFromContext(), ChatRouteStore, ChatSender, DigestStore, HealthStore, MealHistoryStore, Notifier (+37 more)

### Community 19 - "Community 19"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 20 - "Community 20"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 21 - "Community 21"
Cohesion: 0.08
Nodes (16): calcSleepHours(), computeSleepDuration(), formatDuration(), SleepCommand, sourceLabel(), MacroBar(), Matcher, confidenceTier() (+8 more)

### Community 22 - "Community 22"
Cohesion: 0.07
Nodes (2): allEntitiesFakeStore, fakeStore

### Community 23 - "Community 23"
Cohesion: 0.1
Nodes (24): AWS default credential chain (backup), Backup runner, BACKUP_CHECK_INTERVAL, Database-level backup (pg_dump / sqlite3 .backup), internal/exportfmt (shared CSV writer), BACKUP_LOCAL_DIR, local_subdir path-traversal validation, Nudge scheduler (existing background loop) (+16 more)

### Community 24 - "Community 24"
Cohesion: 0.14
Nodes (13): node_crypto, node_http, isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies() (+5 more)

### Community 25 - "Community 25"
Cohesion: 0.15
Nodes (11): go_pkg_encoding_binary, entry, cosineSimilarity(), packF32LE(), sortByScore(), TestCosineSimilarity(), TestPackUnpackF32LE(), TestUnpackBadBlob() (+3 more)

### Community 26 - "Community 26"
Cohesion: 0.16
Nodes (11): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+3 more)

### Community 27 - "Community 27"
Cohesion: 0.22
Nodes (8): Source, bulkDataTypes(), extractMacros(), foodToMatch(), portionsToServingUnits(), TestFoodCategoryUnmarshalObjectShape(), TestFoodCategoryUnmarshalStringShape(), TestFoodPortionsToServingUnits()

### Community 28 - "Community 28"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 29 - "Community 29"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 30 - "Community 30"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 31 - "Community 31"
Cohesion: 0.22
Nodes (7): config, eslint_plugin_react_hooks, eslint_plugin_react_refresh, globals, home_gsaraiva_projects_dietdaemon_web_vite_config, js, typescript_eslint

### Community 32 - "Community 32"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 33 - "Community 33"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 34 - "Community 34"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 35 - "Community 35"
Cohesion: 0.29
Nodes (1): stubStore

### Community 36 - "Community 36"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 37 - "Community 37"
Cohesion: 0.53
Nodes (1): Store

### Community 38 - "Community 38"
Cohesion: 0.6
Nodes (1): Provider

### Community 39 - "Community 39"
Cohesion: 0.4
Nodes (1): SuggestCommand

### Community 40 - "Community 40"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 41 - "Community 41"
Cohesion: 0.33
Nodes (1): fakeHealthStore

### Community 42 - "Community 42"
Cohesion: 0.4
Nodes (3): node_url, plugin_react, vite

### Community 43 - "Community 43"
Cohesion: 0.4
Nodes (1): fakeCommand

### Community 44 - "Community 44"
Cohesion: 0.5
Nodes (3): BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes()

### Community 45 - "Community 45"
Cohesion: 0.4
Nodes (1): FoodCommand

### Community 46 - "Community 46"
Cohesion: 0.4
Nodes (1): NudgeCommand

### Community 47 - "Community 47"
Cohesion: 0.4
Nodes (1): StartCommand

### Community 48 - "Community 48"
Cohesion: 0.4
Nodes (1): TimezoneCommand

### Community 49 - "Community 49"
Cohesion: 0.7
Nodes (1): HelpCommand

### Community 50 - "Community 50"
Cohesion: 0.4
Nodes (1): fakeDigestStore

### Community 51 - "Community 51"
Cohesion: 0.4
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 52 - "Community 52"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 53 - "Community 53"
Cohesion: 0.5
Nodes (1): fakePurgeStore

### Community 54 - "Community 54"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 55 - "Community 55"
Cohesion: 0.5
Nodes (4): AI Compose Profile, docker compose (quick start), .env.example, PostgreSQL Compose Profile

### Community 56 - "Community 56"
Cohesion: 1.0
Nodes (1): react_markdown

### Community 57 - "Community 57"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 58 - "Community 58"
Cohesion: 1.0
Nodes (2): DELETE /api/v1/account, GET /api/v1/export/all

### Community 61 - "Community 61"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 62 - "Community 62"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 63 - "Community 63"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 64 - "Community 64"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 65 - "Community 65"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 66 - "Community 66"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 67 - "Community 67"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 68 - "Community 68"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 69 - "Community 69"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 70 - "Community 70"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

### Community 71 - "Community 71"
Cohesion: 1.0
Nodes (1): home_gsaraiva_projects_dietdaemon_web_src_app_tsx

## Ambiguous Edges - Review These
- `PARSER_TIER` → `ENABLE_STT`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to
- `DELETE /api/v1/account` → `Backup runner`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to

## Knowledge Gaps
- **392 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+387 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 15`** (88 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateFoodServingUnit()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteFoodServingUnit()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterDailyTotals()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 22`** (26 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 29`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (6 nodes): `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (6 nodes): `Provider`, `.AuthCodeURL()`, `.ensure()`, `.Exchange()`, `.UserInfo()`, `.VerifyIDToken()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (6 nodes): `SuggestCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `.resolveIngredients()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (6 nodes): `fakeStore`, `.FrequentFoods()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetRollup()`, `.GetTargets()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (6 nodes): `fakeHealthStore`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetWaterToday()`, `.ListFasts()`, `.ListWorkouts()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (5 nodes): `fakeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (5 nodes): `FoodCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (5 nodes): `NudgeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (5 nodes): `StartCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (5 nodes): `TimezoneCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (5 nodes): `HelpCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (5 nodes): `fakeDigestStore`, `.GetRollups()`, `.GetWaterDailyTotals()`, `.ListWeight()`, `.ListWorkoutsInRange()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (4 nodes): `fakePurgeStore`, `.PurgeAuthAuditEvents()`, `.PurgeDeletedChatSessions()`, `.PurgeLoginAttempts()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (2 nodes): `react_markdown`, `MarkdownText.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (2 nodes): `DELETE /api/v1/account`, `GET /api/v1/export/all`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 61`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 62`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 64`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 67`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 68`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (1 nodes): `home_gsaraiva_projects_dietdaemon_web_src_app_tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `PARSER_TIER` and `ENABLE_STT`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `DELETE /api/v1/account` and `Backup runner`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `Community 1` to `Community 0`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 9`, `Community 10`, `Community 12`, `Community 13`, `Community 16`?**
  _High betweenness centrality (0.089) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 3` to `Community 0`, `Community 1`, `Community 6`, `Community 7`, `Community 12`, `Community 18`, `Community 26`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Why does `Store` connect `Community 4` to `Community 0`, `Community 1`, `Community 3`, `Community 11`, `Community 16`?**
  _High betweenness centrality (0.056) - this node is a cross-community bridge._
- **Are the 275 inferred relationships involving `New()` (e.g. with `adminTempStore()` and `.BackfillEmbeddings()`) actually correct?**
  _`New()` has 275 INFERRED edges - model-reasoned connections that need verification._
- **Are the 209 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 209 INFERRED edges - model-reasoned connections that need verification._