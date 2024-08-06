package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func main() {
	torrentPath := "mine.torrent"

	torrentFile, err := ParseTorrentFile(torrentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing torrent file (%s): %v\n", torrentPath, err)
		os.Exit(1)
	}

	infoHash, err := GetInfoHash(torrentFile.Info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting info hash: %v\n", err)
		os.Exit(1)
	}

	peerID, err := GeneratePeerID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating peer ID: %v\n", err)
		os.Exit(1)
	}

	baseAnnounceURL, err := url.Parse(torrentFile.Announce)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing announce URL: %v\n", err)
		os.Exit(1)
	}

	trackers := []string{baseAnnounceURL.String()}
	for _, tier := range torrentFile.AnnounceList {
		for _, t := range tier {
			tURL, err := url.Parse(t)
			if err == nil && tURL.Scheme != "udp" {
				trackers = append(trackers, tURL.String())
			}
		}
	}

	success := false
	for _, trackerURL := range trackers {
		fmt.Println("Trying tracker:", trackerURL)

		baseURL, err := url.Parse(trackerURL)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			continue
		}

		params := url.Values{}
		params.Add("info_hash", infoHash)
		params.Add("peer_id", url.QueryEscape(peerID))
		params.Add("port", "6881")
		params.Add("uploaded", "0")
		params.Add("downloaded", "0")
		params.Add("left", strconv.Itoa(torrentFile.Info.Length))
		params.Add("event", "started")

		baseURL.RawQuery = params.Encode()

		response, err := http.Get(baseURL.String())
		if err != nil {
			fmt.Println("Error making GET request:", err)
			continue
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			continue
		}

		trackerResponse, err := ParseTrackerResponse(body)
		if err != nil {
			fmt.Println("Error parsing tracker response:", err)
			continue
		}

		if failureReason, ok := trackerResponse["failure reason"].(string); ok {
			fmt.Printf("Tracker response failure: %s\n", failureReason)
			continue
		}

		fmt.Println("Tracker Response:", trackerResponse)
		success = true
		break
	}

	if !success {
		fmt.Println("Failed to get a valid response from any tracker.")
	}
}
