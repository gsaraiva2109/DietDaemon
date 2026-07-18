# Graph Report - DietDaemon  (2026-07-18)

## Corpus Check
- 371 files · ~285,644 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3472 nodes · 6158 edges · 73 communities detected
- Extraction: 72% EXTRACTED · 28% INFERRED · 0% AMBIGUOUS · INFERRED: 1723 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 96|Community 96]]
- [[_COMMUNITY_Community 97|Community 97]]
- [[_COMMUNITY_Community 98|Community 98]]
- [[_COMMUNITY_Community 100|Community 100]]
- [[_COMMUNITY_Community 101|Community 101]]
- [[_COMMUNITY_Community 102|Community 102]]
- [[_COMMUNITY_Community 103|Community 103]]
- [[_COMMUNITY_Community 160|Community 160]]
- [[_COMMUNITY_Community 161|Community 161]]
- [[_COMMUNITY_Community 162|Community 162]]
- [[_COMMUNITY_Community 163|Community 163]]
- [[_COMMUNITY_Community 164|Community 164]]
- [[_COMMUNITY_Community 165|Community 165]]
- [[_COMMUNITY_Community 166|Community 166]]
- [[_COMMUNITY_Community 167|Community 167]]
- [[_COMMUNITY_Community 168|Community 168]]
- [[_COMMUNITY_Community 169|Community 169]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 270 edges
2. `Store` - 216 edges
3. `Handler` - 171 edges
4. `doRequest()` - 114 edges
5. `newHandler()` - 105 edges
6. `newFakeMealStore()` - 101 edges
7. `New()` - 90 edges
8. `contains()` - 89 edges
9. `fakeMealStore` - 87 edges
10. `run()` - 69 edges

## Surprising Connections (you probably didn't know these)
- `NumberField()` --calls--> `parseFloat()`  [INFERRED]
  web/src/components/OnboardingWizard.tsx → adapters/nutrition/taco/taco.go
- `run()` --calls--> `NewStatusCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/status.go
- `run()` --calls--> `NewWeightCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/weight.go
- `run()` --calls--> `NewWaterCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/water.go
- `run()` --calls--> `NewWorkoutCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/workout.go

## Hyperedges (group relationships)
- **Backup/Restore CSV round-trip pattern** — backup_backup_runner, backup_exportfmt, restore_cmd_restore, restore_idempotency [EXTRACTED 0.85]
- **Backup control plane (config, immediate trigger, scheduled loop)** — backup_settings_backup_api, backup_backup_runner, backup_run_now [EXTRACTED 0.80]

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (45): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, fastRow (+37 more)

### Community 1 - "Community 1"
Cohesion: 0.02
Nodes (185): collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory(), TestRouterTextOnly(), TestRouterTextOnly_doneForwarded(), TestRouterToolCallMaxRounds() (+177 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (51): credCreateConfig, credRevokeConfig, customFoodRequest, Handler, hostOnly(), isSixDigit(), readSessionCookie(), writeJSONList() (+43 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (112): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestExtractLabel(), TestExtractLabelHTTPError(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement() (+104 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (96): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthStore, BackupRunner, ChatStore, EmailTokenStore (+88 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (78): allEntitiesFakeStore, newFakeStore(), TestRunFor_ExportsAllEntities(), TestRunFor_MissingDestinationErrors(), TestRunFor_SetsBackupCounts(), TestRunFor_WarnsOnCountDrop(), TestRunOnce_IgnoresIntervalGate(), TestTick_RunsWhenIntervalElapsed() (+70 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (97): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, blockingStore, ChatRouteStore, ChatSender, DigestRule, DigestStore (+89 more)

### Community 7 - "Community 7"
Cohesion: 0.05
Nodes (121): accountRepos, fakeMealLogger, fakeSuggester, fakeVisionAdapter, newHandlerWithAccountStore(), TestHandleDeleteAccountMissingBody(), TestHandleDeleteAccountNotFound(), TestHandleDeleteAccountSuccess() (+113 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (54): Registry, renderModal(), dayLabel(), download(), sourceLabel(), onSubmit(), onAdd(), relativeCaption() (+46 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (24): authHandlerTestStore, fakeAuthStore, mfaEmailTestStore, fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg() (+16 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (53): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+45 more)

### Community 12 - "Community 12"
Cohesion: 0.02
Nodes (55): fakeChatAdapter, fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming() (+47 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (59): emailTestAuthStore, emailToken, erroringCountAuthStore, fakeMailer, buildAuthSecurityHandler(), TestHandleLoginUnknownEmailStillHashes(), TestHandleRegisterDuplicateEmailSkipsHash(), TestHandleTOTPChallengeLockout() (+51 more)

### Community 14 - "Community 14"
Cohesion: 0.07
Nodes (75): fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore, postgresDB() (+67 more)

### Community 15 - "Community 15"
Cohesion: 0.03
Nodes (89): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+81 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (58): macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore, BuildSource(), LocalPaths(), bulkUpserter (+50 more)

### Community 17 - "Community 17"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 18 - "Community 18"
Cohesion: 0.04
Nodes (52): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+44 more)

### Community 19 - "Community 19"
Cohesion: 0.03
Nodes (57): FS(), AuditEvent, BackupConfig, BodyCompositionSummary, CorrectionFeedback, CustomFoodInput, DailyRollup, DailyTargets (+49 more)

### Community 20 - "Community 20"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 21 - "Community 21"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 22 - "Community 22"
Cohesion: 0.05
Nodes (25): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), ParseResponse() (+17 more)

### Community 23 - "Community 23"
Cohesion: 0.07
Nodes (29): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), IsUnit() (+21 more)

### Community 24 - "Community 24"
Cohesion: 0.1
Nodes (18): food, foodCategory, foodNutrient, searchResponse, Source, bulkDataTypes(), extractMacros(), foodToMatch() (+10 more)

### Community 25 - "Community 25"
Cohesion: 0.14
Nodes (12): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), plural(), questionText() (+4 more)

### Community 26 - "Community 26"
Cohesion: 0.12
Nodes (17): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+9 more)

### Community 27 - "Community 27"
Cohesion: 0.1
Nodes (24): AWS default credential chain (backup), Backup runner, BACKUP_CHECK_INTERVAL, Database-level backup (pg_dump / sqlite3 .backup), internal/exportfmt (shared CSV writer), BACKUP_LOCAL_DIR, local_subdir path-traversal validation, Nudge scheduler (existing background loop) (+16 more)

### Community 28 - "Community 28"
Cohesion: 0.09
Nodes (14): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+6 more)

### Community 29 - "Community 29"
Cohesion: 0.15
Nodes (14): sendOut(), Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray(), TestExtractSuggestions_IntArray(), TestExtractSuggestions_MalformedJSON(), TestExtractSuggestions_NoBlock() (+6 more)

### Community 30 - "Community 30"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 31 - "Community 31"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 32 - "Community 32"
Cohesion: 0.16
Nodes (10): entry, cosineSimilarity(), packF32LE(), sortByScore(), TestCosineSimilarity(), TestPackUnpackF32LE(), TestUnpackBadBlob(), unpackF32LE() (+2 more)

### Community 33 - "Community 33"
Cohesion: 0.2
Nodes (8): nutriments, meetsPopularity(), New(), NewBulk(), parseQuantity(), product, searchResponse, Source

### Community 34 - "Community 34"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 35 - "Community 35"
Cohesion: 0.21
Nodes (7): Embedder, fingerprintStore, localFingerprint(), New(), Runner, SourceFactory, Store

### Community 36 - "Community 36"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 37 - "Community 37"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 38 - "Community 38"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 39 - "Community 39"
Cohesion: 0.25
Nodes (4): NewStatusCommand(), pct(), StatusCommand, StatusStore

### Community 40 - "Community 40"
Cohesion: 0.31
Nodes (5): close(), NumberField(), profilePayload(), save(), skipOrCancel()

### Community 41 - "Community 41"
Cohesion: 0.25
Nodes (3): SuggestCommand, SuggestEngine, SuggestFoodSearcher

### Community 42 - "Community 42"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 44 - "Community 44"
Cohesion: 0.36
Nodes (1): Store

### Community 45 - "Community 45"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 47 - "Community 47"
Cohesion: 0.52
Nodes (5): floatPtr(), intPtr(), TestToWorkout(), TestToWorkoutNilSafety(), ToWorkout()

### Community 48 - "Community 48"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 49 - "Community 49"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 55 - "Community 55"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 56 - "Community 56"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 61 - "Community 61"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 62 - "Community 62"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 64 - "Community 64"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 65 - "Community 65"
Cohesion: 0.5
Nodes (4): AI Compose Profile, docker compose (quick start), .env.example, PostgreSQL Compose Profile

### Community 72 - "Community 72"
Cohesion: 0.67
Nodes (2): deleteAccountRequest, UserDataExport

### Community 73 - "Community 73"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 96 - "Community 96"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 97 - "Community 97"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 98 - "Community 98"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 100 - "Community 100"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 101 - "Community 101"
Cohesion: 1.0
Nodes (1): VisionAdapter

### Community 102 - "Community 102"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 103 - "Community 103"
Cohesion: 1.0
Nodes (2): DELETE /api/v1/account, GET /api/v1/export/all

### Community 160 - "Community 160"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 161 - "Community 161"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 162 - "Community 162"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 163 - "Community 163"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 164 - "Community 164"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 165 - "Community 165"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 166 - "Community 166"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 167 - "Community 167"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 168 - "Community 168"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 169 - "Community 169"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Ambiguous Edges - Review These
- `PARSER_TIER` → `ENABLE_STT`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to
- `DELETE /api/v1/account` → `Backup runner`  [AMBIGUOUS]
  README.md · relation: conceptually_related_to

## Knowledge Gaps
- **386 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+381 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 17`** (86 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterDailyTotals()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 62`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 72`** (3 nodes): `deleteAccountRequest`, `UserDataExport`, `handler_account.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 96`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 97`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 98`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 100`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 101`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 102`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 103`** (2 nodes): `DELETE /api/v1/account`, `GET /api/v1/export/all`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 160`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 161`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 162`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 163`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 164`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 165`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 166`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 167`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 168`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 169`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `PARSER_TIER` and `ENABLE_STT`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `DELETE /api/v1/account` and `Backup runner`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 9`, `Community 12`, `Community 13`, `Community 14`, `Community 16`, `Community 18`, `Community 24`, `Community 26`, `Community 29`?**
  _High betweenness centrality (0.345) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 3` to `Community 0`, `Community 1`, `Community 2`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 11`, `Community 13`, `Community 16`, `Community 18`, `Community 22`, `Community 23`, `Community 29`?**
  _High betweenness centrality (0.095) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 2` to `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 18`, `Community 30`?**
  _High betweenness centrality (0.090) - this node is a cross-community bridge._
- **Are the 264 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 264 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._