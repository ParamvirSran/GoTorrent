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

		resp, err := SendGetRequest(requestURL)
		if err != nil {
			fmt.Println("Error sending GET request:", err)
			continue
		}

		trackerResp, err := ParseTrackerResponse(resp)
		if err != nil {
			fmt.Println("Error parsing tracker response:", err)
			continue
		}

		if fail, ok := trackerResp["failure reason"].(string); ok {
			fmt.Println("Tracker response failure:", fail)
			continue
		}
		fmt.Println(trackerResp)
		var peerList []string
		if peers, ok := trackerResp["peers"].(string); ok {
			peerList, err = ParseCompactPeers([]byte(peers))
			if err != nil {
				fmt.Println("Error parsing compact peers:", err)
				continue
			}
		} else if peers, ok := trackerResp["peers"].([]interface{}); ok {
			peerList, err = ParseDictionaryPeers(peers)
			if err != nil {
				fmt.Println("Error parsing dictionary peers:", err)
				continue
			}
		}

		fmt.Println("Peers:", peerList)
		success = true
		event = ""
		break
	}
	if !success {
		fmt.Println("Failed to get a valid response from any tracker.")
	}
}
