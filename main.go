package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	torrentPath := "mine.torrent"

	// Parse the torrent file
	torrentFile, err := ParseTorrentFile(torrentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing torrent file (%s): %v\n", torrentPath, err)
		os.Exit(1)
	}

	// Get the info hash from the torrent file
	infoHash, err := GetInfoHash(torrentFile.Info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting info hash: %v\n", err)
		os.Exit(1)
	}

	// Generate a peer ID
	peerID, err := GeneratePeerID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating peer ID: %v\n", err)
		os.Exit(1)
	}

	// Parse the announce URL
	baseAnnounceURL, err := url.Parse(torrentFile.Announce)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing announce URL: %v\n", err)
		os.Exit(1)
	}

	// Collect all tracker URLs
	trackers := []string{baseAnnounceURL.String()}
	for _, tier := range torrentFile.AnnounceList {
		for _, t := range tier {
			tURL, err := url.Parse(t)
			if err == nil && tURL.Scheme != "udp" {
				trackers = append(trackers, tURL.String())
			}
		}
	}

	// Iterate over all trackers
	success := false
	for _, trackerURL := range trackers {
		fmt.Println("Trying tracker:", trackerURL)

		// Parse the tracker URL
		baseURL, err := url.Parse(trackerURL)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			continue
		}

		// Create query parameters
		params := url.Values{}
		params.Add("info_hash", string(infoHash))
		params.Add("downloaded", "0")
		params.Add("event", "started")
		params.Add("left", "262144")
		params.Add("peer_id", peerID)
		params.Add("port", "1337")
		params.Add("uploaded", "0")

		// Attach query parameters to the URL
		baseURL.RawQuery = params.Encode()

		// Make the GET request
		response, err := http.Get(baseURL.String())
		if err != nil {
			fmt.Println("Error making GET request:", err)
			continue
		}
		defer response.Body.Close()

		// Read and print the response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			continue
		}

		fmt.Println("Response:", string(body))
		success = true
		break
	}

	if !success {
		fmt.Println("Failed to get a valid response from any tracker.")
	}
}
