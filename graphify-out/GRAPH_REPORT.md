# Graph Report - DietDaemon  (2026-07-28)

## Corpus Check
- 478 files · ~647,447 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 5215 nodes · 11900 edges · 69 communities detected
- Extraction: 62% EXTRACTED · 38% INFERRED · 0% AMBIGUOUS · INFERRED: 4485 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 75|Community 75]]
- [[_COMMUNITY_Community 82|Community 82]]
- [[_COMMUNITY_Community 83|Community 83]]
- [[_COMMUNITY_Community 113|Community 113]]
- [[_COMMUNITY_Community 114|Community 114]]
- [[_COMMUNITY_Community 115|Community 115]]
- [[_COMMUNITY_Community 116|Community 116]]
- [[_COMMUNITY_Community 118|Community 118]]
- [[_COMMUNITY_Community 120|Community 120]]

## God Nodes (most connected - your core abstractions)
1. `doRequest()` - 565 edges
2. `New()` - 514 edges
3. `newHandler()` - 466 edges
4. `newFakeMealStore()` - 451 edges
5. `Store` - 260 edges
6. `Handler` - 248 edges
7. `contains()` - 210 edges
8. `ctx()` - 164 edges
9. `tempDB()` - 125 edges
10. `decodeJSON()` - 122 edges

## Surprising Connections (you probably didn't know these)
- `TestAuthenticatedRateLimitCategories()` --calls--> `New()`  [INFERRED]
  internal/api/rate_limit_test.go → adapters/nutrition/taco/taco.go
- `TestAPIRouteFallbackUsesErrorEnvelope()` --calls--> `New()`  [INFERRED]
  internal/api/errors_test.go → adapters/nutrition/taco/taco.go
