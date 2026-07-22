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
	SaveTemplate(ctx context.Context, tmpl types.MealTemplate) error
	LogTemplateUse(ctx context.Context, log types.TemplateLog) error
}

// TemplateMealLogger is the interface for persisting a meal derived from a
// template. The concrete implementation is pipeline.Engine.LogMeal.
type TemplateMealLogger interface {
	LogMeal(ctx context.Context, meal types.Meal) error
}

// TemplateComposer parses free text into resolved items for template creation.
// The concrete implementation is pipeline.Engine.ParseAndResolve.
type TemplateComposer interface {
	ParseAndResolve(ctx context.Context, userID, text, locale string) ([]types.ResolvedItem, int, error)
}

// TemplateCommand handles /template -- log a meal template or list templates.
type TemplateCommand struct {
	store    TemplateStore
	mealLog  TemplateMealLogger
	composer TemplateComposer
	idgen    func() string
}

// NewTemplateCommand creates a TemplateCommand that logs templates through the
// provided meal logger and composes templates through the composer.
func NewTemplateCommand(s TemplateStore, ml TemplateMealLogger, c TemplateComposer) *TemplateCommand {
	return &TemplateCommand{store: s, mealLog: ml, composer: c, idgen: randomID}
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

	// /template save <name>: <free text> — compose a new template from free text.
	if strings.HasPrefix(args, "save ") {
		rest := strings.TrimPrefix(args, "save ")
		idx := strings.Index(rest, ":")
		if idx < 0 {
			return types.Reply{
				Text:        "Usage: /template save <name>: <ingredients, e.g. 200g chicken, 150g rice>",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}
		name := strings.TrimSpace(rest[:idx])
		freeText := strings.TrimSpace(rest[idx+1:])
		if name == "" || freeText == "" {
			return types.Reply{
				Text:        "Usage: /template save <name>: <ingredients, e.g. 200g chicken, 150g rice>",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}

		// ponytail: no retry loop for partial resolution — v1 scope cut.
		// Users with ambiguous items get a clarification reply and can retry manually.
		items, needsClarification, err := c.composer.ParseAndResolve(ctx, msg.UserID, freeText, msg.Locale)
		if err != nil {
			return types.Reply{}, fmt.Errorf("parse and resolve: %w", err)
		}
		if needsClarification > 0 || len(items) == 0 {
			return types.Reply{
				Text:        "Couldn't fully resolve all ingredients. Be more specific (e.g. \"200g grilled chicken breast, 150g white rice\").",
				ChannelMeta: msg.ChannelMeta,
			}, nil
		}

		tmpl := types.MealTemplate{
			ID:        c.idgen(),
			UserID:    msg.UserID,
			Name:      name,
			Items:     items,
			CreatedAt: time.Now().UTC(),
		}
		if err := c.store.SaveTemplate(ctx, tmpl); err != nil {
			return types.Reply{}, fmt.Errorf("save template: %w", err)
		}

		total := macrosSum(items)
		return types.Reply{
			Text: fmt.Sprintf("Saved template %q: %.0f kcal | P %.0fg · C %.0fg · F %.0fg",
				name, total.Calories, total.Protein, total.Carbs, total.Fat),
			ChannelMeta: msg.ChannelMeta,
		}, nil
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
		RawText:    tmpl.Name,
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
