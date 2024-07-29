package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// GetInfoHash takes the Info struct from a torrent and returns the SHA-1 hash of the bencoded info dictionary.
func GetInfoHash(info Info) (string, error) {
	infoMap := map[string]interface{}{
		"name":         info.Name,
		"piece length": info.PieceLength,
		"pieces":       info.Pieces,
	}

	if info.Length <= 0 {
		// Multi-file case
		var files []interface{}
		for _, file := range info.Files {
			fileMap := map[string]interface{}{
				"length": file.Length,
			}
			var path []interface{}
			for _, p := range file.Path {
				path = append(path, p)
			}
			fileMap["path"] = path
			files = append(files, fileMap)
		}
		infoMap["files"] = files
	} else {
		// Single-file case
		infoMap["length"] = info.Length
	}

	// Encode the info dictionary using bencoding
	encodedInfo, err := Encode(infoMap)
	if err != nil {
		return "", fmt.Errorf("failed to encode info dictionary: %w", err)
	}

	// Compute the SHA-1 hash of the bencoded info dictionary
	hash := sha1.Sum(encodedInfo)
	return hex.EncodeToString(hash[:]), nil
}
