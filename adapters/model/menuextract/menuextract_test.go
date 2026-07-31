package menuextract

import (
	"strings"
	"testing"
)

func TestParseResponse(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "unfenced json",
			raw:  `{"dishes":[{"name":"Frango à parmegiana","description":"Peito de frango empanado, molho de tomate, queijo"}],"unreadable":false}`,
		},
		{
			name: "fenced json",
			raw:  "```json\n" + `{"dishes":[{"name":"Burger","description":""}],"unreadable":false}` + "\n```",
		},
		{
			name: "unreadable",
			raw:  `{"dishes":[],"unreadable":true}`,
		},
		{
			name:    "malformed",
			raw:     `not json`,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseResponse(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseResponse(%q) want error, got nil", tc.raw)
				}
				if !strings.Contains(err.Error(), "menuextract:") {
					t.Errorf("error %q does not carry menuextract: prefix", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseResponse(%q) unexpected error: %v", tc.raw, err)
			}
			_ = got
		})
	}
}

func TestParseResponseFieldValues(t *testing.T) {
	got, err := ParseResponse(`{"dishes":[{"name":"Frango à parmegiana","description":"Peito de frango empanado, molho de tomate, queijo"},{"name":"Salada Caesar","description":""}],"unreadable":false}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Unreadable {
		t.Errorf("Unreadable = true, want false")
	}
	if len(got.Dishes) != 2 {
		t.Fatalf("Dishes len = %d, want 2", len(got.Dishes))
	}
	if got.Dishes[0].Name != "Frango à parmegiana" {
		t.Errorf("Dishes[0].Name = %q, want Frango à parmegiana", got.Dishes[0].Name)
	}
	if got.Dishes[0].Description != "Peito de frango empanado, molho de tomate, queijo" {
		t.Errorf("Dishes[0].Description = %q, want the menu's description verbatim", got.Dishes[0].Description)
	}
	if got.Dishes[1].Description != "" {
		t.Errorf("Dishes[1].Description = %q, want empty (menu gave none)", got.Dishes[1].Description)
	}
}

func TestParseResponseUnreadable(t *testing.T) {
	got, err := ParseResponse(`{"dishes":[],"unreadable":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Unreadable {
		t.Errorf("Unreadable = false, want true")
	}
	if len(got.Dishes) != 0 {
		t.Errorf("Dishes = %v, want empty", got.Dishes)
	}
}
