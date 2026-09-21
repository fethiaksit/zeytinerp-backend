package utils

import (
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"05321234567", "+905321234567"},
		{"5321234567", "+905321234567"},
		{"+905321234567", "+905321234567"},
		{"0 (532) 123 45 67", "+905321234567"},
		{"+90 532 123-4567", "+905321234567"},
		{"905321234567", "+905321234567"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		got := NormalizePhone(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestPhoneVariants(t *testing.T) {
	variants := PhoneVariants("05321234567")
	expected := []string{"+905321234567", "05321234567", "5321234567", "905321234567"}

	if len(variants) != len(expected) {
		t.Fatalf("PhoneVariants length = %d, want %d", len(variants), len(expected))
	}

	for i, v := range variants {
		if v != expected[i] {
			t.Errorf("PhoneVariants[%d] = %q, want %q", i, v, expected[i])
		}
	}
}
