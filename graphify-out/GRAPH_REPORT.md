# Graph Report - .  (2026-07-27)

## Corpus Check
- Large corpus: 557 files · ~608,047 words. Semantic extraction will be expensive (many Claude tokens). Consider running on a subfolder, or use --no-semantic to run AST-only.

## Summary
- 5117 nodes · 11731 edges · 63 communities detected
- Extraction: 62% EXTRACTED · 38% INFERRED · 0% AMBIGUOUS · INFERRED: 4435 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 104|Community 104]]
- [[_COMMUNITY_Community 105|Community 105]]
- [[_COMMUNITY_Community 106|Community 106]]
- [[_COMMUNITY_Community 107|Community 107]]
- [[_COMMUNITY_Community 109|Community 109]]
- [[_COMMUNITY_Community 111|Community 111]]

## God Nodes (most connected - your core abstractions)
1. `doRequest()` - 565 edges
2. `New()` - 510 edges
3. `newHandler()` - 455 edges
4. `newFakeMealStore()` - 451 edges
5. `Store` - 260 edges
6. `Handler` - 247 edges
7. `contains()` - 199 edges
8. `ctx()` - 164 edges
9. `tempDB()` - 125 edges
10. `decodeJSON()` - 120 edges

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
Nodes (482): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), fakeMealLogger, fakeSuggester, fakeVisionAdapter, newHandlerWithAccountStore(), TestHandleDeleteAccountClearsSessionCookie(), TestHandleDeleteAccountMissingBody() (+474 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (298): TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), TestExpensiveRequestRoutes(), assertDoneEvent(), assertTextDeltaEvent(), assertToolCallEvent(), assertToolResultEvent(), collectEvents() (+290 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (123): chatStreamState, credCreateConfig, credRevokeConfig, customFoodRequest, dayOverrideBody, deleteAccountRequest, ErrorCode, errorEnvelope (+115 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (52): FS(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, dayTypeRow (+44 more)

### Community 4 - "Community 4"
Cohesion: 0.01
Nodes (101): Registry, renderModal(), dayLabel(), download(), sourceLabel(), onSubmit(), onAdd(), relativeCaption() (+93 more)

### Community 5 - "Community 5"
Cohesion: 0.02
Nodes (198): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), TestExtractLabel(), TestExtractLabelHTTPError(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement() (+190 more)

### Community 6 - "Community 6"
Cohesion: 0.01
Nodes (8): authHandlerTestStore, emailTestAuthStore, fakeAuthStore, fakeMealStore, mfaEmailTestStore, passkeyTestStore, totpTestStore, Store

### Community 7 - "Community 7"
Cohesion: 0.01
Nodes (163): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthRepos, AuthStore, BackupRunner, ChatStore (+155 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (135): extractArgs(), NewChatAdapter(), parseSSEEvent(), sendEvent(), sendProviderError(), drainReadStream(), TestExtractArgsEmptyValue(), TestReadStreamContextCancelledMidStream() (+127 more)

### Community 9 - "Community 9"
Cohesion: 0.04
Nodes (178): IPRateLimiter, TestPendingStoreContract(), postgresDB(), TestFoodImportFingerprintStore(), TestPostgresDietPlanSmoke(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods() (+170 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (101): Adapter, contentBlock, message, messagesRequest, messagesResponse, writeCSV(), capturingDest, Destination (+93 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (120): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, targetsResponse, blockingStore, DigestRule, fakeChatRouteStore, fakeChatSender (+112 more)

### Community 12 - "Community 12"
Cohesion: 0.04
Nodes (118): authTestStore, credAuthStore, emailToken, erroringCountAuthStore, fakeMailer, buildCredHandler(), TestCheckLoginLockoutLocked(), TestCheckLoginLockoutStoreError() (+110 more)

### Community 13 - "Community 13"
Cohesion: 0.02
Nodes (86): Stat(), Config, addProblem(), parseProxyEntry(), validateBulkFile(), Embedder, fingerprintStore, localFingerprint() (+78 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (52): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+44 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (44): CorrectCommand, CorrectResolver, CorrectStore, parsePositiveFloat(), setProfileField(), ProfileCommand, ProfileStore, close() (+36 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (64): adminTempStore(), TestFoodImportAdmin_ImportSource_MaxRowsCap(), TestFoodImportAdmin_ImportSource_TACO(), TestFoodImportAdmin_ImportSource_UnknownSource(), TestFoodImportAdmin_RepairSource(), groupIntoMeals(), importMeals(), main() (+56 more)

### Community 17 - "Community 17"
Cohesion: 0.04
Nodes (53): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+45 more)

### Community 18 - "Community 18"
Cohesion: 0.05
Nodes (38): NewWebAuthnHandle(), randomID(), fakeSource, ChatRouteStore, ChatSender, DigestStore, HealthStore, learnedSchedule (+30 more)

### Community 19 - "Community 19"
Cohesion: 0.04
Nodes (37): appendedChatMessage, buildAdapterForProvider(), buildChatAdapterForProvider(), decryptAIKey(), assertBYOKFailure(), TestBuildBYOKAdaptersRejectUnsupportedProvider(), TestBYOKChatOverrideUsedInsteadOfSharedAdapter(), TestBYOKFailuresDoNotFallBackToSharedAdapters() (+29 more)

### Community 20 - "Community 20"
Cohesion: 0.05
Nodes (36): fakePlanStore, findSlotByLabel(), NewPlanCommand(), nextSlotID(), optionSummary(), parseTimeOfDay(), slotsForDayType(), bundleWithSlots() (+28 more)

### Community 21 - "Community 21"
Cohesion: 0.13
Nodes (61): TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(), TestHandleLoginMFAStepUp(), TestHandleLoginMFAStepUpChallengeCreationFails(), buildTOTPHandler(), defaultTOTPMeals(), enrollTOTPSecret(), newTOTPTestStore(), TestHandleRegenerateRecoveryHasConfirmedError() (+53 more)

### Community 22 - "Community 22"
Cohesion: 0.06
Nodes (46): actionRow, Adapter, buttonComponent, mustMarshal(), readGatewayPayload(), readWSFrame(), buildServerFrame(), genSelfSignedCert() (+38 more)

### Community 23 - "Community 23"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 24 - "Community 24"
Cohesion: 0.13
Nodes (45): buildAuthSecurityHandler(), TestHandleLoginUnknownEmailStillHashes(), TestHandleRegisterDuplicateEmailSkipsHash(), TestRegistrationAllowed(), TestRegistrationAllowedCountUsersError(), doOIDCCallback(), locationParams(), newTestIdP() (+37 more)

### Community 25 - "Community 25"
Cohesion: 0.06
Nodes (30): food, foodCategory, foodNutrient, foodPortion, searchResponse, Source, bulkDataTypes(), emitMatchedFood() (+22 more)

### Community 26 - "Community 26"
Cohesion: 0.12
Nodes (46): doPasskeyLoginFinish(), mfaPasskeyBeginExpiredChallenge(), mfaPasskeyBeginInvalidJSON(), mfaPasskeyBeginMissingToken(), mfaPasskeyBeginNoPasskeysRegistered(), mfaPasskeyBeginSuccess(), mfaPasskeyBeginUnknownChallenge(), mfaPasskeyFinishCeremonyConsumeFails() (+38 more)

### Community 27 - "Community 27"
Cohesion: 0.14
Nodes (20): isMutating(), fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession() (+12 more)

### Community 28 - "Community 28"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 29 - "Community 29"
Cohesion: 0.11
Nodes (14): fakeFoodSearcher, fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_IngredientArgsResolveViaSearch(), TestSuggestCommand_IngredientArgsSkipUnresolvedNames() (+6 more)

### Community 30 - "Community 30"
Cohesion: 0.21
Nodes (17): fakeFoodImportRunner, doAdminRequest(), newAdminTestHandler(), TestAdminFoodImport_BackfillEmbeddings200(), TestAdminFoodImport_BackfillEmbeddingsError(), TestAdminFoodImport_MissingToken401(), TestAdminFoodImport_Repair200(), TestAdminFoodImport_RepairError() (+9 more)

### Community 31 - "Community 31"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 32 - "Community 32"
Cohesion: 0.15
Nodes (17): mapDayTypes(), mapItems(), mapOptions(), mapSlots(), ParseResponse(), assertDayTypeFields(), assertOptionFields(), assertPlanFields() (+9 more)

### Community 33 - "Community 33"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 34 - "Community 34"
Cohesion: 0.11
Nodes (1): fakeStore

### Community 35 - "Community 35"
Cohesion: 0.14
Nodes (8): Adapter, embedRequest, embedResponse, generateRequest, generateResponse, uniqueModels(), pullRequest, tagsResponse

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
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 42 - "Community 42"
Cohesion: 0.31
Nodes (7): fakeCompletionAdapter, doExtractPlanFromText(), TestHandleExtractPlanFromText(), TestHandleExtractPlanFromTextAdapterError(), TestHandleExtractPlanFromTextEmptyBody(), TestHandleExtractPlanFromTextOversized(), TestHandleExtractPlanFromTextUnparseableResponse()

### Community 43 - "Community 43"
Cohesion: 0.36
Nodes (1): Store

### Community 44 - "Community 44"
Cohesion: 0.29
Nodes (3): pct(), StatusCommand, StatusStore

### Community 45 - "Community 45"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 50 - "Community 50"
Cohesion: 0.38
Nodes (4): fakeResponse(), runOptions(), streamOf(), userMessage()

### Community 52 - "Community 52"
Cohesion: 0.29
Nodes (1): stubStore

### Community 53 - "Community 53"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 54 - "Community 54"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 55 - "Community 55"
Cohesion: 0.4
Nodes (2): bundleWith(), plan()

### Community 58 - "Community 58"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 59 - "Community 59"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 66 - "Community 66"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 67 - "Community 67"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 69 - "Community 69"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 76 - "Community 76"
Cohesion: 0.67
Nodes (2): oidcCallbackContext, oidcIdentity

### Community 77 - "Community 77"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 104 - "Community 104"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 106 - "Community 106"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 109 - "Community 109"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 111 - "Community 111"
Cohesion: 1.0
Nodes (1): VisionAdapter

## Knowledge Gaps
- **351 isolated node(s):** `appRuntime`, `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row` (+346 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 34`** (19 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetPhotosData()`, `.GetPlanBundle()`, `.GetRollups()`, `.GetWaterInRange()`, `.GetWorkoutsInRangeWithExercises()`, `.ListDayOverrides()`, `.ListFasts()`, `.ListMeasurements()`, `.ListPhotoMetadata()`, `.ListPlans()`, `.ListSleep()`, `.ListTemplatesForBackup()`, `.ListUsers()`, `.ListWeight()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (12 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.TargetsFor()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (6 nodes): `bundleWith()`, `meal()`, `noPlanView()`, `plan()`, `planDayWithSlots()`, `Dashboard.test.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 67`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (3 nodes): `oidcCallbackContext`, `oidcIdentity`, `handler_oidc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 104`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 106`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 109`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 111`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 11`, `Community 12`, `Community 15`, `Community 16`, `Community 19`, `Community 21`, `Community 22`, `Community 24`, `Community 25`, `Community 26`, `Community 29`, `Community 30`?**
  _High betweenness centrality (0.392) - this node is a cross-community bridge._
- **Why does `newHandler()` connect `Community 0` to `Community 1`, `Community 12`, `Community 19`, `Community 21`, `Community 24`, `Community 30`?**
  _High betweenness centrality (0.118) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 5` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 35`, `Community 7`, `Community 8`, `Community 10`, `Community 11`, `Community 13`, `Community 15`, `Community 16`, `Community 17`, `Community 19`, `Community 20`, `Community 22`, `Community 24`, `Community 29`?**
  _High betweenness centrality (0.108) - this node is a cross-community bridge._
- **Are the 449 inferred relationships involving `doRequest()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`doRequest()` has 449 INFERRED edges - model-reasoned connections that need verification._
- **Are the 505 inferred relationships involving `New()` (e.g. with `TestRunReturnsConfigLoadError()` and `adminTempStore()`) actually correct?**
  _`New()` has 505 INFERRED edges - model-reasoned connections that need verification._
- **Are the 334 inferred relationships involving `newHandler()` (e.g. with `DefaultLockoutConfig()` and `TestMeasurementsRoutesRequireAuth()`) actually correct?**
  _`newHandler()` has 334 INFERRED edges - model-reasoned connections that need verification._
- **Are the 331 inferred relationships involving `newFakeMealStore()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newFakeMealStore()` has 331 INFERRED edges - model-reasoned connections that need verification._