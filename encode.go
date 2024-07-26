package main

import (
	"bytes"
	"fmt"
	"slices"
)

// Encode function to encode Go data types to bencoded data
func Encode(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case int:
		return encodeInt(v)
	case string:
		return encodeString(v)
	case []interface{}:
		return encodeList(v)
	case map[string]interface{}:
		return encodeDict(v)
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

// Helper function to encode integers
func encodeInt(i int) ([]byte, error) {
	return []byte(fmt.Sprintf("i%de", i)), nil
}

// Helper function to encode strings
func encodeString(s string) ([]byte, error) {
	return []byte(fmt.Sprintf("%d:%s", len(s), s)), nil
}

// Helper function to encode lists
func encodeList(l []interface{}) ([]byte, error) {
	buf := bytes.NewBufferString("l")
	for _, item := range l {
		encodedItem, err := Encode(item)
		if err != nil {
			return nil, err
		}
		buf.Write(encodedItem)
	}
	buf.WriteString("e")
	return buf.Bytes(), nil
}

// Helper function to encode dictionaries
func encodeDict(d map[string]interface{}) ([]byte, error) {
	buf := bytes.NewBufferString("d")
	keys := make([]string, 0, len(d))
	for key := range d {
		keys = append(keys, key)
	}

	// alpha order sorting
	slices.Sort(keys)

	for _, key := range keys {
		encodedKey, err := encodeString(key)
		if err != nil {
			return nil, err
		}
		buf.Write(encodedKey)
		encodedVal, err := Encode(d[key])
		if err != nil {
			return nil, err
		}
		buf.Write(encodedVal)
	}
	buf.WriteString("e")
	return buf.Bytes(), nil
}
