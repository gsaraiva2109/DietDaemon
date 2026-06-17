package normalize

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Frango", "frango"},
		{"  Frângó  ", "frango"},
		{"CAFÉ", "cafe"},
		{"açaí", "acai"},
		{"FEIJÃO", "feijao"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range tests {
		got := Normalize(tc.in)
		if got != tc.want {
			t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
