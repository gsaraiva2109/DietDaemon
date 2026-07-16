// TypeScript mirrors of core/types/types.go. The Go API uses the standard
// encoding/json with NO struct tags, so JSON keys are the Go field names
// verbatim (PascalCase). These names must stay exact: a corrected item is
// round-tripped back to POST /meals/{id}/items/{idx}/correct unchanged.

export interface Macros {
  Calories: number
  Protein: number
  Carbs: number
  Fat: number
  Fiber: number
}

export interface ParsedItem {
  RawPhrase: string
  Quantity: number
  Unit: string
  NormalizedGrams: number
  Locale: string
}

export interface FoodMatch {
  FoodID: string
  Name: string
  Source: string // "food_library" | "openfoodfacts" | "taco" | "usda" | ...
  Per100g: Macros
  MatchScore: number // 0..1
}

export interface ResolvedItem {
  Parsed: ParsedItem
  Match: FoodMatch
  Macros: Macros // Per100g scaled to the portion eaten
}

// ParserTier: 0 deterministic, 1 embedding, 2 LLM.
export type ParserTier = 0 | 1 | 2

export interface Meal {
  ID: string
  UserID: string
  At: string // RFC3339
  RawText: string
  Items: ResolvedItem[]
  Confidence: number // 0..1
  ParserTier: ParserTier
  CreatedAt: string // RFC3339
}

export interface DailyRollup {
  UserID: string
  Date: string // "YYYY-MM-DD" in the user's timezone
  Consumed: Macros
  Targets: Macros
}

// BackupConfig is a user's scheduled backup/export settings. LastRunAt is
// Go's zero time ("0001-01-01T00:00:00Z") when it has never run.
export interface BackupConfig {
  UserID: string
  Enabled: boolean
  Destination: 'local' | 's3'
  LocalSubdir: string
  S3Bucket: string
  S3Prefix: string
  S3Region: string
  S3Endpoint: string
  IntervalHrs: number
  LastRunAt: string
}

// The five macros we render, in display order. Keyed to DESIGN.md macro hues.
export const MACRO_KEYS = ['Calories', 'Protein', 'Carbs', 'Fat', 'Fiber'] as const
export type MacroKey = (typeof MACRO_KEYS)[number]

export interface MacroMeta {
  key: MacroKey
  label: string
  unit: string
  // CSS var token name (see index.css @theme)
  colorVar: string
}

export const MACRO_META: Record<MacroKey, MacroMeta> = {
  Calories: { key: 'Calories', label: 'Calories', unit: 'kcal', colorVar: '--color-cal' },
  Protein: { key: 'Protein', label: 'Protein', unit: 'g', colorVar: '--color-protein' },
  Carbs: { key: 'Carbs', label: 'Carbs', unit: 'g', colorVar: '--color-carbs' },
  Fat: { key: 'Fat', label: 'Fat', unit: 'g', colorVar: '--color-fat' },
  Fiber: { key: 'Fiber', label: 'Fiber', unit: 'g', colorVar: '--color-fiber' },
}

// External nutrition sources the resolver can query, in the backend's
// startup-configured default order (see cmd/dietdaemon buildSources). The API
// only ever returns the user's chosen order (or empty for "not customized"),
// so the frontend needs this fixed universe to render the reorder list and to
// seed it the first time a user opens the settings page.
export const NUTRITION_SOURCES = ['openfoodfacts', 'taco'] as const
export const SOURCE_LABELS: Record<string, string> = {
  openfoodfacts: 'Open Food Facts',
  taco: 'TACO (Brazilian food database)',
}

export interface WeeklyStats {
  days: DailyRollup[]
  avg: Macros // element-wise average of Consumed across logged days
  adherence: number // 0..1, fraction of days within ±10% of the calorie target
  calorieTrend: TrendDirection
  proteinTrend: TrendDirection
  bestDay: DailyRollup | null // closest to calorie target
  worstDay: DailyRollup | null // furthest from calorie target
  loggedDays: number
}

export type TrendDirection = 'up' | 'down' | 'flat'

// ---------------------------------------------------------------------------
// NOTE: every type below mirrors a Go struct that DOES carry json tags
// (snake_case), unlike the original PascalCase domain types above. Keys here
// must match those json tags exactly. The one exception is `ResolvedItem`,
// which has no json tags, so its nested fields stay PascalCase even when it
// appears inside a snake_case parent (e.g. MealTemplate.items).
// ---------------------------------------------------------------------------

export interface FoodAlias {
  food_id: string
  alias: string
  normalized: string
}

// PendingAlias is an embedding near-miss awaiting user confirmation before it
// is promoted into a real food alias. food_name is enriched server-side so
// the UI doesn't need a second lookup per row.
export interface PendingAlias {
  id: string
  user_id: string
  phrase: string
  food_id: string
  food_name: string
  match_score: number
  created_at: string
}

export interface FoodDetail {
  food_id: string
  name: string
  source: string
  per_100g: Macros
  category: string
  brand: string
  barcode: string
  image_url: string
  serving_size: number
  serving_unit: string
  query_count: number
  last_used: string
  aliases?: FoodAlias[]
  in_library: boolean
}

