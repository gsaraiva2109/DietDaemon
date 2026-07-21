# Graph Report - .  (2026-07-21)

## Corpus Check
- 312 files · ~99,999 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3828 nodes · 9113 edges · 66 communities detected
- Extraction: 76% EXTRACTED · 24% INFERRED · 0% AMBIGUOUS · INFERRED: 2169 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 278 edges
2. `Store` - 216 edges
3. `New()` - 211 edges
4. `Handler` - 175 edges
5. `doRequest()` - 126 edges
6. `newHandler()` - 117 edges
7. `newFakeMealStore()` - 112 edges
8. `contains()` - 99 edges
9. `New()` - 90 edges
10. `fakeMealStore` - 87 edges

## Surprising Connections (you probably didn't know these)
- `NumberField()` --calls--> `parseFloat()`  [INFERRED]
  web/src/components/OnboardingWizard.tsx → adapters/nutrition/taco/taco.go
- `run()` --calls--> `NewCancelCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/cancel.go
- `run()` --calls--> `NewSleepCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/sleep.go
- `run()` --calls--> `WithWeeklyBudgetRules()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/scheduler/scheduler.go
- `run()` --calls--> `LocalPaths()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/foodimport/sources.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (327): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestToWireMessagesToolRoundTrip() (+319 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (328): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory(), TestRouterTextOnly() (+320 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (157): animatednumber, api, appshell, auth, authcallback, authlayout, chatruntime, chatthreadlistadapter (+149 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (81): ErrorCode, errorEnvelope, errorEnvelopeWriter, errorForStatus(), publicErrorMessage(), TestAPIErrorEnvelope(), TestAPIErrorEnvelopePreservesStreaming(), TestAPIRouteFallbackUsesErrorEnvelope() (+73 more)

### Community 4 - "Community 4"
Cohesion: 0.01
Nodes (38): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, fastRow, foodDetailRow, foodMatchRow (+30 more)

### Community 5 - "Community 5"
Cohesion: 0.04
Nodes (178): accountRepos, assertBYOKFailure(), TestBYOKFailuresDoNotFallBackToSharedAdapters(), TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), emailToken, erroringCountAuthStore, fakeMailer, fakeMealLogger (+170 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (137): TestStreamChatHTTPError(), TestExtractLabel(), TestExtractLabelHTTPError(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal() (+129 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (61): assistant_stream, browser, onAdd(), home_gsaraiva_projects_dietdaemon_web_src_lib_api, home_gsaraiva_projects_dietdaemon_web_src_lib_demodata, home_gsaraiva_projects_dietdaemon_web_src_lib_types, ApiError, blobRequest() (+53 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (29): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, readSessionCookie(), bearerToken(), isMutating(), TestBearerTokenEdgeCases(), mfaEmailTestStore (+21 more)

### Community 9 - "Community 9"
Cohesion: 0.06
Nodes (84): totpChallengeAuthStore, PurgeRunner, IPRateLimiter, LinkCommand, pct(), StatusCommand, macrosSum(), TemplateCommand (+76 more)

### Community 10 - "Community 10"
Cohesion: 0.04
Nodes (57): Adapter, Runner, csvEscape(), PhotoFilename(), parseFloat(), readAll(), ReadFastsCSV(), ReadMealsCSV() (+49 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (48): randomID(), WaterCommand, WorkoutCommand, cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer() (+40 more)

### Community 12 - "Community 12"
Cohesion: 0.03
Nodes (46): CorrectCommand, formatDurationShort(), FastCommand, ProfileCommand, parseTargetArgs(), TargetCommand, WeightCommand, go_pkg_encoding_csv (+38 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (89): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+81 more)

### Community 14 - "Community 14"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 15 - "Community 15"
Cohesion: 0.03
Nodes (66): go_pkg_embed, go_pkg_io_fs, go_pkg_path, FS(), NotFound(), APIKey, AuditEvent, BackupConfig (+58 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (30): fakeChatAdapter, fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming() (+22 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (50): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), go_pkg_sort (+42 more)

### Community 18 - "Community 18"
Cohesion: 0.04
Nodes (41): fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), fakeLoginAttemptRepo, ipBucket, CheckLockout() (+33 more)

### Community 19 - "Community 19"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 20 - "Community 20"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 21 - "Community 21"
Cohesion: 0.07
Nodes (34): go_pkg_github_com_gsaraiva2109_dietdaemon_internal_backup, countRows(), newTestStore(), seedAllEntities(), seedUser(), TestRestoreCLI_DryRun(), TestRestoreCLI_RoundTrip(), ChatRouteStore (+26 more)

### Community 22 - "Community 22"
Cohesion: 0.08
Nodes (18): calcSleepHours(), computeSleepDuration(), formatDuration(), NewSleepCommand(), SleepCommand, SleepStore, sourceLabel(), MacroBar() (+10 more)

### Community 23 - "Community 23"
Cohesion: 0.07
Nodes (2): allEntitiesFakeStore, fakeStore

### Community 24 - "Community 24"
Cohesion: 0.13
Nodes (12): Source, bulkDataTypes(), extractMacros(), foodToMatch(), NewBulk(), TestFetchBulkFile(), TestFetchBulkFileEmitError(), TestFetchBulkFileMaxRows() (+4 more)

### Community 25 - "Community 25"
Cohesion: 0.1
Nodes (24): AWS default credential chain (backup), Backup runner, BACKUP_CHECK_INTERVAL, Database-level backup (pg_dump / sqlite3 .backup), internal/exportfmt (shared CSV writer), BACKUP_LOCAL_DIR, local_subdir path-traversal validation, Nudge scheduler (existing background loop) (+16 more)

### Community 26 - "Community 26"
Cohesion: 0.14
Nodes (13): node_crypto, node_http, isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies() (+5 more)

### Community 27 - "Community 27"
Cohesion: 0.16
Nodes (11): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+3 more)

### Community 28 - "Community 28"
Cohesion: 0.12
Nodes (5): go_pkg_github_com_lib_pq, Dialect, ErrUnsupportedDriver, postgresDialect, sqliteDialect

### Community 29 - "Community 29"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 30 - "Community 30"
Cohesion: 0.29
Nodes (7): newPasskeyHandler(), newPasskeyTestStore(), TestHandlePasskeyLoginBeginCreatesDiscoverableCeremony(), TestHandlePasskeyLoginFinishRejectsMissingOrExpiredCeremony(), TestHandlePasskeyRegisterBeginCreatesCeremony(), WithWebAuthn(), passkeyTestStore

### Community 31 - "Community 31"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 32 - "Community 32"
Cohesion: 0.22
Nodes (7): config, eslint_plugin_react_hooks, eslint_plugin_react_refresh, globals, home_gsaraiva_projects_dietdaemon_web_vite_config, js, typescript_eslint

### Community 33 - "Community 33"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 34 - "Community 34"
Cohesion: 0.25
Nodes (3): NewCancelCommand(), CancelCommand, PendingStore

### Community 35 - "Community 35"
Cohesion: 0.21
Nodes (2): blockingStore, fakeStore

### Community 36 - "Community 36"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 37 - "Community 37"
Cohesion: 0.29
Nodes (1): stubStore

### Community 38 - "Community 38"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 39 - "Community 39"
Cohesion: 0.53
Nodes (1): Store

### Community 40 - "Community 40"
Cohesion: 0.4
Nodes (1): SuggestCommand

### Community 41 - "Community 41"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 42 - "Community 42"
Cohesion: 0.33
Nodes (1): fakeHealthStore

### Community 43 - "Community 43"
Cohesion: 0.4
Nodes (3): node_url, plugin_react, vite

### Community 44 - "Community 44"
Cohesion: 0.4
Nodes (1): fakeCommand

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
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 52 - "Community 52"
Cohesion: 0.5
Nodes (4): AI Compose Profile, docker compose (quick start), .env.example, PostgreSQL Compose Profile

### Community 53 - "Community 53"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 54 - "Community 54"
Cohesion: 1.0
Nodes (2): DELETE /api/v1/account, GET /api/v1/export/all

### Community 55 - "Community 55"
Cohesion: 1.0
Nodes (1): react_markdown

### Community 58 - "Community 58"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 59 - "Community 59"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 60 - "Community 60"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 61 - "Community 61"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 62 - "Community 62"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 63 - "Community 63"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 64 - "Community 64"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 65 - "Community 65"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 66 - "Community 66"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 67 - "Community 67"
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
- **Thin community `Community 14`** (86 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterDailyTotals()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 23`** (26 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (8 nodes): `blockingStore`, `.GetRollup()`, `.GetTargets()`, `.ListUsers()`, `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.ListUsers()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (6 nodes): `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (6 nodes): `SuggestCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`, `.resolveIngredients()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (6 nodes): `fakeStore`, `.FrequentFoods()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetRollup()`, `.GetTargets()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (6 nodes): `fakeHealthStore`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetWaterToday()`, `.ListFasts()`, `.ListWorkouts()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (5 nodes): `fakeCommand`, `.Aliases()`, `.Handle()`, `.Help()`, `.Name()`
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
- **Thin community `Community 53`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (2 nodes): `DELETE /api/v1/account`, `GET /api/v1/export/all`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (2 nodes): `react_markdown`, `MarkdownText.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 61`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 62`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 64`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 67`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `PARSER_TIER` and `ENABLE_STT`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `DELETE /api/v1/account` and `Backup runner`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 9`, `Community 10`, `Community 12`, `Community 16`, `Community 21`, `Community 24`, `Community 30`?**
  _High betweenness centrality (0.089) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 3` to `Community 0`, `Community 1`, `Community 6`, `Community 8`, `Community 10`, `Community 18`, `Community 21`, `Community 27`?**
  _High betweenness centrality (0.077) - this node is a cross-community bridge._
- **Why does `Store` connect `Community 4` to `Community 0`, `Community 1`, `Community 3`, `Community 16`?**
  _High betweenness centrality (0.067) - this node is a cross-community bridge._
- **Are the 273 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 273 INFERRED edges - model-reasoned connections that need verification._
- **Are the 209 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 209 INFERRED edges - model-reasoned connections that need verification._