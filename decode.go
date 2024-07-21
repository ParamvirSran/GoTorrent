package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

func Decode(r io.Reader) (interface{}, error) {
	b := make([]byte, 1)
	_, err := r.Read(b)
	if err != nil {
		return nil, err
	}

	switch b[0] {
	case 'i':
		return decodeInt(r)
	case 'l':
		return decodeList(r)
	case 'd':
		return decodeDict(r)
	default:
		if b[0] >= '0' && b[0] <= 9 {
			return decodeString(r, b[0])
		} else {
			return nil, fmt.Errorf("Invalid bencode prefix: %c", b[0])
		}
	}
}

func decodeString(r io.Reader, firstByte byte) (string, error) {
	var length int
	b := make([]byte, 1)
	buf := bytes.NewBuffer(nil)

	buf.WriteByte(firstByte)
	for {
		_, err := r.Read(b)
		if err != nil {
			return "", err
		}
		if b[0] == ':' {
			break
		}
		buf.WriteByte(b[0])
	}
	length, err := strconv.Atoi(buf.String())
	if err != nil {
		return "", err
	}
	str := make([]byte, length)
	_, err = r.Read(str)
	if err != nil {
		return "", err
	}
	return string(str), nil
}

func decodeInt(r io.Reader) (int, error) {
	var result int
	b := make([]byte, 1)
	buf := bytes.NewBuffer(nil)

	for {
		_, err := r.Read(b)
		if err != nil {
			return 0, err
		}
		if b[0] == 'e' {
			break
		}
		buf.WriteByte(b[0])
	}
	n, err := strconv.Atoi(buf.String())
	if err != nil {
		return 0, err
	}
	result = n
	return result, nil
}

func decodeList(r io.Reader) ([]interface{}, error) {
	var result []interface{}
	for {
		val, err := Decode(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}

func decodeDict(r io.Reader) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for {
		key, err := Decode(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if keyStr, ok := key.(string); ok {
			val, err := Decode(r)
			if err != nil {
				return nil, err
			}
			result[keyStr] = val
		} else {
			return nil, errors.New("Invalid dictionary key, must be a string")
		}
	}
	return result, nil
}
