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
