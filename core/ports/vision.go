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
}
