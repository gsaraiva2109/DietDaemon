# Graph Report - agent-a80c7957d6d7bd729  (2026-07-26)

## Corpus Check
- 438 files · ~377,712 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 4756 nodes · 10855 edges · 62 communities detected
- Extraction: 63% EXTRACTED · 37% INFERRED · 0% AMBIGUOUS · INFERRED: 4070 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 74|Community 74]]
- [[_COMMUNITY_Community 75|Community 75]]
- [[_COMMUNITY_Community 100|Community 100]]
- [[_COMMUNITY_Community 101|Community 101]]
- [[_COMMUNITY_Community 102|Community 102]]
- [[_COMMUNITY_Community 103|Community 103]]
- [[_COMMUNITY_Community 105|Community 105]]
- [[_COMMUNITY_Community 107|Community 107]]

## God Nodes (most connected - your core abstractions)
1. `doRequest()` - 523 edges
2. `New()` - 485 edges
3. `newFakeMealStore()` - 417 edges
4. `newHandler()` - 407 edges
5. `Store` - 252 edges
6. `Handler` - 240 edges
7. `contains()` - 182 edges
8. `ctx()` - 124 edges
9. `decodeJSON()` - 117 edges
10. `fakeMealStore` - 111 edges

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
Cohesion: 0.02
Nodes (433): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), fakeMealLogger, fakeSuggester, fakeVisionAdapter, newHandlerWithAccountStore(), TestHandleDeleteAccountClearsSessionCookie(), TestHandleDeleteAccountMissingBody() (+425 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (301): TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), TestExpensiveRequestRoutes(), assertDoneEvent(), assertTextDeltaEvent(), assertToolCallEvent(), assertToolResultEvent(), collectEvents() (+293 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (116): chatStreamState, credCreateConfig, credRevokeConfig, customFoodRequest, dayOverrideBody, deleteAccountRequest, ErrorCode, errorEnvelope (+108 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (50): FS(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, dayTypeRow (+42 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (185): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestExtractLabel(), TestExtractLabelHTTPError(), NewCancelCommand(), CancelCommand, NewCorrectCommand() (+177 more)

### Community 5 - "Community 5"
Cohesion: 0.01
Nodes (5): fakeAuthStore, fakeMealStore, passkeyTestStore, totpTestStore, Store

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (135): extractArgs(), NewChatAdapter(), parseSSEEvent(), sendEvent(), sendProviderError(), drainReadStream(), TestExtractArgsEmptyValue(), TestReadStreamContextCancelledMidStream() (+127 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (132): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthRepos, AuthStore, authTestStore, BackupRunner (+124 more)

### Community 8 - "Community 8"
Cohesion: 0.06
Nodes (135): IPRateLimiter, TestPendingStoreContract(), postgresDB(), TestFoodImportFingerprintStore(), TestPostgresDietPlanSmoke(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods() (+127 more)

### Community 9 - "Community 9"
Cohesion: 0.04
Nodes (120): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, startBackgroundServices(), blockingStore, DigestRule, fakeChatRouteStore, fakeChatSender (+112 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (57): ProtectedRoute(), ApiError, blobRequest(), buildRequestHeaders(), handleUnauthorized(), multipart(), parseErrorBody(), RateLimitError (+49 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (86): Adapter, contentBlock, message, messagesRequest, messagesResponse, writeCSV(), capturingDest, Destination (+78 more)

### Community 12 - "Community 12"
Cohesion: 0.02
Nodes (59): Registry, confirmReplace(), scaledMacros(), sourceLabel(), renderModal(), dayLabel(), download(), sourceLabel() (+51 more)

### Community 13 - "Community 13"
Cohesion: 0.02
Nodes (81): Stat(), Config, addProblem(), parseProxyEntry(), validateBulkFile(), Embedder, fingerprintStore, localFingerprint() (+73 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (54): CorrectCommand, CorrectResolver, CorrectStore, MealStore, parsePositiveFloat(), setProfileField(), ProfileCommand, ProfileStore (+46 more)

### Community 15 - "Community 15"
Cohesion: 0.06
Nodes (84): authHandlerTestStore, credAuthStore, erroringCountAuthStore, buildCredHandler(), TestCheckLoginLockoutLocked(), TestCheckLoginLockoutStoreError(), TestHandleChangePasswordInvalidJSON(), TestHandleChangePasswordMissingFields() (+76 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (59): NewWebAuthnHandle(), cors(), corsOriginAllowed(), limitRequestBody(), newHTTPHandler(), newHTTPServer(), newRequestID(), observeRequests() (+51 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (53): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+45 more)

### Community 18 - "Community 18"
Cohesion: 0.05
Nodes (61): fakeCmd, NewHelpCommand(), buildTestBundle(), mustRegister(), TestHelpCommand_Detail(), TestHelpCommand_FallbackLocale(), TestHelpCommand_HTMLEscape(), TestHelpCommand_ListAll() (+53 more)

### Community 19 - "Community 19"
Cohesion: 0.04
Nodes (37): appendedChatMessage, buildAdapterForProvider(), buildChatAdapterForProvider(), decryptAIKey(), assertBYOKFailure(), TestBuildBYOKAdaptersRejectUnsupportedProvider(), TestBYOKChatOverrideUsedInsteadOfSharedAdapter(), TestBYOKFailuresDoNotFallBackToSharedAdapters() (+29 more)

### Community 20 - "Community 20"
Cohesion: 0.05
Nodes (49): adminTempStore(), TestFoodImportAdmin_ImportSource_MaxRowsCap(), TestFoodImportAdmin_ImportSource_TACO(), TestFoodImportAdmin_ImportSource_UnknownSource(), TestFoodImportAdmin_RepairSource(), groupIntoMeals(), importMeals(), main() (+41 more)

### Community 21 - "Community 21"
Cohesion: 0.13
Nodes (61): TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(), TestHandleLoginMFAStepUp(), TestHandleLoginMFAStepUpChallengeCreationFails(), buildTOTPHandler(), defaultTOTPMeals(), enrollTOTPSecret(), newTOTPTestStore(), TestHandleRegenerateRecoveryHasConfirmedError() (+53 more)

### Community 22 - "Community 22"
Cohesion: 0.06
Nodes (46): actionRow, Adapter, buttonComponent, mustMarshal(), readGatewayPayload(), readWSFrame(), buildServerFrame(), genSelfSignedCert() (+38 more)

### Community 23 - "Community 23"
Cohesion: 0.05
Nodes (23): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+15 more)

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
Cohesion: 0.08
Nodes (18): findTemplateByName(), macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore, hostFromBaseURL(), entry (+10 more)

### Community 28 - "Community 28"
Cohesion: 0.14
Nodes (20): isMutating(), fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession() (+12 more)

### Community 29 - "Community 29"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 30 - "Community 30"
Cohesion: 0.11
Nodes (14): fakeFoodSearcher, fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_IngredientArgsResolveViaSearch(), TestSuggestCommand_IngredientArgsSkipUnresolvedNames() (+6 more)

### Community 31 - "Community 31"
Cohesion: 0.21
Nodes (17): fakeFoodImportRunner, doAdminRequest(), newAdminTestHandler(), TestAdminFoodImport_BackfillEmbeddings200(), TestAdminFoodImport_BackfillEmbeddingsError(), TestAdminFoodImport_MissingToken401(), TestAdminFoodImport_Repair200(), TestAdminFoodImport_RepairError() (+9 more)

### Community 32 - "Community 32"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 33 - "Community 33"
Cohesion: 0.13
Nodes (1): fakeStore

### Community 34 - "Community 34"
Cohesion: 0.18
Nodes (7): fakePurgeStore, NewPurgeRunner(), TestPurgeRunnerContextCancel(), TestPurgeRunnerTicksAndPurges(), TestPurgeRunnerZeroPurged(), PurgeRunner, PurgeStore

### Community 35 - "Community 35"
Cohesion: 0.26
Nodes (7): appendDelta(), appendToolCall(), applyStreamEvent(), applySuggestions(), applyToolResult(), raiseStreamError(), stripSuggestionsFence()

### Community 36 - "Community 36"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 37 - "Community 37"
Cohesion: 0.18
Nodes (1): fakeStore

### Community 38 - "Community 38"
Cohesion: 0.18
Nodes (1): allEntitiesFakeStore

### Community 39 - "Community 39"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 41 - "Community 41"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 42 - "Community 42"
Cohesion: 0.29
Nodes (4): IDTokenClaims, initResult, Provider, ProviderConfig

### Community 43 - "Community 43"
Cohesion: 0.36
Nodes (1): Store

### Community 44 - "Community 44"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 49 - "Community 49"
Cohesion: 0.38
Nodes (4): fakeResponse(), runOptions(), streamOf(), userMessage()

### Community 51 - "Community 51"
Cohesion: 0.29
Nodes (1): stubStore

### Community 52 - "Community 52"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 53 - "Community 53"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 56 - "Community 56"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 57 - "Community 57"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 63 - "Community 63"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 64 - "Community 64"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 66 - "Community 66"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 69 - "Community 69"
Cohesion: 1.0
Nodes (2): gramsFor(), unitOptionsFor()

### Community 74 - "Community 74"
Cohesion: 0.67
Nodes (2): oidcCallbackContext, oidcIdentity

### Community 75 - "Community 75"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 100 - "Community 100"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

### Community 101 - "Community 101"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 102 - "Community 102"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 103 - "Community 103"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (1): VisionAdapter

## Knowledge Gaps
- **338 isolated node(s):** `appRuntime`, `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row` (+333 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 33`** (15 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotosData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (11 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (11 nodes): `allEntitiesFakeStore`, `.GetMealsInRange()`, `.GetPhotosData()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListSleep()`, `.ListWeight()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 64`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (3 nodes): `gramsFor()`, `unitOptionsFor()`, `servingUnits.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (3 nodes): `oidcCallbackContext`, `oidcIdentity`, `handler_oidc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 100`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 101`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 102`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 103`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 11`, `Community 12`, `Community 14`, `Community 15`, `Community 18`, `Community 19`, `Community 20`, `Community 21`, `Community 22`, `Community 25`, `Community 26`, `Community 30`, `Community 31`?**
  _High betweenness centrality (0.452) - this node is a cross-community bridge._
- **Why does `Store` connect `Community 3` to `Community 16`, `Community 2`, `Community 6`?**
  _High betweenness centrality (0.107) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 2` to `Community 32`, `Community 4`, `Community 7`, `Community 9`, `Community 11`, `Community 16`, `Community 19`, `Community 28`?**
  _High betweenness centrality (0.090) - this node is a cross-community bridge._
- **Are the 407 inferred relationships involving `doRequest()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`doRequest()` has 407 INFERRED edges - model-reasoned connections that need verification._
- **Are the 480 inferred relationships involving `New()` (e.g. with `TestRunReturnsConfigLoadError()` and `adminTempStore()`) actually correct?**
  _`New()` has 480 INFERRED edges - model-reasoned connections that need verification._
- **Are the 297 inferred relationships involving `newFakeMealStore()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newFakeMealStore()` has 297 INFERRED edges - model-reasoned connections that need verification._
- **Are the 286 inferred relationships involving `newHandler()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newHandler()` has 286 INFERRED edges - model-reasoned connections that need verification._