package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Goals & profile
// ---------------------------------------------------------------------------

// GetProfile returns the user profile, or ErrNotFound.
func (s *Store) GetProfile(ctx context.Context, userID string) (types.UserProfile, error) {
	const q = `
		SELECT user_id, height_cm, birth_date, gender, activity_level, goal,
		       target_weight_kg, weekly_rate, onboarded, created_at, updated_at
		FROM user_profiles WHERE user_id = ?
	`
	var row profileRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.UserProfile{}, types.ErrNotFound
		}
		return types.UserProfile{}, fmt.Errorf("store: get profile: %w", err)
	}
	return row.toUserProfile(), nil
}

// profileRow is the flat DB shape of user_profiles; types.UserProfile stores
// Onboarded as bool (DB: int) and CreatedAt/UpdatedAt as time.Time (DB:
// RFC3339 strings).
type profileRow struct {
	UserID         string  `db:"user_id"`
	HeightCm       float64 `db:"height_cm"`
	BirthDate      string  `db:"birth_date"`
	Gender         string  `db:"gender"`
	ActivityLevel  string  `db:"activity_level"`
	Goal           string  `db:"goal"`
	TargetWeightKg float64 `db:"target_weight_kg"`
	WeeklyRate     float64 `db:"weekly_rate"`
	Onboarded      int     `db:"onboarded"`
	CreatedAt      string  `db:"created_at"`
	UpdatedAt      string  `db:"updated_at"`
}

func (r profileRow) toUserProfile() types.UserProfile {
	return types.UserProfile{
		UserID: r.UserID, HeightCm: r.HeightCm, BirthDate: r.BirthDate, Gender: r.Gender,
		ActivityLevel: r.ActivityLevel, Goal: r.Goal, TargetWeightKg: r.TargetWeightKg,
		WeeklyRate: r.WeeklyRate, Onboarded: r.Onboarded != 0,
		CreatedAt: parseUTC(r.CreatedAt), UpdatedAt: parseUTC(r.UpdatedAt),
	}
}

// UpsertProfile inserts or updates the user profile.
func (s *Store) UpsertProfile(ctx context.Context, p types.UserProfile) error {
	onboarded := 0
	if p.Onboarded {
		onboarded = 1
	}
	const q = `
		INSERT INTO user_profiles
			(user_id, height_cm, birth_date, gender, activity_level, goal,
			 target_weight_kg, weekly_rate, onboarded, created_at, updated_at)
		VALUES (:user_id, :height_cm, :birth_date, :gender, :activity_level, :goal,
			:target_weight_kg, :weekly_rate, :onboarded, :created_at, :updated_at)
		ON CONFLICT(user_id) DO UPDATE SET
			height_cm        = excluded.height_cm,
			birth_date       = excluded.birth_date,
			gender           = excluded.gender,
			activity_level   = excluded.activity_level,
			goal             = excluded.goal,
			target_weight_kg = excluded.target_weight_kg,
			weekly_rate      = excluded.weekly_rate,
			onboarded        = excluded.onboarded,
			updated_at       = excluded.updated_at
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": p.UserID, "height_cm": p.HeightCm, "birth_date": p.BirthDate,
		"gender": p.Gender, "activity_level": p.ActivityLevel, "goal": p.Goal,
		"target_weight_kg": p.TargetWeightKg, "weekly_rate": p.WeeklyRate, "onboarded": onboarded,
		"created_at": utcStr(p.CreatedAt), "updated_at": utcStr(p.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("store: bind upsert profile: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}
