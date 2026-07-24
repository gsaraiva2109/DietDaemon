package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ProfileStore is the subset of store methods needed by /profile.
type ProfileStore interface {
	GetProfile(ctx context.Context, userID string) (types.UserProfile, error)
	UpsertProfile(ctx context.Context, p types.UserProfile) error
}

// ProfileCommand handles /profile -- view or set profile fields.
type ProfileCommand struct {
	store ProfileStore
}

// NewProfileCommand creates a ProfileCommand.
func NewProfileCommand(s ProfileStore) *ProfileCommand {
	return &ProfileCommand{store: s}
}

func (c *ProfileCommand) Name() string        { return "/profile" }
func (c *ProfileCommand) Aliases() []string   { return nil }
func (c *ProfileCommand) Help() types.I18nKey { return "cmd.profile.view" }

// profileSetUsage is shown when /profile set is missing "key=value".
const profileSetUsage = "Usage: /profile set key=value\n" +
	"Keys: height_cm (cm), birth_date (YYYY-MM-DD), gender (male/female/other), goal (cut/maintain/bulk), target_weight_kg (kg), weekly_rate (kg/week)\n" +
	"Example: /profile set height_cm=175"

func (c *ProfileCommand) Handle(ctx context.Context, msg types.InboundMessage, args string) (types.Reply, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		return c.viewProfile(ctx, msg)
	}

	// Parse "set key=value" or just "key=value".
	args = strings.TrimPrefix(args, "set ")
	k, v, hasEq := strings.Cut(args, "=")
	if !hasEq {
		return types.Reply{Text: profileSetUsage, ChannelMeta: msg.ChannelMeta}, nil
	}
	k = strings.TrimSpace(k)
	v = strings.TrimSpace(v)

	// Get existing profile.
	p, err := c.store.GetProfile(ctx, msg.UserID)
	if err != nil {
		p = types.UserProfile{UserID: msg.UserID}
	}

	p, errMsg := setProfileField(p, k, v)
	if errMsg != "" {
		return types.Reply{Text: errMsg, ChannelMeta: msg.ChannelMeta}, nil
	}

	if err := c.store.UpsertProfile(ctx, p); err != nil {
		return types.Reply{}, fmt.Errorf("upsert profile: %w", err)
	}

	return types.Reply{
		Text:        fmt.Sprintf("Profile updated: %s = %s", k, v),
		ChannelMeta: msg.ChannelMeta,
	}, nil
}

// viewProfile renders the current profile, or a hint to set one.
func (c *ProfileCommand) viewProfile(ctx context.Context, msg types.InboundMessage) (types.Reply, error) {
	p, err := c.store.GetProfile(ctx, msg.UserID)
	if err != nil || p.HeightCm == 0 {
		return types.Reply{
			Text: "No profile set. Use /profile set key=value to fill it in.\n" +
				"Keys: height_cm, birth_date, gender, goal, target_weight_kg, weekly_rate\n" +
				"Example: /profile set height_cm=175",
			ChannelMeta: msg.ChannelMeta,
		}, nil
	}
	goal := p.Goal
	if goal == "" {
		goal = "not set"
	}
	gender := p.Gender
	if gender == "" {
		gender = "not set"
	}
	text := fmt.Sprintf("Profile\nHeight: %.0f cm\nBirth date: %s\nGender: %s\nGoal: %s\nTarget weight: %.0f kg\nWeekly rate: %.2f kg/week",
		p.HeightCm, p.BirthDate, gender, goal, p.TargetWeightKg, p.WeeklyRate)
	return types.Reply{Text: text, ChannelMeta: msg.ChannelMeta}, nil
}

// setProfileField validates and applies a single "key=value" update to p. On
// validation failure it returns the original profile and a user-facing error
// message; on success it returns the updated profile and an empty message.
func setProfileField(p types.UserProfile, k, v string) (types.UserProfile, string) {
	switch k {
	case "height_cm":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f <= 0 {
			return p, "height_cm must be a positive number (e.g. 175)"
		}
		p.HeightCm = f
	case "birth_date":
		p.BirthDate = v
	case "gender":
		if v != "male" && v != "female" && v != "other" {
			return p, "gender must be male, female, or other"
		}
		p.Gender = v
	case "goal":
		if v != "cut" && v != "maintain" && v != "bulk" {
			return p, "goal must be cut, maintain, or bulk"
		}
		p.Goal = v
	case "target_weight_kg":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f <= 0 {
			return p, "target_weight_kg must be a positive number (e.g. 75)"
		}
		p.TargetWeightKg = f
	case "weekly_rate":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f <= 0 {
			return p, "weekly_rate must be a positive number (e.g. 0.5)"
		}
		p.WeeklyRate = f
	default:
		return p, fmt.Sprintf("Unknown profile key: %s\nValid keys: height_cm, birth_date, gender, goal, target_weight_kg, weekly_rate", k)
	}
	return p, ""
}
