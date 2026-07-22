# Graph Report - DietDaemon  (2026-07-22)

## Corpus Check
- 394 files · ~300,771 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3461 nodes · 6297 edges · 59 communities detected
- Extraction: 72% EXTRACTED · 28% INFERRED · 0% AMBIGUOUS · INFERRED: 1766 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 95|Community 95]]
- [[_COMMUNITY_Community 96|Community 96]]
- [[_COMMUNITY_Community 97|Community 97]]
- [[_COMMUNITY_Community 98|Community 98]]
- [[_COMMUNITY_Community 100|Community 100]]
- [[_COMMUNITY_Community 101|Community 101]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 285 edges
2. `Store` - 221 edges
3. `Handler` - 181 edges
4. `doRequest()` - 127 edges
5. `newHandler()` - 117 edges
6. `newFakeMealStore()` - 113 edges
7. `contains()` - 99 edges
8. `fakeMealStore` - 89 edges
9. `run()` - 70 edges
10. `ctx()` - 68 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `NewWeightCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/weight.go
- `run()` --calls--> `NewProfileCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/profile.go
- `run()` --calls--> `NewWaterCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/water.go
- `run()` --calls--> `NewWorkoutCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/workout.go
- `run()` --calls--> `NewSleepCommand()`  [INFERRED]
  cmd/dietdaemon/main.go → internal/commands/sleep.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (46): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, fastRow (+38 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (208): erroringCountAuthStore, TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory() (+200 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (74): credCreateConfig, credRevokeConfig, customFoodRequest, ErrorCode, errorEnvelope, errorEnvelopeWriter, errorForStatus(), publicErrorMessage() (+66 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (176): accountRepos, assertBYOKFailure(), TestBYOKFailuresDoNotFallBackToSharedAdapters(), TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), emailToken, fakeMailer, fakeMealLogger, fakeSuggester (+168 more)

### Community 4 - "Community 4"
Cohesion: 0.03
Nodes (118): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestExtractLabel(), TestExtractLabelHTTPError(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement() (+110 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (103): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, countRows(), newTestStore(), seedAllEntities(), seedUser(), TestRestoreCLI_DryRun() (+95 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (28): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, readSessionCookie(), bearerToken(), isMutating(), TestBearerTokenEdgeCases(), mfaEmailTestStore (+20 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (82): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthStore, BackupRunner, ChatStore, EmailTokenStore (+74 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 9 - "Community 9"
Cohesion: 0.03
Nodes (68): Adapter, contentBlock, message, messagesRequest, messagesResponse, Destination, Runner, Store (+60 more)

### Community 10 - "Community 10"
Cohesion: 0.06
Nodes (94): totpChallengeAuthStore, fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore (+86 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (54): Registry, renderModal(), dayLabel(), download(), sourceLabel(), onSubmit(), onAdd(), relativeCaption() (+46 more)

### Community 12 - "Community 12"
Cohesion: 0.03
Nodes (60): fakeChatAdapter, sendOut(), blockingChatAdapter, fakeChatAdapter, Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray() (+52 more)

### Community 13 - "Community 13"
Cohesion: 0.02
Nodes (1): fakeMealStore

### Community 14 - "Community 14"
Cohesion: 0.04
Nodes (52): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+44 more)

### Community 15 - "Community 15"
Cohesion: 0.03
Nodes (62): FS(), NotFound(), AuditEvent, BackupConfig, BodyCompositionSummary, CorrectionFeedback, CustomFoodInput, DailyRollup (+54 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (32): CorrectCommand, CorrectResolver, CorrectStore, MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand() (+24 more)

### Community 17 - "Community 17"
Cohesion: 0.05
Nodes (45): adminTempStore(), TestFoodImportAdmin_ImportSource_MaxRowsCap(), TestFoodImportAdmin_ImportSource_TACO(), TestFoodImportAdmin_ImportSource_UnknownSource(), TestFoodImportAdmin_RepairSource(), groupIntoMeals(), importMeals(), main() (+37 more)

### Community 18 - "Community 18"
Cohesion: 0.05
Nodes (25): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, calcSleepHours(), computeSleepDuration(), formatDuration(), NewSleepCommand() (+17 more)

### Community 19 - "Community 19"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 20 - "Community 20"
Cohesion: 0.05
Nodes (28): randomID(), NewWaterCommand(), WaterCommand, WaterStore, NewWeightCommand(), WeightCommand, WeightStore, NewWorkoutCommand() (+20 more)

### Community 21 - "Community 21"
Cohesion: 0.1
Nodes (25): fakeParser, fakePending, fakeReplier, fakeResolver, fakeStore, fakeTranscriber, newFakePending(), newFakeStore() (+17 more)

### Community 22 - "Community 22"
Cohesion: 0.07
Nodes (31): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), IsUnit() (+23 more)

### Community 23 - "Community 23"
Cohesion: 0.09
Nodes (27): cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer(), newRequestID(), observeRequests(), recoverPanics() (+19 more)

### Community 24 - "Community 24"
Cohesion: 0.09
Nodes (25): Embedder, fakeEmbedder, fakeSource, fingerprintStore, localFingerprint(), New(), NewWithLocalPaths(), replaceDataset() (+17 more)

### Community 25 - "Community 25"
Cohesion: 0.09
Nodes (21): food, foodCategory, foodNutrient, foodPortion, searchResponse, Source, bulkDataTypes(), extractMacros() (+13 more)

### Community 26 - "Community 26"
Cohesion: 0.13
Nodes (13): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), parseGrams(), plural() (+5 more)

### Community 27 - "Community 27"
Cohesion: 0.1
Nodes (10): fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming(), TestHandleChatMessageStreamError() (+2 more)

### Community 28 - "Community 28"
Cohesion: 0.09
Nodes (14): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+6 more)

### Community 29 - "Community 29"
Cohesion: 0.12
Nodes (12): fakeFoodSearcher, fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_IngredientArgsResolveViaSearch(), TestSuggestCommand_IngredientArgsSkipUnresolvedNames() (+4 more)

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
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 38 - "Community 38"
Cohesion: 0.18
Nodes (1): allEntitiesFakeStore

### Community 39 - "Community 39"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 40 - "Community 40"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 42 - "Community 42"
Cohesion: 0.36
Nodes (1): Store

### Community 43 - "Community 43"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 47 - "Community 47"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 52 - "Community 52"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 53 - "Community 53"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 59 - "Community 59"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 60 - "Community 60"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 62 - "Community 62"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 65 - "Community 65"
Cohesion: 1.0
Nodes (2): gramsFor(), unitOptionsFor()

### Community 69 - "Community 69"
Cohesion: 0.67
Nodes (2): deleteAccountRequest, UserDataExport

### Community 70 - "Community 70"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 95 - "Community 95"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

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

## Knowledge Gaps
- **307 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+302 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 13`** (88 nodes): `fakeMealStore`, `.AddFoodAlias()`, `.AddMealItem()`, `.AddToLibrary()`, `.ConfirmPendingAlias()`, `.ConsumeLinkingCode()`, `.CorrectMealItem()`, `.CreateCustomFood()`, `.CreateFoodServingUnit()`, `.CreateLinkingCode()`, `.DeleteCustomFood()`, `.DeleteFoodAlias()`, `.DeleteFoodServingUnit()`, `.DeleteMealItem()`, `.DeleteMeasurement()`, `.DeletePhoto()`, `.DeleteSleep()`, `.DeleteTemplate()`, `.DeleteUserAIKey()`, `.DeleteUserHevyKey()`, `.DeleteWater()`, `.DeleteWeight()`, `.DeleteWorkout()`, `.EndFast()`, `.EndSleep()`, `.FrequentFoods()`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetBackupConfig()`, `.GetFood()`, `.GetFoodDetail()`, `.GetFoodForUser()`, `.GetFoodImportStatuses()`, `.GetMeal()`, `.GetMealsInRange()`, `.GetNudgeRuleConfig()`, `.GetPhotoData()`, `.GetProfile()`, `.GetRollup()`, `.GetRollups()`, `.GetSourcePrecedence()`, `.GetTargets()`, `.GetTemplate()`, `.GetTemplates()`, `.GetUser()`, `.GetUserAIKey()`, `.GetUserHevyKey()`, `.GetWaterDailyTotals()`, `.GetWaterToday()`, `.GetWorkout()`, `.ImportWorkout()`, `.LatestMealTime()`, `.ListFasts()`, `.ListFoods()`, `.ListMeasurements()`, `.ListPendingAliases()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`, `.ListWorkouts()`, `.LogMeasurement()`, `.LogSleep()`, `.LogTemplateUse()`, `.LogWater()`, `.LogWeight()`, `.LogWorkout()`, `.LookupLinkingCode()`, `.LookupLinkingCodeAny()`, `.RecentMeals()`, `.RejectPendingAlias()`, `.RemoveFromLibrary()`, `.SaveMeal()`, `.SaveTemplate()`, `.SearchCatalog()`, `.SearchFoods()`, `.SetBackupConfig()`, `.SetNudgeRuleConfig()`, `.SetSourcePrecedence()`, `.SetTargets()`, `.SetUserAIKey()`, `.SetUserHevyKey()`, `.StartFast()`, `.UpdateCustomFood()`, `.UpdateRollupTargets()`, `.UploadPhoto()`, `.UpsertProfile()`, `.UpsertUser()`, `.WeightTrend()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (15 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (11 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotoData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (3 nodes): `gramsFor()`, `unitOptionsFor()`, `servingUnits.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (3 nodes): `deleteAccountRequest`, `UserDataExport`, `handler_account.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 95`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
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

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 9`, `Community 10`, `Community 11`, `Community 12`, `Community 14`, `Community 16`, `Community 17`, `Community 21`, `Community 23`, `Community 24`, `Community 25`, `Community 27`, `Community 29`?**
  _High betweenness centrality (0.408) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 4` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 5`, `Community 7`, `Community 9`, `Community 12`, `Community 14`, `Community 15`, `Community 17`, `Community 20`, `Community 21`, `Community 22`, `Community 23`, `Community 24`, `Community 26`, `Community 29`, `Community 33`?**
  _High betweenness centrality (0.129) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 7` to `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 35`, `Community 10`, `Community 11`, `Community 15`, `Community 16`, `Community 17`, `Community 18`, `Community 20`, `Community 23`, `Community 24`, `Community 29`?**
  _High betweenness centrality (0.121) - this node is a cross-community bridge._
- **Are the 280 inferred relationships involving `New()` (e.g. with `adminTempStore()` and `.BackfillEmbeddings()`) actually correct?**
  _`New()` has 280 INFERRED edges - model-reasoned connections that need verification._
- **Are the 4 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 38 inferred relationships involving `doRequest()` (e.g. with `TestEmailVerifySuccess()` and `TestEmailVerifyInvalidToken()`) actually correct?**
  _`doRequest()` has 38 INFERRED edges - model-reasoned connections that need verification._
- **Are the 26 inferred relationships involving `newHandler()` (e.g. with `newAdminTestHandler()` and `TestAdminFoodImport_RunnerNil503()`) actually correct?**
  _`newHandler()` has 26 INFERRED edges - model-reasoned connections that need verification._