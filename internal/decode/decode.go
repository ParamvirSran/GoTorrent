package decode

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

func Decode(r io.Reader) (interface{}, error) {
	b := make([]byte, 1)
	_, err := r.Read(b)
	if err != nil {
		return nil, fmt.Errorf("error reading bencode prefix: %w", err)
	}

	switch b[0] {
	case 'i':
		return decodeInt(r)
	case 'l':
		return decodeList(r)
	case 'd':
		return decodeDict(r)
	default:
		return decodeString(r, b[0])
	}
}

func decodeInt(r io.Reader) (interface{}, error) {
	buf := &bytes.Buffer{}
	for {
		b := make([]byte, 1)
		_, err := r.Read(b)
		if err != nil {
			return nil, fmt.Errorf("error reading integer: %w", err)
		}
		if b[0] == 'e' {
			break
		}
		buf.WriteByte(b[0])
	}

	result, err := strconv.Atoi(buf.String())
	if err != nil {
		return nil, fmt.Errorf("error converting string to integer: %w", err)
	}
	return result, nil
}

func decodeString(r io.Reader, firstByte byte) (interface{}, error) {
	buf := &bytes.Buffer{}
	buf.WriteByte(firstByte)

	for {
		b := make([]byte, 1)
		_, err := r.Read(b)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error reading string length: %w", err)
		}
		if b[0] == ':' {
			break
		}
		buf.WriteByte(b[0])
	}

	length, err := strconv.Atoi(buf.String())
	if err != nil {
		return "", fmt.Errorf("invalid string length: %w", err)
	}

	str := make([]byte, length)
	_, err = io.ReadFull(r, str)
	if err != nil {
		return "", fmt.Errorf("error reading string with parsed length: %w", err)
	}

	return string(str), nil
}

func decodeList(r io.Reader) (interface{}, error) {
	var result []interface{}

	for {
		b := make([]byte, 1)
		_, err := r.Read(b)
		if err != nil {
			return nil, fmt.Errorf("error reading list: %w", err)
		}
		if b[0] == 'e' {
			break
		}
		r = io.MultiReader(bytes.NewReader(b), r)
		val, err := Decode(r)
		if err != nil {
			return nil, fmt.Errorf("error reading list item: %w", err)
		}
		result = append(result, val)
	}
	return result, nil
}

func decodeDict(r io.Reader) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for {
		b := make([]byte, 1)
		_, err := r.Read(b)
		if err != nil {
			return nil, fmt.Errorf("error reading dictionary key prefix: %w", err)
		}

		if b[0] == 'e' {
			break
		}

		r = io.MultiReader(bytes.NewReader(b), r)

		key, err := Decode(r)
		if err != nil {
			return nil, fmt.Errorf("error reading dictionary key: %w", err)
		}

		keyStr, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %T", key)
		}

		val, err := Decode(r)
		if err != nil {
			return nil, fmt.Errorf("error reading dictionary value: %w", err)
		}

		result[keyStr] = val
	}

	return result, nil
}
