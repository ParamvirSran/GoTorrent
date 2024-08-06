package main

import (
	"bytes"
	"fmt"
	"slices"
)

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
		return nil, fmt.Errorf("unsupported type for encoding: %T", v)
	}
}

func encodeInt(i int) ([]byte, error) {
	return []byte(fmt.Sprintf("i%de", i)), nil
}

func encodeString(s string) ([]byte, error) {
	return []byte(fmt.Sprintf("%d:%s", len(s), s)), nil
}

func encodeList(l []interface{}) ([]byte, error) {
	buf := bytes.NewBufferString("l")
	for _, item := range l {
		if item == nil {
			return nil, fmt.Errorf("nil value found in list; cannot encode nil as bencoded data")
		}
		encodedItem, err := Encode(item)
		if err != nil {
			return nil, fmt.Errorf("error encoding list item: %w", err)
		}
		buf.Write(encodedItem)
	}
	buf.WriteString("e")
	return buf.Bytes(), nil
}

func encodeDict(d map[string]interface{}) ([]byte, error) {
	buf := bytes.NewBufferString("d")
	keys := make([]string, 0, len(d))
	for key := range d {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	for _, key := range keys {
		val := d[key]
		if val == nil {
			return nil, fmt.Errorf("nil value for key '%s'; cannot encode nil as bencoded data", key)
		}
		encodedKey, err := encodeString(key)
		if err != nil {
			return nil, fmt.Errorf("error encoding dictionary key '%s': %w", key, err)
		}
		buf.Write(encodedKey)

		encodedVal, err := Encode(val)
		if err != nil {
			return nil, fmt.Errorf("error encoding dictionary value for key '%s': %w", key, err)
		}
		buf.Write(encodedVal)
	}
	buf.WriteString("e")
	return buf.Bytes(), nil
}
