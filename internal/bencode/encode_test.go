package bencode

import (
	"testing"
)

func TestEncodeInt(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{123, "i123e"},
		{-456, "i-456e"},
		{0, "i0e"},
		{2147483647, "i2147483647e"},   // Max int32 value
		{-2147483648, "i-2147483648e"}, // Min int32 value
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result, err := encodeInt(test.input)
			if err != nil {
				t.Errorf("unexpected error for input %d: %v", test.input, err)
			}
			if string(result) != test.expected {
				t.Errorf("expected %s, got %s for input %d", test.expected, result, test.input)
			}
		})
	}
}

func TestEncodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"spam", "4:spam", false},
		{"", "0:", false},
		{"hello", "5:hello", false},
		{" ", "1: ", false},       // Space character
		{"a\nb", "3:a\nb", false}, // String with newline
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result, err := encodeString(test.input)
			if test.hasError {
				if err == nil {
					t.Errorf("expected error for input %s, but got none", test.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %s: %v", test.input, err)
				}
				if string(result) != test.expected {
					t.Errorf("expected %s, got %s for input %s", test.expected, result, test.input)
				}
			}
		})
	}
}

func TestEncodeList(t *testing.T) {
	tests := []struct {
		input    []any
		expected string
		hasError bool
	}{
		{[]any{123, "spam", []any{"nested", 456}}, "li123e4:spaml6:nestedi456eee", false},
		{[]any{}, "le", false},
		{[]any{"a", "b", "c"}, "l1:a1:b1:ce", false},
		{[]any{"spam", "eggs"}, "l4:spam4:eggse", false},
		{[]any{nil}, "", true}, // Handling nil as error
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result, err := encodeList(test.input)
			if test.hasError {
				if err == nil {
					t.Errorf("expected error for input %v, but got none", test.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %v: %v", test.input, err)
				}
				if string(result) != test.expected {
					t.Errorf("expected %s, got %s for input %v", test.expected, result, test.input)
				}
			}
		})
	}
}

func TestEncodeDict(t *testing.T) {
	tests := []struct {
		input    map[string]any
		expected string
		hasError bool
	}{
		{map[string]any{"cow": "moo", "spam": "eggs"}, "d3:cow3:moo4:spam4:eggse", false},
		{map[string]any{}, "de", false},
		{map[string]any{"key": "value", "nested": map[string]any{"nkey": "nvalue"}}, "d3:key5:value6:nestedd4:nkey6:nvalueee", false},
		{map[string]any{"key": nil}, "", true}, // Handling nil as error
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result, err := encodeDict(test.input)
			if test.hasError {
				if err == nil {
					t.Errorf("expected error for input %v, but got none", test.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %v: %v", test.input, err)
				}
				if string(result) != test.expected {
					t.Errorf("expected %s, got %s for input %v", test.expected, result, test.input)
				}
			}
		})
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		input    any
		expected string
		hasError bool
	}{
		{123, "i123e", false},
		{"spam", "4:spam", false},
		{[]any{123, "spam", []any{"nested", 456}}, "li123e4:spaml6:nestedi456eee", false},
		{map[string]any{"cow": "moo", "spam": "eggs"}, "d3:cow3:moo4:spam4:eggse", false},
		{3.14, "", true}, // Unsupported type
		{nil, "", true},  // Nil value
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result, err := Encode(test.input)
			if test.hasError {
				if err == nil {
					t.Errorf("expected an error for input %v, but got none", test.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %v: %v", test.input, err)
				}
				if string(result) != test.expected {
					t.Errorf("expected %s, got %s for input %v", test.expected, result, test.input)
				}
			}
		})
	}
}
