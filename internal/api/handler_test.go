package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// --- fakes ---

type fakeMealStore struct {
	meals        map[string]types.Meal
	recentMeals  []types.Meal
	rollup       types.DailyRollup
	rollups      []types.DailyRollup
	user         types.User
	correctErr   error
	getMealErr   error
	rollupErr    error
	rollupsErr   error
	recentErr    error
	getUserErr   error
	targets      types.DailyTargets
	targetsErr   error
	addErr       error
	deleteErr    error
	backupConfig types.BackupConfig

	nudgeRuleConfig map[string]types.NudgeRuleConfig

	// Latest meal.
	latestMealTime    string
	latestMealTimeErr error

	// Food discovery.
	foodList             []types.FoodDetail
	foodListErr          error
	foodDetail           types.FoodDetail
	foodDetailErr        error
	addAliasErr          error
	deleteAliasErr       error
	foodsByID            map[string]types.FoodMatch
	removeFromLibraryErr error
	addToLibraryErr      error
	customFood           types.FoodDetail
	createCustomFoodErr  error
	updateCustomFoodErr  error
	deleteCustomFoodErr  error
	createServingUnitErr error
	deleteServingUnitErr error
	createCustomFoodUser string
	updateCustomFoodUser string
	deleteCustomFoodUser string
	createCustomFoodIn   types.CustomFoodInput
	updateCustomFoodIn   types.CustomFoodInput

	// Pending aliases.
	pendingAliases         []types.PendingAlias
	pendingAliasesErr      error
	confirmPendingAliasErr error
	rejectPendingAliasErr  error

	// Nutrition source precedence.
	precedence       []string
	precedenceErr    error
	setPrecedenceErr error

	// Bulk food-import status.
	foodImportStatuses    []types.FoodImportStatus
	foodImportStatusesErr error

	// Meal templates.
	templates         []types.MealTemplate
	templatesErr      error
	template          types.MealTemplate
	templateErr       error
	saveTemplateErr   error
	deleteTemplateErr error
	logTemplateErr    error
	saveMealErr       error

	// Body tracking.
	weights              []types.WeightEntry
	weightsErr           error
	logWeightErr         error
	deleteWeightErr      error
	weightTrend          []types.WeightTrend
	weightTrendErr       error
	activeFast           *types.Fast
	fasts                []types.Fast
	startFastErr         error
	endFastErr           error
	activeFastErr        error
	listFastsErr         error
	measurements         []types.MeasurementEntry
	measurementsErr      error
	logMeasurementErr    error
	deleteMeasurementErr error
	photoMetadata        []types.ProgressPhoto
	photoMetadataErr     error
	photoData            types.ProgressPhoto
	photoDataErr         error
	uploadPhotoErr       error
	deletePhotoErr       error

	// BYOK AI keys.
	aiKeyProvider  string
	aiKeyEncrypted string
	aiKeyFound     bool
	aiKeyErr       error

	// Goals & profile.
	profile          types.UserProfile
	profileErr       error
	upsertProfileErr error

	// Body tracking / shared.
	mealsInRange    []types.Meal
	mealsInRangeErr error

	// Water — daily aggregates (export).
	waterDailyTotals    []types.WaterDayTotal
	waterDailyTotalsErr error
}

func newFakeMealStore() *fakeMealStore {
	return &fakeMealStore{
		meals: map[string]types.Meal{},
	}
}

