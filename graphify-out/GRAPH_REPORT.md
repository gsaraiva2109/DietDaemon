# Graph Report - DietDaemon  (2026-07-31)

## Corpus Check
- 480 files · ~661,187 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 5343 nodes · 12329 edges · 70 communities detected
- Extraction: 62% EXTRACTED · 38% INFERRED · 0% AMBIGUOUS · INFERRED: 4729 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 83|Community 83]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 114|Community 114]]
- [[_COMMUNITY_Community 115|Community 115]]
- [[_COMMUNITY_Community 116|Community 116]]
- [[_COMMUNITY_Community 117|Community 117]]
- [[_COMMUNITY_Community 119|Community 119]]
- [[_COMMUNITY_Community 121|Community 121]]

## God Nodes (most connected - your core abstractions)
1. `doRequest()` - 578 edges
2. `New()` - 533 edges
3. `newHandler()` - 470 edges
4. `newFakeMealStore()` - 467 edges
5. `Store` - 271 edges
6. `Handler` - 250 edges
7. `contains()` - 213 edges
8. `ctx()` - 188 edges
9. `tempDB()` - 145 edges
10. `decodeJSON()` - 123 edges

## Surprising Connections (you probably didn't know these)
- `TestAPIRouteFallbackUsesErrorEnvelope()` --calls--> `New()`  [INFERRED]
  internal/api/errors_test.go → adapters/nutrition/taco/taco.go
