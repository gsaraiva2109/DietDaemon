# Graph Report - DietDaemon  (2026-08-09)

## Corpus Check
- 503 files · ~586,335 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 5618 nodes · 13066 edges · 77 communities detected
- Extraction: 61% EXTRACTED · 39% INFERRED · 0% AMBIGUOUS · INFERRED: 5064 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 81|Community 81]]
- [[_COMMUNITY_Community 83|Community 83]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 92|Community 92]]
- [[_COMMUNITY_Community 94|Community 94]]
- [[_COMMUNITY_Community 95|Community 95]]
- [[_COMMUNITY_Community 96|Community 96]]
- [[_COMMUNITY_Community 127|Community 127]]
- [[_COMMUNITY_Community 128|Community 128]]
- [[_COMMUNITY_Community 129|Community 129]]
- [[_COMMUNITY_Community 130|Community 130]]
- [[_COMMUNITY_Community 131|Community 131]]
- [[_COMMUNITY_Community 132|Community 132]]

## God Nodes (most connected - your core abstractions)
1. `doRequest()` - 601 edges
2. `New()` - 571 edges
3. `newHandler()` - 504 edges
4. `newFakeMealStore()` - 485 edges
5. `Store` - 278 edges
6. `Handler` - 257 edges
7. `contains()` - 225 edges
8. `ctx()` - 197 edges
9. `tempDB()` - 155 edges
10. `decodeJSON()` - 138 edges

## Surprising Connections (you probably didn't know these)
- `TestStreamChatHTTPError()` --calls--> `contains()`  [INFERRED]
  adapters/model/anthropic/chat_test.go → internal/auth/totp_test.go