func (s *fakeMealStore) GetMeal(_ context.Context, mealID string) (types.Meal, error) {
	if s.getMealErr != nil {
		return types.Meal{}, s.getMealErr
	}
	if m, ok := s.meals[mealID]; ok {
		return m, nil
	}
	return types.Meal{}, types.ErrNotFound
}
func (s *fakeMealStore) RecentMeals(_ context.Context, _ string, _ int) ([]types.Meal, error) {
	if s.recentErr != nil {
		return nil, s.recentErr
	}
	return s.recentMeals, nil
}
func (s *fakeMealStore) GetRollup(_ context.Context, _, _ string) (types.DailyRollup, error) {
	if s.rollupErr != nil {
		return types.DailyRollup{}, s.rollupErr
	}
	return s.rollup, nil
}
func (s *fakeMealStore) GetRollups(_ context.Context, _, _, _ string) ([]types.DailyRollup, error) {
	if s.rollupsErr != nil {
		return nil, s.rollupsErr
	}
	return s.rollups, nil
}
func (s *fakeMealStore) CorrectMealItem(_ context.Context, _ string, _ string, _ int, _ types.ResolvedItem) error {
	return s.correctErr
}
func (s *fakeMealStore) AddMealItem(_ context.Context, _, mealID string, item types.ResolvedItem) error {
	if s.addErr != nil {
		return s.addErr
	}
	m, ok := s.meals[mealID]
	if !ok {
		return types.ErrNotFound
	}
	m.Items = append(m.Items, item)
	s.meals[mealID] = m
	return nil
}
func (s *fakeMealStore) DeleteMealItem(_ context.Context, _, mealID string, idx int) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	m, ok := s.meals[mealID]
	if !ok {
		return types.ErrNotFound
	}
	if idx < 0 || idx >= len(m.Items) {
		return types.ErrNotFound
	}
	m.Items = append(m.Items[:idx], m.Items[idx+1:]...)
	s.meals[mealID] = m
	return nil
}
func (s *fakeMealStore) GetTargets(_ context.Context, _ string) (types.DailyTargets, error) {
	if s.targetsErr != nil {
		return types.DailyTargets{}, s.targetsErr
	}
	return s.targets, nil
}
func (s *fakeMealStore) SetTargets(_ context.Context, t types.DailyTargets) error {
	if s.targetsErr != nil {
		return s.targetsErr
	}
	s.targets = t
	return nil
}
func (s *fakeMealStore) UpdateRollupTargets(_ context.Context, _, _ string, t types.Macros) error {
	s.rollup.Targets = t
	return nil
}
func (s *fakeMealStore) GetNudgeRuleConfig(_ context.Context, userID string) ([]types.NudgeRuleConfig, error) {
	var out []types.NudgeRuleConfig
	for _, c := range s.nudgeRuleConfig {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (s *fakeMealStore) SetNudgeRuleConfig(_ context.Context, userID, ruleID string, enabled bool, params json.RawMessage) error {
	if s.nudgeRuleConfig == nil {
		s.nudgeRuleConfig = map[string]types.NudgeRuleConfig{}
	}
	s.nudgeRuleConfig[userID+"|"+ruleID] = types.NudgeRuleConfig{UserID: userID, RuleID: ruleID, Enabled: enabled, Params: params}
	return nil
}
func (s *fakeMealStore) DeleteNudgeRuleConfig(_ context.Context, userID, ruleID string) error {
	delete(s.nudgeRuleConfig, userID+"|"+ruleID)
	return nil
}
func (s *fakeMealStore) GetBackupConfig(_ context.Context, userID string) (types.BackupConfig, error) {
	if s.backupConfig.UserID == "" {
		return types.BackupConfig{}, types.ErrNotFound
	}
	return s.backupConfig, nil
}
func (s *fakeMealStore) SetBackupConfig(_ context.Context, cfg types.BackupConfig) error {
	s.backupConfig = cfg
	return nil
}
func (s *fakeMealStore) GetUser(_ context.Context, _ string) (types.User, error) {
	if s.getUserErr != nil {
		return types.User{}, s.getUserErr
	}
	return s.user, nil
}
func (s *fakeMealStore) UpsertUser(_ context.Context, u types.User) error { s.user = u; return nil }

// Latest meal.
func (s *fakeMealStore) LatestMealTime(_ context.Context, _ string) (string, error) {
	return s.latestMealTime, s.latestMealTimeErr
}

// Food discovery.
func (s *fakeMealStore) ListFoods(_ context.Context, _, _ string, _, _ int) ([]types.FoodDetail, error) {
	return s.foodList, s.foodListErr
}
func (s *fakeMealStore) SearchFoods(_ context.Context, _, _ string) ([]types.FoodDetail, error) {
	return s.foodList, s.foodListErr
}
func (s *fakeMealStore) FrequentFoods(_ context.Context, _ string, _ int) ([]types.FoodDetail, error) {
	return s.foodList, s.foodListErr
}
func (s *fakeMealStore) GetFoodDetail(_ context.Context, _, _ string) (types.FoodDetail, error) {
	return s.foodDetail, s.foodDetailErr
}
func (s *fakeMealStore) GetFood(_ context.Context, foodID string) (types.FoodMatch, error) {
	if fm, ok := s.foodsByID[foodID]; ok {
		return fm, nil
	}
	return types.FoodMatch{}, types.ErrNoMatch
}
func (s *fakeMealStore) GetFoodForUser(_ context.Context, _, foodID string) (types.FoodMatch, error) {
	return s.GetFood(context.Background(), foodID)
}
func (s *fakeMealStore) SearchCatalog(_ context.Context, _, _, _ string, _, _ int) ([]types.FoodDetail, error) {
	return s.foodList, s.foodListErr
}
func (s *fakeMealStore) RemoveFromLibrary(_ context.Context, _, _ string) error {
	return s.removeFromLibraryErr
}
func (s *fakeMealStore) AddToLibrary(_ context.Context, _, _ string) error {
	return s.addToLibraryErr
}
func (s *fakeMealStore) AddFoodAlias(_ context.Context, _, _, _ string) error {
	return s.addAliasErr
}
func (s *fakeMealStore) DeleteFoodAlias(_ context.Context, _, _, _ string) error {
	return s.deleteAliasErr
}
func (s *fakeMealStore) CreateCustomFood(_ context.Context, userID string, input types.CustomFoodInput) (types.FoodDetail, error) {
	s.createCustomFoodUser, s.createCustomFoodIn = userID, input
	return s.customFood, s.createCustomFoodErr
}
func (s *fakeMealStore) UpdateCustomFood(_ context.Context, userID, foodID string, input types.CustomFoodInput) (types.FoodDetail, error) {
	s.updateCustomFoodUser, s.updateCustomFoodIn = userID, input
	if s.customFood.FoodID == "" {
		s.customFood.FoodID = foodID
	}
	return s.customFood, s.updateCustomFoodErr
}
func (s *fakeMealStore) DeleteCustomFood(_ context.Context, userID, _ string) error {
	s.deleteCustomFoodUser = userID
	return s.deleteCustomFoodErr
}
func (s *fakeMealStore) CreateFoodServingUnit(_ context.Context, _, _, label string, grams float64) (types.FoodServingUnit, error) {
	return types.FoodServingUnit{ID: "unit-1", Label: label, Grams: grams, Custom: true}, s.createServingUnitErr
}
func (s *fakeMealStore) DeleteFoodServingUnit(_ context.Context, _, _ string) error {
	return s.deleteServingUnitErr
}

// Pending aliases.
func (s *fakeMealStore) ListPendingAliases(_ context.Context, _ string) ([]types.PendingAlias, error) {
	return s.pendingAliases, s.pendingAliasesErr
}
func (s *fakeMealStore) ConfirmPendingAlias(_ context.Context, _, _ string) error {
	return s.confirmPendingAliasErr
}
func (s *fakeMealStore) RejectPendingAlias(_ context.Context, _, _ string) error {
	return s.rejectPendingAliasErr
}

// Nutrition source precedence.
func (s *fakeMealStore) GetSourcePrecedence(_ context.Context, _ string) ([]string, error) {
	return s.precedence, s.precedenceErr
}
func (s *fakeMealStore) SetSourcePrecedence(_ context.Context, _ string, order []string) error {
	s.precedence = order
	return s.setPrecedenceErr
}

func (s *fakeMealStore) GetFoodImportStatuses(_ context.Context) ([]types.FoodImportStatus, error) {
	return s.foodImportStatuses, s.foodImportStatusesErr
}

// Meal templates.
func (s *fakeMealStore) SaveTemplate(_ context.Context, _ types.MealTemplate) error {
	return s.saveTemplateErr
}
func (s *fakeMealStore) GetTemplates(_ context.Context, _ string) ([]types.MealTemplate, error) {
	return s.templates, s.templatesErr
}
func (s *fakeMealStore) GetTemplate(_ context.Context, _ string) (types.MealTemplate, error) {
	return s.template, s.templateErr
}
func (s *fakeMealStore) DeleteTemplate(_ context.Context, _, _ string) error {
	return s.deleteTemplateErr
}
func (s *fakeMealStore) LogTemplateUse(_ context.Context, _ types.TemplateLog) error {
	return s.logTemplateErr
}
func (s *fakeMealStore) SaveMeal(_ context.Context, _ types.Meal) error {
	return s.saveMealErr
}

// Body tracking — weight.
func (s *fakeMealStore) ListWeight(_ context.Context, _ string, _ int) ([]types.WeightEntry, error) {
	return s.weights, s.weightsErr
}
func (s *fakeMealStore) LogWeight(_ context.Context, _ types.WeightEntry) (string, error) {
	return "", s.logWeightErr
}
func (s *fakeMealStore) DeleteWeight(_ context.Context, _, _ string) error {
	return s.deleteWeightErr
}
func (s *fakeMealStore) WeightTrend(_ context.Context, _ string, _ int) ([]types.WeightTrend, error) {
	return s.weightTrend, s.weightTrendErr
}

// Fasting.
func (s *fakeMealStore) StartFast(_ context.Context, f types.Fast) error {
	if s.startFastErr != nil {
		return s.startFastErr
	}
	cp := f
	s.activeFast = &cp
	s.fasts = append([]types.Fast{cp}, s.fasts...)
	return nil
}
func (s *fakeMealStore) GetActiveFast(_ context.Context, _ string) (types.Fast, error) {
	if s.activeFastErr != nil {
		return types.Fast{}, s.activeFastErr
	}
	if s.activeFast == nil {
		return types.Fast{}, types.ErrNotFound
	}
	return *s.activeFast, nil
}
func (s *fakeMealStore) EndFast(_ context.Context, _, fastID string, endAt time.Time, completed bool) (types.Fast, error) {
	if s.endFastErr != nil {
		return types.Fast{}, s.endFastErr
	}
	if s.activeFast == nil || s.activeFast.ID != fastID {
		return types.Fast{}, types.ErrNotFound
	}
	f := *s.activeFast
	f.EndAt = &endAt
	f.Completed = completed
	s.activeFast = nil
	if len(s.fasts) > 0 {
		s.fasts[0] = f
	}
	return f, nil
}
func (s *fakeMealStore) ListFasts(_ context.Context, _ string, _ int) ([]types.Fast, error) {
	return s.fasts, s.listFastsErr
}

// Body tracking — measurements.
func (s *fakeMealStore) ListMeasurements(_ context.Context, _ string, _ int) ([]types.MeasurementEntry, error) {
	return s.measurements, s.measurementsErr
}
func (s *fakeMealStore) LogMeasurement(_ context.Context, _ types.MeasurementEntry) (string, error) {
	return "", s.logMeasurementErr
}
func (s *fakeMealStore) DeleteMeasurement(_ context.Context, _, _ string) error {
	return s.deleteMeasurementErr
}

// Body tracking — photos.
func (s *fakeMealStore) ListPhotoMetadata(_ context.Context, _ string) ([]types.ProgressPhoto, error) {
	return s.photoMetadata, s.photoMetadataErr
}
func (s *fakeMealStore) GetPhotoData(_ context.Context, _ string) (types.ProgressPhoto, error) {
	return s.photoData, s.photoDataErr
}
func (s *fakeMealStore) UploadPhoto(_ context.Context, _ types.ProgressPhoto) error {
	return s.uploadPhotoErr
}
func (s *fakeMealStore) DeletePhoto(_ context.Context, _, _ string) error {
	return s.deletePhotoErr
}

// Body tracking / shared.
func (s *fakeMealStore) GetMealsInRange(_ context.Context, _, _, _ string) ([]types.Meal, error) {
	return s.mealsInRange, s.mealsInRangeErr
}

// BYOK: per-user AI API keys.
func (s *fakeMealStore) GetUserAIKey(_ context.Context, _ string) (string, string, bool, error) {
	return s.aiKeyProvider, s.aiKeyEncrypted, s.aiKeyFound, s.aiKeyErr
}
func (s *fakeMealStore) SetUserAIKey(_ context.Context, _, _, _ string) error {
	return nil
}
func (s *fakeMealStore) DeleteUserAIKey(_ context.Context, _ string) error {
	return nil
}

func (s *fakeMealStore) GetUserHevyKey(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}
func (s *fakeMealStore) SetUserHevyKey(_ context.Context, _, _ string) error {
	return nil
}
func (s *fakeMealStore) DeleteUserHevyKey(_ context.Context, _ string) error {
	return nil
}

// Goals & profile.
func (s *fakeMealStore) GetProfile(_ context.Context, _ string) (types.UserProfile, error) {
	return s.profile, s.profileErr
}
func (s *fakeMealStore) UpsertProfile(_ context.Context, _ types.UserProfile) error {
	return s.upsertProfileErr
}

// Linking codes.
func (s *fakeMealStore) CreateLinkingCode(_ context.Context, _, _, _ string) error {
	return nil
}
func (s *fakeMealStore) LookupLinkingCode(_ context.Context, _ string) (types.LinkingCode, error) {
	return types.LinkingCode{}, s.rollupErr
}
func (s *fakeMealStore) LookupLinkingCodeAny(_ context.Context, _ string) (types.LinkingCode, error) {
	return types.LinkingCode{}, s.rollupErr
}
func (s *fakeMealStore) ConsumeLinkingCode(_ context.Context, _ string) error {
	return nil
}

func (s *fakeMealStore) LogWater(_ context.Context, _ types.WaterLog) error {
	return nil
}
func (s *fakeMealStore) GetWaterToday(_ context.Context, _, _ string) ([]types.WaterLog, int, error) {
	return nil, 0, nil
}
func (s *fakeMealStore) DeleteWater(_ context.Context, _, _ string) error {
	return nil
}
func (s *fakeMealStore) GetWaterDailyTotals(_ context.Context, _, _, _ string) ([]types.WaterDayTotal, error) {
	return s.waterDailyTotals, s.waterDailyTotalsErr
}
func (s *fakeMealStore) LogWorkout(_ context.Context, _ types.Workout) error {
	return nil
}
func (s *fakeMealStore) ImportWorkout(_ context.Context, _ types.Workout) error {
	return nil
}
func (s *fakeMealStore) GetWorkout(_ context.Context, _ string) (types.Workout, error) {
	return types.Workout{}, types.ErrNotFound
}
func (s *fakeMealStore) ListWorkouts(_ context.Context, _ string, _ int) ([]types.Workout, error) {
	return nil, nil
}
func (s *fakeMealStore) DeleteWorkout(_ context.Context, _, _ string) error {
	return nil
}
func (s *fakeMealStore) LogSleep(_ context.Context, _ types.SleepLog) error {
	return nil
}
func (s *fakeMealStore) GetActiveSleep(_ context.Context, _ string) (*types.SleepLog, error) {
	return nil, types.ErrNotFound
}
func (s *fakeMealStore) EndSleep(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (s *fakeMealStore) ListSleep(_ context.Context, _ string, _ int) ([]types.SleepLog, error) {
	return nil, nil
}
func (s *fakeMealStore) DeleteSleep(_ context.Context, _, _ string) error {
	return nil
}

// fakeAuthStore implements AuthStore for tests.
type fakeAuthStore struct {
	users                   map[string]types.User
	userByEmail             map[string]types.User
	phcHash                 map[string]string
	userCount               int
	apiKeys                 map[string][]types.APIKey
	keyUserID               map[string]string // hashed key -> userID
	shareTokens             map[string][]types.ShareToken
	shareUserID             map[string]string // hashed token -> userID
	loginAttempts           []loginAttemptEntry
	recentFailedAttemptsErr error
	deleteUserSessionsErr   error
	auditEvents             []types.AuditEvent
}

type loginAttemptEntry struct {
	identifier string
	succeeded  bool
	at         time.Time
}

func newFakeAuthStore() *fakeAuthStore {
	s := &fakeAuthStore{
		users:       make(map[string]types.User),
		userByEmail: make(map[string]types.User),
		phcHash:     make(map[string]string),
		apiKeys:     make(map[string][]types.APIKey),
		keyUserID:   make(map[string]string),
		shareTokens: make(map[string][]types.ShareToken),
		shareUserID: make(map[string]string),
	}
	// Pre-register a test user for existing test compatibility.
	s.users["test-user"] = types.User{ID: "test-user", Email: "test@example.com", Status: "active", CreatedAt: time.Now().UTC()}
	s.keyUserID["4c806362b613f7496abf284146efd31da90e4b16169fe001841ca17290f427c4"] = "test-user"
	return s
}

func (s *fakeAuthStore) GetUserByEmail(_ context.Context, email string) (types.User, error) {
	if u, ok := s.userByEmail[email]; ok {
		return u, nil
	}
	return types.User{}, types.ErrNotFound
}

func (s *fakeAuthStore) CreateUserWithPassword(_ context.Context, accountID, userID, email, displayName, phcHash string) (types.User, error) {
	u := types.User{ID: userID, AccountID: accountID, Email: email, DisplayName: displayName, Status: "active", CreatedAt: time.Now().UTC()}
	s.users[userID] = u
	s.userByEmail[email] = u
	s.phcHash[userID] = phcHash
	s.userCount++
	return u, nil
}

func (s *fakeAuthStore) GetPasswordHash(_ context.Context, userID string) (string, error) {
	if h, ok := s.phcHash[userID]; ok {
		return h, nil
	}
	return "", types.ErrNotFound
}

func (s *fakeAuthStore) SetPasswordHash(_ context.Context, userID, phcHash string) error {
	s.phcHash[userID] = phcHash
	return nil
}

func (s *fakeAuthStore) CountUsers(_ context.Context) (int, error) {
	return s.userCount, nil
}

func (s *fakeAuthStore) DeleteAccount(_ context.Context, userID string) error {
	delete(s.users, userID)
	return nil
}

func (s *fakeAuthStore) GetUserByAPIKey(_ context.Context, hashedKey string) (types.User, error) {
	if userID, ok := s.keyUserID[hashedKey]; ok {
		if u, ok := s.users[userID]; ok {
			return u, nil
		}
	}
	return types.User{}, types.ErrNotFound
}

func (s *fakeAuthStore) CreateAPIKey(_ context.Context, id, userID, hashedKey, label string) error {
	s.keyUserID[hashedKey] = userID
	s.apiKeys[userID] = append(s.apiKeys[userID], types.APIKey{ID: id, UserID: userID, Label: label, CreatedAt: time.Now().UTC()})
	return nil
}

func (s *fakeAuthStore) ListAPIKeys(_ context.Context, userID string) ([]types.APIKey, error) {
	return s.apiKeys[userID], nil
}

func (s *fakeAuthStore) RevokeAPIKey(_ context.Context, userID, keyID string) error {
	return nil
}

func (s *fakeAuthStore) GetUserByShareToken(_ context.Context, hashedToken string) (types.User, error) {
	if userID, ok := s.shareUserID[hashedToken]; ok {
		if u, ok := s.users[userID]; ok {
			return u, nil
		}
	}
	return types.User{}, types.ErrNotFound
}

func (s *fakeAuthStore) CreateShareToken(_ context.Context, id, userID, hashedToken, label string) error {
	s.shareUserID[hashedToken] = userID
	s.shareTokens[userID] = append(s.shareTokens[userID], types.ShareToken{ID: id, UserID: userID, Label: label, CreatedAt: time.Now().UTC()})
	return nil
}

func (s *fakeAuthStore) ListShareTokens(_ context.Context, userID string) ([]types.ShareToken, error) {
	return s.shareTokens[userID], nil
}

func (s *fakeAuthStore) RevokeShareToken(_ context.Context, userID, tokenID string) error {
	for hashed, uid := range s.shareUserID {
		if uid != userID {
			continue
		}
		for _, t := range s.shareTokens[userID] {
			if t.ID == tokenID {
				delete(s.shareUserID, hashed)
				return nil
			}
		}
	}
	return types.ErrNotFound
}

func (s *fakeAuthStore) WriteAuditEvent(_ context.Context, ev types.AuditEvent) error {
	s.auditEvents = append(s.auditEvents, ev)
	return nil
}

func (s *fakeAuthStore) RecordLoginAttempt(_ context.Context, identifier string, succeeded bool) error {
	s.loginAttempts = append(s.loginAttempts, loginAttemptEntry{identifier, succeeded, time.Now().UTC()})
	return nil
}

// --- auth.SessionRepo ---

func (s *fakeAuthStore) CreateSession(_ context.Context, sess auth.Session) error { return nil }
func (s *fakeAuthStore) GetSession(_ context.Context, id string) (auth.Session, error) {
	return auth.Session{}, errors.New("not found")
}
func (s *fakeAuthStore) TouchSession(_ context.Context, id string, lastSeen, idleExpires time.Time) error {
	return nil
}
func (s *fakeAuthStore) DeleteSession(_ context.Context, id string) error { return nil }
func (s *fakeAuthStore) DeleteUserSessions(_ context.Context, userID string) error {
	return s.deleteUserSessionsErr
}

// --- auth.LoginAttemptRepo ---

func (s *fakeAuthStore) RecentFailedAttempts(_ context.Context, identifier string, since time.Time) (int, error) {
	if s.recentFailedAttemptsErr != nil {
		return 0, s.recentFailedAttemptsErr
	}
	count := 0
	for _, a := range s.loginAttempts {
		if a.identifier == identifier && !a.succeeded && a.at.After(since) {
			count++
		}
	}
	return count, nil
}

// --- auth.TOTPRepo ---

func (s *fakeAuthStore) UpsertTOTPSecret(_ context.Context, userID, encSecret string) error {
	return nil
}
func (s *fakeAuthStore) ConfirmTOTP(_ context.Context, userID string) error { return nil }
func (s *fakeAuthStore) GetTOTPSecret(_ context.Context, userID string) (string, bool, error) {
	return "", false, types.ErrNotFound
}
func (s *fakeAuthStore) DeleteTOTP(_ context.Context, userID string) error { return nil }
func (s *fakeAuthStore) HasConfirmedTOTP(_ context.Context, userID string) (bool, error) {
	return false, nil
}

// --- auth.MFAChallengeRepo ---

func (s *fakeAuthStore) CreateMFAChallenge(_ context.Context, id, userID string, remember bool, expiresAt string) error {
	return nil
}
func (s *fakeAuthStore) GetMFAChallenge(_ context.Context, id string) (string, bool, string, error) {
	return "", false, "", types.ErrNotFound
}
func (s *fakeAuthStore) DeleteMFAChallenge(_ context.Context, id string) error { return nil }

// --- auth.RecoveryCodeRepo ---

func (s *fakeAuthStore) ReplaceRecoveryCodes(_ context.Context, userID string, hashes []string) error {
	return nil
}
func (s *fakeAuthStore) ConsumeRecoveryCode(_ context.Context, userID, hash string) (bool, error) {
	return false, nil
}

// OIDC stubs.
func (s *fakeAuthStore) GetUserByOIDCIdentity(_ context.Context, provider, subject string) (types.User, error) {
	return types.User{}, types.ErrNotFound
}
func (s *fakeAuthStore) LinkOIDCIdentity(_ context.Context, id, userID, provider, subject, email string) error {
	return nil
}
func (s *fakeAuthStore) ListOIDCIdentities(_ context.Context, userID string) ([]types.OIDCIdentity, error) {
	return nil, nil
}
func (s *fakeAuthStore) DeleteOIDCIdentity(_ context.Context, userID, id string) error {
	return nil
}
func (s *fakeAuthStore) CreateUserWithOIDC(_ context.Context, accountID, userID, email, displayName, identityID, provider, subject string) (types.User, error) {
	u := types.User{ID: userID, AccountID: accountID, Email: email, DisplayName: displayName, Status: "active", CreatedAt: time.Now().UTC()}
	return u, nil
}
func (s *fakeAuthStore) CreateOIDCState(_ context.Context, id, nonce, pkceVerifier, linkUserID, next, expiresAt string) error {
	return nil
}
func (s *fakeAuthStore) ConsumeOIDCState(_ context.Context, id string) (nonce, pkceVerifier, linkUserID, next string, err error) {
	return "", "", "", "", types.ErrNotFound
}
func (s *fakeAuthStore) DeleteOIDCState(_ context.Context, id string) error {
	return nil
}

// Email tokens.
func (s *fakeAuthStore) MarkEmailVerified(_ context.Context, userID string) error      { return nil }
func (s *fakeAuthStore) UpdateUserEmail(_ context.Context, userID, email string) error { return nil }
func (s *fakeAuthStore) CreateEmailToken(_ context.Context, id, userID, purpose, expiresAt string) error {
	return nil
}
func (s *fakeAuthStore) ConsumeEmailToken(_ context.Context, id, purpose string) (string, error) {
	return "", types.ErrNotFound
}

// Magic codes.
func (s *fakeAuthStore) UpsertMagicCode(_ context.Context, userID, codeHash, expiresAt string) error {
	return nil
}
func (s *fakeAuthStore) GetMagicCode(_ context.Context, userID string) (string, string, int, error) {
	return "", "", 0, types.ErrNotFound
}
func (s *fakeAuthStore) IncrementMagicCodeAttempts(_ context.Context, userID string) error {
	return nil
}
func (s *fakeAuthStore) DeleteMagicCode(_ context.Context, userID string) error { return nil }
func (s *fakeAuthStore) DeleteEmailTokensByUserAndPurpose(_ context.Context, userID, purpose string) error {
	return nil
}

type fakeMealLogger struct {
	lastMsg  types.InboundMessage
	lastMeal types.Meal
	err      error
}

func (l *fakeMealLogger) Handle(_ context.Context, msg types.InboundMessage) error {
	l.lastMsg = msg
	return l.err
}

func (l *fakeMealLogger) LogMeal(_ context.Context, meal types.Meal) error {
	l.lastMeal = meal
	return l.err
}

type fakeSuggester struct {
	sug types.MealSuggestion
	err error
}

func (s *fakeSuggester) Suggest(_ context.Context, _ string) (types.MealSuggestion, error) {
	return s.sug, s.err
}

func (s *fakeSuggester) SuggestFromIngredients(_ context.Context, _ string, _ []string) (types.MealSuggestion, error) {
	return s.sug, s.err
}

// --- helpers ---

// sug is optional (variadic) so existing call sites that don't care about
// suggestions don't need to change.
func newHandler(store MealStore, logger MealLogger, sug ...Suggester) *Handler {
	store2 := newFakeAuthStore()
	var suggester Suggester
	if len(sug) > 0 {
		suggester = sug[0]
	}
	return New(store, logger, time.UTC, suggester, nil,
		WithAuth(store2, store2, store2, store2, store2, store2, nil, "DietDaemon", AuthConfig{
			SessionCfg: auth.SessionConfig{
				IdleTTL:     1 * time.Hour,
				AbsoluteTTL: 24 * time.Hour,
				RememberTTL: 72 * time.Hour,
			},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
			CookieSecure:     false,
		}),
	)
}

func doRequest(h *Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var rBody []byte
	if body != nil {
		rBody, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(rBody))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// If no Authorization header provided, use test API key.
	if _, ok := headers["Authorization"]; !ok {
		req.Header.Set("Authorization", "Bearer test-api-key")
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode JSON: %v (body=%q)", err, rec.Body.String())
	}
	return v
}

// --- auth tests ---

func TestBearerTokenEdgeCases(t *testing.T) {
	// Shorter than "Bearer "
	if got := bearerToken(&http.Request{Header: http.Header{"Authorization": {"Bear x"}}}); got != "" {
		t.Errorf("short auth header = %q, want empty", got)
	}
	// No Bearer prefix.
	if got := bearerToken(&http.Request{Header: http.Header{"Authorization": {"Basic xyz"}}}); got != "" {
		t.Errorf("Basic auth = %q, want empty", got)
	}
}

// --- handler tests ---

func TestHandleRollupsToday(t *testing.T) {
	store := newFakeMealStore()
	store.rollup = types.DailyRollup{
		UserID:   "test-user",
		Date:     "2026-06-17",
		Consumed: types.Macros{Calories: 2100, Protein: 140},
		Targets:  types.Macros{Calories: 3000, Protein: 180},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rollup := decodeJSON[types.DailyRollup](t, rec)
	if rollup.Consumed.Calories != 2100 {
		t.Errorf("calories = %v, want 2100", rollup.Consumed.Calories)
	}
}

func TestHandleGetFoodCatalogFallback(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetailErr = types.ErrNotFound
	store.foodsByID = map[string]types.FoodMatch{
		"catalog-1": {
			FoodID: "catalog-1", Name: "Catalog Only Food", Source: "usda",
			Per100g: types.Macros{Calories: 100, Protein: 10},
		},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/catalog-1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	fd := decodeJSON[types.FoodDetail](t, rec)
	if fd.Name != "Catalog Only Food" || fd.InLibrary {
		t.Errorf("unexpected fallback food detail: %+v", fd)
	}
	if len(fd.Aliases) != 0 {
		t.Errorf("expected no aliases, got %v", fd.Aliases)
	}
}

func TestHandleGetFoodNotFoundAnywhere(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetailErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/nonexistent", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRollupsRange(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "test-user", Date: "2026-06-15", Consumed: types.Macros{Calories: 2000}},
		{UserID: "test-user", Date: "2026-06-16", Consumed: types.Macros{Calories: 2200}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/rollups/range?start=2026-06-15&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rollups := decodeJSON[[]types.DailyRollup](t, rec)
	if len(rollups) != 2 {
		t.Errorf("rollups count = %d, want 2", len(rollups))
	}
}

func TestHandleRollupsRangeMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/rollups/range", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing params expected 400, got %d", rec.Code)
	}
}

func TestHandleRollupsRangeNullReturn(t *testing.T) {
	store := newFakeMealStore()
	// rollups is nil (not initialized) — handler should return [] not null.
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/rollups/range?start=2026-06-15&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "[") {
		t.Errorf("expected JSON array, got %s", rec.Body.String())
	}
}

func TestHandleMealsList(t *testing.T) {
	store := newFakeMealStore()
	store.recentMeals = []types.Meal{
		{ID: "m1", UserID: "test-user", RawText: "200g chicken"},
		{ID: "m2", UserID: "test-user", RawText: "2 eggs"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals?limit=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	meals := decodeJSON[[]types.Meal](t, rec)
	if len(meals) != 2 {
		t.Errorf("meals count = %d, want 2", len(meals))
	}
}

func TestHandleMealsListDefaultLimit(t *testing.T) {
	store := newFakeMealStore()
	store.recentMeals = []types.Meal{}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	meals := decodeJSON[[]types.Meal](t, rec)
	if meals == nil {
		t.Error("expected empty array, got null")
	}
}

func TestHandleMealDetail(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID:      "m1",
		UserID:  "test-user",
		RawText: "200g chicken",
		Items:   []types.ResolvedItem{},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals/m1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	meal := decodeJSON[types.Meal](t, rec)
	if meal.ID != "m1" {
		t.Errorf("meal ID = %q, want m1", meal.ID)
	}
}

func TestHandleMealDetailWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID:     "m1",
		UserID: "other-user",
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals/m1", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-user meal access expected 404, got %d", rec.Code)
	}
}

func TestHandleMealDetailNotFound(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals/nonexistent", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("missing meal expected 404, got %d", rec.Code)
	}
}

