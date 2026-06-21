// Package i18n provides locale-aware string resolution for DietDaemon. It loads
// flat JSON translation bundles (map[string]string keyed by locale) and resolves
// keys with text/template interpolation. Fallback: exact locale -> base language
// (strip region) -> "en" -> the bare key string.
package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Bundle holds locale strings. Keys are JSON paths; values are text/template
// strings. Thread-safe after construction; Load and T are safe for concurrent
// reads once initial loading is complete.
type Bundle struct {
	translations map[string]map[string]string // locale -> key -> template
}

// NewBundle creates an empty Bundle ready for Load calls.
func NewBundle() *Bundle {
	return &Bundle{translations: make(map[string]map[string]string)}
}

// Load reads a JSON locale file (map[string]string) into the bundle. Multiple
// calls for the same locale merge keys; later calls overwrite earlier ones.
// Returns an error when the data is not valid JSON of the expected shape.
func (b *Bundle) Load(locale string, data []byte) error {
	var kv map[string]string
	if err := json.Unmarshal(data, &kv); err != nil {
		return fmt.Errorf("i18n: load %s: %w", locale, err)
	}
	if b.translations[locale] == nil {
		b.translations[locale] = make(map[string]string)
	}
	for k, v := range kv {
		b.translations[locale][k] = v
	}
	return nil
}

// T resolves a translation key for the given locale, applying optional
// template data. Fallback order:
//  1. Exact locale match (e.g. "pt-BR")
//  2. Base language match (e.g. "pt" when locale is "pt-BR")
//  3. English ("en")
//  4. The key itself as the last resort
//
// Template execution errors are silently swallowed and return the unresolved
// template string.
func (b *Bundle) T(locale string, key types.I18nKey, data map[string]interface{}) string {
	k := string(key)

	// Try requested locale first.
	if tmpl, ok := b.translations[locale][k]; ok {
		return b.exec(tmpl, data)
	}

	// Try stripping region: "pt-BR" -> "pt".
	if i := strings.Index(locale, "-"); i > 0 {
		base := locale[:i]
		if tmpl, ok := b.translations[base][k]; ok {
			return b.exec(tmpl, data)
		}
	}

	// Fall back to English.
	if tmpl, ok := b.translations["en"][k]; ok {
		return b.exec(tmpl, data)
	}

	// Last resort: return the key.
	return k
}

func (b *Bundle) exec(tmpl string, data map[string]interface{}) string {
	t, err := template.New("").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return tmpl
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return tmpl
	}
	return buf.String()
}

// LoadEmbedded reads all .json files from an embed.FS and loads them as locale
// bundles. Each file name (without extension) becomes the locale code (e.g.
// "en.json" -> locale "en").
func (b *Bundle) LoadEmbedded(fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("i18n: read embedded locales: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		locale := strings.TrimSuffix(entry.Name(), ".json")
		data, err := fs.ReadFile(fsys, entry.Name())
		if err != nil {
			return fmt.Errorf("i18n: read %s: %w", entry.Name(), err)
		}
		if err := b.Load(locale, data); err != nil {
			return fmt.Errorf("i18n: load %s: %w", entry.Name(), err)
		}
	}
	return nil
}
