package ports

import (
	"context"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// VisionAdapter extracts a nutrition label from a photographed image. Optional;
// only used when OCR_ADAPTER is set. Kept separate from ModelAdapter because
// ModelAdapter.Complete is text-only and scoped to parser tiers — not every
// completion model is vision-capable, and vision calls have different
// cost/latency characteristics than text ones.
type VisionAdapter interface {
	// ExtractLabel reads image (raw bytes, given mimeType e.g. "image/jpeg")
	// and returns the nutrition values it can find. It must never invent or
	// estimate a value: unreadable fields are nil, not guessed.
	ExtractLabel(ctx context.Context, image []byte, mimeType string) (types.NutritionLabelDraft, error)

	// ExtractPlan reads one or more pages (raw bytes + mimeType each) — a
	// photographed or scanned diet plan, possibly spanning several pages in
	// submission order — and returns the plan draft it can find. It must
	// never invent or estimate a value: unreadable fields are nil, not
	// guessed.
	ExtractPlan(ctx context.Context, pages []types.PlanImagePage) (types.PlanDraft, error)

	// ExtractMenu reads image (raw bytes, given mimeType e.g. "image/jpeg") — a
	// photographed restaurant menu — and returns the dish candidates it can
	// find. It must never invent a dish that isn't visible.
	ExtractMenu(ctx context.Context, image []byte, mimeType string) (types.MenuDraft, error)
}
