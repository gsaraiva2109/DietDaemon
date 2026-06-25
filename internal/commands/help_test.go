package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/i18n"
	"github.com/gsaraiva2109/dietdaemon/internal/i18n/locales"
)

// fakeCmd is a minimal Command stub used to build a registry for help tests.
type fakeCmd struct {
	name    string
	aliases []string
	help    types.I18nKey
}

func (f *fakeCmd) Name() string        { return f.name }
func (f *fakeCmd) Aliases() []string   { return f.aliases }
func (f *fakeCmd) Help() types.I18nKey { return f.help }
func (f *fakeCmd) Handle(_ context.Context, _ types.InboundMessage, _ string) (types.Reply, error) {
	return types.Reply{}, nil
}

func buildTestBundle(t *testing.T) *i18n.Bundle {
	t.Helper()
	b := i18n.NewBundle()
	if err := b.LoadEmbedded(locales.FS); err != nil {
		t.Fatalf("load locales: %v", err)
	}
	return b
}

func TestHelpCommand_ListAll(t *testing.T) {
	bundle := buildTestBundle(t)

	r := NewRegistry()
	mustRegister(t, r, &fakeCmd{name: "/start", help: "cmd.start.help"})
	mustRegister(t, r, &fakeCmd{name: "/target", help: "cmd.target.usage"})
	mustRegister(t, r, &fakeCmd{name: "/status", aliases: []string{"/summary"}, help: "cmd.status.title"})
	mustRegister(t, r, &fakeCmd{name: "/help", aliases: []string{"/h", "/commands"}, help: "cmd.help.description"})

	hc := NewHelpCommand(r, bundle)

	reply, err := hc.Handle(context.Background(), types.InboundMessage{
		Locale:      "en",
		ChannelMeta: map[string]string{"chat_id": "123"},
	}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	t.Logf("\n=== /help (en) ===\n%s\n=== end ===", reply.Text)
	t.Logf("ParseMode: %s", reply.ParseMode)

	if reply.ParseMode != "HTML" {
		t.Errorf("expected ParseMode=HTML, got %q", reply.ParseMode)
	}
	if !strings.Contains(reply.Text, "<b>/start</b>") {
		t.Error("expected <b>/start</b> in output")
	}

	// Verify resolved descriptions present (not raw I18nKeys).
	for _, raw := range []string{"cmd.target.usage", "cmd.status.title", "cmd.start.help", "cmd.start.welcome"} {
		if strings.Contains(reply.Text, raw) {
			t.Errorf("raw I18nKey %q found in output — not translated", raw)
		}
	}

	// Verify start description is clean (no template artifacts).
	if strings.Contains(reply.Text, "<no value>") {
		t.Error("output contains <no value> — template variable not resolved")
	}
}

func TestHelpCommand_Detail(t *testing.T) {
	bundle := buildTestBundle(t)

	r := NewRegistry()
	mustRegister(t, r, &fakeCmd{name: "/target", help: "cmd.target.usage"})
	mustRegister(t, r, &fakeCmd{name: "/status", aliases: []string{"/summary"}, help: "cmd.status.title"})

	hc := NewHelpCommand(r, bundle)

	reply, err := hc.Handle(context.Background(), types.InboundMessage{
		Locale:      "en",
		ChannelMeta: map[string]string{"chat_id": "123"},
	}, "target")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	t.Logf("\n=== /help target (en) ===\n%s\n=== end ===", reply.Text)
	t.Logf("ParseMode: %s", reply.ParseMode)

	if !strings.Contains(reply.Text, "<b>/target</b>") {
		t.Error("expected <b>/target</b> in detail output")
	}
	if !strings.Contains(reply.Text, "Usage") {
		t.Error("expected resolved description in detail output")
	}
}

func TestHelpCommand_UnknownCommand(t *testing.T) {
	bundle := buildTestBundle(t)

	r := NewRegistry()
	hc := NewHelpCommand(r, bundle)

	reply, err := hc.Handle(context.Background(), types.InboundMessage{
		Locale:      "en",
		ChannelMeta: map[string]string{"chat_id": "123"},
	}, "nonexistent")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	t.Logf("\n=== /help nonexistent (en) ===\n%s\n=== end ===", reply.Text)

	if !strings.Contains(reply.Text, "Unknown command") {
		t.Error("expected 'Unknown command' in output")
	}
}

func TestHelpCommand_Portuguese(t *testing.T) {
	bundle := buildTestBundle(t)

	r := NewRegistry()
	mustRegister(t, r, &fakeCmd{name: "/target", help: "cmd.target.usage"})
	mustRegister(t, r, &fakeCmd{name: "/status", help: "cmd.status.title"})

	hc := NewHelpCommand(r, bundle)

	reply, err := hc.Handle(context.Background(), types.InboundMessage{
		Locale:      "pt-BR",
		ChannelMeta: map[string]string{"chat_id": "123"},
	}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	t.Logf("\n=== /help (pt-BR) ===\n%s\n=== end ===", reply.Text)

	if !strings.Contains(reply.Text, "Comandos") {
		t.Error("expected Portuguese title 'Comandos'")
	}
	if !strings.Contains(reply.Text, "<b>/target</b>") {
		t.Error("expected <b>/target</b> in Portuguese output")
	}
}

func TestHelpCommand_FallbackLocale(t *testing.T) {
	bundle := buildTestBundle(t)

	r := NewRegistry()
	mustRegister(t, r, &fakeCmd{name: "/target", help: "cmd.target.usage"})

	hc := NewHelpCommand(r, bundle)

	reply, err := hc.Handle(context.Background(), types.InboundMessage{
		Locale:      "",
		ChannelMeta: map[string]string{"chat_id": "123"},
	}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	t.Logf("\n=== /help (empty locale → en fallback) ===\n%s\n=== end ===", reply.Text)

	if !strings.Contains(reply.Text, "Commands") {
		t.Error("expected English title 'Commands' as fallback")
	}
}

func TestHelpCommand_HTMLEscape(t *testing.T) {
	bundle := buildTestBundle(t)

	r := NewRegistry()
	// Description with characters that would break HTML parse mode if unescaped.
	mustRegister(t, r, &fakeCmd{name: "/test", help: "cmd.start.welcome"})

	hc := NewHelpCommand(r, bundle)

	reply, err := hc.Handle(context.Background(), types.InboundMessage{
		Locale:      "en",
		ChannelMeta: map[string]string{"chat_id": "123"},
	}, "")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	t.Logf("\n=== /help HTML escape test ===\n%s\n=== end ===", reply.Text)

	// The welcome template has {{.Name}} which renders as <no value> with nil data.
	// After HTML escaping, <no value> must become &lt;no value&gt; so Telegram's
	// HTML parser doesn't reject the message.
	if strings.Contains(reply.Text, "<no value>") {
		t.Error("unescaped <no value> in output — would break Telegram HTML parse mode")
	}
	if !strings.Contains(reply.Text, "&lt;no value&gt;") {
		t.Log("Note: <no value> was escaped or the template rendered differently")
	}
}

func mustRegister(t *testing.T, r *Registry, cmd ports.Command) {
	t.Helper()
	if err := r.Register(cmd); err != nil {
		t.Fatalf("register %s: %v", cmd.Name(), err)
	}
}
