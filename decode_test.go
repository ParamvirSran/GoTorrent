package main

import (
	"bytes"
	"testing"
)

// Helper function to create a new reader for testing
func newReader(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"i123e", 123, false},
		{"i-456e", -456, false},
		{"ie", 0, true},
		{"i12x3e", 0, true},
	}

	for _, test := range tests {
		reader := newReader(test.input)
		result, err := Decode(reader)
		if test.hasError {
			if err == nil {
				t.Errorf("expected an error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", test.input, err)
			} else if result != test.expected {
				t.Errorf("expected %d, got %d for input %s", test.expected, result, test.input)
			}
		}
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"4:spam", "spam", false},
		{"0:", "", false},
		{"5:hello", "hello", false},
		{"3:ab", "", true},
		{"3:apple", "app", false},
	}

	for _, test := range tests {
		reader := newReader(test.input)
		result, err := Decode(reader)
		if test.hasError {
			if err == nil {
				t.Errorf("expected an error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", test.input, err)
			} else if result != test.expected {
				t.Errorf("expected %s, got %s for input %s", test.expected, result, test.input)
			}
		}
	}
}

func TestDecodeList(t *testing.T) {
	tests := []struct {
		input    string
		expected []interface{}
		hasError bool
	}{
		{"li123ei456ee", []interface{}{123, 456}, false},
		{"l4:spam4:eggse", []interface{}{"spam", "eggs"}, false},
		{"le", []interface{}{}, false},
		{"li123ei45e", nil, true},
	}

	for _, test := range tests {
		reader := newReader(test.input)
		result, err := Decode(reader)
		if test.hasError {
			if err == nil {
				t.Errorf("expected an error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", test.input, err)
			} else {
				resultList, ok := result.([]interface{})
				if !ok {
					t.Errorf("expected a list for input %s, but got %T", test.input, result)
				} else if len(resultList) != len(test.expected) {
					t.Errorf("expected length %d, got length %d for input %s", len(test.expected), len(resultList), test.input)
				} else {
					for i, v := range resultList {
						if v != test.expected[i] {
							t.Errorf("expected %v, got %v at index %d for input %s", test.expected[i], v, i, test.input)
						}
					}
				}
			}
		}
	}
}

func TestDecodeDict(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]interface{}
		hasError bool
	}{
		{"d3:cow3:moo4:spam4:eggse", map[string]interface{}{"cow": "moo", "spam": "eggs"}, false},
		{"d4:bull3:cow3:cow3:mooe", map[string]interface{}{"bull": "cow", "cow": "moo"}, false},
		{"de", map[string]interface{}{}, false},
		{"d3:cowi123ee", map[string]interface{}{"cow": 123}, false},
		{"d3:cow3:moo", nil, true},
	}

	for _, test := range tests {
		reader := newReader(test.input)
		result, err := Decode(reader)
		if test.hasError {
			if err == nil {
				t.Errorf("expected an error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", test.input, err)
			} else {
				resultDict, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("expected a dictionary for input %s, but got %T", test.input, result)
				} else if len(resultDict) != len(test.expected) {
					t.Errorf("expected length %d, got length %d for input %s", len(test.expected), len(resultDict), test.input)
				} else {
					for k, v := range resultDict {
						if v != test.expected[k] {
							t.Errorf("expected %v, got %v for key %s in input %s", test.expected[k], v, k, test.input)
						}
					}
				}
			}
		}
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
		hasError bool
	}{
		{"i123e", 123, false},
		{"4:spam", "spam", false},
		{"li123ei456ee", []interface{}{123, 456}, false},
		{"d3:cow3:moo4:spam4:eggse", map[string]interface{}{"cow": "moo", "spam": "eggs"}, false},
		{"x", nil, true},
	}

	for _, test := range tests {
		reader := newReader(test.input)
		result, err := Decode(reader)
		if test.hasError {
			if err == nil {
				t.Errorf("expected an error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", test.input, err)
			} else {
				switch expected := test.expected.(type) {
				case int:
					if result != expected {
						t.Errorf("expected %d, got %v for input %s", expected, result, test.input)
					}
				case string:
					if result != expected {
						t.Errorf("expected %s, got %v for input %s", expected, result, test.input)
					}
				case []interface{}:
					resultList, ok := result.([]interface{})
					if !ok {
						t.Errorf("expected a list for input %s, but got %T", test.input, result)
					} else if len(resultList) != len(expected) {
						t.Errorf("expected length %d, got length %d for input %s", len(expected), len(resultList), test.input)
					} else {
						for i, v := range resultList {
							if v != expected[i] {
								t.Errorf("expected %v, got %v at index %d for input %s", expected[i], v, i, test.input)
							}
						}
					}
				case map[string]interface{}:
					resultDict, ok := result.(map[string]interface{})
					if !ok {
						t.Errorf("expected a dictionary for input %s, but got %T", test.input, result)
					} else if len(resultDict) != len(expected) {
						t.Errorf("expected length %d, got length %d for input %s", len(expected), len(resultDict), test.input)
					} else {
						for k, v := range resultDict {
							if v != expected[k] {
								t.Errorf("expected %v, got %v for key %s in input %s", expected[k], v, k, test.input)
							}
						}
					}
				}
			}
		}
	}
}
