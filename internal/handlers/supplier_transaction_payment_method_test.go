package handlers

import "testing"

func TestNormalizeSupplierPaymentMethodCashAliases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "cash", want: "cash"},
		{input: " NAKİT ", want: "cash"},
		{input: "bank", want: "bank_transfer"},
		{input: "BANK_TRANSFER", want: "bank_transfer"},
	}

	for _, test := range tests {
		got, err := normalizeSupplierPaymentMethod(test.input, "payment")
		if err != nil {
			t.Errorf("normalize %q: %v", test.input, err)
			continue
		}
		if got != test.want {
			t.Errorf("normalize %q = %q, want %q", test.input, got, test.want)
		}
	}
}
