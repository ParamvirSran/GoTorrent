package main

import (
	"fmt"
	"net/url"
	"os"
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

	uploaded := 0
	downloaded := 0
	left := torrentFile.Info.PieceLength
	event := "started"

	trackers := []string{torrentFile.Announce}
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
		requestURL, err := BuildAnnounceURL(trackerURL, infoHash, peerID, event, uploaded, downloaded, left)
		if err != nil {
			continue
		}

		response, err := SendGetRequest(requestURL)
		if err != nil {
			continue
		}

		trackerResponse, err := ParseTrackerResponse(response)
		if err != nil {
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