func TestHandleCorrectItem(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID:      "m1",
		UserID:  "test-user",
		RawText: "200g chicken",
		Items: []types.ResolvedItem{
			{Parsed: types.ParsedItem{RawPhrase: "chicken"}},
		},
	}
	h := newHandler(store, &fakeMealLogger{})

	body := types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: "chicken", NormalizedGrams: 200},
		Match:  types.FoodMatch{FoodID: "chicken", Name: "Chicken Breast"},
		Macros: types.Macros{Calories: 330, Protein: 62},
	}
	rec := doRequest(h, "POST", "/api/v1/meals/m1/items/0/correct", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCorrectItemBadIndex(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/meals/m1/items/abc/correct", types.ResolvedItem{}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad index expected 400, got %d", rec.Code)
	}
}

func TestHandleCorrectItemNegativeIndex(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/meals/m1/items/-1/correct", types.ResolvedItem{}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("negative index expected 400, got %d", rec.Code)
	}
}

func TestHandleLogMeal(t *testing.T) {
	logger := &fakeMealLogger{}
	store := newFakeMealStore()
	h := newHandler(store, logger)

	body := map[string]string{"text": "200g chicken, 2 eggs"}
	rec := doRequest(h, "POST", "/api/v1/meals/log", body, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMsg.Text != "200g chicken, 2 eggs" {
		t.Errorf("logged text = %q, want %q", logger.lastMsg.Text, "200g chicken, 2 eggs")
	}
	if logger.lastMsg.UserID != "test-user" {
		t.Errorf("logged userID = %q, want test-user", logger.lastMsg.UserID)
	}
}

func TestHandleLogMealEmptyText(t *testing.T) {
	logger := &fakeMealLogger{}
	store := newFakeMealStore()
	h := newHandler(store, logger)

	body := map[string]string{"text": ""}
	rec := doRequest(h, "POST", "/api/v1/meals/log", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty text expected 400, got %d", rec.Code)
	}
}

func TestHandleLogMealInvalidJSON(t *testing.T) {
	logger := &fakeMealLogger{}
	store := newFakeMealStore()
	h := newHandler(store, logger)

	req := httptest.NewRequest("POST", "/api/v1/meals/log", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleLogMealLoggerError(t *testing.T) {
	logger := &fakeMealLogger{err: errors.New("pipeline busy")}
	store := newFakeMealStore()
	h := newHandler(store, logger)

	body := map[string]string{"text": "200g chicken"}
	rec := doRequest(h, "POST", "/api/v1/meals/log", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("logger error expected 500, got %d", rec.Code)
	}
}

func TestHandleRollupsTodayNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.rollupErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for not-found rollup, got %d", rec.Code)
	}
}

// TestShareTokenReadOnlyFlow exercises the full share-link lifecycle:
// create → read via the /shared/{token}/... prefix with no cookie/API key →
// mutation on that prefix is rejected → revoke → subsequent reads 401.
func TestShareTokenReadOnlyFlow(t *testing.T) {
	store := newFakeMealStore()
	store.rollup = types.DailyRollup{
		UserID:   "test-user",
		Date:     "2026-06-17",
		Consumed: types.Macros{Calories: 2100, Protein: 140},
		Targets:  types.Macros{Calories: 3000, Protein: 180},
	}
	h := newHandler(store, &fakeMealLogger{})

	// Create a share token as the authenticated user.
	createRec := doRequest(h, "POST", "/api/v1/auth/share-tokens", map[string]string{"label": "test"}, nil)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create share token: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeJSON[types.NewShareTokenResponse](t, createRec)
	if created.Token == "" {
		t.Fatal("create share token: empty raw token")
	}

	// Read the shared dashboard with no Authorization header at all.
	readRec := doRequest(h, "GET", "/api/v1/shared/"+created.Token+"/rollups/today", nil, map[string]string{"Authorization": ""})
	if readRec.Code != http.StatusOK {
		t.Fatalf("shared read: expected 200, got %d: %s", readRec.Code, readRec.Body.String())
	}
	rollup := decodeJSON[types.DailyRollup](t, readRec)
	if rollup.Consumed.Calories != 2100 {
		t.Errorf("shared read: calories = %v, want 2100", rollup.Consumed.Calories)
	}

	// Mutations are not exposed on the shared prefix.
	mutateRec := doRequest(h, "POST", "/api/v1/shared/"+created.Token+"/rollups/today", nil, map[string]string{"Authorization": ""})
	if mutateRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("shared mutate: expected 405, got %d", mutateRec.Code)
	}

	// Revoke, then the same link must stop working.
	revokeRec := doRequest(h, "DELETE", "/api/v1/auth/share-tokens/"+created.ID, nil, nil)
	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("revoke share token: expected 204, got %d: %s", revokeRec.Code, revokeRec.Body.String())
	}
	postRevokeRec := doRequest(h, "GET", "/api/v1/shared/"+created.Token+"/rollups/today", nil, map[string]string{"Authorization": ""})
	if postRevokeRec.Code != http.StatusUnauthorized {
		t.Errorf("shared read after revoke: expected 401, got %d", postRevokeRec.Code)
	}
}

