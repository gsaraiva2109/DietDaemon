# Graph Report - DietDaemon  (2026-08-27)

## Corpus Check
- 521 files · ~489,608 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 5726 nodes · 13268 edges · 68 communities detected
- Extraction: 62% EXTRACTED · 38% INFERRED · 0% AMBIGUOUS · INFERRED: 5092 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 75|Community 75]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 88|Community 88]]
- [[_COMMUNITY_Community 90|Community 90]]
- [[_COMMUNITY_Community 91|Community 91]]
- [[_COMMUNITY_Community 92|Community 92]]
- [[_COMMUNITY_Community 131|Community 131]]
- [[_COMMUNITY_Community 132|Community 132]]
- [[_COMMUNITY_Community 133|Community 133]]
- [[_COMMUNITY_Community 134|Community 134]]
- [[_COMMUNITY_Community 135|Community 135]]
- [[_COMMUNITY_Community 136|Community 136]]

## God Nodes (most connected - your core abstractions)
1. `doRequest()` - 601 edges
2. `New()` - 578 edges
3. `newHandler()` - 504 edges
4. `newFakeMealStore()` - 485 edges
5. `Store` - 278 edges
6. `Handler` - 259 edges
7. `contains()` - 231 edges
8. `ctx()` - 222 edges
9. `tempDB()` - 155 edges
10. `decodeJSON()` - 138 edges

## Surprising Connections (you probably didn't know these)
- `contains()` --calls--> `TestStreamChatHTTPError()`  [INFERRED]
  internal/auth/totp_test.go → adapters/model/anthropic/chat_test.go
- `contains()` --calls--> `TestPhotoPromptCallsOutMultiColumnCarbCycling()`  [INFERRED]
  internal/auth/totp_test.go → adapters/model/planextract/planextract_test.go
- `TestAPIRouteFallbackUsesErrorEnvelope()` --calls--> `New()`  [INFERRED]
  internal/api/errors_test.go → adapters/nutrition/taco/taco.go
