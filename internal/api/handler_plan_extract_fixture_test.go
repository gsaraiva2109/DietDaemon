package api

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/planextract"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Multi-page regression coverage (issue #224) — a real Dietbox-style 4-page
// plan, fed through both import paths, asserting every page's content
// actually reaches the model/vision adapter in order and that the resulting
// draft exposes every day type, slot, option, item, substitution and note
// the fixture describes. This is the test that must fail if a page is
// silently dropped anywhere between the request and the parsed draft.
// ---------------------------------------------------------------------------

const fixtureDir = "testdata/plan_import"

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fixtureDir + "/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestHandleExtractPlanFromText_MultiPageFixtureRegression(t *testing.T) {
	text := string(mustReadFixture(t, "dietbox_pasted_text.txt"))
	response := string(mustReadFixture(t, "dietbox_response.json"))

	adapter := &fakeCompletionAdapter{response: response}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = adapter

	rec := doExtractPlanFromText(h, text)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	// Every page marker must reach the prompt, in strictly increasing order —
	// proves no page was dropped or reordered on the way into the request.
	// Search sequentially from the end of the previous match: the prompt's
	// own instructions cite "--- Page 2 ---" as a formatting example, so an
	// unanchored strings.Index would find that instructional text instead of
	// the pasted payload's marker.
	markers := []string{"--- Page 1 ---", "--- Page 2 ---", "--- Page 3 ---", "--- Page 4 ---"}
	markerIdx := make([]int, len(markers))
	searchFrom := 0
	for i, m := range markers {
		idx := strings.Index(adapter.calledPrompt[searchFrom:], m)
		if idx < 0 {
			t.Fatalf("calledPrompt missing marker %q after position %d", m, searchFrom)
		}
		idx += searchFrom
		markerIdx[i] = idx
		searchFrom = idx + len(m)
	}

	// Each page's unique content must land after its own marker and before
	// the next one — proves content, not just the marker, made it across.
	pageContent := []string{
		"Aveia com Morango Silvestre",               // page 1
		"Salmão Grelhado com Quinoa Dourada",        // page 2
		"Omelete de Claras com Espinafre Fresco",    // page 3
		"Beber pelo menos 2 litros de água por dia", // page 4
	}
	for i, content := range pageContent {
		idx := strings.Index(adapter.calledPrompt, content)
		if idx < 0 {
			t.Fatalf("calledPrompt missing page %d content %q", i+1, content)
		}
		if idx < markerIdx[i] {
			t.Errorf("page %d content %q at index %d, want it after its own marker (index %d)", i+1, content, idx, markerIdx[i])
		}
		if i+1 < len(markerIdx) && idx > markerIdx[i+1] {
			t.Errorf("page %d content %q at index %d, want it before next page's marker (index %d)", i+1, content, idx, markerIdx[i+1])
		}
	}

	got := decodeJSON[types.PlanDraft](t, rec)
	assertDietboxDraft(t, got)
}

func TestHandleExtractPlanFromImage_MultiPageFixtureRegression(t *testing.T) {
	response := string(mustReadFixture(t, "dietbox_response.json"))
	wantDraft, err := planextract.ParseResponse(response)
	if err != nil {
		t.Fatalf("ParseResponse(fixture): %v", err)
	}

	names := []string{"page1.png", "page2.png", "page3.png", "page4.png"}
	contents := make([][]byte, len(names))
	for i, name := range names {
		contents[i] = mustReadFixture(t, name)
	}

	adapter := &fakeVisionAdapter{planDraft: wantDraft}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = adapter

	rec := doExtractPlanFromImagesOrdered(h, names, contents)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	if len(adapter.calledPages) != len(contents) {
		t.Fatalf("calledPages len = %d, want %d", len(adapter.calledPages), len(contents))
	}
	for i, p := range adapter.calledPages {
		if !bytes.Equal(p.Data, contents[i]) {
			t.Errorf("calledPages[%d].Data does not match %s bytes exactly", i, names[i])
		}
		if p.MimeType != "image/png" {
			t.Errorf("calledPages[%d].MimeType = %q, want image/png", i, p.MimeType)
		}
	}

	got := decodeJSON[types.PlanDraft](t, rec)
	assertDietboxDraft(t, got)
}

