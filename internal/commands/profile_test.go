package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeProfileStore is a minimal ProfileStore stub for /profile tests.
type fakeProfileStore struct {
	profile   types.UserProfile
	getErr    error
	upserted  *types.UserProfile
	upsertErr error
}

func (f *fakeProfileStore) GetProfile(_ context.Context, _ string) (types.UserProfile, error) {
	return f.profile, f.getErr
}

func (f *fakeProfileStore) UpsertProfile(_ context.Context, p types.UserProfile) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserted = &p
	return nil
}

func TestProfileCommand_ViewNoProfile(t *testing.T) {
	store := &fakeProfileStore{getErr: errors.New("not found")}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "No profile set") {
		t.Errorf("expected no-profile hint, got %q", reply.Text)
	}
}

func TestProfileCommand_ViewExisting(t *testing.T) {
	store := &fakeProfileStore{profile: types.UserProfile{
		HeightCm: 180, BirthDate: "1990-01-01", Gender: "male", Goal: "cut",
		TargetWeightKg: 75, WeeklyRate: 0.5,
	}}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "  ")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "180 cm") {
		t.Errorf("expected height in reply, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "cut") {
		t.Errorf("expected goal in reply, got %q", reply.Text)
	}
}

func TestProfileCommand_ViewExistingUnsetFields(t *testing.T) {
	store := &fakeProfileStore{profile: types.UserProfile{HeightCm: 180}}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Goal: not set") || !strings.Contains(reply.Text, "Gender: not set") {
		t.Errorf("expected 'not set' placeholders, got %q", reply.Text)
	}
}

func TestProfileCommand_SetMissingEquals(t *testing.T) {
	store := &fakeProfileStore{}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "set height_cm")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Usage:") {
		t.Errorf("expected usage reply, got %q", reply.Text)
	}
	if store.upserted != nil {
		t.Fatal("expected no upsert on malformed input")
	}
}

func TestProfileCommand_SetHeightValid(t *testing.T) {
	store := &fakeProfileStore{getErr: errors.New("no profile yet")}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "set height_cm=175")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.upserted == nil {
		t.Fatal("expected UpsertProfile to be called")
	}
	if store.upserted.HeightCm != 175 {
		t.Errorf("HeightCm = %v, want 175", store.upserted.HeightCm)
	}
	if store.upserted.UserID != "u1" {
		t.Errorf("UserID = %q, want u1", store.upserted.UserID)
	}
	if !strings.Contains(reply.Text, "height_cm = 175") {
		t.Errorf("expected confirmation, got %q", reply.Text)
	}
}

func TestProfileCommand_SetHeightInvalid(t *testing.T) {
	store := &fakeProfileStore{}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "set height_cm=-5")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "height_cm must be a positive number") {
		t.Errorf("expected validation error, got %q", reply.Text)
	}
	if store.upserted != nil {
		t.Fatal("expected no upsert on invalid height")
	}
}

func TestProfileCommand_SetGenderInvalid(t *testing.T) {
	store := &fakeProfileStore{}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "set gender=robot")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "gender must be male, female, or other") {
		t.Errorf("expected validation error, got %q", reply.Text)
	}
}

func TestProfileCommand_SetGoalInvalid(t *testing.T) {
	store := &fakeProfileStore{}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "set goal=shred")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "goal must be cut, maintain, or bulk") {
		t.Errorf("expected validation error, got %q", reply.Text)
	}
}

func TestProfileCommand_SetWithoutSetPrefix(t *testing.T) {
	store := &fakeProfileStore{}
	cmd := NewProfileCommand(store)

	// "key=value" without the "set " prefix is also accepted.
	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "weekly_rate=0.5")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if store.upserted == nil || store.upserted.WeeklyRate != 0.5 {
		t.Fatalf("expected WeeklyRate=0.5 to be upserted, got %+v", store.upserted)
	}
	if !strings.Contains(reply.Text, "weekly_rate = 0.5") {
		t.Errorf("expected confirmation, got %q", reply.Text)
	}
}

func TestProfileCommand_SetUnknownKey(t *testing.T) {
	store := &fakeProfileStore{}
	cmd := NewProfileCommand(store)

	reply, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "set favorite_color=blue")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(reply.Text, "Unknown profile key: favorite_color") {
		t.Errorf("expected unknown-key error, got %q", reply.Text)
	}
}

func TestProfileCommand_UpsertError(t *testing.T) {
	store := &fakeProfileStore{upsertErr: errors.New("db down")}
	cmd := NewProfileCommand(store)

	_, err := cmd.Handle(context.Background(), types.InboundMessage{UserID: "u1"}, "set height_cm=175")
	if err == nil {
		t.Fatal("expected error when upsert fails")
	}
}

func TestProfileCommand_Metadata(t *testing.T) {
	cmd := NewProfileCommand(&fakeProfileStore{})
	if cmd.Name() != "/profile" {
		t.Errorf("Name() = %q, want /profile", cmd.Name())
	}
	if cmd.Aliases() != nil {
		t.Errorf("Aliases() = %v, want nil", cmd.Aliases())
	}
	if cmd.Help() != types.I18nKey("cmd.profile.view") {
		t.Errorf("Help() = %q, want cmd.profile.view", cmd.Help())
	}
}