- `TestHandleListChatSessionsError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go
- `TestHandleGetChatMessagesError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (528): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), fakeSuggester, deleteAccountTestHandler(), newHandlerWithAccountStore(), TestHandleDeleteAccountClearsSessionCookie(), TestHandleDeleteAccountEmailSendFailureStillSucceeds(), TestHandleDeleteAccountEmailSkippedOnUserLookupFailure() (+520 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (328): TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), TestExpensiveRequestRoutes(), assertDoneEvent(), assertTextDeltaEvent(), assertToolCallEvent(), assertToolResultEvent(), collectEvents() (+320 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (135): auditActor, chatStreamState, credCreateConfig, credRevokeConfig, customFoodRequest, dayOverrideBody, deleteAccountRequest, ErrorCode (+127 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (57): normalize(), FS(), Normalize(), Unaccent(), AccountDeletionStatus, backupConfigRow, catalogRow, credRow (+49 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (280): fakeAccount, fakeAuditEvent, fakeBackupDest, fakeMailer, fakePurgeStore, orderLog, NewPurgeRunner(), TestPurgeAccountBackups_DeleteErrorStillPurgesAccount() (+272 more)

### Community 5 - "Community 5"
Cohesion: 0.01
Nodes (117): Registry, renderShell(), addFood(), renderModal(), typeSearch(), renderModal(), renderModal(), dayLabel() (+109 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (212): checkCompleteRequest(), TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), checkExtractLabelDraft(), checkExtractLabelRequest(), checkExtractMenuRequest(), checkExtractPlanRequest() (+204 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (150): Adapter, contentBlock, message, messagesRequest, messagesResponse, writeCSV(), capturingDest, Destination (+142 more)

### Community 8 - "Community 8"
Cohesion: 0.01
Nodes (7): authHandlerTestStore, fakeAuthStore, fakeMealStore, passkeyTestStore, totpTestStore, Dest, Store

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (143): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthRepos, AuthStore, BackupRunner, ChatStore (+135 more)

### Community 10 - "Community 10"
Cohesion: 0.03
Nodes (129): authTestStore, credAuthStore, emailTestAuthStore, emailToken, erroringCountAuthStore, fakeMailer, buildCredHandler(), TestCheckLoginLockoutConcurrentNoRace() (+121 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (116): NewChatAdapter(), parseSSEEvent(), sendProviderError(), assertAssistantToolUseMessage(), assertToolResultMessage(), drainReadStream(), TestReadStreamContextCancelledMidStream(), TestReadStreamErrorEvent() (+108 more)

### Community 12 - "Community 12"
Cohesion: 0.03
Nodes (131): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, targetsResponse, startBackgroundServices(), blockingStore, DigestRule, fakeChatRouteStore (+123 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (109): OpenSQLiteStore(), SignalContext(), TestOpenSQLiteStore(), TestSignalContextHonorsCanceledParent(), assertPendingMealReplaceAndDelete(), assertPendingMealSaveAndGet(), assertPendingStoreLifecycle(), newTestPendingMeal() (+101 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (58): CorrectCommand, CorrectResolver, CorrectStore, fakeMealStore, MealStore, parsePositiveFloat(), setProfileField(), ProfileCommand (+50 more)

### Community 15 - "Community 15"
Cohesion: 0.02
Nodes (82): Config, addProblem(), parseProxyEntry(), validateBulkFile(), NotFound(), AuditEvent, BackupConfig, BodyCompositionSummary (+74 more)

### Community 16 - "Community 16"
Cohesion: 0.03
Nodes (57): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, confirmReplace(), scaledMacros(), sourceLabel(), MacroBar() (+49 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (44): classifyGoalDivergence(), newAuditID(), PurgeRunner, PurgeStore, New(), ChatRouteStore, ChatSender, DigestStore (+36 more)

### Community 18 - "Community 18"
Cohesion: 0.04
Nodes (53): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+45 more)

### Community 19 - "Community 19"
Cohesion: 0.04
Nodes (54): fakeCompletionAdapter, assertDayType(), assertDietboxDraft(), assertWeekdaySchedule(), derefOrNilStr(), itemCount(), mustReadFixture(), TestHandleExtractPlanFromImage_MultiPageFixtureRegression() (+46 more)

### Community 20 - "Community 20"
Cohesion: 0.04
Nodes (37): appendedChatMessage, buildAdapterForProvider(), buildChatAdapterForProvider(), decryptAIKey(), assertBYOKFailure(), TestBuildBYOKAdaptersRejectUnsupportedProvider(), TestBYOKChatOverrideUsedInsteadOfSharedAdapter(), TestBYOKFailuresDoNotFallBackToSharedAdapters() (+29 more)

### Community 21 - "Community 21"
Cohesion: 0.05
Nodes (48): actionRow, Adapter, buttonComponent, dialWebSocket(), dialWebSocketWithTLSConfig(), mustMarshal(), readGatewayPayload(), awaitClosed() (+40 more)

### Community 22 - "Community 22"
Cohesion: 0.13
Nodes (61): TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(), TestHandleLoginMFAStepUp(), TestHandleLoginMFAStepUpChallengeCreationFails(), buildTOTPHandler(), defaultTOTPMeals(), enrollTOTPSecret(), newTOTPTestStore(), TestHandleRegenerateRecoveryHasConfirmedError() (+53 more)

### Community 23 - "Community 23"
Cohesion: 0.05
Nodes (36): Client, NewClient(), listResponse, captureHandler, Config, Mailer, New(), TestNew() (+28 more)

### Community 24 - "Community 24"
Cohesion: 0.05
Nodes (42): corsTestCase, cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer(), newRequestID(), observeRequests() (+34 more)

### Community 25 - "Community 25"
Cohesion: 0.14
Nodes (45): buildAuthSecurityHandler(), TestHandleLoginUnknownEmailStillHashes(), TestHandleRegisterDuplicateEmailSkipsHash(), doOIDCCallback(), locationParams(), newTestIdP(), oidcCallbackRequest(), oidcHandler() (+37 more)

### Community 26 - "Community 26"
Cohesion: 0.06
Nodes (31): food, foodCategory, foodNutrient, foodPortion, searchResponse, Source, bulkDataTypes(), emitMatchedFood() (+23 more)

### Community 27 - "Community 27"
Cohesion: 0.12
Nodes (46): doPasskeyLoginFinish(), mfaPasskeyBeginExpiredChallenge(), mfaPasskeyBeginInvalidJSON(), mfaPasskeyBeginMissingToken(), mfaPasskeyBeginNoPasskeysRegistered(), mfaPasskeyBeginSuccess(), mfaPasskeyBeginUnknownChallenge(), mfaPasskeyFinishCeremonyConsumeFails() (+38 more)

### Community 28 - "Community 28"
Cohesion: 0.08
Nodes (48): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+40 more)

### Community 29 - "Community 29"
Cohesion: 0.15
Nodes (20): isMutating(), fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession() (+12 more)

### Community 30 - "Community 30"
Cohesion: 0.15
Nodes (15): sendOut(), sendSuggestions(), Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray(), TestExtractSuggestions_IntArray(), TestExtractSuggestions_MalformedJSON() (+7 more)

### Community 31 - "Community 31"
Cohesion: 0.21
Nodes (17): fakeFoodImportRunner, doAdminRequest(), newAdminTestHandler(), TestAdminFoodImport_BackfillEmbeddings200(), TestAdminFoodImport_BackfillEmbeddingsError(), TestAdminFoodImport_MissingToken401(), TestAdminFoodImport_Repair200(), TestAdminFoodImport_RepairError() (+9 more)

### Community 32 - "Community 32"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 33 - "Community 33"
Cohesion: 0.1
Nodes (1): fakeStore

### Community 34 - "Community 34"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 35 - "Community 35"
Cohesion: 0.17
Nodes (1): fakeStore

### Community 36 - "Community 36"
Cohesion: 0.26
Nodes (7): appendDelta(), appendToolCall(), applyStreamEvent(), applySuggestions(), applyToolResult(), raiseStreamError(), stripSuggestionsFence()

### Community 37 - "Community 37"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 38 - "Community 38"
Cohesion: 0.36
Nodes (9): bundle(), bundleWithOption(), bundleWithSlot(), dayType(), mealTemplate(), option(), plan(), resolvedItem() (+1 more)

### Community 39 - "Community 39"
Cohesion: 0.27
Nodes (7): fakeVisionAdapter, doOCRUpload(), TestHandleOCRExtractCustomFood(), TestHandleOCRExtractCustomFoodAdapterError(), TestHandleOCRExtractCustomFoodDisabled(), TestHandleOCRExtractCustomFoodMissingFile(), TestHandleOCRExtractCustomFoodNonImage()

### Community 40 - "Community 40"
Cohesion: 0.18
Nodes (1): allEntitiesFakeStore

### Community 41 - "Community 41"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 43 - "Community 43"
Cohesion: 0.28
Nodes (3): parseWorkoutArgs(), WorkoutCommand, WorkoutStore

### Community 44 - "Community 44"
Cohesion: 0.32
Nodes (4): applyExtractedDraft(), buildCustomFoodInput(), onExtracted(), submit()

### Community 45 - "Community 45"
Cohesion: 0.36
Nodes (1): Store

### Community 46 - "Community 46"
Cohesion: 0.29
Nodes (3): pct(), StatusCommand, StatusStore

### Community 47 - "Community 47"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 52 - "Community 52"
Cohesion: 0.38
Nodes (4): fakeResponse(), runOptions(), streamOf(), userMessage()

### Community 54 - "Community 54"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 55 - "Community 55"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 56 - "Community 56"
Cohesion: 0.4
Nodes (2): bundleWith(), plan()

### Community 58 - "Community 58"
Cohesion: 0.5
Nodes (2): focusSoon(), resetSearchState()

### Community 61 - "Community 61"
Cohesion: 0.4
Nodes (1): fakeMealLogger

### Community 62 - "Community 62"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 63 - "Community 63"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 73 - "Community 73"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 75 - "Community 75"
Cohesion: 0.5
Nodes (1): crossUserPhotoStore

### Community 76 - "Community 76"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 88 - "Community 88"
Cohesion: 1.0
Nodes (2): chunk(), solidPng()

### Community 90 - "Community 90"
Cohesion: 0.67
Nodes (1): Memory

### Community 91 - "Community 91"
Cohesion: 0.67
Nodes (2): oidcCallbackContext, oidcIdentity

### Community 92 - "Community 92"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 131 - "Community 131"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

### Community 132 - "Community 132"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 133 - "Community 133"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 134 - "Community 134"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 135 - "Community 135"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 136 - "Community 136"
Cohesion: 1.0
Nodes (1): VisionAdapter

## Knowledge Gaps
- **363 isolated node(s):** `corsTestCase`, `appRuntime`, `phraseEntry`, `mealSaver`, `Row` (+358 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 33`** (20 nodes): `fakeStore`, `.AccountDeletedAt()`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotosData()`, `.GetPlanBundle()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListDayOverrides()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListPlans()`, `.ListSleep()`, `.ListTemplatesForBackup()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (13 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SaveMealAndAddToRollup()`, `.SetTargets()`, `.TargetsFor()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (11 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotosData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (6 nodes): `bundleWith()`, `meal()`, `noPlanView()`, `plan()`, `planDayWithSlots()`, `Dashboard.test.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (5 nodes): `focusSoon()`, `onKey()`, `onListKey()`, `resetSearchState()`, `CommandPalette.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 61`** (5 nodes): `fakeMealLogger`, `.Handle()`, `.LogMeal()`, `.LogMealFromItems()`, `.ParseAndResolve()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (4 nodes): `crossUserPhotoStore`, `.DeletePhoto()`, `.GetPhotoData()`, `.ListPhotoMetadata()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 88`** (3 nodes): `chunk()`, `solidPng()`, `generate-page-pngs.mjs`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 90`** (3 nodes): `queue.go`, `Memory`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 91`** (3 nodes): `oidcCallbackContext`, `oidcIdentity`, `handler_oidc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 92`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 131`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 132`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 133`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 134`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 135`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 136`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 16`, `Community 19`, `Community 20`, `Community 21`, `Community 22`, `Community 23`, `Community 24`, `Community 25`, `Community 26`, `Community 27`, `Community 30`, `Community 31`, `Community 39`, `Community 43`?**
  _High betweenness centrality (0.404) - this node is a cross-community bridge._
- **Why does `newHandler()` connect `Community 0` to `Community 1`, `Community 39`, `Community 10`, `Community 19`, `Community 20`, `Community 22`, `Community 25`, `Community 31`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 6` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 7`, `Community 9`, `Community 10`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 15`, `Community 18`, `Community 19`, `Community 20`, `Community 21`, `Community 22`, `Community 24`, `Community 25`, `Community 30`?**
  _High betweenness centrality (0.096) - this node is a cross-community bridge._
- **Are the 485 inferred relationships involving `doRequest()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`doRequest()` has 485 INFERRED edges - model-reasoned connections that need verification._
- **Are the 573 inferred relationships involving `New()` (e.g. with `TestRunReturnsConfigLoadError()` and `adminTempStore()`) actually correct?**
  _`New()` has 573 INFERRED edges - model-reasoned connections that need verification._
- **Are the 383 inferred relationships involving `newHandler()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newHandler()` has 383 INFERRED edges - model-reasoned connections that need verification._
- **Are the 365 inferred relationships involving `newFakeMealStore()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newFakeMealStore()` has 365 INFERRED edges - model-reasoned connections that need verification._