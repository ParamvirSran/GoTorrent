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

func GetInfoHash(info Info) (string, error) {
	infoMap := make(map[string]interface{})
	infoMap["name"] = info.Name
	infoMap["piece length"] = info.PieceLength
	infoMap["pieces"] = info.Pieces

	if info.Length > 0 {
		// Single-file case
		infoMap["length"] = info.Length
	} else {
		// Multi-file case
		var files []interface{}
		for _, file := range info.Files {
			pathInterfaceList := make([]interface{}, len(file.Path))
			for i, p := range file.Path {
				pathInterfaceList[i] = p
			}
			fileMap := map[string]interface{}{
				"length": file.Length,
				"path":   pathInterfaceList,
			}
			files = append(files, fileMap)
		}
		infoMap["files"] = files
	}

	encodedInfo, err := Encode(infoMap)
	if err != nil {
		return "", fmt.Errorf("failed to encode info dictionary: %w", err)
	}

	hash := sha1.Sum(encodedInfo)
	return string(hash[:]), nil
}

func GeneratePeerID() (string, error) {
	random := make([]byte, 20)
	_, err := rand.Read(random)
	if err != nil {
		return "", fmt.Errorf("error generating random peer ID: %w", err)
	}
	prefix := "MYCLI"
	peerID := prefix + hex.EncodeToString(random)[:20-len(prefix)]
	return peerID, nil
}

func BuildAnnounceURL(baseURL, infoHash, peerID, event string, uploaded, downloaded, left int) (string, error) {
	trackerURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("info_hash", infoHash)
	params.Add("peer_id", peerID)
	params.Add("port", trackerURL.Port())
	params.Add("uploaded", strconv.Itoa(uploaded))
	params.Add("downloaded", strconv.Itoa(downloaded))
	params.Add("left", strconv.Itoa(left))
	params.Add("compact", "1")

	if event != "" {
		params.Add("event", event)
	}

	trackerURL.RawQuery = params.Encode()
	return trackerURL.String(), nil
}

func SendGetRequest(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error sending GET request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

func ParseTrackerResponse(response []byte) (map[string]interface{}, error) {
	decoded, err := Decode(bytes.NewReader(response))
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %w", err)
	}

	trackerResponse, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tracker response format: expected dictionary but got %T", decoded)
	}

	return trackerResponse, nil
}