- `TestPhotoPromptCallsOutMultiColumnCarbCycling()` --calls--> `contains()`  [INFERRED]
  adapters/model/planextract/planextract_test.go → internal/auth/totp_test.go
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
Nodes (403): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), targetsResponse, NewWebAuthn(), Stat() (+395 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (122): auditActor, chatStreamState, credCreateConfig, credRevokeConfig, customFoodRequest, dayOverrideBody, deleteAccountRequest, ErrorCode (+114 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (57): normalize(), FS(), Normalize(), Unaccent(), AccountDeletionStatus, backupConfigRow, catalogRow, credRow (+49 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (212): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), checkExtractMenuRequest(), checkExtractPlanRequest(), TestExtractLabel(), TestExtractLabelHTTPError(), TestExtractMenu() (+204 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (244): fakeAccount, fakeAuditEvent, fakeMailer, NewPurgeRunner(), TestPurgeAccountBackups_DeleteErrorStillPurgesAccount(), TestPurgeAccountBackups_DeletesFromConfiguredDestination(), TestPurgeAccountBackups_GetConfigErrorSkipsThatUser(), TestPurgeAccountBackups_ListUsersErrorStillPurgesAccount() (+236 more)

### Community 6 - "Community 6"
Cohesion: 0.01
Nodes (11): authHandlerTestStore, fakeAuthStore, fakeMealStore, mfaEmailTestStore, passkeyTestStore, totpTestStore, fakeBackupDest, fakePurgeStore (+3 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (150): Adapter, contentBlock, message, messagesRequest, messagesResponse, writeCSV(), Destination, Runner (+142 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (116): NewChatAdapter(), parseSSEEvent(), sendProviderError(), drainReadStream(), TestReadStreamContextCancelledMidStream(), TestReadStreamErrorEvent(), TestReadStreamMalformedJSONSkipped(), TestReadStreamNonDataLinesSkipped() (+108 more)

### Community 9 - "Community 9"
Cohesion: 0.03
Nodes (128): authTestStore, credAuthStore, emailTestAuthStore, emailToken, erroringCountAuthStore, fakeMailer, buildCredHandler(), TestCheckLoginLockoutConcurrentNoRace() (+120 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (45): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+37 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (66): addFood(), renderModal(), typeSearch(), renderModal(), renderModal(), dayLabel(), renderModal(), download() (+58 more)

### Community 12 - "Community 12"
Cohesion: 0.02
Nodes (67): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, calcSleepHours(), computeSleepDuration(), formatDuration(), parseQualityAndNote() (+59 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (100): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthRepos, AuthStore, BackupRunner, ChatStore (+92 more)

### Community 14 - "Community 14"
Cohesion: 0.02
Nodes (82): Config, addProblem(), parseProxyEntry(), validateBulkFile(), NotFound(), AuditEvent, BackupConfig, BodyCompositionSummary (+74 more)

### Community 15 - "Community 15"
Cohesion: 0.03
Nodes (59): appendedChatMessage, buildAdapterForProvider(), buildChatAdapterForProvider(), decryptAIKey(), assertBYOKFailure(), TestBuildBYOKAdaptersRejectUnsupportedProvider(), TestBYOKChatOverrideUsedInsteadOfSharedAdapter(), TestBYOKFailuresDoNotFallBackToSharedAdapters() (+51 more)

### Community 16 - "Community 16"
Cohesion: 0.03
Nodes (46): CorrectCommand, CorrectResolver, CorrectStore, parsePositiveFloat(), setProfileField(), ProfileCommand, ProfileStore, close() (+38 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (73): OpenSQLiteStore(), SignalContext(), TestOpenSQLiteStore(), TestSignalContextHonorsCanceledParent(), main(), run(), runBackfill(), runRepair() (+65 more)

### Community 18 - "Community 18"
Cohesion: 0.04
Nodes (44): classifyGoalDivergence(), newAuditID(), PurgeRunner, PurgeStore, New(), ChatRouteStore, ChatSender, DigestStore (+36 more)

### Community 19 - "Community 19"
Cohesion: 0.03
Nodes (31): NewCancelCommand(), CancelCommand, fakeMealStore, NewFoodCommand(), FoodCommand, FoodStore, NewLinkCommand(), LinkCodeStore (+23 more)

### Community 20 - "Community 20"
Cohesion: 0.04
Nodes (54): fakeCompletionAdapter, assertDayType(), assertDietboxDraft(), assertWeekdaySchedule(), derefOrNilStr(), itemCount(), mustReadFixture(), TestHandleExtractPlanFromImage_MultiPageFixtureRegression() (+46 more)

### Community 21 - "Community 21"
Cohesion: 0.04
Nodes (30): allEntitiesFakeStore, New(), newFakeStore(), TestRun_ChecksImmediatelyBeforeCancelledContextReturns(), TestRunFor_ExportsAllEntities(), TestRunFor_LoadErrorAbortsRemainingEntities(), TestRunFor_MissingDestinationErrors(), TestRunFor_PhotoBlobWrittenBeforeIndex() (+22 more)

### Community 22 - "Community 22"
Cohesion: 0.13
Nodes (61): TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(), TestHandleLoginMFAStepUp(), TestHandleLoginMFAStepUpChallengeCreationFails(), buildTOTPHandler(), defaultTOTPMeals(), enrollTOTPSecret(), newTOTPTestStore(), TestHandleRegenerateRecoveryHasConfirmedError() (+53 more)

### Community 23 - "Community 23"
Cohesion: 0.05
Nodes (36): Client, NewClient(), listResponse, captureHandler, Config, Mailer, New(), TestNew() (+28 more)

### Community 24 - "Community 24"
Cohesion: 0.06
Nodes (38): cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer(), newRequestID(), observeRequests(), recoverPanics() (+30 more)

### Community 25 - "Community 25"
Cohesion: 0.07
Nodes (45): actionRow, Adapter, buttonComponent, mustMarshal(), readGatewayPayload(), readWSFrame(), buildServerFrame(), genSelfSignedCert() (+37 more)

### Community 26 - "Community 26"
Cohesion: 0.14
Nodes (45): buildAuthSecurityHandler(), TestHandleLoginUnknownEmailStillHashes(), TestHandleRegisterDuplicateEmailSkipsHash(), doOIDCCallback(), locationParams(), newTestIdP(), oidcCallbackRequest(), oidcHandler() (+37 more)

### Community 27 - "Community 27"
Cohesion: 0.06
Nodes (31): food, foodCategory, foodNutrient, foodPortion, searchResponse, Source, bulkDataTypes(), emitMatchedFood() (+23 more)

### Community 28 - "Community 28"
Cohesion: 0.12
Nodes (46): doPasskeyLoginFinish(), mfaPasskeyBeginExpiredChallenge(), mfaPasskeyBeginInvalidJSON(), mfaPasskeyBeginMissingToken(), mfaPasskeyBeginNoPasskeysRegistered(), mfaPasskeyBeginSuccess(), mfaPasskeyBeginUnknownChallenge(), mfaPasskeyFinishCeremonyConsumeFails() (+38 more)

### Community 29 - "Community 29"
Cohesion: 0.08
Nodes (48): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+40 more)

### Community 30 - "Community 30"
Cohesion: 0.07
Nodes (32): sendOut(), sendSuggestions(), assertDoneEvent(), assertTextDeltaEvent(), assertToolCallEvent(), assertToolResultEvent(), collectEvents(), TestRouterContextCancellation() (+24 more)

### Community 31 - "Community 31"
Cohesion: 0.07
Nodes (31): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), IsUnit() (+23 more)

### Community 32 - "Community 32"
Cohesion: 0.15
Nodes (14): fakeCmd, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape(), TestHelpCommand_ListAll() (+6 more)

### Community 33 - "Community 33"
Cohesion: 0.19
Nodes (17): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+9 more)

### Community 34 - "Community 34"
Cohesion: 0.21
Nodes (17): fakeFoodImportRunner, doAdminRequest(), newAdminTestHandler(), TestAdminFoodImport_BackfillEmbeddings200(), TestAdminFoodImport_BackfillEmbeddingsError(), TestAdminFoodImport_MissingToken401(), TestAdminFoodImport_Repair200(), TestAdminFoodImport_RepairError() (+9 more)

### Community 35 - "Community 35"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 36 - "Community 36"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 37 - "Community 37"
Cohesion: 0.2
Nodes (7): Embedder, fingerprintStore, localFingerprint(), New(), Runner, SourceFactory, Store

### Community 38 - "Community 38"
Cohesion: 0.27
Nodes (11): addSortIndicators(), enableUI(), getNthColumn(), getTable(), getTableBody(), getTableHeader(), loadColumns(), loadData() (+3 more)

### Community 39 - "Community 39"
Cohesion: 0.23
Nodes (10): failingReader, cryptoRand5Digits(), GenerateRecoveryCodes(), TestCryptoRand5DigitsPanicsOnRandFailure(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount() (+2 more)

### Community 40 - "Community 40"
Cohesion: 0.17
Nodes (1): fakeStore

### Community 41 - "Community 41"
Cohesion: 0.26
Nodes (7): appendDelta(), appendToolCall(), applyStreamEvent(), applySuggestions(), applyToolResult(), raiseStreamError(), stripSuggestionsFence()

### Community 42 - "Community 42"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 43 - "Community 43"
Cohesion: 0.36
Nodes (9): bundle(), bundleWithOption(), bundleWithSlot(), dayType(), mealTemplate(), option(), plan(), resolvedItem() (+1 more)

### Community 44 - "Community 44"
Cohesion: 0.35
Nodes (8): a(), B(), D(), g(), i(), k(), Q(), y()

### Community 45 - "Community 45"
Cohesion: 0.27
Nodes (7): fakeVisionAdapter, doOCRUpload(), TestHandleOCRExtractCustomFood(), TestHandleOCRExtractCustomFoodAdapterError(), TestHandleOCRExtractCustomFoodDisabled(), TestHandleOCRExtractCustomFoodMissingFile(), TestHandleOCRExtractCustomFoodNonImage()

### Community 46 - "Community 46"
Cohesion: 0.24
Nodes (5): writeCandidates(), writeOtherOptions(), SuggestCommand, SuggestEngine, SuggestFoodSearcher

### Community 47 - "Community 47"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 49 - "Community 49"
Cohesion: 0.22
Nodes (2): blockingStore, fakeStore

### Community 50 - "Community 50"
Cohesion: 0.28
Nodes (3): parseWorkoutArgs(), WorkoutCommand, WorkoutStore

### Community 51 - "Community 51"
Cohesion: 0.36
Nodes (1): Store

### Community 52 - "Community 52"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 57 - "Community 57"
Cohesion: 0.38
Nodes (4): fakeResponse(), runOptions(), streamOf(), userMessage()

### Community 59 - "Community 59"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 60 - "Community 60"
Cohesion: 0.29
Nodes (1): stubStore

### Community 61 - "Community 61"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 62 - "Community 62"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 63 - "Community 63"
Cohesion: 0.4
Nodes (2): bundleWith(), plan()

### Community 65 - "Community 65"
Cohesion: 0.33
Nodes (1): fakeHealthStore

### Community 66 - "Community 66"
Cohesion: 0.5
Nodes (2): focusSoon(), resetSearchState()

### Community 70 - "Community 70"
Cohesion: 0.7
Nodes (4): goToNext(), goToPrevious(), makeCurrent(), toggleClass()

### Community 71 - "Community 71"
Cohesion: 0.4
Nodes (1): fakeMealLogger

### Community 72 - "Community 72"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 73 - "Community 73"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 81 - "Community 81"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 83 - "Community 83"
Cohesion: 0.5
Nodes (1): crossUserPhotoStore

### Community 84 - "Community 84"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 92 - "Community 92"
Cohesion: 1.0
Nodes (2): chunk(), solidPng()

### Community 94 - "Community 94"
Cohesion: 0.67
Nodes (1): Memory

### Community 95 - "Community 95"
Cohesion: 0.67
Nodes (2): oidcCallbackContext, oidcIdentity

### Community 96 - "Community 96"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 127 - "Community 127"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

### Community 128 - "Community 128"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 129 - "Community 129"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 130 - "Community 130"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 131 - "Community 131"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 132 - "Community 132"
Cohesion: 1.0
Nodes (1): VisionAdapter

## Knowledge Gaps
- **361 isolated node(s):** `appRuntime`, `phraseEntry`, `mealSaver`, `Row`, `HevyWorkout` (+356 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 40`** (13 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SaveMealAndAddToRollup()`, `.SetTargets()`, `.TargetsFor()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (10 nodes): `blockingStore`, `.GetRollup()`, `.GetTargets()`, `.ListUsers()`, `.TargetsFor()`, `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.ListUsers()`, `.TargetsFor()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (6 nodes): `bundleWith()`, `meal()`, `noPlanView()`, `plan()`, `planDayWithSlots()`, `Dashboard.test.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (6 nodes): `fakeHealthStore`, `.GetActiveFast()`, `.GetActiveSleep()`, `.GetWaterToday()`, `.ListFasts()`, `.ListWorkouts()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (5 nodes): `focusSoon()`, `onKey()`, `onListKey()`, `resetSearchState()`, `CommandPalette.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (5 nodes): `fakeMealLogger`, `.Handle()`, `.LogMeal()`, `.LogMealFromItems()`, `.ParseAndResolve()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (4 nodes): `crossUserPhotoStore`, `.DeletePhoto()`, `.GetPhotoData()`, `.ListPhotoMetadata()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 92`** (3 nodes): `chunk()`, `solidPng()`, `generate-page-pngs.mjs`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 94`** (3 nodes): `queue.go`, `Memory`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 95`** (3 nodes): `oidcCallbackContext`, `oidcIdentity`, `handler_oidc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 96`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 127`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 128`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 129`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 130`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 131`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 132`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 11`, `Community 12`, `Community 13`, `Community 15`, `Community 16`, `Community 17`, `Community 19`, `Community 20`, `Community 21`, `Community 22`, `Community 23`, `Community 24`, `Community 25`, `Community 26`, `Community 27`, `Community 28`, `Community 30`, `Community 34`, `Community 39`, `Community 45`, `Community 50`?**
  _High betweenness centrality (0.421) - this node is a cross-community bridge._
- **Why does `newHandler()` connect `Community 0` to `Community 1`, `Community 34`, `Community 9`, `Community 45`, `Community 15`, `Community 20`, `Community 22`, `Community 26`?**
  _High betweenness centrality (0.116) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 4` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 5`, `Community 7`, `Community 8`, `Community 9`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 17`, `Community 19`, `Community 20`, `Community 21`, `Community 22`, `Community 24`, `Community 25`, `Community 26`, `Community 30`, `Community 31`, `Community 32`, `Community 39`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Are the 485 inferred relationships involving `doRequest()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`doRequest()` has 485 INFERRED edges - model-reasoned connections that need verification._
- **Are the 566 inferred relationships involving `New()` (e.g. with `TestRunReturnsConfigLoadError()` and `adminTempStore()`) actually correct?**
  _`New()` has 566 INFERRED edges - model-reasoned connections that need verification._
- **Are the 383 inferred relationships involving `newHandler()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newHandler()` has 383 INFERRED edges - model-reasoned connections that need verification._
- **Are the 365 inferred relationships involving `newFakeMealStore()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newFakeMealStore()` has 365 INFERRED edges - model-reasoned connections that need verification._