// A complete nutrition label entered per the selected serving basis. The API
// stores the nutrients normalized per 100g while preserving basis_grams as the
// food's serving size.
export interface CustomFoodInput {
  name: string
  calories: number
  protein: number
  carbs: number
  fat: number
  fiber: number
  basis_grams: number
}

export interface MealTemplate {
  id: string
  user_id: string
  name: string
  items: ResolvedItem[]
  created_at: string // RFC3339
  last_used: string // RFC3339
}

export interface WeightEntry {
  id: string
  user_id: string
  date: string // YYYY-MM-DD
  weight_kg: number
  note: string
  created_at: string
}

export interface MeasurementEntry {
  id: string
  user_id: string
  date: string // YYYY-MM-DD
  waist_cm: number
  hips_cm: number
  chest_cm: number
  left_arm_cm: number
  right_arm_cm: number
  left_thigh_cm: number
  right_thigh_cm: number
  note: string
  created_at: string
}

export interface ProgressPhoto {
  id: string
  user_id: string
  date: string // YYYY-MM-DD
  view: string // front | side | back
  mime_type: string
  created_at: string
}

export interface WeightTrend {
  date: string // YYYY-MM-DD
  weight_kg: number
  rolling_avg: number
}

export interface BodyCompositionSummary {
  current_weight_kg: number
  start_weight_kg: number
  change_kg: number
  trend_direction: string // up | down | stable
  latest_trend_point?: WeightTrend | null
}

// -----------------------------------------------------------------------------
// Health tracking, fasting (live backend) + water / workout / sleep (endpoints;
// endpoints 404 until shipped, cards fall back to empty state).
// -----------------------------------------------------------------------------

// A single intermittent-fasting window. end_at is absent while in progress.
export interface Fast {
  id: string
  user_id: string
  start_at: string // RFC3339
  end_at?: string | null // RFC3339; null/absent while fasting
  target_hours: number
  completed: boolean
  created_at: string
}

export interface WaterLog {
  id: string
  user_id: string
  amount_ml: number
  logged_at: string
  note?: string
}

// GET /body/water, today's running total against the daily goal.
export interface WaterToday {
  logs: WaterLog[]
  today_ml: number
  goal_ml: number
}

export type WorkoutIntensity = 'light' | 'moderate' | 'heavy'

export interface Workout {
  id: string
  user_id: string
  name: string
  duration_min: number
  intensity: WorkoutIntensity
  calories_burned?: number
  note?: string
  logged_at: string
}

export type SleepQuality = 'poor' | 'fair' | 'good' | 'great'

export interface SleepLog {
  id: string
  user_id: string
  sleep_at: string // HH:MM
  wake_at: string // HH:MM
  duration_hours: number
  quality: SleepQuality
  note?: string
  logged_at: string
}

// The numeric measurement fields, for chart series + form rendering.
export const MEASUREMENT_FIELDS = [
  { key: 'waist_cm', label: 'Waist' },
  { key: 'hips_cm', label: 'Hips' },
  { key: 'chest_cm', label: 'Chest' },
  { key: 'left_arm_cm', label: 'Left arm' },
  { key: 'right_arm_cm', label: 'Right arm' },
  { key: 'left_thigh_cm', label: 'Left thigh' },
  { key: 'right_thigh_cm', label: 'Right thigh' },
] as const
export type MeasurementField = (typeof MEASUREMENT_FIELDS)[number]['key']

export interface UserProfile {
  user_id: string
  height_cm: number
  birth_date: string // YYYY-MM-DD
  gender: string // male | female
  activity_level: string // sedentary | light | moderate | active | very_active
  goal: string // cut | maintain | bulk
  target_weight_kg: number
  weekly_rate: number // kg/week
  onboarded: boolean
  created_at: string
  updated_at: string
}

export interface TDEEResult {
  bmr: number
  tdee: number
  cut_cal: number
  maintain_cal: number
  bulk_cal: number
  protein_g: number
  fat_g: number
  carbs_g: number
}

export interface GoalSuggestion {
  current_intake_kcal: number
  recommended_kcal: number
  current_loss_kg: number
  target_loss_kg: number
  message: string
}

export const ACTIVITY_LEVELS = [
  { value: 'sedentary', label: 'Sedentary', hint: 'Little or no exercise, desk job' },
  { value: 'light', label: 'Lightly active', hint: 'Light exercise 1 to 3 days/week' },
  { value: 'moderate', label: 'Moderately active', hint: 'Moderate exercise 3 to 5 days/week' },
  { value: 'active', label: 'Very active', hint: 'Hard exercise 6 to 7 days/week' },
  { value: 'very_active', label: 'Extra active', hint: 'Physical job or 2× daily training' },
] as const

export const GOALS = [
  { value: 'cut', label: 'Cut', hint: 'Lose fat in a calorie deficit' },
  { value: 'maintain', label: 'Maintain', hint: 'Hold your current weight' },
  { value: 'bulk', label: 'Bulk', hint: 'Build muscle in a calorie surplus' },
] as const

