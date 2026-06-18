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

// ---------------------------------------------------------------------------
// Phase 1 — computed weekly stats (frontend-only, from DailyRollup[]).
// ---------------------------------------------------------------------------

export interface WeeklyStats {
  days: DailyRollup[]
  avg: Macros // element-wise average of Consumed across logged days
  adherence: number // 0..1 — fraction of days within ±10% of the calorie target
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

// Phase 2 — Food Discovery -------------------------------------------------

export interface FoodAlias {
  food_id: string
  alias: string
  normalized: string
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
}

// Phase 3 — Meal Templates -------------------------------------------------

export interface MealTemplate {
  id: string
  user_id: string
  name: string
  items: ResolvedItem[]
  created_at: string // RFC3339
  last_used: string // RFC3339
}

// Phase 4 — Body Tracking --------------------------------------------------

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

// Phase 5 — Goals & Planning ----------------------------------------------

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
  { value: 'light', label: 'Lightly active', hint: 'Light exercise 1–3 days/week' },
  { value: 'moderate', label: 'Moderately active', hint: 'Moderate exercise 3–5 days/week' },
  { value: 'active', label: 'Very active', hint: 'Hard exercise 6–7 days/week' },
  { value: 'very_active', label: 'Extra active', hint: 'Physical job or 2× daily training' },
] as const

export const GOALS = [
  { value: 'cut', label: 'Cut', hint: 'Lose fat in a calorie deficit' },
  { value: 'maintain', label: 'Maintain', hint: 'Hold your current weight' },
  { value: 'bulk', label: 'Bulk', hint: 'Build muscle in a calorie surplus' },
] as const
