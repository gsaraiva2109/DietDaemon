// Command tune benchmarks the embedding matcher against the fixture phrases in
// fixtures/test_phrases.json to help choose optimal EMBED_MATCH_THRESHOLD and
// ALIAS_WRITE_BACK_THRESHOLD values. It requires a running Ollama instance.
//
// Usage:
//
//	go run ./cmd/tune -ollama-url http://localhost:11434 -embed-model nomic-embed-text
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/ollama"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/index"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver/embedding"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// phraseEntry is one row from fixtures/test_phrases.json.
type phraseEntry struct {
	Phrase       string `json:"phrase"`
	ExpectedID   string `json:"expected_food_id"`
	ExpectedName string `json:"expected_name"`
}

func main() {
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama base URL")
	embedModel := flag.String("embed-model", "nomic-embed-text", "Embedding model name")
	dbPath := flag.String("db", ":memory:", "SQLite database path (use :memory: for ephemeral)")
	flag.Parse()

	if err := run(*ollamaURL, *embedModel, *dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "tune: %v\n", err)
		os.Exit(1)
	}
}

func run(ollamaURL, embedModel, dbPath string) error {
	phrases, err := loadPhrases("fixtures/test_phrases.json")
	if err != nil {
		return fmt.Errorf("load fixtures: %w", err)
	}
	fmt.Printf("Loaded %d benchmark phrases.\n", len(phrases))

	st, err := store.New(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	src, err := taco.New("")
	if err != nil {
		return fmt.Errorf("load taco: %w", err)
	}

	model := ollama.New(ollamaURL, embedModel, "", 30*time.Second)
	idx := index.New(st.DB())
	matcher := embedding.New(model, idx, st, 0.0) // threshold 0 = return everything for benchmark

	ctx := context.Background()

	// Pre-index expected foods so embeddings are available for matching.
	indexed := map[string]bool{}
	for _, p := range phrases {
		if indexed[p.ExpectedName] {
			continue
		}
		item := types.ParsedItem{RawPhrase: p.ExpectedName}
		match, err := src.Resolve(ctx, item)
		if err != nil {
			fmt.Printf("  skip %q: taco resolve: %v\n", p.ExpectedName, err)
			continue
		}
		_ = st.UpsertFood(ctx, "tune", match, []string{p.ExpectedName})
		_ = matcher.EmbedFood(ctx, "tune", match.FoodID, match.Name)
		indexed[p.ExpectedName] = true
	}
	fmt.Printf("Indexed %d foods.\n", len(indexed))

	thresholds := []float64{0.50, 0.55, 0.60, 0.65, 0.70, 0.75, 0.80, 0.85, 0.90, 0.92, 0.95, 0.98, 0.99}

	fmt.Printf("\n%-8s | %-8s | %-8s | %-8s | %s\n", "Thresh", "Prec", "Recall", "F1", "Matches")
	fmt.Println(strings.Repeat("-", 60))

	for _, th := range thresholds {
		prec, rec, f1, matched := evaluate(ctx, matcher, phrases, th)
		fmt.Printf("  %.2f   |  %.4f  |  %.4f  |  %.4f  | %d/%d\n",
			th, prec, rec, f1, matched, len(phrases))
	}

	return nil
}

func loadPhrases(path string) ([]phraseEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var phrases []phraseEntry
	if err := json.Unmarshal(data, &phrases); err != nil {
		return nil, err
	}
	return phrases, nil
}

func evaluate(
	ctx context.Context,
	matcher *embedding.Matcher,
	phrases []phraseEntry,
	threshold float64,
) (precision, recall, f1 float64, matched int) {
	matcher.SetThreshold(threshold)

	var tp, fp, fn int
	userID := "tune"

	for _, p := range phrases {
		match, err := matcher.Match(ctx, userID, p.Phrase)
		if err != nil {
			fn++
			continue
		}
		if match.FoodID == p.ExpectedID {
			tp++
			matched++
		} else {
			fp++
		}
	}

	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return
}