export interface User {
  id: string
  email: string
  display_name: string
  email_verified: boolean
  created_at: string
  totp_enabled?: boolean
}

// GET /auth/session, the authenticated user, or 401 when anonymous.
export interface SessionResponse {
  user: User
}

// How new accounts may be created. Drives login/register screen gating.
//   open, anyone may register
//   invite, only the bootstrap (first) user; closed thereafter
//   oidc-only, no password form; sign in with a provider
export type RegistrationMode = 'open' | 'invite' | 'oidc-only'

// A configured OIDC provider. `id` is the route slug used in
// /auth/oidc/{id}/start; `name` is the human label on the button.
export interface OidcProvider {
  id: string
  name: string
}

// GET /auth/providers, drives the login screen. `providers` is empty until
export interface ProvidersResponse {
  registration_mode: RegistrationMode
  providers: OidcProvider[]
}

// A provider account linked to the current user.
export interface LinkedIdentity {
  id: string
  provider: string // matches OidcProvider.id
  email: string
  linked_at: string
}

// Machine API key. The raw `key` is returned ONCE on create and never listed.
export interface ApiKey {
  id: string
  label: string
  created_at: string
  last_used_at: string | null
  revoked_at: string | null
}

export interface NewApiKey extends ApiKey {
  key: string // "ddk_…", shown once, never stored client-side
}

// Read-only share link. The raw `token` is returned ONCE on create and never listed.
export interface ShareToken {
  id: string
  label: string
  created_at: string
  last_used_at: string | null
  revoked_at: string | null
}

export interface NewShareToken extends ShareToken {
  token: string // shown once, never stored client-side
}

// Per-source bulk food-import status.
export interface FoodImportStatus {
  source: string
  fingerprint?: string
  last_result: string // "imported" | "skipped" | "failed" | "changed_during_import"
  last_run_at: string
  last_error?: string
}

// ---------------------------------------------------------------------------
// TOTP / MFA
// ---------------------------------------------------------------------------

// POST /auth/totp/enroll, provisioning data for the authenticator app.
export interface TotpEnrollResponse {
  otpauth_url: string // otpauth://totp/… (rendered as a QR)
  secret: string // base32, for manual entry
}

// Returned once after enroll-verify and on regenerate. Show, then forget.
export interface RecoveryCodesResponse {
  recovery_codes: string[]
}

// Login can defer to a second factor instead of issuing a session.
export interface MfaChallenge {
  mfa_required: true
  challenge_token: string
}

// POST /auth/login → either a session (1FA done) or an MFA challenge.
export type LoginResponse = SessionResponse | MfaChallenge

export function isMfaChallenge(r: LoginResponse): r is MfaChallenge {
  return 'mfa_required' in r && r.mfa_required
}

// ---------------------------------------------------------------------------
// Passkeys / WebAuthn
// ---------------------------------------------------------------------------

export interface Passkey {
  id: string
  label: string
  created_at: string
  last_used_at: string | null
}

// ---------------------------------------------------------------------------
// Nudge rules
// ---------------------------------------------------------------------------

export type NudgeRuleKind = 'macro' | 'health' | 'digest' | 'weekly-budget' | 'smart-meal'

// Rule is a JSON blob of the underlying Go rule struct's own fields (Rule,
// HealthRule, or DigestRule — shape depends on `kind`), so field names are
// Go's PascalCase, mirrored verbatim per the note atop this file.
export interface NudgeRuleView {
  rule_id: string
  kind: NudgeRuleKind
  enabled: boolean
  rule: Record<string, unknown>
}

export interface NudgeRuleUpdate {
  rule_id: string
  enabled: boolean
  params?: Record<string, unknown>
  reset?: boolean
}

// ---------------------------------------------------------------------------
// Streak & weekly budget
// ---------------------------------------------------------------------------

export interface StreakResponse {
  current_days: number
}

export interface WeeklyBudgetView {
  plain: number
  effective: number
}

export interface WeeklyBudgetResponse {
  calories: WeeklyBudgetView
  protein: WeeklyBudgetView
}

// ---------------------------------------------------------------------------
// AI Key settings (BYOK)
// ---------------------------------------------------------------------------

export interface AIKeyStatus {
  has_key: boolean
  provider: string
}

// ---------------------------------------------------------------------------
// Hevy integration
// ---------------------------------------------------------------------------

export interface HevyKeyStatus {
  has_key: boolean
}

export interface HevyImportResult {
  imported: number
  skipped_duplicates: number
  total: number
}

// ---------------------------------------------------------------------------
// AI chat assistant
// ---------------------------------------------------------------------------

export interface ChatSession {
  id: string
  title: string
  created_at: string
  updated_at: string
}

export type ChatRole = 'user' | 'assistant' | 'tool'

export interface ChatMessageRecord {
  id: string
  role: ChatRole
  content: string
  tool_name?: string
  created_at: string
}

export interface AssistantSettings {
  custom_instructions: string
  base_prompt: string
}
