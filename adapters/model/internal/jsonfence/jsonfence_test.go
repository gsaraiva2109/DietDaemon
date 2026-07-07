package jsonfence

import "testing"

func TestStrip(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no fence", `{"a":1}`, `{"a":1}`},
		{"plain fence", "```\n{\"a\":1}\n```", `{"a":1}`},
		{"json language tag", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"surrounding whitespace", "  \n```json\n{\"a\":1}\n```\n  ", `{"a":1}`},
		{"no trailing fence", "```json\n{\"a\":1}", `{"a":1}`},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Strip(tc.in); got != tc.want {
				t.Errorf("Strip(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
