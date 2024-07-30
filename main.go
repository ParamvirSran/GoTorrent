package main

import (
	"fmt"
	"os"
)

func main() {
	torrentPath := "test.torrent"

	// Parse the .torrent file
	torrentFile, err := ParseTorrentFile(torrentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing torrent file (%s): %v\n", torrentPath, err)
		os.Exit(1)
	}

	// // Get the info hash
	// infoHash, err := GetInfoHash(torrentFile.Info)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error getting info hash: %v\n", err)
	// 	os.Exit(1)
	// }
	// fmt.Println(infoHash)
	//
	// Generate a unique peer ID
	peerID, err := GeneratePeerID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating peer ID: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(peerID)

	// Build the announce URL
	announceURL := BuildAnnounceURL(
		torrentFile.Announce,
		"47c7105178eb58088980a4ea7fa7f5dee491dd18",
		peerID,
		6881,                    // Port
		0,                       // Uploaded
		0,                       // Downloaded
		torrentFile.Info.Length, // Left
		"started",               // Event
	)
	fmt.Println(announceURL)

	// Send the GET request to the tracker
	response, err := SendGetRequest(announceURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending GET request to tracker: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(response)

	// Parse the tracker response
	trackerResponse, err := ParseTrackerResponse(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing tracker response: %v\n", err)
		os.Exit(1)
	}

	// Print the tracker response
	fmt.Println("Tracker Response:")
	for key, value := range trackerResponse {
		fmt.Printf("%s: %v\n", key, value)
	}

	// Extract and print the peer list
	peers, ok := trackerResponse["peers"].([]interface{})
	if !ok {
		fmt.Println("No peers found in the tracker response.")
		return
	}

	fmt.Println("Peers:")
	for _, peer := range peers {
		peerInfo, ok := peer.(map[string]interface{})
		if !ok {
			fmt.Println("Error: Peer info not in the expected format.")
			continue
		}
		fmt.Printf("Peer ID: %s, IP: %s, Port: %d\n",
			peerInfo["peer id"].(string),
			peerInfo["ip"].(string),
			int(peerInfo["port"].(int64)),
		)
	}

}
