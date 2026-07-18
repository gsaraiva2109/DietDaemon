package labelextract

import "testing"

func TestParseResponse(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "unfenced full label",
			raw:  `{"name":"Whole Milk","basis_grams":100,"calories":61,"protein_g":3.2,"carbs_g":4.8,"fat_g":3.3,"fiber_g":0,"low_confidence_fields":[],"unreadable":false}`,
		},
		{
			name: "fenced json",
			raw:  "```json\n{\"name\":\"Leite Integral\",\"basis_grams\":200,\"calories\":122,\"protein_g\":6.4,\"carbs_g\":9.6,\"fat_g\":6.6,\"fiber_g\":null,\"low_confidence_fields\":[\"calories\"],\"unreadable\":false}\n```",
		},
		{
			name: "partial nulls",
			raw:  `{"name":"Mystery Bar","basis_grams":null,"calories":null,"protein_g":null,"carbs_g":null,"fat_g":null,"fiber_g":null,"low_confidence_fields":[],"unreadable":false}`,
		},
		{
			name: "unreadable",
			raw:  `{"name":null,"basis_grams":null,"calories":null,"protein_g":null,"carbs_g":null,"fat_g":null,"fiber_g":null,"low_confidence_fields":[],"unreadable":true}`,
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
	got, err := ParseResponse(`{"name":"Whole Milk","basis_grams":100,"calories":61,"protein_g":3.2,"carbs_g":4.8,"fat_g":3.3,"fiber_g":0,"low_confidence_fields":["calories"],"unreadable":false}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name == nil || *got.Name != "Whole Milk" {
		t.Errorf("Name = %v, want Whole Milk", got.Name)
	}
	if got.BasisGrams == nil || *got.BasisGrams != 100 {
		t.Errorf("BasisGrams = %v, want 100", got.BasisGrams)
	}
	if got.Calories == nil || *got.Calories != 61 {
		t.Errorf("Calories = %v, want 61", got.Calories)
	}
	if len(got.LowConfidenceFields) != 1 || got.LowConfidenceFields[0] != "calories" {
		t.Errorf("LowConfidenceFields = %v, want [calories]", got.LowConfidenceFields)
	}
	if got.Unreadable {
		t.Errorf("Unreadable = true, want false")
	}
}

func TestParseResponseUnreadable(t *testing.T) {
	got, err := ParseResponse(`{"name":null,"basis_grams":null,"calories":null,"protein_g":null,"carbs_g":null,"fat_g":null,"fiber_g":null,"low_confidence_fields":[],"unreadable":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Unreadable {
		t.Errorf("Unreadable = false, want true")
	}
	if got.Name != nil {
		t.Errorf("Name = %v, want nil", got.Name)
	}
}
