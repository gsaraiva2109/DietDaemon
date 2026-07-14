package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/normalize"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Meals
// ---------------------------------------------------------------------------

// SaveMeal inserts a meal and all its resolved items inside a transaction.
func (s *Store) SaveMeal(ctx context.Context, m types.Meal) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const mealQ = `
		INSERT INTO meals (id, user_id, at_utc, raw_text, confidence, parser_tier, created_at, external_id)
		VALUES (:id, :user_id, :at_utc, :raw_text, :confidence, :parser_tier, :created_at, :external_id)
	`
	mealQuery, mealArgs, err := sqlx.Named(mealQ, map[string]any{
		"id":          m.ID,
		"user_id":     m.UserID,
		"at_utc":      utcStr(m.At),
		"raw_text":    m.RawText,
		"confidence":  m.Confidence,
		"parser_tier": int(m.ParserTier),
		"created_at":  utcStr(m.CreatedAt),
		"external_id": m.ExternalID,
	})
	if err != nil {
		return fmt.Errorf("store: bind meal: %w", err)
	}
	if _, err = tx.ExecContext(ctx, s.rewrite(mealQuery), mealArgs...); err != nil {
		if isUniqueViolation(err) {
			return nil // safe no-op: already imported
		}
		return fmt.Errorf("store: insert meal: %w", err)
	}

	const itemQ = `
		INSERT INTO resolved_items
			(id, meal_id, position, raw_phrase, quantity, unit, normalized_grams,
			 food_id, food_name, source, match_score,
			 kcal, protein, carbs, fat, fiber)
		VALUES (:id, :meal_id, :position, :raw_phrase, :quantity, :unit, :normalized_grams,
			:food_id, :food_name, :source, :match_score,
			:kcal, :protein, :carbs, :fat, :fiber)
	`
	for i, it := range m.Items {
		itemQuery, itemArgs, err := sqlx.Named(itemQ, resolvedItemNamedArgs(newID(), m.ID, i, it))
		if err != nil {
			return fmt.Errorf("store: bind resolved_item: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(itemQuery), itemArgs...); err != nil {
			return fmt.Errorf("store: insert resolved_item: %w", err)
		}
	}

	return tx.Commit()
}

// resolvedItemNamedArgs builds the named-parameter map shared by every insert
// of a resolved_items row (SaveMeal, AddMealItem). position is the item's
// 0-based ordinal within the meal, replacing reliance on SQLite's implicit
// rowid for ordering.
func resolvedItemNamedArgs(id, mealID string, position int, it types.ResolvedItem) map[string]any {
	return map[string]any{
		"id":               id,
		"meal_id":          mealID,
		"position":         position,
		"raw_phrase":       it.Parsed.RawPhrase,
		"quantity":         it.Parsed.Quantity,
		"unit":             it.Parsed.Unit,
		"normalized_grams": it.Parsed.NormalizedGrams,
		"food_id":          it.Match.FoodID,
		"food_name":        it.Match.Name,
		"source":           it.Match.Source,
		"match_score":      it.Match.MatchScore,
		"kcal":             it.Macros.Calories,
		"protein":          it.Macros.Protein,
		"carbs":            it.Macros.Carbs,
		"fat":              it.Macros.Fat,
		"fiber":            it.Macros.Fiber,
	}
}