// assertDietboxDraft asserts a decoded PlanDraft matches the ground truth in
// dietbox_response.json / describes what's actually written in
// dietbox_pasted_text.txt: both day types, their slot/option/item shape, the
// substitution note, the general note, and the weekday schedule.
func assertDietboxDraft(t *testing.T, got types.PlanDraft) {
	t.Helper()

	if got.PlanName == nil || *got.PlanName != "Plano Alimentar Exemplo" {
		t.Errorf("PlanName = %v, want Plano Alimentar Exemplo", got.PlanName)
	}
	if len(got.DayTypes) != 2 {
		t.Fatalf("DayTypes len = %d, want 2", len(got.DayTypes))
	}

	assertDayType(t, got.DayTypes[0], "Dia de treino", []int{2, 3, 2})
	assertDayType(t, got.DayTypes[1], "Dia de descanso", []int{2, 2, 1})

	if len(got.Substitutions) != 1 || got.Substitutions[0] != "Pode substituir arroz por batata doce" {
		t.Errorf("Substitutions = %v, want [Pode substituir arroz por batata doce]", got.Substitutions)
	}
	if got.Notes == nil || !strings.Contains(*got.Notes, "Beber pelo menos 2 litros de água por dia") {
		t.Errorf("Notes = %v, want it to contain the water-intake note", got.Notes)
	}

	wantSchedule := []*string{new("Dia de treino"), new("Dia de descanso"), new("Dia de treino"), new("Dia de descanso"), new("Dia de treino"), nil, nil}
	assertWeekdaySchedule(t, got.WeekdaySchedule, wantSchedule)
}

func assertDayType(t *testing.T, day types.PlanDraftDayType, wantName string, wantSlotItemCounts []int) {
	t.Helper()

	if day.Name != wantName {
		t.Errorf("DayTypes.Name = %q, want %s", day.Name, wantName)
	}
	if len(day.Slots) != len(wantSlotItemCounts) {
		t.Fatalf("%s slots len = %d, want %d", wantName, len(day.Slots), len(wantSlotItemCounts))
	}
	for i, slot := range day.Slots {
		if len(slot.Options) != 1 || len(slot.Options[0].Items) != wantSlotItemCounts[i] {
			t.Errorf("%s slot %d (%s) item count = %d, want %d", wantName, i, slot.Label, itemCount(slot), wantSlotItemCounts[i])
		}
	}
}

func assertWeekdaySchedule(t *testing.T, got, want []*string) {
	t.Helper()

	if len(got) != 7 {
		t.Fatalf("WeekdaySchedule len = %d, want 7", len(got))
	}
	for i, w := range want {
		g := got[i]
		if (w == nil) != (g == nil) || (w != nil && *w != *g) {
			t.Errorf("WeekdaySchedule[%d] = %v, want %v", i, derefOrNilStr(g), derefOrNilStr(w))
		}
	}
}

func itemCount(slot types.PlanDraftSlot) int {
	if len(slot.Options) == 0 {
		return 0
	}
	return len(slot.Options[0].Items)
}

func derefOrNilStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// TestHandleExtractPlanFromText_OutOfOrderMarkersPassthrough guards that the
// handler never silently reorders or "fixes" page markers itself — a client
// bug that submits pages out of numeric order must reach the model exactly
// as received, not be rewritten into a false sense of order.
func TestHandleExtractPlanFromText_OutOfOrderMarkersPassthrough(t *testing.T) {
	text := "--- Page 2 ---\nsegunda página\n\n--- Page 1 ---\nprimeira página"

	adapter := &fakeCompletionAdapter{response: validPlanDraftResponse}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = adapter

	rec := doExtractPlanFromText(h, text)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(adapter.calledPrompt, text) {
		t.Errorf("calledPrompt does not contain the out-of-order text verbatim: %q", adapter.calledPrompt)
	}
	idx2 := strings.Index(adapter.calledPrompt, "--- Page 2 ---")
	idx1 := strings.Index(adapter.calledPrompt, "--- Page 1 ---")
	if idx2 >= idx1 {
		t.Errorf("handler reordered markers: want Page 2 marker (idx %d) before Page 1 marker (idx %d), matching input order", idx2, idx1)
	}
}