func TestHandleSuggest(t *testing.T) {
	store := newFakeMealStore()
	sug := &fakeSuggester{sug: types.MealSuggestion{
		Remaining: types.Macros{Calories: 500, Protein: 30},
		Candidates: []types.SuggestedCombo{
			{Score: 0.9},
		},
		Message: "test message",
		Source:  "llm",
	}}
	h := newHandler(store, &fakeMealLogger{}, sug)

	rec := doRequest(h, "GET", "/api/v1/suggest", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	got := decodeJSON[types.MealSuggestion](t, rec)
	if got.Message != "test message" {
		t.Errorf("message = %q, want %q", got.Message, "test message")
	}
	if got.Source != "llm" {
		t.Errorf("source = %q, want %q", got.Source, "llm")
	}
}

func TestHandleSuggestError(t *testing.T) {
	store := newFakeMealStore()
	sug := &fakeSuggester{err: types.ErrNotFound}
	h := newHandler(store, &fakeMealLogger{}, sug)

	rec := doRequest(h, "GET", "/api/v1/suggest", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSuggestFromIngredients(t *testing.T) {
	store := newFakeMealStore()
	sug := &fakeSuggester{sug: types.MealSuggestion{
		Remaining: types.Macros{Calories: 500, Protein: 30},
		Message:   "on-hand suggestion",
		Source:    "rules",
	}}
	h := newHandler(store, &fakeMealLogger{}, sug)

	rec := doRequest(h, "POST", "/api/v1/suggest/ingredients", map[string]any{"food_ids": []string{"f1", "f2"}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	got := decodeJSON[types.MealSuggestion](t, rec)
	if got.Message != "on-hand suggestion" {
		t.Errorf("message = %q, want %q", got.Message, "on-hand suggestion")
	}
}

func TestHandleSuggestFromIngredientsRequiresFoodIDs(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}, &fakeSuggester{})

	rec := doRequest(h, "POST", "/api/v1/suggest/ingredients", map[string]any{"food_ids": []string{}}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty food_ids, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerWriteErrGeneric(t *testing.T) {
	store := newFakeMealStore()
	store.rollupErr = errors.New("db connection lost")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("generic error expected 500, got %d", rec.Code)
	}
}

func TestMealsListNullReturn(t *testing.T) {
	store := newFakeMealStore()
	// recentMeals is nil (not initialized slice).
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "[") {
		t.Errorf("expected JSON array, got %s", rec.Body.String())
	}
}

func TestAuthHeaderEdgeCases(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	// Authorization header with exactly "Bearer " and nothing else.
	rec := doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Bearer ",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("empty Bearer token expected 401, got %d", rec.Code)
	}

	// No "Bearer" prefix at all.
	rec = doRequest(h, "GET", "/api/v1/rollups/today", nil, map[string]string{
		"Authorization": "Basic dGVzdDp0ZXN0",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Basic auth expected 401, got %d", rec.Code)
	}
}

// --- targets + item add/delete tests ---

func TestSetTargets(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := types.Macros{Calories: 3000, Protein: 180, Carbs: 360, Fat: 90, Fiber: 38}
	rec := doRequest(h, "PUT", "/api/v1/targets", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT targets expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.targets.Targets.Protein != 180 {
		t.Errorf("targets not persisted, got %+v", store.targets)
	}
	if store.rollup.Targets.Calories != 3000 {
		t.Errorf("rollup targets not refreshed, got %+v", store.rollup.Targets)
	}
}

// TestSetTargetsWaterGoalFirstCallDefaults verifies that a first-ever PUT
// with no water_goal_ml in the body (old-frontend bare-Macros payload)
// defaults the stored water goal sensibly rather than leaving it zero.
func TestSetTargetsWaterGoalFirstCallDefaults(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := types.Macros{Calories: 2500, Protein: 150, Carbs: 300, Fat: 70, Fiber: 30}
	rec := doRequest(h, "PUT", "/api/v1/targets", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT targets expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.targets.WaterGoalMl != defaultWaterGoalMl {
		t.Errorf("expected default water goal %d on first call, got %d", defaultWaterGoalMl, store.targets.WaterGoalMl)
	}
}

// TestSetTargetsWaterGoalPreservedWhenOmitted verifies a subsequent PUT that
// omits water_goal_ml (backward-compatible bare Macros body) keeps the
// previously stored value instead of resetting it.
func TestSetTargetsWaterGoalPreservedWhenOmitted(t *testing.T) {
	store := newFakeMealStore()
	store.targets = types.DailyTargets{UserID: "test-user", WaterGoalMl: 2500}
	h := newHandler(store, &fakeMealLogger{})

	body := types.Macros{Calories: 2500, Protein: 150, Carbs: 300, Fat: 70, Fiber: 30}
	rec := doRequest(h, "PUT", "/api/v1/targets", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT targets expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.targets.WaterGoalMl != 2500 {
		t.Errorf("expected water goal preserved at 2500, got %d", store.targets.WaterGoalMl)
	}
}

// TestSetTargetsWaterGoalUpdatedWhenPresent verifies an explicit, positive
// water_goal_ml in the body is stored.
func TestSetTargetsWaterGoalUpdatedWhenPresent(t *testing.T) {
	store := newFakeMealStore()
	store.targets = types.DailyTargets{UserID: "test-user", WaterGoalMl: 2000}
	h := newHandler(store, &fakeMealLogger{})

	body := struct {
		types.Macros
		WaterGoalMl int `json:"water_goal_ml"`
	}{
		Macros:      types.Macros{Calories: 2500, Protein: 150, Carbs: 300, Fat: 70, Fiber: 30},
		WaterGoalMl: 3000,
	}
	rec := doRequest(h, "PUT", "/api/v1/targets", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT targets expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.targets.WaterGoalMl != 3000 {
		t.Errorf("expected water goal updated to 3000, got %d", store.targets.WaterGoalMl)
	}
}

// TestSetTargetsWaterGoalRejectsNonPositive verifies a present but <= 0
// water_goal_ml is rejected as a validation error, matching the handler's
// existing macro validation error style/status code.
func TestSetTargetsWaterGoalRejectsNonPositive(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := struct {
		types.Macros
		WaterGoalMl int `json:"water_goal_ml"`
	}{
		Macros:      types.Macros{Calories: 2500, Protein: 150, Carbs: 300, Fat: 70, Fiber: 30},
		WaterGoalMl: 0,
	}
	rec := doRequest(h, "PUT", "/api/v1/targets", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT targets with water_goal_ml=0 expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestGetWaterToday verifies the goal_ml field falls back to the package
// default when there's no stored targets row, and otherwise reflects the
// stored (possibly non-default) water goal.
func TestGetWaterToday(t *testing.T) {
	t.Run("no targets row falls back to default", func(t *testing.T) {
		store := newFakeMealStore()
		store.targetsErr = types.ErrNotFound
		h := newHandler(store, &fakeMealLogger{})

		rec := doRequest(h, "GET", "/api/v1/body/water", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET water expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out["goal_ml"] != float64(defaultWaterGoalMl) {
			t.Errorf("expected default goal_ml %d, got %v", defaultWaterGoalMl, out["goal_ml"])
		}
	})

	t.Run("stored non-default water goal", func(t *testing.T) {
		store := newFakeMealStore()
		store.targets = types.DailyTargets{UserID: "test-user", WaterGoalMl: 3200}
		h := newHandler(store, &fakeMealLogger{})

		rec := doRequest(h, "GET", "/api/v1/body/water", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET water expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out["goal_ml"] != float64(3200) {
			t.Errorf("expected stored goal_ml 3200, got %v", out["goal_ml"])
		}
	})
}

func TestAddMealItem(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "test-user", Items: []types.ResolvedItem{}}
	h := newHandler(store, &fakeMealLogger{})

	item := types.ResolvedItem{
		Parsed: types.ParsedItem{RawPhrase: "banana", NormalizedGrams: 120},
		Match:  types.FoodMatch{Name: "Banana", Source: "taco"},
		Macros: types.Macros{Calories: 107, Protein: 1, Carbs: 27},
	}
	rec := doRequest(h, "POST", "/api/v1/meals/m1/items", item, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST item expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.Meal](t, rec)
	if len(got.Items) != 1 || got.Items[0].Match.Name != "Banana" {
		t.Errorf("item not added, got %+v", got.Items)
	}
}

func TestDeleteMealItem(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "test-user", Items: []types.ResolvedItem{
		{Match: types.FoodMatch{Name: "A"}},
		{Match: types.FoodMatch{Name: "B"}},
	}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/meals/m1/items/0", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE item expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.Meal](t, rec)
	if len(got.Items) != 1 || got.Items[0].Match.Name != "B" {
		t.Errorf("wrong item deleted, got %+v", got.Items)
	}
}

func TestDeleteMealItemForbiddenOtherUser(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{ID: "m1", UserID: "someone-else", Items: []types.ResolvedItem{{}}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/meals/m1/items/0", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("deleting another user's item expected 404, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Meals — latest
// ---------------------------------------------------------------------------

func TestMealsLatest(t *testing.T) {
	store := newFakeMealStore()
	store.latestMealTime = "2026-06-17T12:00:00Z"
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals/latest", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["latest"] != "2026-06-17T12:00:00Z" {
		t.Errorf("latest = %q, want 2026-06-17T12:00:00Z", got["latest"])
	}
}

func TestMealsLatestEmpty(t *testing.T) {
	store := newFakeMealStore()
	store.latestMealTimeErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/meals/latest", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["latest"] != "" {
		t.Errorf("latest = %q, want empty", got["latest"])
	}
}

// ---------------------------------------------------------------------------
// Food discovery
// ---------------------------------------------------------------------------

func TestListFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{
		{FoodID: "f1", Name: "Chicken", Source: "food_library"},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 || foods[0].Name != "Chicken" {
		t.Errorf("unexpected foods: %+v", foods)
	}
}

func TestListFoodsNullReturn(t *testing.T) {
	store := newFakeMealStore()
	// foodList is nil.
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "[") {
		t.Errorf("expected JSON array, got %s", rec.Body.String())
	}
}

func TestSearchFoodsMissingQ(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/search", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing q, got %d", rec.Code)
	}
}

func TestSearchFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Banana"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/search?q=banana", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 1 {
		t.Errorf("expected 1 result, got %d", len(foods))
	}
}

func TestFrequentFoods(t *testing.T) {
	store := newFakeMealStore()
	store.foodList = []types.FoodDetail{{FoodID: "f1", Name: "Rice"}, {FoodID: "f2", Name: "Beans"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/frequent?limit=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	foods := decodeJSON[[]types.FoodDetail](t, rec)
	if len(foods) != 2 {
		t.Errorf("expected 2, got %d", len(foods))
	}
}

func TestGetFoodNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetailErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetFood(t *testing.T) {
	store := newFakeMealStore()
	store.foodDetail = types.FoodDetail{
		FoodID:  "f1",
		Name:    "Egg",
		Aliases: []types.FoodAlias{{FoodID: "f1", Normalized: "egg"}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/foods/f1", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[types.FoodDetail](t, rec)
	if got.Name != "Egg" || len(got.Aliases) != 1 {
		t.Errorf("unexpected food detail: %+v", got)
	}
}

func TestAddAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/f1/aliases", map[string]string{"alias": "ovo"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddAliasMissing(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/f1/aliases", map[string]string{"alias": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestDeleteAlias(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/foods/f1/aliases/egg", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func customFoodBody() map[string]any {
	return map[string]any{
		"name": "Protein oats", "calories": 210.0, "protein": 12.0,
		"carbs": 31.0, "fat": 5.0, "fiber": 6.0, "basis_grams": 60.0,
	}
}

func TestCreateCustomFood(t *testing.T) {
	store := newFakeMealStore()
	store.customFood = types.FoodDetail{FoodID: "custom-1", Name: "Protein oats", Source: "custom", ServingSize: 60, ServingUnit: "g", InLibrary: true}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/foods/custom", customFoodBody(), nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.createCustomFoodUser != "test-user" || store.createCustomFoodIn.BasisGrams != 60 || store.createCustomFoodIn.Macros.Protein != 12 {
		t.Fatalf("unexpected custom food input: user=%q input=%+v", store.createCustomFoodUser, store.createCustomFoodIn)
	}
	if got := decodeJSON[types.FoodDetail](t, rec); got.FoodID != "custom-1" {
		t.Errorf("food_id = %q, want custom-1", got.FoodID)
	}
}

func TestCreateCustomFoodValidation(t *testing.T) {
	for name, body := range map[string]map[string]any{
		"missing nutrient":  {"name": "Oats", "calories": 1, "protein": 1, "carbs": 1, "fat": 1, "basis_grams": 100},
		"negative nutrient": {"name": "Oats", "calories": -1, "protein": 1, "carbs": 1, "fat": 1, "fiber": 1, "basis_grams": 100},
		"zero basis":        {"name": "Oats", "calories": 1, "protein": 1, "carbs": 1, "fat": 1, "fiber": 1, "basis_grams": 0},
	} {
		t.Run(name, func(t *testing.T) {
			rec := doRequest(newHandler(newFakeMealStore(), &fakeMealLogger{}), "POST", "/api/v1/foods/custom", body, nil)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCustomFoodUpdateDeleteAndErrors(t *testing.T) {
	store := newFakeMealStore()
	store.customFood = types.FoodDetail{FoodID: "custom-1", Name: "Protein oats", Source: "custom"}
	h := newHandler(store, &fakeMealLogger{})

	if rec := doRequest(h, "PUT", "/api/v1/foods/custom-1/custom", customFoodBody(), nil); rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.updateCustomFoodUser != "test-user" || store.updateCustomFoodIn.BasisGrams != 60 {
		t.Fatalf("update did not use authenticated user/input: %q %+v", store.updateCustomFoodUser, store.updateCustomFoodIn)
	}
	if rec := doRequest(h, "DELETE", "/api/v1/foods/custom-1/custom", nil, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.deleteCustomFoodUser != "test-user" {
		t.Fatalf("delete user = %q, want test-user", store.deleteCustomFoodUser)
	}

	store.createCustomFoodErr = types.ErrConflict
	if rec := doRequest(h, "POST", "/api/v1/foods/custom", customFoodBody(), nil); rec.Code != http.StatusConflict {
		t.Errorf("conflict: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	store.deleteCustomFoodErr = types.ErrNotFound
	if rec := doRequest(h, "DELETE", "/api/v1/foods/custom-1/custom", nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("not found: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Meal templates
// ---------------------------------------------------------------------------

func TestListTemplates(t *testing.T) {
	store := newFakeMealStore()
	store.templates = []types.MealTemplate{{ID: "t1", Name: "Morning"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	templates := decodeJSON[[]types.MealTemplate](t, rec)
	if len(templates) != 1 || templates[0].Name != "Morning" {
		t.Errorf("unexpected templates: %+v", templates)
	}
}

func TestCreateTemplate(t *testing.T) {
	store := newFakeMealStore()
	logger := &fakeMealLogger{}
	h := newHandler(store, logger)

	body := map[string]any{
		"name":  "My Template",
		"items": []types.ResolvedItem{{Match: types.FoodMatch{Name: "Egg"}}},
	}
	rec := doRequest(h, "POST", "/api/v1/templates", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTemplateValidation(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates", map[string]string{"name": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty name expected 400, got %d", rec.Code)
	}
}

func TestComposeTemplate(t *testing.T) {
	store := newFakeMealStore()
	store.foodsByID = map[string]types.FoodMatch{
		"egg": {FoodID: "egg", Name: "Egg", Source: "food_library", Per100g: types.Macros{Calories: 155, Protein: 13, Carbs: 1.1, Fat: 11}},
	}
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "Breakfast",
		"items": []map[string]any{
			{"food_id": "egg", "grams": 200},
		},
	}
	rec := doRequest(h, "POST", "/api/v1/templates/compose", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	tmpl := decodeJSON[types.MealTemplate](t, rec)
	if len(tmpl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(tmpl.Items))
	}
	item := tmpl.Items[0]
	if item.Macros.Calories != 310 {
		t.Errorf("expected scaled calories 310, got %v", item.Macros.Calories)
	}
	if item.Parsed.NormalizedGrams != 200 {
		t.Errorf("expected 200g, got %v", item.Parsed.NormalizedGrams)
	}
}

func TestComposeTemplateUnknownFood(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"name": "Breakfast",
		"items": []map[string]any{
			{"food_id": "missing-food", "grams": 100},
		},
	}
	rec := doRequest(h, "POST", "/api/v1/templates/compose", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if body := decodeJSON[errorEnvelope](t, rec); body.Error.Code != ErrorValidation {
		t.Errorf("expected validation error, got %#v", body)
	}
}

func TestComposeTemplateValidation(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/templates/compose", map[string]string{"name": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty name/items expected 400, got %d", rec.Code)
	}
}

func TestGetTemplateNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.templateErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/templates/missing", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteTemplate(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/templates/t1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestLogTemplate(t *testing.T) {
	store := newFakeMealStore()
	store.template = types.MealTemplate{
		ID:     "t1",
		UserID: "test-user",
		Name:   "Morning",
		Items:  []types.ResolvedItem{{Match: types.FoodMatch{Name: "Egg"}}},
	}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger)

	rec := doRequest(h, "POST", "/api/v1/templates/t1/log", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMeal.UserID != "test-user" {
		t.Errorf("LogMeal not called")
	}
}

func TestDuplicateMeal(t *testing.T) {
	store := newFakeMealStore()
	store.meals["m1"] = types.Meal{
		ID: "m1", UserID: "test-user", RawText: "200g chicken",
		Items: []types.ResolvedItem{{Match: types.FoodMatch{Name: "Chicken"}}},
	}
	logger := &fakeMealLogger{}
	h := newHandler(store, logger)

	rec := doRequest(h, "POST", "/api/v1/meals/m1/duplicate", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if logger.lastMeal.Items == nil {
		t.Errorf("LogMeal not called with items")
	}
}

// ---------------------------------------------------------------------------
// Body tracking
// ---------------------------------------------------------------------------

func TestListWeight(t *testing.T) {
	store := newFakeMealStore()
	store.weights = []types.WeightEntry{{ID: "w1", WeightKg: 80.5, Date: "2026-06-17"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/weight?days=30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	weights := decodeJSON[[]types.WeightEntry](t, rec)
	if len(weights) != 1 || weights[0].WeightKg != 80.5 {
		t.Errorf("unexpected weights: %+v", weights)
	}
}

func TestLogWeightValidation(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/weight", map[string]any{"weight_kg": 0}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("zero weight expected 400, got %d", rec.Code)
	}
}

func TestLogWeight(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/body/weight", map[string]any{
		"weight_kg": 80.5, "date": "2026-06-17", "note": "morning",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWeightTrend(t *testing.T) {
	store := newFakeMealStore()
	store.weightTrend = []types.WeightTrend{
		{Date: "2026-06-15", WeightKg: 80, RollingAvg: 80},
		{Date: "2026-06-16", WeightKg: 79.5, RollingAvg: 79.75},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/weight/trend?days=14", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	trend := decodeJSON[[]types.WeightTrend](t, rec)
	if len(trend) != 2 {
		t.Errorf("expected 2, got %d", len(trend))
	}
}

func TestDeleteWeight(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/weight/w1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestListMeasurements(t *testing.T) {
	store := newFakeMealStore()
	store.measurements = []types.MeasurementEntry{{ID: "m1", WaistCm: 90, Date: "2026-06-17"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/measurements?days=30", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	got := decodeJSON[[]types.MeasurementEntry](t, rec)
	if len(got) != 1 || got[0].WaistCm != 90 {
		t.Errorf("unexpected measurements: %+v", got)
	}
}

func TestLogMeasurements(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := types.MeasurementEntry{Date: "2026-06-17", WaistCm: 90, HipsCm: 100}
	rec := doRequest(h, "POST", "/api/v1/body/measurements", body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteMeasurement(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/measurements/m1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestListPhotos(t *testing.T) {
	store := newFakeMealStore()
	store.photoMetadata = []types.ProgressPhoto{{ID: "p1", View: "front"}}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/photos", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	photos := decodeJSON[[]types.ProgressPhoto](t, rec)
	if len(photos) != 1 {
		t.Errorf("expected 1, got %d", len(photos))
	}
}

func TestPhotoDataNotFound(t *testing.T) {
	store := newFakeMealStore()
	store.photoDataErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/photos/missing/data", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDeletePhoto(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/body/photos/p1", nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestBodySummaryEmpty(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/body/summary", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Goals & profile
// ---------------------------------------------------------------------------

func TestGetProfileNotOnboarded(t *testing.T) {
	store := newFakeMealStore()
	store.profileErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/profile", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	profile := decodeJSON[types.UserProfile](t, rec)
	if profile.Onboarded {
		t.Errorf("expected onboarded=false for missing profile")
	}
}

func TestGetProfile(t *testing.T) {
	store := newFakeMealStore()
	store.profile = types.UserProfile{UserID: "test-user", HeightCm: 175, Onboarded: true}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/profile", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	profile := decodeJSON[types.UserProfile](t, rec)
	if profile.HeightCm != 175 || !profile.Onboarded {
		t.Errorf("unexpected profile: %+v", profile)
	}
}

func TestUpsertProfile(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := types.UserProfile{HeightCm: 180, Gender: "male", ActivityLevel: "moderate"}
	rec := doRequest(h, "PUT", "/api/v1/profile", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCalculateTDEEMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/tdee", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing params, got %d", rec.Code)
	}
}

func TestCalculateTDEE(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/tdee?weight_kg=80&height_cm=175&age=30&gender=male&activity=moderate", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	result := decodeJSON[types.TDEEResult](t, rec)
	if result.BMR <= 0 || result.TDEE <= 0 {
		t.Errorf("unexpected TDEE result: %+v", result)
	}
}

func TestGoalSuggestionsNoProfile(t *testing.T) {
	store := newFakeMealStore()
	store.profileErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/goals/suggestions", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Data export
// ---------------------------------------------------------------------------

func TestExportMealsJSON(t *testing.T) {
	store := newFakeMealStore()
	store.mealsInRange = []types.Meal{
		{ID: "m1", UserID: "test-user", RawText: "200g chicken",
			Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 330, Protein: 62}}}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExportMealsCSV(t *testing.T) {
	store := newFakeMealStore()
	store.mealsInRange = []types.Meal{
		{ID: "m1", UserID: "test-user", RawText: "200g chicken",
			Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 330, Protein: 62}}}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals?start=2026-06-01&end=2026-06-17&format=csv", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("expected text/csv, got %s", ct)
	}
	// Exact byte body: proves the internal/exportfmt extraction changed nothing
	// about the REST endpoint's on-the-wire CSV output.
	const wantMeals = "id,date,raw_text,kcal,protein,carbs,fat,fiber\n" +
		"m1,0001-01-01,\"200g chicken\",330,62.0,0.0,0.0,0.0\n"
	if got := rec.Body.String(); got != wantMeals {
		t.Errorf("csv body mismatch:\ngot:  %q\nwant: %q", got, wantMeals)
	}
}

func TestExportRollupsJSON(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "test-user", Date: "2026-06-17", Consumed: types.Macros{Calories: 2100}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-17", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExportRollupsCSV(t *testing.T) {
	store := newFakeMealStore()
	store.rollups = []types.DailyRollup{
		{UserID: "test-user", Date: "2026-06-17", Consumed: types.Macros{Calories: 2100}},
	}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/rollups?start=2026-06-01&end=2026-06-17&format=csv", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("expected text/csv, got %s", ct)
	}
	// Exact byte body: proves the internal/exportfmt extraction changed nothing
	// about the REST endpoint's on-the-wire CSV output.
	const wantRollups = "date,consumed_kcal,consumed_protein,consumed_carbs,consumed_fat,consumed_fiber,target_kcal,target_protein,target_carbs,target_fat,target_fiber\n" +
		"2026-06-17,2100,0.0,0.0,0.0,0.0,0,0.0,0.0,0.0,0.0\n"
	if got := rec.Body.String(); got != wantRollups {
		t.Errorf("csv body mismatch:\ngot:  %q\nwant: %q", got, wantRollups)
	}
}

func TestExportMissingParams(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/export/meals", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing params, got %d", rec.Code)
	}
}

// Stubs — not exercised by existing tests.

func (s *fakeAuthStore) GetOrCreateWebAuthnHandle(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (s *fakeAuthStore) GetUserByWebAuthnHandle(_ context.Context, _ string) (types.User, error) {
	return types.User{}, types.ErrNotFound
}
func (s *fakeAuthStore) CreateWebAuthnCredential(_ context.Context, _, _, _, _ string, _ int, _ string) error {
	return nil
}
func (s *fakeAuthStore) ListWebAuthnCredentials(_ context.Context, _ string) ([]types.Passkey, error) {
	return nil, nil
}
func (s *fakeAuthStore) GetWebAuthnCredentialsRaw(_ context.Context, _ string) ([]types.WebAuthnCredential, error) {
	return nil, nil
}
func (s *fakeAuthStore) UpdateWebAuthnCredentialOnAuth(_ context.Context, _, _ string, _ int, _ string) error {
	return nil
}
func (s *fakeAuthStore) RenameWebAuthnCredential(_ context.Context, _, _, _ string) error { return nil }
func (s *fakeAuthStore) DeleteWebAuthnCredential(_ context.Context, _, _ string) error    { return nil }
func (s *fakeAuthStore) CreateWebAuthnSession(_ context.Context, _, _, _, _ string) error { return nil }
func (s *fakeAuthStore) ConsumeWebAuthnSession(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (s *fakeAuthStore) UpsertMFAEmailCode(_ context.Context, _, _, _ string) error { return nil }
func (s *fakeAuthStore) GetMFAEmailCode(_ context.Context, _ string) (string, string, int, error) {
	return "", "", 0, nil
}
func (s *fakeAuthStore) IncrementMFAEmailCodeAttempts(_ context.Context, _ string) error { return nil }
func (s *fakeAuthStore) DeleteMFAEmailCode(_ context.Context, _ string) error            { return nil }

// --- Fasting ---

func TestStartAndEndFast(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/fasting/start", map[string]any{"target_hours": 16}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	started := decodeJSON[types.Fast](t, rec)
	if started.EndAt != nil {
		t.Errorf("started fast should have nil end_at")
	}
	if started.TargetHours != 16 {
		t.Errorf("expected target_hours 16, got %v", started.TargetHours)
	}

	rec = doRequest(h, "GET", "/api/v1/fasting/active", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("active: expected 200, got %d", rec.Code)
	}

	rec = doRequest(h, "POST", "/api/v1/fasting/end", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("end: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	ended := decodeJSON[types.Fast](t, rec)
	if ended.EndAt == nil {
		t.Errorf("ended fast should have end_at set")
	}

	rec = doRequest(h, "GET", "/api/v1/fasting/active", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("active after end: expected 404, got %d", rec.Code)
	}
}

func TestStartFastDefaultsTarget(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/fasting/start", nil, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	f := decodeJSON[types.Fast](t, rec)
	if f.TargetHours != 16 {
		t.Errorf("expected default target_hours 16, got %v", f.TargetHours)
	}
}

func TestStartFastConflict(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	if rec := doRequest(h, "POST", "/api/v1/fasting/start", nil, nil); rec.Code != http.StatusCreated {
		t.Fatalf("first start: expected 201, got %d", rec.Code)
	}
	rec := doRequest(h, "POST", "/api/v1/fasting/start", nil, nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("second start: expected 409, got %d", rec.Code)
	}
}

func TestEndFastNoActive(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/fasting/end", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestListFastsEmpty(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/fasting/history", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	fasts := decodeJSON[[]types.Fast](t, rec)
	if len(fasts) != 0 {
		t.Errorf("expected empty history, got %d", len(fasts))
	}
}
