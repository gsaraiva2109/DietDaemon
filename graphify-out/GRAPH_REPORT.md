# Graph Report - DietDaemon  (2026-08-01)

## Corpus Check
- 489 files · ~673,637 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 5432 nodes · 12522 edges · 69 communities detected
- Extraction: 62% EXTRACTED · 38% INFERRED · 0% AMBIGUOUS · INFERRED: 4803 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 74|Community 74]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 85|Community 85]]
- [[_COMMUNITY_Community 115|Community 115]]
- [[_COMMUNITY_Community 116|Community 116]]
- [[_COMMUNITY_Community 117|Community 117]]
- [[_COMMUNITY_Community 118|Community 118]]
- [[_COMMUNITY_Community 120|Community 120]]
- [[_COMMUNITY_Community 122|Community 122]]

## God Nodes (most connected - your core abstractions)
1. `doRequest()` - 584 edges
2. `New()` - 541 edges
3. `newHandler()` - 486 edges
4. `newFakeMealStore()` - 472 edges
5. `Store` - 271 edges
6. `Handler` - 253 edges
7. `contains()` - 219 edges
8. `ctx()` - 188 edges
9. `tempDB()` - 145 edges
10. `decodeJSON()` - 131 edges

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
Nodes (526): accountRepos, TestBYOKKeyAbsenceRetainsSharedAdapterFallback(), fakeCompletionAdapter, fakeSuggester, deleteAccountTestHandler(), newHandlerWithAccountStore(), TestHandleDeleteAccountClearsSessionCookie(), TestHandleDeleteAccountEmailSendFailureStillSucceeds() (+518 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (325): TestAuthenticatedRateLimitCategories(), TestAuthenticatedRateLimitReturnsStructuredError(), TestExpensiveRequestRoutes(), fakeAccount, fakeAuditEvent, fakeMailer, NewPurgeRunner(), TestPurgeAccountPhotosListError() (+317 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (125): chatStreamState, credCreateConfig, credRevokeConfig, customFoodRequest, dayOverrideBody, deleteAccountRequest, exportLogData, Handler (+117 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (56): FS(), Normalize(), TestNormalize(), unaccent(), AccountDeletionStatus, backupConfigRow, catalogRow, credRow (+48 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (222): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), checkExtractMenuRequest(), checkExtractPlanRequest(), TestExtractLabel(), TestExtractLabelHTTPError(), TestExtractMenu() (+214 more)

### Community 5 - "Community 5"
Cohesion: 0.01
Nodes (8): authHandlerTestStore, fakeAuthStore, fakeMealStore, mfaEmailTestStore, passkeyTestStore, totpTestStore, fakePurgeStore, Store

### Community 6 - "Community 6"
Cohesion: 0.04
Nodes (204): IPRateLimiter, TestPendingStoreContract(), postgresDB(), TestFoodImportFingerprintStore(), TestPostgresDietPlanSmoke(), TestPostgresDualDriverSmoke(), TestPostgresMealLifecycle(), TestPostgresSearchFoods() (+196 more)

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (138): Adapter, contentBlock, message, messagesRequest, messagesResponse, writeCSV(), Destination, Runner (+130 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (152): extractArgs(), NewChatAdapter(), parseSSEEvent(), sendEvent(), sendProviderError(), drainReadStream(), TestExtractArgsEmptyValue(), TestReadStreamContextCancelledMidStream() (+144 more)

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (127): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthRepos, AuthStore, BackupRunner, ChatStore (+119 more)

### Community 10 - "Community 10"
Cohesion: 0.03
Nodes (130): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, targetsResponse, startBackgroundServices(), blockingStore, DigestRule, fakeChatRouteStore (+122 more)

### Community 11 - "Community 11"
Cohesion: 0.02
Nodes (45): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+37 more)

### Community 12 - "Community 12"
Cohesion: 0.02
Nodes (68): Registry, addFood(), renderModal(), typeSearch(), renderModal(), renderModal(), dayLabel(), renderModal() (+60 more)

### Community 13 - "Community 13"
Cohesion: 0.02
Nodes (82): Stat(), Config, addProblem(), parseProxyEntry(), validateBulkFile(), NotFound(), AuditEvent, BackupConfig (+74 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (56): NewWebAuthnHandle(), randomID(), calcSleepHours(), computeSleepDuration(), formatDuration(), parseQualityAndNote(), TestCalcSleepHours_NilWake(), TestCalcSleepHours_Overnight() (+48 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (81): authTestStore, emailTestAuthStore, emailToken, fakeMailer, TestHandleRegisterCreateEmailTokenFailure(), TestHandleRegisterSendsVerificationEmailWhenMailerConfigured(), TestHandleTOTPChallengeLockout(), containsStr() (+73 more)

### Community 16 - "Community 16"
Cohesion: 0.07
Nodes (83): credAuthStore, erroringCountAuthStore, buildCredHandler(), TestCheckLoginLockoutLocked(), TestCheckLoginLockoutStoreError(), TestHandleChangePasswordInvalidJSON(), TestHandleChangePasswordMissingFields(), TestHandleChangePasswordNewPasswordTooShort() (+75 more)

### Community 17 - "Community 17"
Cohesion: 0.03
Nodes (60): fakePlanStore, findSlotByLabel(), NewPlanCommand(), nextSlotID(), optionSummary(), parseTimeOfDay(), slotsForDayType(), bundleWithSlots() (+52 more)

### Community 18 - "Community 18"
Cohesion: 0.03
Nodes (44): CorrectCommand, CorrectResolver, CorrectStore, fakeMealStore, MealStore, parsePositiveFloat(), setProfileField(), ProfileCommand (+36 more)

### Community 19 - "Community 19"
Cohesion: 0.04
Nodes (53): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), llmItem (+45 more)

### Community 20 - "Community 20"
Cohesion: 0.03
Nodes (39): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, confirmReplace(), scaledMacros(), sourceLabel(), MacroBar() (+31 more)

### Community 21 - "Community 21"
Cohesion: 0.05
Nodes (44): ErrorCode, errorEnvelope, errorEnvelopeWriter, errorForStatus(), publicErrorMessage(), TestAPIErrorEnvelope(), TestAPIErrorEnvelopePreservesStreaming(), TestAPIRouteFallbackUsesErrorEnvelope() (+36 more)

### Community 22 - "Community 22"
Cohesion: 0.04
Nodes (30): allEntitiesFakeStore, New(), newFakeStore(), TestRun_ChecksImmediatelyBeforeCancelledContextReturns(), TestRunFor_ExportsAllEntities(), TestRunFor_LoadErrorAbortsRemainingEntities(), TestRunFor_MissingDestinationErrors(), TestRunFor_PhotoBlobWrittenBeforeIndex() (+22 more)

### Community 23 - "Community 23"
Cohesion: 0.04
Nodes (37): appendedChatMessage, buildAdapterForProvider(), buildChatAdapterForProvider(), decryptAIKey(), assertBYOKFailure(), TestBuildBYOKAdaptersRejectUnsupportedProvider(), TestBYOKChatOverrideUsedInsteadOfSharedAdapter(), TestBYOKFailuresDoNotFallBackToSharedAdapters() (+29 more)

### Community 24 - "Community 24"
Cohesion: 0.13
Nodes (61): TestHandleLoginHasConfirmedTOTPErrorFallsThroughToNormalLogin(), TestHandleLoginMFAStepUp(), TestHandleLoginMFAStepUpChallengeCreationFails(), buildTOTPHandler(), defaultTOTPMeals(), enrollTOTPSecret(), newTOTPTestStore(), TestHandleRegenerateRecoveryHasConfirmedError() (+53 more)

### Community 25 - "Community 25"
Cohesion: 0.06
Nodes (46): actionRow, Adapter, buttonComponent, mustMarshal(), readGatewayPayload(), readWSFrame(), buildServerFrame(), genSelfSignedCert() (+38 more)

### Community 26 - "Community 26"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 27 - "Community 27"
Cohesion: 0.12
Nodes (46): doPasskeyLoginFinish(), mfaPasskeyBeginExpiredChallenge(), mfaPasskeyBeginInvalidJSON(), mfaPasskeyBeginMissingToken(), mfaPasskeyBeginNoPasskeysRegistered(), mfaPasskeyBeginSuccess(), mfaPasskeyBeginUnknownChallenge(), mfaPasskeyFinishCeremonyConsumeFails() (+38 more)

### Community 28 - "Community 28"
Cohesion: 0.06
Nodes (30): food, foodCategory, foodNutrient, foodPortion, searchResponse, Source, bulkDataTypes(), emitMatchedFood() (+22 more)

### Community 29 - "Community 29"
Cohesion: 0.12
Nodes (14): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), messageTime(), parseGrams() (+6 more)

### Community 30 - "Community 30"
Cohesion: 0.08
Nodes (15): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+7 more)

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
Cohesion: 0.14
Nodes (4): Dialect, ErrUnsupportedDriver, postgresDialect, sqliteDialect

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
Cohesion: 0.27
Nodes (7): fakeVisionAdapter, doOCRUpload(), TestHandleOCRExtractCustomFood(), TestHandleOCRExtractCustomFoodAdapterError(), TestHandleOCRExtractCustomFoodDisabled(), TestHandleOCRExtractCustomFoodMissingFile(), TestHandleOCRExtractCustomFoodNonImage()

### Community 42 - "Community 42"
Cohesion: 0.24
Nodes (5): writeCandidates(), writeOtherOptions(), SuggestCommand, SuggestEngine, SuggestFoodSearcher

### Community 43 - "Community 43"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 44 - "Community 44"
Cohesion: 0.31
Nodes (7): loadPdfjs(), pdfToImages(), renderPage(), handleFile(), onExtracted(), onFailed(), t()

### Community 46 - "Community 46"
Cohesion: 0.36
Nodes (1): Store

### Community 47 - "Community 47"
Cohesion: 0.29
Nodes (3): pct(), StatusCommand, StatusStore

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 53 - "Community 53"
Cohesion: 0.38
Nodes (4): fakeResponse(), runOptions(), streamOf(), userMessage()

### Community 55 - "Community 55"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 56 - "Community 56"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 57 - "Community 57"
Cohesion: 0.4
Nodes (2): bundleWith(), plan()

### Community 58 - "Community 58"
Cohesion: 0.5
Nodes (2): focusSoon(), resetSearchState()

### Community 62 - "Community 62"
Cohesion: 0.7
Nodes (4): goToNext(), goToPrevious(), makeCurrent(), toggleClass()

### Community 63 - "Community 63"
Cohesion: 0.4
Nodes (1): fakeMealLogger

### Community 64 - "Community 64"
Cohesion: 0.4
Nodes (4): imageURL, visionContentPart, visionMessage, visionRequest

### Community 65 - "Community 65"
Cohesion: 0.4
Nodes (4): imageSource, visionContentBlock, visionMessage, visionRequest

### Community 73 - "Community 73"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 74 - "Community 74"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 76 - "Community 76"
Cohesion: 0.5
Nodes (1): crossUserPhotoStore

### Community 77 - "Community 77"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 84 - "Community 84"
Cohesion: 0.67
Nodes (2): oidcCallbackContext, oidcIdentity

### Community 85 - "Community 85"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 115 - "Community 115"
Cohesion: 1.0
Nodes (1): adminFoodImportRequest

### Community 116 - "Community 116"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 117 - "Community 117"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 118 - "Community 118"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 120 - "Community 120"
Cohesion: 1.0
Nodes (1): visionRequest

### Community 122 - "Community 122"
Cohesion: 1.0
Nodes (1): VisionAdapter

## Knowledge Gaps
- **361 isolated node(s):** `appRuntime`, `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row` (+356 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 37`** (12 nodes): `fakeStore`, `.GetRollup()`, `.GetTargets()`, `.GetUser()`, `.GetUserIDByChannel()`, `.MapChannelUser()`, `.SaveMeal()`, `.SetTargets()`, `.TargetsFor()`, `.UpsertChatRoute()`, `.UpsertRollup()`, `.UpsertUser()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (6 nodes): `bundleWith()`, `meal()`, `noPlanView()`, `plan()`, `planDayWithSlots()`, `Dashboard.test.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (5 nodes): `focusSoon()`, `onKey()`, `onListKey()`, `resetSearchState()`, `CommandPalette.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (5 nodes): `fakeMealLogger`, `.Handle()`, `.LogMeal()`, `.LogMealFromItems()`, `.ParseAndResolve()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (4 nodes): `crossUserPhotoStore`, `.DeletePhoto()`, `.GetPhotoData()`, `.ListPhotoMetadata()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (3 nodes): `oidcCallbackContext`, `oidcIdentity`, `handler_oidc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 85`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 115`** (2 nodes): `adminFoodImportRequest`, `handler_admin_import.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 116`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 117`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 118`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 120`** (2 nodes): `vision.go`, `visionRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 122`** (2 nodes): `vision.go`, `VisionAdapter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 12`, `Community 15`, `Community 16`, `Community 17`, `Community 18`, `Community 21`, `Community 22`, `Community 23`, `Community 24`, `Community 25`, `Community 27`, `Community 28`, `Community 31`, `Community 41`?**
  _High betweenness centrality (0.438) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 4` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 7`, `Community 8`, `Community 9`, `Community 10`, `Community 13`, `Community 16`, `Community 17`, `Community 18`, `Community 19`, `Community 21`, `Community 22`, `Community 23`, `Community 25`, `Community 29`?**
  _High betweenness centrality (0.121) - this node is a cross-community bridge._
- **Why does `newHandler()` connect `Community 0` to `Community 1`, `Community 41`, `Community 15`, `Community 16`, `Community 23`, `Community 24`, `Community 31`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Are the 468 inferred relationships involving `doRequest()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`doRequest()` has 468 INFERRED edges - model-reasoned connections that need verification._
- **Are the 536 inferred relationships involving `New()` (e.g. with `TestRunReturnsConfigLoadError()` and `adminTempStore()`) actually correct?**
  _`New()` has 536 INFERRED edges - model-reasoned connections that need verification._
- **Are the 365 inferred relationships involving `newHandler()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newHandler()` has 365 INFERRED edges - model-reasoned connections that need verification._
- **Are the 352 inferred relationships involving `newFakeMealStore()` (e.g. with `TestMeasurementsRoutesRequireAuth()` and `TestListMeasurementsStoreError()`) actually correct?**
  _`newFakeMealStore()` has 352 INFERRED edges - model-reasoned connections that need verification._