// RecentMeals returns the most recent meals for a user, each with its resolved
// items populated. Meals are ordered newest-first.
func (s *Store) RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error) {
	const mealQ = `
		SELECT id, user_id, at_utc, raw_text, confidence, parser_tier, created_at
		FROM meals
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	var rows []mealRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(mealQ), userID, limit); err != nil {
		return nil, fmt.Errorf("store: query meals: %w", err)
	}

	var meals []types.Meal
	var mealIDs []string
	for _, r := range rows {
		m := r.toMeal()
		meals = append(meals, m)
		mealIDs = append(mealIDs, m.ID)
	}

	if len(meals) == 0 {
		return meals, nil
	}

	// Fetch all resolved items for the retrieved meals.
	itemsByMeal, err := s.loadItems(ctx, mealIDs)
	if err != nil {
		return nil, err
	}

	for i := range meals {
		meals[i].Items = itemsByMeal[meals[i].ID]
	}

	return meals, nil
}

// RecentMealTimes returns logged meal timestamps since since, newest first.
func (s *Store) RecentMealTimes(ctx context.Context, userID string, since time.Time) ([]time.Time, error) {
	var rows []string
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(`SELECT at_utc FROM meals WHERE user_id = ? AND at_utc >= ? ORDER BY at_utc DESC`), userID, utcStr(since)); err != nil {
		return nil, fmt.Errorf("store: recent meal times: %w", err)
	}
	out := make([]time.Time, len(rows))
	for i, row := range rows {
		out[i] = parseUTC(row)
	}
	return out, nil
}

// GetMeal returns a single meal by ID with its resolved items populated.
// Returns types.ErrNotFound when the meal does not exist.
func (s *Store) GetMeal(ctx context.Context, mealID string) (types.Meal, error) {
	const q = `
		SELECT id, user_id, at_utc, raw_text, confidence, parser_tier, created_at
		FROM meals WHERE id = ?
	`
	var row mealRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), mealID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Meal{}, types.ErrNotFound
		}
		return types.Meal{}, fmt.Errorf("store: get meal: %w", err)
	}
	m := row.toMeal()

	itemsByMeal, err := s.loadItems(ctx, []string{m.ID})
	if err != nil {
		return types.Meal{}, err
	}
	m.Items = itemsByMeal[m.ID]
	return m, nil
}

// GetRollups returns daily rollups for a user between startDate and endDate
// (inclusive, "YYYY-MM-DD" format). Ordered by date ascending.
func (s *Store) GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error) {
	const q = `
		SELECT user_id, date,
		       consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
		       target_kcal, target_protein, target_carbs, target_fat, target_fiber
		FROM daily_rollups
		WHERE user_id = ? AND date >= ? AND date <= ?
		ORDER BY date ASC
	`
	var rows []rollupRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: query rollups: %w", err)
	}
	out := make([]types.DailyRollup, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toRollup())
	}
	return out, nil
}

// mealRow is the flat DB shape of the meals table; types.Meal additionally
// carries a non-column Items slice populated by loadItems.
type mealRow struct {
	ID         string  `db:"id"`
	UserID     string  `db:"user_id"`
	AtUTC      string  `db:"at_utc"`
	RawText    string  `db:"raw_text"`
	Confidence float64 `db:"confidence"`
	ParserTier int     `db:"parser_tier"`
	CreatedAt  string  `db:"created_at"`
}

func (r mealRow) toMeal() types.Meal {
	return types.Meal{
		ID:         r.ID,
		UserID:     r.UserID,
		At:         parseUTC(r.AtUTC),
		RawText:    r.RawText,
		Confidence: r.Confidence,
		ParserTier: types.ParserTier(r.ParserTier),
		CreatedAt:  parseUTC(r.CreatedAt),
	}
}

// resolvedItemRow is the flat DB shape of resolved_items; types.ResolvedItem
// groups these columns into nested Parsed/Match/Macros structs, and Per100g
// is reconstructed rather than stored.
type resolvedItemRow struct {
	MealID          string  `db:"meal_id"`
	RawPhrase       string  `db:"raw_phrase"`
	Quantity        float64 `db:"quantity"`
	Unit            string  `db:"unit"`
	NormalizedGrams float64 `db:"normalized_grams"`
	FoodID          string  `db:"food_id"`
	FoodName        string  `db:"food_name"`
	Source          string  `db:"source"`
	MatchScore      float64 `db:"match_score"`
	Kcal            float64 `db:"kcal"`
	Protein         float64 `db:"protein"`
	Carbs           float64 `db:"carbs"`
	Fat             float64 `db:"fat"`
	Fiber           float64 `db:"fiber"`
}

func (r resolvedItemRow) toResolvedItem() types.ResolvedItem {
	macros := types.Macros{Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber}
	return types.ResolvedItem{
		Parsed: types.ParsedItem{
			RawPhrase: r.RawPhrase, Quantity: r.Quantity, Unit: r.Unit, NormalizedGrams: r.NormalizedGrams,
		},
		Match: types.FoodMatch{
			FoodID: r.FoodID, Name: r.FoodName, Source: r.Source, MatchScore: r.MatchScore,
			// Reconstruct Per100g from the absolute macros and portion grams.
			Per100g: macrosPer100g(macros, r.NormalizedGrams),
		},
		Macros: macros,
	}
}

func (s *Store) loadItems(ctx context.Context, mealIDs []string) (map[string][]types.ResolvedItem, error) {
	if len(mealIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(mealIDs))
	args := make([]any, len(mealIDs))
	for i, id := range mealIDs {
		placeholders[i] = s.dialect.Placeholder(i + 1)
		args[i] = id
	}

	// #nosec G201 -- placeholder expansion is ? only, values are args
	q := fmt.Sprintf(`
		SELECT meal_id, raw_phrase, quantity, unit, normalized_grams,
		       food_id, food_name, source, match_score,
		       kcal, protein, carbs, fat, fiber
		FROM resolved_items
		WHERE meal_id IN (%s)
		ORDER BY meal_id, position
	`, strings.Join(placeholders, ","))

	var rows []resolvedItemRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: query items: %w", err)
	}

	out := make(map[string][]types.ResolvedItem)
	for _, r := range rows {
		out[r.MealID] = append(out[r.MealID], r.toResolvedItem())
	}
	return out, nil
}

// macrosPer100g back-calculates per-100g macros from the absolute portion
// macros. If grams is zero or negative the absolute macros are returned as-is.
func macrosPer100g(m types.Macros, grams float64) types.Macros {
	if grams <= 0 {
		return m
	}
	return m.Scale(100.0 / grams)
}

// mealOwnerRow is the flat DB shape used to fetch a meal's timestamp while
// checking that it belongs to the calling user.
type mealOwnerRow struct {
	AtUTC  string `db:"at_utc"`
	UserID string `db:"user_id"`
}

// mealOwner loads mealID's at_utc/user_id within tx and confirms it belongs
// to userID, returning types.ErrNotFound otherwise. Shared by CorrectMealItem,
// AddMealItem, and DeleteMealItem, which all gate a rollup mutation on the
// same ownership check.
func (s *Store) mealOwner(ctx context.Context, tx *sqlx.Tx, mealID, userID string) (mealOwnerRow, error) {
	const q = `SELECT at_utc, user_id FROM meals WHERE id = ?`
	var row mealOwnerRow
	if err := tx.GetContext(ctx, &row, s.rewrite(q), mealID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mealOwnerRow{}, types.ErrNotFound
		}
		return mealOwnerRow{}, fmt.Errorf("store: get meal: %w", err)
	}
	if row.UserID != userID {
		return mealOwnerRow{}, types.ErrNotFound
	}
	return row, nil
}

// CorrectMealItem updates one resolved item's macros for a meal, then
// recalculates the daily rollup and refreshes the global foods cache so future
// logs use the corrected values. itemIndex is the 0-based position of the item
// within the meal's items (ordered by the position column).
func (s *Store) CorrectMealItem(ctx context.Context, userID string, mealID string, itemIndex int, corrected types.ResolvedItem) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = s.correctMealItemTx(ctx, tx, userID, mealID, itemIndex, corrected)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// CorrectMealItemWithFeedback corrects an item and learns its original phrase
// atomically. Conflicting aliases are queued for an explicit replacement.
func (s *Store) CorrectMealItemWithFeedback(ctx context.Context, userID, mealID string, itemIndex int, corrected types.ResolvedItem) (types.CorrectionFeedback, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return types.CorrectionFeedback{}, fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	phrase, err := s.correctMealItemTx(ctx, tx, userID, mealID, itemIndex, corrected)
	if err != nil {
		return types.CorrectionFeedback{}, err
	}
	normalized := normalize.Normalize(phrase)
	if normalized == "" || corrected.Match.FoodID == "" {
		if err := tx.Commit(); err != nil {
			return types.CorrectionFeedback{}, err
		}
		return types.CorrectionFeedback{}, nil
	}
	var existing string
	err = tx.GetContext(ctx, &existing, s.rewrite(`SELECT food_id FROM food_aliases WHERE user_id = ? AND alias_normalized = ?`), userID, normalized)
	if errors.Is(err, sql.ErrNoRows) || existing == corrected.Match.FoodID {
		if _, err := tx.ExecContext(ctx, s.rewrite(`INSERT INTO food_aliases (user_id, alias_normalized, food_id) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`), userID, normalized, corrected.Match.FoodID); err != nil {
			return types.CorrectionFeedback{}, fmt.Errorf("store: insert correction alias: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return types.CorrectionFeedback{}, err
		}
		return types.CorrectionFeedback{}, nil
	}
	if err != nil {
		return types.CorrectionFeedback{}, fmt.Errorf("store: lookup correction alias: %w", err)
	}
	id := newID()
	if _, err := tx.ExecContext(ctx, s.rewrite(`INSERT INTO pending_aliases (id, user_id, phrase, food_id, match_score, replacement, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`), id, userID, phrase, corrected.Match.FoodID, corrected.Match.MatchScore, true, utcStr(time.Now())); err != nil {
		return types.CorrectionFeedback{}, fmt.Errorf("store: add replacement alias: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return types.CorrectionFeedback{}, err
	}
	return types.CorrectionFeedback{PendingAliasID: id}, nil
}

// correctMealItemTx performs the shared direct/API correction work and returns
// the original phrase of the corrected item for chat feedback learning.
func (s *Store) correctMealItemTx(ctx context.Context, tx *sqlx.Tx, userID string, mealID string, itemIndex int, corrected types.ResolvedItem) (string, error) {
	// Load the meal to get the at time for rollup lookup and the original items.
	meal, err := s.mealOwner(ctx, tx, mealID, userID)
	if err != nil {
		return "", err
	}
	mealAt := parseUTC(meal.AtUTC)

	// Load items ordered by position so we can find and update the target item
	// by its real id.
	const itemsQ = `
		SELECT id, raw_phrase, quantity, unit, normalized_grams,
		       food_id, food_name, source, match_score,
		       kcal, protein, carbs, fat, fiber
		FROM resolved_items
		WHERE meal_id = ?
		ORDER BY position
	`
	var itemRows []mealItemRow
	if err := tx.SelectContext(ctx, &itemRows, s.rewrite(itemsQ), mealID); err != nil {
		return "", fmt.Errorf("store: query items: %w", err)
	}

	type item struct {
		id string
		ri types.ResolvedItem
	}
	items := make([]item, len(itemRows))
	var oldTotal types.Macros
	for i, r := range itemRows {
		items[i] = item{id: r.ID, ri: r.toResolvedItem()}
		oldTotal = oldTotal.Add(items[i].ri.Macros)
	}
	if itemIndex < 0 || itemIndex >= len(items) {
		return "", fmt.Errorf("store: item index %d out of range [0, %d)", itemIndex, len(items))
	}

	// Replace the target item's macros and recalculate the new total.
	oldItemMacros := items[itemIndex].ri.Macros
	originalPhrase := items[itemIndex].ri.Parsed.RawPhrase
	items[itemIndex].ri = corrected

	var newTotal types.Macros
	for _, it := range items {
		newTotal = newTotal.Add(it.ri.Macros)
	}

	// Update the resolved_items row.
	const updateQ = `
		UPDATE resolved_items SET
			normalized_grams = :normalized_grams, food_id = :food_id, food_name = :food_name,
			source = :source, match_score = :match_score,
			kcal = :kcal, protein = :protein, carbs = :carbs, fat = :fat, fiber = :fiber
		WHERE id = :id
	`
	updateQuery, updateArgs, err := sqlx.Named(updateQ, map[string]any{
		"normalized_grams": corrected.Parsed.NormalizedGrams,
		"food_id":          corrected.Match.FoodID,
		"food_name":        corrected.Match.Name,
		"source":           corrected.Match.Source,
		"match_score":      corrected.Match.MatchScore,
		"kcal":             corrected.Macros.Calories,
		"protein":          corrected.Macros.Protein,
		"carbs":            corrected.Macros.Carbs,
		"fat":              corrected.Macros.Fat,
		"fiber":            corrected.Macros.Fiber,
		"id":               items[itemIndex].id,
	})
	if err != nil {
		return "", fmt.Errorf("store: bind update item: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(updateQuery), updateArgs...); err != nil {
		return "", fmt.Errorf("store: update item: %w", err)
	}

	// Update the daily rollup: remove old macros, add new ones.
	localDate := mealAt.Format("2006-01-02")
	const rollupQ = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (:user_id, :date,
		        :new_total_kcal, :new_total_protein, :new_total_carbs, :new_total_fat, :new_total_fiber,
		        0, 0, 0, 0, 0)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal    = consumed_kcal    - :old_kcal    + :corrected_kcal,
			consumed_protein = consumed_protein - :old_protein + :corrected_protein,
			consumed_carbs   = consumed_carbs   - :old_carbs   + :corrected_carbs,
			consumed_fat     = consumed_fat     - :old_fat     + :corrected_fat,
			consumed_fiber   = consumed_fiber   - :old_fiber   + :corrected_fiber
	`
	rollupQuery, rollupArgs, err := sqlx.Named(rollupQ, map[string]any{
		"user_id":           userID,
		"date":              localDate,
		"new_total_kcal":    newTotal.Calories,
		"new_total_protein": newTotal.Protein,
		"new_total_carbs":   newTotal.Carbs,
		"new_total_fat":     newTotal.Fat,
		"new_total_fiber":   newTotal.Fiber,
		"old_kcal":          oldItemMacros.Calories,
		"old_protein":       oldItemMacros.Protein,
		"old_carbs":         oldItemMacros.Carbs,
		"old_fat":           oldItemMacros.Fat,
		"old_fiber":         oldItemMacros.Fiber,
		"corrected_kcal":    corrected.Macros.Calories,
		"corrected_protein": corrected.Macros.Protein,
		"corrected_carbs":   corrected.Macros.Carbs,
		"corrected_fat":     corrected.Macros.Fat,
		"corrected_fiber":   corrected.Macros.Fiber,
	})
	if err != nil {
		return "", fmt.Errorf("store: bind update rollup: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(rollupQuery), rollupArgs...); err != nil {
		return "", fmt.Errorf("store: update rollup: %w", err)
	}

	// Refresh the global food catalog: upsert the corrected macros so future
	// alias lookups (by any user) use the corrected values.
	if corrected.Match.FoodID != "" {
		const foodQ = `
			INSERT INTO foods
				(food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, created_at, updated_at)
			VALUES (:food_id, :name, :source, :kcal_100g, :protein_100g, :carbs_100g, :fat_100g, :fiber_100g, :now, :now)
			ON CONFLICT(food_id) DO UPDATE SET
				kcal_100g    = excluded.kcal_100g,
				protein_100g = excluded.protein_100g,
				carbs_100g   = excluded.carbs_100g,
				fat_100g     = excluded.fat_100g,
				fiber_100g   = excluded.fiber_100g,
				updated_at   = excluded.updated_at
		`
		foodQuery, foodArgs, err := sqlx.Named(foodQ, foodNamedArgs(corrected.Match))
		if err != nil {
			return "", fmt.Errorf("store: bind upsert food: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(foodQuery), foodArgs...); err != nil {
			return "", fmt.Errorf("store: upsert food: %w", err)
		}
	}

	return originalPhrase, nil
}

// mealItemRow is resolvedItemRow plus its own id, used where the caller must
// locate and later update or delete a specific resolved_items row
// (CorrectMealItem, DeleteMealItem).
type mealItemRow struct {
	ID string `db:"id"`
	resolvedItemRow
}

// AddMealItem appends a resolved item to an existing meal and adds its macros
// to that day's rollup. Mirrors CorrectMealItem's delta approach.
func (s *Store) AddMealItem(ctx context.Context, userID, mealID string, item types.ResolvedItem) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	meal, err := s.mealOwner(ctx, tx, mealID, userID)
	if err != nil {
		return err
	}

	// Next position is one past the current max for this meal (-1 when empty,
	// so the first item lands at 0).
	const posQ = `SELECT COALESCE(MAX(position), -1) + 1 FROM resolved_items WHERE meal_id = ?`
	var nextPosition int
	if err := tx.GetContext(ctx, &nextPosition, s.rewrite(posQ), mealID); err != nil {
		return fmt.Errorf("store: next item position: %w", err)
	}

	const itemQ = `
		INSERT INTO resolved_items
			(id, meal_id, position, raw_phrase, quantity, unit, normalized_grams,
			 food_id, food_name, source, match_score,
			 kcal, protein, carbs, fat, fiber)
		VALUES (:id, :meal_id, :position, :raw_phrase, :quantity, :unit, :normalized_grams,
			:food_id, :food_name, :source, :match_score,
			:kcal, :protein, :carbs, :fat, :fiber)
	`
	itemQuery, itemArgs, err := sqlx.Named(itemQ, resolvedItemNamedArgs(newID(), mealID, nextPosition, item))
	if err != nil {
		return fmt.Errorf("store: bind resolved_item: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(itemQuery), itemArgs...); err != nil {
		return fmt.Errorf("store: insert resolved_item: %w", err)
	}

	localDate := parseUTC(meal.AtUTC).Format("2006-01-02")
	const rollupQ = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (:user_id, :date, :kcal, :protein, :carbs, :fat, :fiber, 0, 0, 0, 0, 0)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal    = consumed_kcal    + :kcal,
			consumed_protein = consumed_protein + :protein,
			consumed_carbs   = consumed_carbs   + :carbs,
			consumed_fat     = consumed_fat     + :fat,
			consumed_fiber   = consumed_fiber   + :fiber
	`
	m := item.Macros
	rollupQuery, rollupArgs, err := sqlx.Named(rollupQ, map[string]any{
		"user_id": userID, "date": localDate,
		"kcal": m.Calories, "protein": m.Protein, "carbs": m.Carbs, "fat": m.Fat, "fiber": m.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind update rollup: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(rollupQuery), rollupArgs...); err != nil {
		return fmt.Errorf("store: update rollup: %w", err)
	}

	return tx.Commit()
}

// DeleteMealItem removes the item at itemIndex (zero-based, position order)
// from a meal and subtracts its macros from that day's rollup.
func (s *Store) DeleteMealItem(ctx context.Context, userID, mealID string, itemIndex int) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	meal, err := s.mealOwner(ctx, tx, mealID, userID)
	if err != nil {
		return err
	}

	const itemsQ = `
		SELECT id, kcal, protein, carbs, fat, fiber
		FROM resolved_items WHERE meal_id = ? ORDER BY position
	`
	var items []mealItemRow
	if err := tx.SelectContext(ctx, &items, s.rewrite(itemsQ), mealID); err != nil {
		return fmt.Errorf("store: query items: %w", err)
	}
	if itemIndex < 0 || itemIndex >= len(items) {
		return fmt.Errorf("store: item index %d out of range [0, %d): %w", itemIndex, len(items), types.ErrNotFound)
	}

	target := items[itemIndex]
	if _, err := tx.ExecContext(ctx, s.rewrite(`DELETE FROM resolved_items WHERE id = ?`), target.ID); err != nil {
		return fmt.Errorf("store: delete item: %w", err)
	}

	localDate := parseUTC(meal.AtUTC).Format("2006-01-02")
	const rollupQ = `
		UPDATE daily_rollups SET
			consumed_kcal    = consumed_kcal    - ?,
			consumed_protein = consumed_protein - ?,
			consumed_carbs   = consumed_carbs   - ?,
			consumed_fat     = consumed_fat     - ?,
			consumed_fiber   = consumed_fiber   - ?
		WHERE user_id = ? AND date = ?
	`
	m := types.Macros{Calories: target.Kcal, Protein: target.Protein, Carbs: target.Carbs, Fat: target.Fat, Fiber: target.Fiber}
	if _, err := tx.ExecContext(ctx, s.rewrite(rollupQ),
		m.Calories, m.Protein, m.Carbs, m.Fat, m.Fiber, userID, localDate,
	); err != nil {
		return fmt.Errorf("store: update rollup: %w", err)
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Latest meal
// ---------------------------------------------------------------------------

// LatestMealTime returns the most recent meal timestamp for a user, or
// types.ErrNotFound when no meals exist.
func (s *Store) LatestMealTime(ctx context.Context, userID string) (string, error) {
	const q = `SELECT at_utc FROM meals WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`
	var at string
	if err := s.db.GetContext(ctx, &at, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", types.ErrNotFound
		}
		return "", fmt.Errorf("store: latest meal time: %w", err)
	}
	return at, nil
}

// GetMealsInRange returns meals for a user within a date range (inclusive).
func (s *Store) GetMealsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Meal, error) {
	const mealQ = `
		SELECT id, user_id, at_utc, raw_text, confidence, parser_tier, created_at
		FROM meals
		WHERE user_id = ? AND date(at_utc) >= ? AND date(at_utc) <= ?
		ORDER BY at_utc ASC
	`
	var rows []mealRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(mealQ), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: query meals in range: %w", err)
	}

	var meals []types.Meal
	var mealIDs []string
	for _, r := range rows {
		m := r.toMeal()
		meals = append(meals, m)
		mealIDs = append(mealIDs, m.ID)
	}

	if len(meals) == 0 {
		return []types.Meal{}, nil
	}

	itemsByMeal, err := s.loadItems(ctx, mealIDs)
	if err != nil {
		return nil, err
	}
	for i := range meals {
		meals[i].Items = itemsByMeal[meals[i].ID]
	}
	return meals, nil
}
