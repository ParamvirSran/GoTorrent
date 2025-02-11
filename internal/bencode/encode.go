package bencode

import (
	"bytes"
	"fmt"
	"slices"
)

// Encode serializes data into a bencoded format
func Encode(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case int64:
		return encodeInt(int(v))
	case int:
		return encodeInt(v)
	case string:
		return encodeString(v)
	case []byte:
		return encodeString(string(v)) // Treat []byte as string for bencoding
	case []interface{}:
		return encodeList(v)
	case map[string]interface{}:
		return encodeDict(v)
	default:
		return nil, fmt.Errorf("unsupported type for encoding: %T", v)
	}
}

// encodeInt serializes an integer into bencoded format
func encodeInt(i int) ([]byte, error) {
	return []byte(fmt.Sprintf("i%de", i)), nil
}

// encodeString serializes a string into bencoded format
func encodeString(s string) ([]byte, error) {
	return []byte(fmt.Sprintf("%d:%s", len(s), s)), nil
}

// encodeList serializes a list of items into bencoded format
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

// encodeDict serializes a dictionary into bencoded format
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
