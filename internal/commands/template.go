package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TemplateStore is the subset of store methods needed by /template.
type TemplateStore interface {
	GetTemplates(ctx context.Context, userID string) ([]types.MealTemplate, error)
	GetTemplate(ctx context.Context, id string) (types.MealTemplate, error)
	LogTemplateUse(ctx context.Context, log types.TemplateLog) error
}

// TemplateMealLogger is the interface for persisting a meal derived from a
// template. The concrete implementation is pipeline.Engine.LogMeal.
type TemplateMealLogger interface {
	LogMeal(ctx context.Context, meal types.Meal) error
}

// TemplateCommand handles /template -- log a meal template or list templates.
type TemplateCommand struct {
	store   TemplateStore
	mealLog TemplateMealLogger
	idgen   func() string
}

// NewTemplateCommand creates a TemplateCommand that logs templates through the
// provided meal logger.
func NewTemplateCommand(s TemplateStore, ml TemplateMealLogger) *TemplateCommand {
	return &TemplateCommand{store: s, mealLog: ml, idgen: randomID}
}

func (c *TemplateCommand) Name() string        { return "/template" }
func (c *TemplateCommand) Aliases() []string   { return nil }
func (c *TemplateCommand) Help() types.I18nKey { return "cmd.template.usage" }

func (c *TemplateCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" || args == "list" {
		templates, err := c.store.GetTemplates(ctx, msg.UserID)
		if err != nil || len(templates) == 0 {
			return types.Reply{
				Text:        "No templates saved yet. Save one from the dashboard or from a logged meal.",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		var b strings.Builder
		b.WriteString("Templates:\n\n")
		for _, t := range templates {
			total := macrosSum(t.Items)
			fmt.Fprintf(&b, "  - %s -- %.0f kcal (P%.0f/C%.0f/F%.0f)\n", t.Name, total.Calories, total.Protein, total.Carbs, total.Fat)
		}
		b.WriteString("\nUse /template <name> to log one.")
		return types.Reply{Text: b.String(), ChannelMeta: msg.ChannelMeta}, nil
	}

	// Find template by name (case-insensitive).
	templates, err := c.store.GetTemplates(ctx, msg.UserID)
	if err != nil {
		return types.Reply{}, fmt.Errorf("get templates: %w", err)
	}
	var tmpl *types.MealTemplate
	for i, t := range templates {
		if strings.EqualFold(t.Name, args) {
			tmpl = &templates[i]
			break
		}
	}
	if tmpl == nil {
		return types.Reply{
			Text:        fmt.Sprintf("Template not found: %s", args),
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}

	// Log the template as a meal.
	now := time.Now().UTC()
	meal := types.Meal{
		ID:         c.idgen(),
		UserID:     msg.UserID,
		At:         now,
		RawText:    fmt.Sprintf("[template] %s", tmpl.Name),
		Items:      tmpl.Items,
		Confidence: 1.0,
		ParserTier: types.TierDeterministic,
		CreatedAt:  now,
	}
	if err := c.mealLog.LogMeal(ctx, meal); err != nil {
		return types.Reply{}, fmt.Errorf("log template meal: %w", err)
	}

	// Record template usage.
	_ = c.store.LogTemplateUse(ctx, types.TemplateLog{
		ID:         c.idgen(),
		UserID:     msg.UserID,
		TemplateID: tmpl.ID,
		LoggedAt:   now,
	})

	total := meal.Total()
	return types.Reply{
		Text: fmt.Sprintf("Logged template \"%s\": %.0f kcal | P %.0fg . C %.0fg . F %.0fg",
			tmpl.Name, total.Calories, total.Protein, total.Carbs, total.Fat),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}

// macrosSum sums the macros across all resolved items.
func macrosSum(items []types.ResolvedItem) types.Macros {
	var sum types.Macros
	for _, it := range items {
		sum = sum.Add(it.Macros)
	}
	return sum
}
