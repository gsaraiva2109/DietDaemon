-- Diet plans: a prescribed meal schedule the user transcribes from a
-- nutritionist, with named day-types (e.g. low-carb / high-carb) cycled by a
-- repeating pattern. The app never generates these numbers; every value here
-- is typed in by the user from what a professional prescribed. Cycle length
-- is len(cycle_pattern) — a 7-element array anchored on a Monday is a weekday
-- plan; no separate cycle table.
CREATE TABLE diet_plans (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    notes             TEXT NOT NULL DEFAULT '',
    valid_from        TEXT NOT NULL,
    valid_to          TEXT NOT NULL DEFAULT '', -- '' = open-ended (current plan)
    cycle_pattern     TEXT NOT NULL,             -- JSON array of diet_plan_day_types.id
    cycle_anchor_date TEXT NOT NULL,
    created_at        TEXT NOT NULL DEFAULT '',
    updated_at        TEXT NOT NULL DEFAULT '',
    CHECK (valid_to = '' OR valid_to >= valid_from)
);
CREATE INDEX idx_diet_plans_user_range ON diet_plans(user_id, valid_from, valid_to);

-- Named day-types within a plan (e.g. "Low-carb", "High-carb"). Macros are
-- typed by the user, never derived from foods. water_goal_ml is per day-type
-- because hydration genuinely differs on training days.
CREATE TABLE diet_plan_day_types (
    id            TEXT PRIMARY KEY,
    plan_id       TEXT NOT NULL REFERENCES diet_plans(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    position      INTEGER NOT NULL CHECK (position >= 0),
    kcal          REAL NOT NULL CHECK (kcal >= 0),
    protein       REAL NOT NULL CHECK (protein >= 0),
    carbs         REAL NOT NULL CHECK (carbs >= 0),
    fat           REAL NOT NULL CHECK (fat >= 0),
    fiber         REAL NOT NULL CHECK (fiber >= 0),
    water_goal_ml INTEGER NOT NULL CHECK (water_goal_ml >= 0)
);
CREATE INDEX idx_diet_plan_day_types_plan ON diet_plan_day_types(plan_id);

-- Prescribed meal slots within a day-type (e.g. "Café da manhã").
CREATE TABLE diet_plan_slots (
    id          TEXT PRIMARY KEY,
    day_type_id TEXT NOT NULL REFERENCES diet_plan_day_types(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL CHECK (position >= 0),
    time_of_day TEXT NOT NULL DEFAULT '',
    label       TEXT NOT NULL
);
CREATE INDEX idx_diet_plan_slots_day_type ON diet_plan_slots(day_type_id);

-- Alternatives for a slot ("Opção 1 / Opção 2"), each backed by a
-- meal_templates row (owner_kind = 'plan') for its prescribed foods.
CREATE TABLE diet_plan_slot_options (
    id          TEXT PRIMARY KEY,
    slot_id     TEXT NOT NULL REFERENCES diet_plan_slots(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL CHECK (position >= 0),
    label       TEXT NOT NULL,
    template_id TEXT NOT NULL REFERENCES meal_templates(id) ON DELETE CASCADE
);
CREATE INDEX idx_diet_plan_slot_options_slot ON diet_plan_slot_options(slot_id);

-- Per-date day-type override (one-tap "pick today's type" on the dashboard),
-- takes precedence over the plan's cycle_pattern for that date.
CREATE TABLE diet_plan_day_overrides (
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date        TEXT NOT NULL,
    day_type_id TEXT NOT NULL REFERENCES diet_plan_day_types(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, date)
);

-- 'plan' rows are hidden from the Templates list endpoint; they exist only
-- to back a diet_plan_slot_options row.
ALTER TABLE meal_templates ADD COLUMN owner_kind TEXT NOT NULL DEFAULT 'user' CHECK (owner_kind IN ('user', 'plan'));

-- Explicit slot/option link for a logged meal. Left empty ('') for bot-logged
-- meals, which are matched to a slot by time only at read time — a wrong
-- guess must never persist, so only an explicit SPA log or correction may
-- write these.
ALTER TABLE meals ADD COLUMN plan_slot_id TEXT NOT NULL DEFAULT '';
ALTER TABLE meals ADD COLUMN plan_option_id TEXT NOT NULL DEFAULT '';
