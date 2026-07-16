# Graph Report - DietDaemon  (2026-07-16)

## Corpus Check
- 341 files · ~258,641 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3156 nodes · 5475 edges · 73 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 1471 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 96|Community 96]]
- [[_COMMUNITY_Community 97|Community 97]]
- [[_COMMUNITY_Community 98|Community 98]]
- [[_COMMUNITY_Community 99|Community 99]]
- [[_COMMUNITY_Community 101|Community 101]]
- [[_COMMUNITY_Community 154|Community 154]]
- [[_COMMUNITY_Community 155|Community 155]]
- [[_COMMUNITY_Community 156|Community 156]]
- [[_COMMUNITY_Community 157|Community 157]]
- [[_COMMUNITY_Community 158|Community 158]]
- [[_COMMUNITY_Community 159|Community 159]]
- [[_COMMUNITY_Community 160|Community 160]]
- [[_COMMUNITY_Community 161|Community 161]]
- [[_COMMUNITY_Community 162|Community 162]]
- [[_COMMUNITY_Community 163|Community 163]]

## God Nodes (most connected - your core abstractions)
1. `New()` - 243 edges
2. `Store` - 203 edges
3. `Handler` - 163 edges
4. `doRequest()` - 100 edges
5. `New()` - 90 edges
6. `newFakeMealStore()` - 89 edges
7. `newHandler()` - 88 edges
8. `fakeMealStore` - 82 edges
9. `contains()` - 82 edges
10. `run()` - 67 edges

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
Nodes (112): credCreateConfig, credRevokeConfig, Handler, hostOnly(), isSixDigit(), readSessionCookie(), writeJSONList(), calculateTDEE() (+104 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (43): parseTier(), Normalize(), TestNormalize(), unaccent(), backupConfigRow, catalogRow, credRow, fastRow (+35 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (212): collectEvents(), TestRouterContextCancellation(), TestRouterErrorPropagation(), TestRouterMidStreamError(), TestRouterSeedsHistory(), TestRouterTextOnly(), TestRouterTextOnly_doneForwarded(), TestRouterToolCallMaxRounds() (+204 more)

### Community 3 - "Community 3"
Cohesion: 0.05
Nodes (129): emailToken, fakeMailer, fakeMealLogger, fakeSuggester, buildAuthSecurityHandler(), TestHandleLoginUnknownEmailStillHashes(), TestHandleRegisterDuplicateEmailSkipsHash(), TestHandleTOTPChallengeLockout() (+121 more)

### Community 4 - "Community 4"
Cohesion: 0.03
Nodes (94): TestComplete(), TestCompleteHTTPError(), TestEmbedNotSupported(), NewCorrectCommand(), TestCorrectCommand_BadGramsFormat(), TestCorrectCommand_ConflictOffersReplacement(), TestCorrectCommand_HappyPath(), TestCorrectCommand_NoRecentMeal() (+86 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (97): buildNudgeRuleView(), buildNudgeRuleViewWeeklyBudget(), nudgeRuleView, blockingStore, ChatRouteStore, ChatSender, DigestRule, DigestStore (+89 more)

### Community 6 - "Community 6"
Cohesion: 0.02
Nodes (6): authHandlerTestStore, emailTestAuthStore, fakeMealStore, mfaEmailTestStore, Store, fakePending

### Community 7 - "Community 7"
Cohesion: 0.02
Nodes (74): AccountStore, APIKeyStore, AuditStore, AuthConfig, AuthStore, BackupRunner, ChatStore, EmailTokenStore (+66 more)

### Community 8 - "Community 8"
Cohesion: 0.02
Nodes (44): ProtectedRoute(), AuthProvider(), useAuth(), useDemo(), useActiveFast(), useAIKey(), useApiKeys(), useBodySummary() (+36 more)

### Community 9 - "Community 9"
Cohesion: 0.02
Nodes (53): Registry, dayLabel(), download(), sourceLabel(), onSubmit(), onAdd(), relativeCaption(), copy() (+45 more)

### Community 10 - "Community 10"
Cohesion: 0.02
Nodes (55): fakeChatAdapter, fakeChatStore, newChatHandler(), parseSSE(), TestHandleChatMessageAdapterError(), TestHandleChatMessageBasic(), TestHandleChatMessageEmptyText(), TestHandleChatMessageSSEStreaming() (+47 more)

### Community 11 - "Community 11"
Cohesion: 0.03
Nodes (35): formatDurationShort(), NewFastCommand(), FastCommand, FastStore, randomID(), calcSleepHours(), computeSleepDuration(), formatDuration() (+27 more)

### Community 12 - "Community 12"
Cohesion: 0.04
Nodes (52): macrosSum(), TemplateCommand, TemplateComposer, TemplateMealLogger, TemplateStore, bulkUpserter, main(), run() (+44 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (1): fakeAuthStore

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (55): FS(), AuditEvent, BackupConfig, BodyCompositionSummary, CorrectionFeedback, DailyRollup, DailyTargets, Fast (+47 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (29): CorrectCommand, CorrectResolver, CorrectStore, MealStore, NewProfileCommand(), ProfileCommand, ProfileStore, NewTargetCommand() (+21 more)

### Community 16 - "Community 16"
Cohesion: 0.05
Nodes (57): Environment-Driven Configuration, Feature-Flagged Capabilities, Modular Monolith Architecture, Provider-Agnostic Design, Honest about uncertainty design principle, No-CGO stance, Backup Documentation, CLAUDE.md Project Instructions (+49 more)

### Community 17 - "Community 17"
Cohesion: 0.07
Nodes (49): AppleIcon(), Auth0Icon(), AuthentikIcon(), base(), BodyIcon(), brand(), CameraIcon(), ChatIcon() (+41 more)

### Community 18 - "Community 18"
Cohesion: 0.05
Nodes (47): DietDaemon, Open Food Facts, TACO (Brazilian Food Composition Table), DietDaemon Container Service, Ollama Sidecar Service, DietDaemon Spoon Favicon, DietDaemon Web App Entry Point, Optional Dashboard (+39 more)

### Community 19 - "Community 19"
Cohesion: 0.07
Nodes (29): Parser, consumeUnit(), parseNumber(), parseSegment(), refineColher(), stripConnector(), stripLeadingFiller(), IsUnit() (+21 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (30): extractArgs(), NewChatAdapter(), sendEvent(), TestExtractArgsEmptyValue(), TestStreamChatHTTPError(), TestToWireMessagesToolRoundTrip(), toWireMessages(), ChatAdapter (+22 more)

### Community 21 - "Community 21"
Cohesion: 0.09
Nodes (22): llmItem, llmResponse, Parser, ModelOverrideFromContext(), Candidate, CandidateItem, Engine, describeCombo() (+14 more)

### Community 22 - "Community 22"
Cohesion: 0.1
Nodes (18): food, foodCategory, foodNutrient, searchResponse, Source, bulkDataTypes(), extractMacros(), foodToMatch() (+10 more)

### Community 23 - "Community 23"
Cohesion: 0.12
Nodes (13): Engine, MealStore, Parser, PendingStore, askText(), isNotFound(), parseGrams(), plural() (+5 more)

### Community 24 - "Community 24"
Cohesion: 0.13
Nodes (19): entry, cosineSimilarity(), packF32LE(), sortByScore(), openTestDB(), requireNoErr(), TestCacheUpdatesOnUpsertAndInvalidatesOnDelete(), TestCosineSimilarity() (+11 more)

### Community 25 - "Community 25"
Cohesion: 0.08
Nodes (16): Client, NewClient(), listResponse, Config, Mailer, New(), smtpPortOrDefault(), Message (+8 more)

### Community 26 - "Community 26"
Cohesion: 0.15
Nodes (19): fakeSessionRepo, Session, CreateSession(), RotateSession(), cfg(), newFakeSessionRepo(), TestCreateSession(), TestCreateSessionRemember() (+11 more)

### Community 27 - "Community 27"
Cohesion: 0.12
Nodes (16): actionRow, Adapter, buttonComponent, dialWebSocket(), mustMarshal(), readGatewayPayload(), readWSFrame(), writeGatewayFrame() (+8 more)

### Community 28 - "Community 28"
Cohesion: 0.12
Nodes (17): MFAChallengeRepo, GenerateSecret(), contains(), TestGenerateSecret(), TestGenerateSecretEmptyAccount(), TestGenerateSecretEmptyIssuer(), TestValidateCode(), TestValidateCodeEmptySecret() (+9 more)

### Community 29 - "Community 29"
Cohesion: 0.12
Nodes (12): fakeFoodSearcher, fakeSuggestEngine, NewSuggestCommand(), TestSuggestCommand_EmptyMessage(), TestSuggestCommand_EngineError(), TestSuggestCommand_HappyPath(), TestSuggestCommand_IngredientArgsResolveViaSearch(), TestSuggestCommand_IngredientArgsSkipUnresolvedNames() (+4 more)

### Community 30 - "Community 30"
Cohesion: 0.09
Nodes (12): Adapter, contentBlock, message, messagesRequest, messagesResponse, Strip(), TestStrip(), Adapter (+4 more)

### Community 31 - "Community 31"
Cohesion: 0.15
Nodes (14): sendOut(), Router, ExtractSuggestions(), TestExtractSuggestions_BlockNotAtEnd(), TestExtractSuggestions_EmptyArray(), TestExtractSuggestions_IntArray(), TestExtractSuggestions_MalformedJSON(), TestExtractSuggestions_NoBlock() (+6 more)

### Community 32 - "Community 32"
Cohesion: 0.13
Nodes (12): isPrevDay(), Streak(), TestStreak_AboveCeilStops(), TestStreak_AllInBand(), TestStreak_DateGap(), TestStreak_Empty(), TestStreak_ExactBoundary(), TestStreak_MissingTarget() (+4 more)

### Community 33 - "Community 33"
Cohesion: 0.16
Nodes (11): isLockedOut(), issueMagic(), issueResetToken(), issueVerifyToken(), now(), parseCookies(), recordFailure(), seed() (+3 more)

### Community 34 - "Community 34"
Cohesion: 0.17
Nodes (5): Destination, Runner, Store, WriteMealsCSV(), WriteRollupsCSV()

### Community 35 - "Community 35"
Cohesion: 0.18
Nodes (7): IDTokenClaims, initResult, Provider, BuildRegistry(), TestBuildRegistry(), TestBuildRegistryCustomScopes(), ProviderConfig

### Community 36 - "Community 36"
Cohesion: 0.16
Nodes (8): Adapter, embedRequest, embedResponse, generateRequest, generateResponse, uniqueModels(), pullRequest, tagsResponse

### Community 37 - "Community 37"
Cohesion: 0.21
Nodes (7): Embedder, fingerprintStore, localFingerprint(), New(), Runner, SourceFactory, Store

### Community 38 - "Community 38"
Cohesion: 0.17
Nodes (11): BulkFilter, BulkSource, Command, MessagingAdapter, ModelAdapter, Notifier, NutritionSource, Parser (+3 more)

### Community 39 - "Community 39"
Cohesion: 0.24
Nodes (4): demoRange(), fd(), hoursAgo(), m()

### Community 40 - "Community 40"
Cohesion: 0.31
Nodes (8): cryptoRand5Digits(), GenerateRecoveryCodes(), TestGenerateRecoveryCodesCount(), TestGenerateRecoveryCodesFormat(), TestGenerateRecoveryCodesHashRoundtrip(), TestGenerateRecoveryCodesInvalidCount(), TestGenerateRecoveryCodesUniqueness(), RecoveryCodeRepo

### Community 41 - "Community 41"
Cohesion: 0.28
Nodes (9): Color System (OKLCH, Sage/Amber), Macro Color Hues, Macro Ring UI Component, Motion System (Framer Motion, Spring/Tick), Accessibility & Inclusion, Brand Personality, Design Principles, Alias Review UI (+1 more)

### Community 43 - "Community 43"
Cohesion: 0.36
Nodes (1): Store

### Community 44 - "Community 44"
Cohesion: 0.29
Nodes (4): priorityInt(), TestPriorityMapping(), message, Notifier

### Community 46 - "Community 46"
Cohesion: 0.52
Nodes (5): floatPtr(), intPtr(), TestToWorkout(), TestToWorkoutNilSafety(), ToWorkout()

### Community 47 - "Community 47"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 48 - "Community 48"
Cohesion: 0.29
Nodes (1): fakeStore

### Community 49 - "Community 49"
Cohesion: 0.29
Nodes (1): stubStore

### Community 50 - "Community 50"
Cohesion: 0.33
Nodes (3): Notifier, priorityString(), TestPriorityMapping()

### Community 51 - "Community 51"
Cohesion: 0.29
Nodes (6): ChatAdapter, ChatEvent, ChatMessage, ChatRequest, ToolCallEvent, ToolSpec

### Community 52 - "Community 52"
Cohesion: 0.33
Nodes (1): fakeStore

### Community 56 - "Community 56"
Cohesion: 0.4
Nodes (2): inferenceResponse, Provider

### Community 57 - "Community 57"
Cohesion: 0.5
Nodes (5): MULTI_USER (Product Deployment Mode), Users, Auth, MULTI_USER, Family/Household Multi-user Sharing

### Community 62 - "Community 62"
Cohesion: 0.5
Nodes (3): HevyExercise, HevySet, HevyWorkout

### Community 63 - "Community 63"
Cohesion: 0.5
Nodes (2): Memory, Queue

### Community 65 - "Community 65"
Cohesion: 0.5
Nodes (3): Message, Session, Store

### Community 66 - "Community 66"
Cohesion: 0.5
Nodes (1): Dest

### Community 73 - "Community 73"
Cohesion: 0.67
Nodes (1): notifierFactory

### Community 96 - "Community 96"
Cohesion: 1.0
Nodes (1): pendingAliasView

### Community 97 - "Community 97"
Cohesion: 1.0
Nodes (1): aiKeyStatus

### Community 98 - "Community 98"
Cohesion: 1.0
Nodes (1): sentNudgeRow

### Community 99 - "Community 99"
Cohesion: 1.0
Nodes (1): ProviderKey

### Community 101 - "Community 101"
Cohesion: 1.0
Nodes (2): STT Error Behaviour, STT Troubleshooting

### Community 154 - "Community 154"
Cohesion: 1.0
Nodes (1): Typography (Plus Jakarta Sans)

### Community 155 - "Community 155"
Cohesion: 1.0
Nodes (1): Anti-references

### Community 156 - "Community 156"
Cohesion: 1.0
Nodes (1): Recipe / Multi-ingredient Composition

### Community 157 - "Community 157"
Cohesion: 1.0
Nodes (1): Weekly/Monthly Digest Notification

### Community 158 - "Community 158"
Cohesion: 1.0
Nodes (1): Health Platform Import/Export

### Community 159 - "Community 159"
Cohesion: 1.0
Nodes (1): Configurable Nudge Rules

### Community 160 - "Community 160"
Cohesion: 1.0
Nodes (1): Scheduled Data Export/Backup

### Community 161 - "Community 161"
Cohesion: 1.0
Nodes (1): Precedence UI

### Community 162 - "Community 162"
Cohesion: 1.0
Nodes (1): Group 2 — Food Logging & Resolution

### Community 163 - "Community 163"
Cohesion: 1.0
Nodes (1): Group 3 — Scheduler & Data Ops

## Knowledge Gaps
- **338 isolated node(s):** `phraseEntry`, `bulkUpserter`, `mealSaver`, `Row`, `HevyWorkout` (+333 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 13`** (62 nodes): `fakeAuthStore`, `.ConfirmTOTP()`, `.ConsumeEmailToken()`, `.ConsumeOIDCState()`, `.ConsumeRecoveryCode()`, `.ConsumeWebAuthnSession()`, `.CountUsers()`, `.CreateAPIKey()`, `.CreateEmailToken()`, `.CreateMFAChallenge()`, `.CreateOIDCState()`, `.CreateSession()`, `.CreateShareToken()`, `.CreateUserWithOIDC()`, `.CreateUserWithPassword()`, `.CreateWebAuthnCredential()`, `.CreateWebAuthnSession()`, `.DeleteEmailTokensByUserAndPurpose()`, `.DeleteMagicCode()`, `.DeleteMFAChallenge()`, `.DeleteMFAEmailCode()`, `.DeleteOIDCIdentity()`, `.DeleteOIDCState()`, `.DeleteSession()`, `.DeleteTOTP()`, `.DeleteUserSessions()`, `.DeleteWebAuthnCredential()`, `.GetMagicCode()`, `.GetMFAChallenge()`, `.GetMFAEmailCode()`, `.GetOrCreateWebAuthnHandle()`, `.GetPasswordHash()`, `.GetSession()`, `.GetTOTPSecret()`, `.GetUserByAPIKey()`, `.GetUserByEmail()`, `.GetUserByOIDCIdentity()`, `.GetUserByShareToken()`, `.GetUserByWebAuthnHandle()`, `.GetWebAuthnCredentialsRaw()`, `.HasConfirmedTOTP()`, `.IncrementMagicCodeAttempts()`, `.IncrementMFAEmailCodeAttempts()`, `.LinkOIDCIdentity()`, `.ListAPIKeys()`, `.ListOIDCIdentities()`, `.ListShareTokens()`, `.ListWebAuthnCredentials()`, `.MarkEmailVerified()`, `.RecentFailedAttempts()`, `.RecordLoginAttempt()`, `.RenameWebAuthnCredential()`, `.ReplaceRecoveryCodes()`, `.RevokeAPIKey()`, `.SetPasswordHash()`, `.TouchSession()`, `.UpdateUserEmail()`, `.UpdateWebAuthnCredentialOnAuth()`, `.UpsertMagicCode()`, `.UpsertMFAEmailCode()`, `.UpsertTOTPSecret()`, `.WriteAuditEvent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (8 nodes): `pendingstore.go`, `New()`, `Store`, `.Delete()`, `.deleteRow()`, `.expired()`, `.Get()`, `.Save()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (7 nodes): `fakeStore`, `.GetBackupConfig()`, `.GetMealsInRange()`, `.GetRollups()`, `.ListUsers()`, `.SetBackupCounts()`, `.SetBackupLastRun()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (7 nodes): `fakeStore`, `.AddPendingAlias()`, `.GetFood()`, `.GetSourcePrecedence()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (7 nodes): `stubStore`, `.AddPendingAlias()`, `.GetFood()`, `.ListFoodsWithoutVectors()`, `.LookupFood()`, `.RecordFoodQuery()`, `.UpsertFood()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (6 nodes): `fakeStore`, `.FrequentFoods()`, `.GetFood()`, `.GetFoodDetail()`, `.GetRollup()`, `.GetTargets()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (5 nodes): `whisper.go`, `inferenceResponse`, `Provider`, `.Transcribe()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (4 nodes): `queue.go`, `Memory`, `Queue`, `NewMemory()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (4 nodes): `s3dest.go`, `Dest`, `.Write()`, `New()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (3 nodes): `TestNotifierContract()`, `notifierFactory`, `notifier_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 96`** (2 nodes): `pendingAliasView`, `handler_food.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 97`** (2 nodes): `aiKeyStatus`, `handler_settings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 98`** (2 nodes): `store_nudges.go`, `sentNudgeRow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 99`** (2 nodes): `store_provider_keys.go`, `ProviderKey`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 101`** (2 nodes): `STT Error Behaviour`, `STT Troubleshooting`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 154`** (1 nodes): `Typography (Plus Jakarta Sans)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 155`** (1 nodes): `Anti-references`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 156`** (1 nodes): `Recipe / Multi-ingredient Composition`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 157`** (1 nodes): `Weekly/Monthly Digest Notification`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 158`** (1 nodes): `Health Platform Import/Export`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 159`** (1 nodes): `Configurable Nudge Rules`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 160`** (1 nodes): `Scheduled Data Export/Backup`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 161`** (1 nodes): `Precedence UI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 162`** (1 nodes): `Group 2 — Food Logging & Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 163`** (1 nodes): `Group 3 — Scheduler & Data Ops`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 2` to `Community 0`, `Community 1`, `Community 3`, `Community 4`, `Community 5`, `Community 7`, `Community 9`, `Community 10`, `Community 12`, `Community 13`, `Community 15`, `Community 20`, `Community 22`, `Community 24`, `Community 28`, `Community 29`, `Community 31`?**
  _High betweenness centrality (0.334) - this node is a cross-community bridge._
- **Why does `run()` connect `Community 7` to `Community 0`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 35`, `Community 9`, `Community 11`, `Community 12`, `Community 14`, `Community 15`, `Community 29`?**
  _High betweenness centrality (0.122) - this node is a cross-community bridge._
- **Why does `Handler` connect `Community 0` to `Community 32`, `Community 34`, `Community 4`, `Community 5`, `Community 7`, `Community 20`?**
  _High betweenness centrality (0.103) - this node is a cross-community bridge._
- **Are the 238 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 238 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `Handler` (e.g. with `run()` and `TestHandlerServesSPA()`) actually correct?**
  _`Handler` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 19 inferred relationships involving `doRequest()` (e.g. with `TestEmailVerifySuccess()` and `TestEmailVerifyInvalidToken()`) actually correct?**
  _`doRequest()` has 19 INFERRED edges - model-reasoned connections that need verification._
- **Are the 88 inferred relationships involving `New()` (e.g. with `run()` and `buildEmbedAdapter()`) actually correct?**
  _`New()` has 88 INFERRED edges - model-reasoned connections that need verification._