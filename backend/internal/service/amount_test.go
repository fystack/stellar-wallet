package service

import "testing"

func TestParseAmountKeepsExactStellarPrecision(t *testing.T) {
	tests := []struct {
		raw, want string
		valid     bool
	}{
		{raw: "12.345", want: "12.345", valid: true},
		{raw: " 0.0000001 ", want: "0.0000001", valid: true},
		{raw: "1.0000000", want: "1", valid: true},
		{raw: "100", want: "100", valid: true},
		{raw: "0.00000001"},
		{raw: "0"},
		{raw: "-1"},
		{raw: "abc"},
		{raw: ""},
	}
	for _, tt := range tests {
		got, err := parseAmount(tt.raw)
		if tt.valid != (err == nil) || got != tt.want {
			t.Errorf("parseAmount(%q) = %q, %v", tt.raw, got, err)
		}
	}
}
