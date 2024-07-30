package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// GetInfoHash takes the Info struct from a torrent and returns the SHA-1 hash of the bencoded info dictionary.
func GetInfoHash(info Info) (string, error) {
	infoMap := map[string]interface{}{
		"name":         info.Name,
		"piece length": info.PieceLength,
		"pieces":       info.Pieces,
	}

	if info.Length <= 0 {
		// multifile case
		var files []interface{}
		for _, file := range info.Files {

			var pathInterfaceList []interface{}
			for _, p := range file.Path {
				pathInterfaceList = append(pathInterfaceList, p)
			}

			fileMap := map[string]interface{}{
				"length": file.Length,
				"path":   pathInterfaceList,
			}
			files = append(files, fileMap)
		}
		infoMap["files"] = files
	} else {
		// single file case
		infoMap["length"] = info.Length
	}

	encodedInfo, err := Encode(infoMap)
	if err != nil {
		return "", fmt.Errorf("failed to encode info dictionary: %w", err)
	}
	fmt.Printf("Bencoded Info: %x\n", encodedInfo)

	hash := sha1.Sum(encodedInfo)
	infoHash := hex.EncodeToString(hash[:])
	fmt.Printf("Generated Info Hash: %s\n", infoHash)
	return infoHash, nil
}

// GeneratePeerID generates a unique peer ID of length 20 bytes
func GeneratePeerID() (string, error) {
	random := make([]byte, 20)
	_, err := rand.Read(random)
	if err != nil {
		return "", err
	}

	prefix := "-MYCLI-"
	return prefix + hex.EncodeToString(random)[:20-len(prefix)], nil
}

// BuildAnnounceURL constructs the GET request URL for the tracker
func BuildAnnounceURL(baseURL, infoHash, peerID string, port, uploaded, downloaded, left int, event string) string {
	params := url.Values{}
	params.Add("info_hash", url.QueryEscape(infoHash))
	params.Add("peer_id", url.QueryEscape(peerID))
	params.Add("port", strconv.Itoa(port))
	params.Add("uploaded", strconv.Itoa(uploaded))
	params.Add("downloaded", strconv.Itoa(downloaded))
	params.Add("left", strconv.Itoa(left))

	if event != "" {
		params.Add("event", event)
	}

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// SendGetRequest sends the GET request to the tracker and returns the response body
func SendGetRequest(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error sending get request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

// ParseTrackerResponse parses the tracker's response into a map
func ParseTrackerResponse(response []byte) (map[string]interface{}, error) {
	decoded, err := Decode(bytes.NewReader(response))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	trackerResponse, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tracker response format")
	}

	return trackerResponse, nil
}
