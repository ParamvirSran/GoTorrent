package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// Decode function to decode bencoded data from an io.Reader
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

// Helper function to decode integers
func decodeInt(r io.Reader) (interface{}, error) {
	b := make([]byte, 1)
	buf := bytes.NewBuffer(nil)

	for {
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

// Helper function to decode strings
func decodeString(r io.Reader, firstByte byte) (interface{}, error) {
	b := make([]byte, 1)
	buf := bytes.NewBuffer([]byte{firstByte})

	for {
		_, err := r.Read(b)
		if err != nil {
			return "", fmt.Errorf("error reading string length: %w", err)
		}
		if b[0] == ':' {
			break
		}
		buf.WriteByte(b[0])
	}

	len, err := strconv.Atoi(buf.String())
	if err != nil {
		return "", fmt.Errorf("invalid string length: %w", err)
	}

	str := make([]byte, len)
	_, err = io.ReadFull(r, str)
	if err != nil {
		return "", fmt.Errorf("error reading string with parsed length: %w", err)
	}
	return string(str), nil
}

// Helper function to decode lists
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
			return nil, fmt.Errorf("error reading list: %w", err)
		}
		result = append(result, val)
	}
	return result, nil
}

// Helper function to decode dictionaries
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

		val, err := Decode(r)
		if err != nil {
			return nil, fmt.Errorf("error reading dictionary value: %w", err)
		}

		result[key.(string)] = val
	}

	return result, nil
}