- `TestHandleListChatSessionsError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go
- `TestHandleGetChatMessagesError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go
- `TestHandleGetChatSettingsError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go
- `TestHandleSetChatSettingsError()` --calls--> `New()`  [INFERRED]
  internal/api/handler_chat_test.go → adapters/nutrition/taco/taco.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (507): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), fakeCompletionAdapter, fakeMealLogger, fakeSuggester, deleteAccountTestHandler(), newHandlerWithAccountStore(), TestHandleDeleteAccountClearsSessionCookie() (+499 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (355): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), TestExpensiveRequestRoutes(), targetsResponse, assertDoneEvent() (+347 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (135): chatStreamState, credCreateConfig, credRevokeConfig, customFoodRequest, dayOverrideBody, deleteAccountRequest, ErrorCode, errorEnvelope (+127 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (56): FS(), Normalize(), TestNormalize(), unaccent(), AccountDeletionStatus, backupConfigRow, catalogRow, credRow (+48 more)

### Community 4 - "Community 4"
Cohesion: 0.01
Nodes (160): Adapter, contentBlock, message, messagesRequest, messagesResponse, allEntitiesFakeStore, New(), newFakeStore() (+152 more)

### Community 5 - "Community 5"
Cohesion: 0.02
Nodes (193): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), checkExtractPlanRequest(), TestExtractLabel(), TestExtractLabelHTTPError(), TestExtractPlan(), TestExtractPlanHTTPError() (+185 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (224): fakeAccount, fakeAuditEvent, fakeMailer, NewPurgeRunner(), TestPurgeAccountPhotosListError(), TestPurgeAccountPhotosPerAccountErrorContinues(), TestPurgeAccountsListError(), TestPurgeAccountsPerAccountErrorContinues() (+216 more)

### Community 7 - "Community 7"
Cohesion: 0.01
Nodes (8): authHandlerTestStore, fakeAuthStore, fakeMealStore, mfaEmailTestStore, passkeyTestStore, totpTestStore, fakePurgeStore, Store

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (135): extractArgs(), NewChatAdapter(), parseSSEEvent(), sendEvent(), sendProviderError(), drainReadStream(), TestExtractArgsEmptyValue(), TestReadStreamContextCancelledMidStream() (+127 more)

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (143): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthRepos, AuthStore, BackupRunner, ChatStore (+135 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (65): addFood(), renderModal(), typeSearch(), renderModal(), renderModal(), dayLabel(), renderModal(), download() (+57 more)

### Community 12 - "Community 12"
Cohesion: 0.02
Nodes (86): Stat(), Config, addProblem(), parseProxyEntry(), validateBulkFile(), Embedder, fingerprintStore, localFingerprint() (+78 more)

### Community 13 - "Community 13"
Cohesion: 0.04
Nodes (56): fakeCmd, fakeMealStore, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape() (+48 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (59): appendedChatMessage, buildAdapterForProvider(), buildChatAdapterForProvider(), decryptAIKey(), assertBYOKFailure(), TestBuildBYOKAdaptersRejectUnsupportedProvider(), TestBYOKChatOverrideUsedInsteadOfSharedAdapter(), TestBYOKFailuresDoNotFallBackToSharedAdapters() (+51 more)

### Community 15 - "Community 15"
Cohesion: 0.07
Nodes (83): credAuthStore, erroringCountAuthStore, buildCredHandler(), TestCheckLoginLockoutLocked(), TestCheckLoginLockoutStoreError(), TestHandleChangePasswordInvalidJSON(), TestHandleChangePasswordMissingFields(), TestHandleChangePasswordNewPasswordTooShort() (+75 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (50): NewWebAuthnHandle(), randomID(), calcSleepHours(), computeSleepDuration(), formatDuration(), parseQualityAndNote(), TestCalcSleepHours_NilWake(), TestCalcSleepHours_Overnight() (+42 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (75): adminTempStore(), TestFoodImportAdmin_ImportSource_MaxRowsCap(), TestFoodImportAdmin_ImportSource_TACO(), TestFoodImportAdmin_ImportSource_UnknownSource(), TestFoodImportAdmin_RepairSource(), bulkUpserter, main(), run() (+67 more)

### Community 18 - "Community 18"
Cohesion: 0.04
Nodes (44): CorrectCommand, CorrectResolver, CorrectStore, parsePositiveFloat(), setProfileField(), ProfileCommand, ProfileStore, close() (+36 more)

### Community 19 - "Community 19"
Cohesion: 0.03
Nodes (39): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, confirmReplace(), scaledMacros(), sourceLabel(), MacroBar() (+31 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (48): authTestStore, emailTestAuthStore, emailToken, fakeMailer, containsStr(), TestHandleEmailChangeConflict(), TestHandleEmailChangeCreateEmailTokenError(), TestHandleEmailChangeGetPasswordHashError() (+40 more)

### Community 21 - "Community 21"
Cohesion: 0.05
Nodes (36): fakePlanStore, findSlotByLabel(), NewPlanCommand(), nextSlotID(), optionSummary(), parseTimeOfDay(), slotsForDayType(), bundleWithSlots() (+28 more)

### Community 22 - "Community 22"
Cohesion: 0.13
Nodes (61): TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(), TestHandleLoginMFAStepUp(), TestHandleLoginMFAStepUpChallengeCreationFails(), buildTOTPHandler(), defaultTOTPMeals(), enrollTOTPSecret(), newTOTPTestStore(), TestHandleRegenerateRecoveryHasConfirmedError() (+53 more)

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
Cohesion: 0.07
Nodes (31): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), IsUnit() (+23 more)

### Community 28 - "Community 28"
Cohesion: 0.09
Nodes (28): cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer(), newRequestID(), observeRequests(), recoverPanics() (+20 more)

### Community 29 - "Community 29"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 30 - "Community 30"
Cohesion: 0.17
Nodes (17): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+9 more)

### Community 31 - "Community 31"
Cohesion: 0.11
Nodes (14): fakeFoodSearcher, fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_IngredientArgsResolveViaSearch(), TestSuggestCommand_IngredientArgsSkipUnresolvedNames() (+6 more)

### Community 32 - "Community 32"
Cohesion: 0.21
Nodes (17): fakeFoodImportRunner, doAdminRequest(), newAdminTestHandler(), TestAdminFoodImport_BackfillEmbeddings200(), TestAdminFoodImport_BackfillEmbeddingsError(), TestAdminFoodImport_MissingToken401(), TestAdminFoodImport_Repair200(), TestAdminFoodImport_RepairError() (+9 more)

### Community 33 - "Community 33"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 34 - "Community 34"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 35 - "Community 35"
Cohesion: 0.27
Nodes (11): addSortIndicators(), enableUI(), getNthColumn(), getTable(), getTableBody(), getTableHeader(), loadColumns(), loadData() (+3 more)

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
Cohesion: 0.35
Nodes (8): a(), B(), D(), g(), i(), k(), Q(), y()

### Community 40 - "Community 40"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 41 - "Community 41"
Cohesion: 0.31
Nodes (7): loadPdfjs(), pdfToImages(), renderPage(), handleFile(), onExtracted(), onFailed(), t()

### Community 43 - "Community 43"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 44 - "Community 44"
Cohesion: 0.31
Nodes (7): fakeVisionAdapter, doOCRUpload(), TestHandleOCRExtractCustomFood(), TestHandleOCRExtractCustomFoodAdapterError(), TestHandleOCRExtractCustomFoodDisabled(), TestHandleOCRExtractCustomFoodMissingFile(), TestHandleOCRExtractCustomFoodNonImage()

### Community 45 - "Community 45"
Cohesion: 0.29
Nodes (4): IDTokenClaims, initResult, Provider, ProviderConfig

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
Nodes (1): crossUserPhotoStore

### Community 76 - "Community 76"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 83 - "Community 83"
Cohesion: 0.67
Nodes (2): oidcCallbackContext, oidcIdentity

### Community 84 - "Community 84"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 114 - "Community 114"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

### Community 115 - "Community 115"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 116 - "Community 116"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 117 - "Community 117"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 119 - "Community 119"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 121 - "Community 121"
Cohesion: 1.0
Nodes (1): VisionAdapter

## Knowledge Gaps
- **354 isolated node(s):** `appRuntime`, `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row` (+349 more)
  These have ≤1 connection - possible missing edges or undocumented components.
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
- **Thin community `Community 75`** (4 nodes): `crossUserPhotoStore`, `.DeletePhoto()`, `.GetPhotoData()`, `.ListPhotoMetadata()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (3 nodes): `oidcCallbackContext`, `oidcIdentity`, `handler_oidc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 114`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 115`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 116`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 117`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 119`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 121`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 11`, `Community 13`, `Community 14`, `Community 15`, `Community 17`, `Community 18`, `Community 20`, `Community 22`, `Community 23`, `Community 25`, `Community 26`, `Community 28`, `Community 31`, `Community 32`, `Community 44`?**
  _High betweenness centrality (0.429) - this node is a cross-community bridge._
- **Why does `newHandler()` connect `Community 0` to `Community 32`, `Community 1`, `Community 9`, `Community 44`, `Community 14`, `Community 15`, `Community 22`?**
  _High betweenness centrality (0.110) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 2` to `Community 33`, `Community 1`, `Community 4`, `Community 9`, `Community 14`, `Community 16`?**
  _High betweenness centrality (0.103) - this node is a cross-community bridge._
- **Are the 462 inferred relationships involving `doRequest()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`doRequest()` has 462 INFERRED edges - model-reasoned connections that need verification._
- **Are the 528 inferred relationships involving `New()` (e.g. with `TestRunReturnsConfigLoadError()` and `adminTempStore()`) actually correct?**
  _`New()` has 528 INFERRED edges - model-reasoned connections that need verification._
- **Are the 349 inferred relationships involving `newHandler()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newHandler()` has 349 INFERRED edges - model-reasoned connections that need verification._
- **Are the 347 inferred relationships involving `newFakeMealStore()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newFakeMealStore()` has 347 INFERRED edges - model-reasoned connections that need verification._