- `TestHandleListChatSessionsError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go
- `TestHandleGetChatMessagesError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go
- `TestHandleGetChatSettingsError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (489): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), fakeCompletionAdapter, fakeMealLogger, fakeSuggester, newHandlerWithAccountStore(), TestHandleDeleteAccountClearsSessionCookie(), TestHandleDeleteAccountMissingBody() (+481 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (377): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, targetsResponse, assertDoneEvent(), assertTextDeltaEvent(), assertToolCallEvent(), assertToolResultEvent() (+369 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (129): chatStreamState, credCreateConfig, credRevokeConfig, customFoodRequest, dayOverrideBody, deleteAccountRequest, ErrorCode, errorEnvelope (+121 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (52): FS(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, dayTypeRow (+44 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (206): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), checkExtractPlanRequest(), TestExtractLabel(), TestExtractLabelHTTPError(), TestExtractPlan(), TestExtractPlanHTTPError() (+198 more)

### Community 5 - "Community 5"
Cohesion: 0.01
Nodes (7): authHandlerTestStore, fakeAuthStore, fakeMealStore, mfaEmailTestStore, passkeyTestStore, totpTestStore, Store

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (126): Adapter, contentBlock, message, messagesRequest, messagesResponse, writeCSV(), Destination, Runner (+118 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (135): extractArgs(), NewChatAdapter(), parseSSEEvent(), sendEvent(), sendProviderError(), drainReadStream(), TestExtractArgsEmptyValue(), TestReadStreamContextCancelledMidStream() (+127 more)

### Community 8 - "Community 8"
Cohesion: 0.04
Nodes (178): IPRateLimiter, TestPendingStoreContract(), postgresDB(), TestFoodImportFingerprintStore(), TestPostgresDietPlanSmoke(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods() (+170 more)

### Community 9 - "Community 9"
Cohesion: 0.01
Nodes (77): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, Registry, addFood(), renderModal(), typeSearch() (+69 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (57): ProtectedRoute(), ApiError, blobRequest(), buildRequestHeaders(), handleUnauthorized(), multipart(), parseErrorBody(), RateLimitError (+49 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (104): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthRepos, AuthStore, BackupRunner, ChatStore (+96 more)

### Community 12 - "Community 12"
Cohesion: 0.02
Nodes (89): Stat(), Config, addProblem(), parseProxyEntry(), validateBulkFile(), TestRunStopsCleanlyWithCanceledContext(), Embedder, fingerprintStore (+81 more)

### Community 13 - "Community 13"
Cohesion: 0.02
Nodes (58): CorrectCommand, CorrectResolver, CorrectStore, fakeMealStore, MealStore, parsePositiveFloat(), setProfileField(), ProfileCommand (+50 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (61): NewWebAuthnHandle(), randomID(), calcSleepHours(), computeSleepDuration(), formatDuration(), parseQualityAndNote(), TestCalcSleepHours_NilWake(), TestCalcSleepHours_Overnight() (+53 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (81): authTestStore, emailTestAuthStore, emailToken, fakeMailer, TestHandleRegisterCreateEmailTokenFailure(), TestHandleRegisterSendsVerificationEmailWhenMailerConfigured(), TestHandleTOTPChallengeLockout(), containsStr() (+73 more)

### Community 16 - "Community 16"
Cohesion: 0.07
Nodes (83): credAuthStore, erroringCountAuthStore, buildCredHandler(), TestCheckLoginLockoutLocked(), TestCheckLoginLockoutStoreError(), TestHandleChangePasswordInvalidJSON(), TestHandleChangePasswordMissingFields(), TestHandleChangePasswordNewPasswordTooShort() (+75 more)

### Community 17 - "Community 17"
Cohesion: 0.03
Nodes (73): adminTempStore(), TestFoodImportAdmin_ImportSource_MaxRowsCap(), TestFoodImportAdmin_ImportSource_TACO(), TestFoodImportAdmin_ImportSource_UnknownSource(), TestFoodImportAdmin_RepairSource(), bulkUpserter, main(), run() (+65 more)

### Community 18 - "Community 18"
Cohesion: 0.04
Nodes (53): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+45 more)

### Community 19 - "Community 19"
Cohesion: 0.04
Nodes (45): fakePlanStore, findSlotByLabel(), NewPlanCommand(), nextSlotID(), optionSummary(), parseTimeOfDay(), slotsForDayType(), bundleWithSlots() (+37 more)

### Community 20 - "Community 20"
Cohesion: 0.04
Nodes (37): appendedChatMessage, buildAdapterForProvider(), buildChatAdapterForProvider(), decryptAIKey(), assertBYOKFailure(), TestBuildBYOKAdaptersRejectUnsupportedProvider(), TestBYOKChatOverrideUsedInsteadOfSharedAdapter(), TestBYOKFailuresDoNotFallBackToSharedAdapters() (+29 more)

### Community 21 - "Community 21"
Cohesion: 0.13
Nodes (61): TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(), TestHandleLoginMFAStepUp(), TestHandleLoginMFAStepUpChallengeCreationFails(), buildTOTPHandler(), defaultTOTPMeals(), enrollTOTPSecret(), newTOTPTestStore(), TestHandleRegenerateRecoveryHasConfirmedError() (+53 more)

### Community 22 - "Community 22"
Cohesion: 0.04
Nodes (26): allEntitiesFakeStore, New(), newFakeStore(), TestRun_ChecksImmediatelyBeforeCancelledContextReturns(), TestRunFor_ExportsAllEntities(), TestRunFor_LoadErrorAbortsRemainingEntities(), TestRunFor_MissingDestinationErrors(), TestRunFor_PhotoBlobWrittenBeforeIndex() (+18 more)

### Community 23 - "Community 23"
Cohesion: 0.06
Nodes (46): actionRow, Adapter, buttonComponent, mustMarshal(), readGatewayPayload(), readWSFrame(), buildServerFrame(), genSelfSignedCert() (+38 more)

### Community 24 - "Community 24"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 25 - "Community 25"
Cohesion: 0.06
Nodes (30): food, foodCategory, foodNutrient, foodPortion, searchResponse, Source, bulkDataTypes(), emitMatchedFood() (+22 more)

### Community 26 - "Community 26"
Cohesion: 0.12
Nodes (46): doPasskeyLoginFinish(), mfaPasskeyBeginExpiredChallenge(), mfaPasskeyBeginInvalidJSON(), mfaPasskeyBeginMissingToken(), mfaPasskeyBeginNoPasskeysRegistered(), mfaPasskeyBeginSuccess(), mfaPasskeyBeginUnknownChallenge(), mfaPasskeyFinishCeremonyConsumeFails() (+38 more)

### Community 27 - "Community 27"
Cohesion: 0.09
Nodes (28): cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer(), newRequestID(), observeRequests(), recoverPanics() (+20 more)

### Community 28 - "Community 28"
Cohesion: 0.14
Nodes (20): isMutating(), fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession() (+12 more)

### Community 29 - "Community 29"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 30 - "Community 30"
Cohesion: 0.11
Nodes (16): scaleMacros(), gramsFor(), unitOptionsFor(), addPosition(), apply(), formatNumber(), fromResolvedItem(), isAdLibitum() (+8 more)

### Community 31 - "Community 31"
Cohesion: 0.21
Nodes (17): fakeFoodImportRunner, doAdminRequest(), newAdminTestHandler(), TestAdminFoodImport_BackfillEmbeddings200(), TestAdminFoodImport_BackfillEmbeddingsError(), TestAdminFoodImport_MissingToken401(), TestAdminFoodImport_Repair200(), TestAdminFoodImport_RepairError() (+9 more)

### Community 32 - "Community 32"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 33 - "Community 33"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 34 - "Community 34"
Cohesion: 0.27
Nodes (11): addSortIndicators(), enableUI(), getNthColumn(), getTable(), getTableBody(), getTableHeader(), loadColumns(), loadData() (+3 more)

### Community 35 - "Community 35"
Cohesion: 0.18
Nodes (7): fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore

### Community 36 - "Community 36"
Cohesion: 0.26
Nodes (7): appendDelta(), appendToolCall(), applyStreamEvent(), applySuggestions(), applyToolResult(), raiseStreamError(), stripSuggestionsFence()

### Community 37 - "Community 37"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 38 - "Community 38"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 39 - "Community 39"
Cohesion: 0.36
Nodes (9): bundle(), bundleWithOption(), bundleWithSlot(), dayType(), mealTemplate(), option(), plan(), resolvedItem() (+1 more)

### Community 40 - "Community 40"
Cohesion: 0.35
Nodes (8): a(), B(), D(), g(), i(), k(), Q(), y()

### Community 41 - "Community 41"
Cohesion: 0.24
Nodes (5): writeCandidates(), writeOtherOptions(), SuggestCommand, SuggestEngine, SuggestFoodSearcher

### Community 42 - "Community 42"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 43 - "Community 43"
Cohesion: 0.31
Nodes (7): loadPdfjs(), pdfToImages(), renderPage(), handleFile(), onExtracted(), onFailed(), t()

### Community 45 - "Community 45"
Cohesion: 0.31
Nodes (7): fakeVisionAdapter, doOCRUpload(), TestHandleOCRExtractCustomFood(), TestHandleOCRExtractCustomFoodAdapterError(), TestHandleOCRExtractCustomFoodDisabled(), TestHandleOCRExtractCustomFoodMissingFile(), TestHandleOCRExtractCustomFoodNonImage()

### Community 46 - "Community 46"
Cohesion: 0.22
Nodes (2): blockingStore, fakeStore

### Community 47 - "Community 47"
Cohesion: 0.36
Nodes (1): Store

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 53 - "Community 53"
Cohesion: 0.38
Nodes (4): fakeResponse(), runOptions(), streamOf(), userMessage()

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
Cohesion: 0.4
Nodes (2): bundleWith(), plan()

### Community 60 - "Community 60"
Cohesion: 0.5
Nodes (2): focusSoon(), resetSearchState()

### Community 64 - "Community 64"
Cohesion: 0.7
Nodes (4): goToNext(), goToPrevious(), makeCurrent(), toggleClass()

### Community 65 - "Community 65"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 66 - "Community 66"
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

### Community 82 - "Community 82"
Cohesion: 0.67
Nodes (2): oidcCallbackContext, oidcIdentity

### Community 83 - "Community 83"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 113 - "Community 113"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

### Community 114 - "Community 114"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 115 - "Community 115"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 116 - "Community 116"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 118 - "Community 118"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 120 - "Community 120"
Cohesion: 1.0
Nodes (1): VisionAdapter

## Knowledge Gaps
- **351 isolated node(s):** `appRuntime`, `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row` (+346 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 37`** (12 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.TargetsFor()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (10 nodes): `blockingStore`, `.GetRollup()`, `.GetTargets()`, `.ListUsers()`, `.TargetsFor()`, `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.ListUsers()`, `.TargetsFor()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (6 nodes): `bundleWith()`, `meal()`, `noPlanView()`, `plan()`, `planDayWithSlots()`, `Dashboard.test.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (5 nodes): `focusSoon()`, `onKey()`, `onListKey()`, `resetSearchState()`, `CommandPalette.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 82`** (3 nodes): `oidcCallbackContext`, `oidcIdentity`, `handler_oidc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 113`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 114`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 115`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 116`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 118`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 120`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 17`, `Community 19`, `Community 20`, `Community 21`, `Community 22`, `Community 23`, `Community 25`, `Community 26`, `Community 27`, `Community 31`, `Community 45`?**
  _High betweenness centrality (0.424) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 2` to `Community 32`, `Community 1`, `Community 4`, `Community 6`, `Community 11`, `Community 14`, `Community 20`, `Community 28`?**
  _High betweenness centrality (0.102) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 4` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 6`, `Community 7`, `Community 11`, `Community 12`, `Community 13`, `Community 16`, `Community 17`, `Community 18`, `Community 19`, `Community 20`, `Community 22`, `Community 23`, `Community 27`?**
  _High betweenness centrality (0.100) - this node is a cross-community bridge._
- **Are the 449 inferred relationships involving `doRequest()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`doRequest()` has 449 INFERRED edges - model-reasoned connections that need verification._
- **Are the 509 inferred relationships involving `New()` (e.g. with `TestRunReturnsConfigLoadError()` and `adminTempStore()`) actually correct?**
  _`New()` has 509 INFERRED edges - model-reasoned connections that need verification._
- **Are the 345 inferred relationships involving `newHandler()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newHandler()` has 345 INFERRED edges - model-reasoned connections that need verification._
- **Are the 331 inferred relationships involving `newFakeMealStore()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newFakeMealStore()` has 331 INFERRED edges - model-reasoned connections that need verification._