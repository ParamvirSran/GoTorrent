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

	// Get the info hash
	infoHash, err := GetInfoHash(torrentFile.Info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting info hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Info Hash: %s\n", infoHash)
}
