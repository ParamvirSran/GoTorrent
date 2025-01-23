package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
)

// CalculateSHA1 calculates the SHA-1 hash of the given data
func CalculateSHA1(data []byte) []byte {
	hash := sha1.Sum(data)
	return hash[:]
}

// CompareHashes compares two hashes and returns true if they are equal
func CompareHashes(hash1 []byte, hash2 []byte) bool {
	return hex.EncodeToString(hash1) == hex.EncodeToString(hash2)
}

// DebugLog logs messages when debug mode is enabled
func DebugLog(debug bool, args ...interface{}) {
	if debug {
		log.Println(args...)
	}
}
