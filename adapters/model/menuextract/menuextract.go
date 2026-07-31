// Package menuextract holds the prompt and response contract for the
// photographed-restaurant-menu import path (#201), so the JSON schema the
// model is asked for lives in exactly one place. It mirrors planextract's
// shape: a Prompt constant, private wire types, and ParseResponse — kept
// flatter than planextract since a menu is just a list of dishes, no
// day-type/slot/option tree.
package menuextract

import (
	"encoding/json"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/jsonfence"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Prompt instructs the model to read a photographed restaurant menu — in any
// language — and return the JSON contract ParseResponse expects. It is
// deliberately language-agnostic: the model reads the menu natively, no
// locale branching is needed here.
const Prompt = `You are reading a photographed restaurant menu. The menu may be in any language — read it natively, no locale branching is needed.

List every distinct dish you can find. Each dish has:
- a name, exactly as written on the menu
- a short description: ingredients or preparation, but ONLY if visible on the menu. Do NOT invent details that aren't printed there — if the menu gives no description, leave it empty.

Never invent a dish that isn't visible on the menu.

If the photo is not a legible menu — too blurry, cropped, glare, or not a menu at all — set unreadable to true and dishes to an empty array.

Respond with ONLY this JSON object, no markdown fences, no commentary:
{
  "dishes": [
    {"name": string, "description": string}
  ],
  "unreadable": boolean
}`

type wireDish struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type wireResponse struct {
	Dishes     []wireDish `json:"dishes"`
	Unreadable bool       `json:"unreadable"`
}

// ParseResponse parses a model's raw text response (optionally
// markdown-fenced) into a MenuDraft.
func ParseResponse(raw string) (types.MenuDraft, error) {
	stripped := jsonfence.Strip(raw)

	var wr wireResponse
	if err := json.Unmarshal([]byte(stripped), &wr); err != nil {
		return types.MenuDraft{}, fmt.Errorf("menuextract: decode response: %w", err)
	}

	return types.MenuDraft{
		Dishes:     mapDishes(wr.Dishes),
		Unreadable: wr.Unreadable,
	}, nil
}

func mapDishes(in []wireDish) []types.MenuDishCandidate {
	out := make([]types.MenuDishCandidate, len(in))
	for i, d := range in {
		out[i] = types.MenuDishCandidate{
			Name:        d.Name,
			Description: d.Description,
		}
	}
	return out
}
