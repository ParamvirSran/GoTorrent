package main

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
	}

	for _, test := range tests {
		result, err := encodeInt(test.input)
		if err != nil {
			t.Errorf("unexpected error for input %d: %v", test.input, err)
		}
		if string(result) != test.expected {
			t.Errorf("expected %s, got %s for input %d", test.expected, result, test.input)
		}
	}
}

func TestEncodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"spam", "4:spam"},
		{"", "0:"},
		{"hello", "5:hello"},
	}

	for _, test := range tests {
		result, err := encodeString(test.input)
		if err != nil {
			t.Errorf("unexpected error for input %s: %v", test.input, err)
		}
		if string(result) != test.expected {
			t.Errorf("expected %s, got %s for input %s", test.expected, result, test.input)
		}
	}
}

func TestEncodeList(t *testing.T) {
	tests := []struct {
		input    []interface{}
		expected string
	}{
		{[]interface{}{123, "spam", []interface{}{"nested", 456}}, "li123e4:spaml6:nestedi456eee"},
		{[]interface{}{}, "le"},
		{[]interface{}{"a", "b", "c"}, "l1:a1:b1:ce"},
		{[]interface{}{"spam", "eggs"}, "l4:spam4:eggse"},
	}

	for _, test := range tests {
		result, err := encodeList(test.input)
		if err != nil {
			t.Errorf("unexpected error for input %v: %v", test.input, err)
		}
		if string(result) != test.expected {
			t.Errorf("expected %s, got %s for input %v", test.expected, result, test.input)
		}
	}
}

func TestEncodeDict(t *testing.T) {
	tests := []struct {
		input    map[string]interface{}
		expected string
	}{
		{map[string]interface{}{"cow": "moo", "spam": "eggs"}, "d3:cow3:moo4:spam4:eggse"},
		{map[string]interface{}{}, "de"},
		{map[string]interface{}{"key": "value", "nested": map[string]interface{}{"nkey": "nvalue"}}, "d3:key5:value6:nestedd4:nkey6:nvalueee"},
	}

	for _, test := range tests {
		result, err := encodeDict(test.input)
		if err != nil {
			t.Errorf("unexpected error for input %v: %v", test.input, err)
		}
		if string(result) != test.expected {
			t.Errorf("expected %s, got %s for input %v", test.expected, result, test.input)
		}
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
		hasError bool
	}{
		{123, "i123e", false},
		{"spam", "4:spam", false},
		{[]interface{}{123, "spam", []interface{}{"nested", 456}}, "li123e4:spaml6:nestedi456eee", false},
		{map[string]interface{}{"cow": "moo", "spam": "eggs"}, "d3:cow3:moo4:spam4:eggse", false},
		{3.14, "", true}, // Unsupported type
	}

	for _, test := range tests {
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
	}
}
