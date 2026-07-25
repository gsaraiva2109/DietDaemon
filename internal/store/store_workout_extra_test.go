package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Query-counting sqlite driver, used only to prove GetWorkoutsInRangeWithExercises
// batches its exercises lookup into a single query instead of one per workout.
// ---------------------------------------------------------------------------

var countingDriverSeq int64

// countingDriver wraps the real "sqlite" driver and counts how many times a
// SELECT touching workout_exercises is issued.
type countingDriver struct {
	driver.Driver
	count *int64
}

func (d *countingDriver) Open(name string) (driver.Conn, error) {
	c, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return &countingConn{Conn: c, count: d.count}, nil
}

type countingConn struct {
	driver.Conn
	count *int64
}

func (c *countingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "SELECT") && strings.Contains(query, "workout_exercises") {
		atomic.AddInt64(c.count, 1)
	}
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}

// tempDBCountingExerciseQueries opens a temp SQLite-backed Store through a
// wrapped driver instance that counts SELECT queries against
// workout_exercises, so tests can assert batching actually happened instead
// of trusting code inspection.
//
// Store is built by hand rather than via New(): New()'s driver argument name
// is also used verbatim as the embedded migrations directory ("sqlite" /
// "postgres"), so a uniquely-named wrapped driver (needed so sql.Register
// doesn't collide across tests) would break migration lookup. Constructing
// *Store directly (this file is in package store) decouples the sql.Open
// driver name from the "sqlite" dialect/migrations selection.
func tempDBCountingExerciseQueries(t *testing.T) (*Store, *int64, func()) {
	t.Helper()

	base, err := sql.Open("sqlite", "")
	if err != nil {
		t.Fatalf("open base sqlite driver: %v", err)
	}
	underlying := base.Driver()
	_ = base.Close()

	var count int64
	name := "sqlite-querycount-" + strconv.FormatInt(atomic.AddInt64(&countingDriverSeq, 1), 10)
	sql.Register(name, &countingDriver{Driver: underlying, count: &count})

	f, err := os.CreateTemp("", "dietdaemon-test-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path)

	db, err := sql.Open(name, path)
	if err != nil {
		t.Fatalf("sql.Open(%q): %v", name, err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer, mirrors New()'s setup
	for _, pragma := range []string{
		"PRAGMA locking_mode = EXCLUSIVE",
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			t.Fatalf("%s: %v", pragma, err)
		}
	}

	s := &Store{db: sqlx.NewDb(db, "sqlite"), dialect: SQLiteDialect(), driver: "sqlite", loc: time.UTC}
	if err := s.runMigrations(); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}
	return s, &count, func() {
		_ = s.Close()
		_ = os.Remove(path)
	}
}

// ---------------------------------------------------------------------------
// Fix 1: N+1 batching
// ---------------------------------------------------------------------------

func TestGetWorkoutsInRangeWithExercises_BatchesExerciseQuery(t *testing.T) {
	s, queryCount, cleanup := tempDBCountingExerciseQueries(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-1", CreatedAt: time.Now().UTC()})

	const numWorkouts = 5
	for i := range numWorkouts {
		w := types.Workout{
			ID: "workout-" + strconv.Itoa(i), UserID: "user-1", Name: "Workout",
			DurationMin: 30, Intensity: "medium", LoggedAt: "2026-06-17T12:00:00Z",
			Exercises: []types.WorkoutExercise{
				{Name: "Exercise A"},
				{Name: "Exercise B"},
			},
		}
		if err := s.LogWorkout(ctx(), w); err != nil {
			t.Fatalf("LogWorkout(%d): %v", i, err)
		}
	}

	atomic.StoreInt64(queryCount, 0)

	got, err := s.GetWorkoutsInRangeWithExercises(ctx(), "user-1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetWorkoutsInRangeWithExercises: %v", err)
	}
	if len(got) != numWorkouts {
		t.Fatalf("len(workouts) = %d, want %d", len(got), numWorkouts)
	}
	for _, w := range got {
		if len(w.Exercises) != 2 {
			t.Errorf("workout %s: len(exercises) = %d, want 2", w.ID, len(w.Exercises))
		}
	}

	if n := atomic.LoadInt64(queryCount); n != 1 {
		t.Errorf("workout_exercises SELECT count = %d, want 1 (batched query did not happen)", n)
	}
}

// ---------------------------------------------------------------------------
// Fix 2: UTC-day bug
// ---------------------------------------------------------------------------

// TestListWorkoutsInRange_UsesLocalDateNotUTCDate is the regression test for
// the bug fixed alongside #143: a workout logged near local midnight in a
// non-UTC timezone must be bucketed by the user's local calendar day, not the
// UTC day the timestamp happens to fall on.
func TestListWorkoutsInRange_UsesLocalDateNotUTCDate(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	// America/Sao_Paulo is UTC-3 (no DST as of 2026). 2026-06-18T02:30:00Z is
	// 2026-06-17 23:30 local time -- local day 06-17, UTC day 06-18.
	mustUser(t, s, types.User{ID: "user-1", Timezone: "America/Sao_Paulo", CreatedAt: time.Now().UTC()})

	w := types.Workout{
		ID: "workout-midnight", UserID: "user-1", Name: "Late run",
		DurationMin: 20, Intensity: "low", LoggedAt: "2026-06-18T02:30:00Z",
	}
	if err := s.LogWorkout(ctx(), w); err != nil {
		t.Fatalf("LogWorkout: %v", err)
	}

	got, err := s.ListWorkoutsInRange(ctx(), "user-1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("ListWorkoutsInRange(local day): %v", err)
	}
	if len(got) != 1 || got[0].ID != "workout-midnight" {
		t.Fatalf("ListWorkoutsInRange(2026-06-17) = %+v, want the workout bucketed on the local day", got)
	}

	gotWrongDay, err := s.ListWorkoutsInRange(ctx(), "user-1", "2026-06-18", "2026-06-18")
	if err != nil {
		t.Fatalf("ListWorkoutsInRange(UTC day): %v", err)
	}
	if len(gotWrongDay) != 0 {
		t.Fatalf("ListWorkoutsInRange(2026-06-18) = %+v, want empty (workout must not be bucketed on the UTC day)", gotWrongDay)
	}

	withExercises, err := s.GetWorkoutsInRangeWithExercises(ctx(), "user-1", "2026-06-17", "2026-06-17")
	if err != nil {
		t.Fatalf("GetWorkoutsInRangeWithExercises: %v", err)
	}
	if len(withExercises) != 1 || withExercises[0].ID != "workout-midnight" {
		t.Fatalf("GetWorkoutsInRangeWithExercises = %+v, want the workout bucketed on the local day", withExercises)
	}
}

// TestListWorkoutsInRange_WellWithinDayRoundTrips pins the ordinary,
// pre-migration-style case: a workout logged well within a day (far from any
// midnight boundary) round-trips through ListWorkoutsInRange and
// GetWorkoutsInRangeWithExercises for both UTC and non-UTC users.
func TestListWorkoutsInRange_WellWithinDayRoundTrips(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	mustUser(t, s, types.User{ID: "user-utc", CreatedAt: time.Now().UTC()})
	mustUser(t, s, types.User{ID: "user-tz", Timezone: "America/Sao_Paulo", CreatedAt: time.Now().UTC()})

	for _, userID := range []string{"user-utc", "user-tz"} {
		w := types.Workout{
			ID: "workout-" + userID, UserID: userID, Name: "Midday session",
			DurationMin: 45, Intensity: "high", LoggedAt: "2026-06-17T18:00:00Z",
			Exercises: []types.WorkoutExercise{{Name: "Deadlift"}},
		}
		if err := s.LogWorkout(ctx(), w); err != nil {
			t.Fatalf("LogWorkout(%s): %v", userID, err)
		}

		got, err := s.ListWorkoutsInRange(ctx(), userID, "2026-06-17", "2026-06-17")
		if err != nil {
			t.Fatalf("ListWorkoutsInRange(%s): %v", userID, err)
		}
		if len(got) != 1 || got[0].ID != w.ID {
			t.Fatalf("ListWorkoutsInRange(%s) = %+v, want 1 workout %q", userID, got, w.ID)
		}

		withExercises, err := s.GetWorkoutsInRangeWithExercises(ctx(), userID, "2026-06-17", "2026-06-17")
		if err != nil {
			t.Fatalf("GetWorkoutsInRangeWithExercises(%s): %v", userID, err)
		}
		if len(withExercises) != 1 || len(withExercises[0].Exercises) != 1 {
			t.Fatalf("GetWorkoutsInRangeWithExercises(%s) = %+v, want 1 workout with 1 exercise", userID, withExercises)
		}
	}
}
