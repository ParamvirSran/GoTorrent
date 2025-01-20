package main

import (
	"log"
	"os"

	"github.com/ParamvirSran/GoTorrent/internal/peers"
	"github.com/ParamvirSran/GoTorrent/internal/torrent"
	"github.com/ParamvirSran/GoTorrent/internal/utils"
)

var debugMode = false

const (
	DefaultPort = 6881
	StartEvent  = "started"
)

func main() {
	// parse command-line arguments
	torrentPath := parseArgs()

	// parse .torrent file and initialize download parameters
	torrentFile, infoHash, peerID := initializeTorrent(torrentPath)

	// gather trackers and get peers
	peerList := getPeers(torrentFile, infoHash, peerID)

	// Connect to discovered peers and begin downloading/uploading
	connectToPeers(peerList, infoHash, peerID)

	select {}
}

// parseArgs handles command-line argument parsing and validation
func parseArgs() string {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <torrent-file>", os.Args[0])
	}
	if os.Getenv("DEBUG") == "1" {
		debugMode = true
	}
	return os.Args[1]
}

// initializeTorrent parses and sets up the torrent download process
func initializeTorrent(torrentPath string) (*torrent.TorrentFile, []byte, []byte) {
	torrentFile, err := torrent.ParseTorrentFile(torrentPath)
	if err != nil {
		log.Fatalf("Error parsing torrent file (%s): %v", torrentPath, err)
	}

	infoHash, err := torrent.GetInfoHash(torrentFile.Info)
	if err != nil {
		log.Fatalf("Error getting info hash: %v", err)
	}

	peerID, err := torrent.GeneratePeerID()
	if err != nil {
		log.Fatalf("Error generating peer ID: %v", err)
	}

	return torrentFile, infoHash, []byte(peerID)
}

// getPeers returns the compact peer list from the tracker
func getPeers(torrentFile *torrent.TorrentFile, infoHash []byte, peerID []byte) []string {
	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		log.Fatal("No valid trackers found")
	}

	uploaded, downloaded, left := 0, 0, torrentFile.Info.PieceLength // TODO: calculate left properly
	peerList, err := torrent.ContactTrackers(trackers, string(infoHash), string(peerID), StartEvent, uploaded, downloaded, left, DefaultPort)
	if err != nil {
		log.Fatalf("Error contacting trackers: %v", err)
	}

	if len(peerList) == 0 {
		log.Fatal("No peers found")
	}

	utils.DebugLog(debugMode, "Peers:", peerList)
	return peerList
}

// connectToPeers establishes connections to each peer in the peer list
func connectToPeers(peerList []string, infoHash []byte, peerID []byte) {
	for _, peerAddress := range peerList {
		go func(peerAddress string) {
			peers.HandlePeerConnection(infoHash, peerID, peerAddress)
		}(peerAddress)
	}
}

// trackAndDownload tracks the download progress, requesting and storing pieces
func trackAndDownload(torrentFile *torrent.TorrentFile, infoHash []byte, peerID []byte) {
	// Assuming pieces are split into chunks; tracking missing pieces
	totalPieces := len(torrentFile.Info.Pieces) / 20 // Adjust accordingly
	utils.DebugLog(debugMode, "", +totalPieces)
	return
}
