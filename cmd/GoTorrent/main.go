package main

import (
	"fmt"
	"github.com/ParamvirSran/GoTorrent/internal/peers"
	"github.com/ParamvirSran/GoTorrent/internal/torrent"
	"log"
	"net"
	"os"
	"time"
)

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

	// Start listening for incoming peer connections
	go startPeerListener(DefaultPort, infoHash, peerID)

	// Connect to discovered peers and begin downloading/uploading
	connectToPeers(peerList, infoHash, peerID)

	// Start downloading the torrent (track progress, request pieces)
	trackAndDownload(torrentFile, infoHash, peerID)

	select {}
}

// parseArgs handles command-line argument parsing and validation
func parseArgs() string {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <torrent-file>", os.Args[0])
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

	fmt.Println("Peers:", peerList)
	return peerList
}

// startPeerListener starts a listener to handle incoming connections from peers
func startPeerListener(port int, infoHash []byte, peerID []byte) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Error listening on port %d: %v\n", port, err)
	}
	defer listener.Close()

	log.Printf("Listening for peer connections on port %d\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		go peers.HandlePeerConnection(conn, string(infoHash), string(peerID))
	}
}

// connectToPeers establishes connections to each peer in the peer list
func connectToPeers(peerList []string, infoHash []byte, peerID []byte) {
	for _, peerAddress := range peerList {
		go func(peerAddress string) {
			conn, err := net.Dial("tcp", peerAddress)
			if err != nil {
				log.Printf("Failed to connect to peer %s: %v\n", peerAddress, err)
				return
			}
			peers.HandlePeerConnection(conn, string(infoHash), string(peerID))
		}(peerAddress)
	}
}

// trackAndDownload tracks the download progress, requesting and storing pieces
func trackAndDownload(torrentFile *torrent.TorrentFile, infoHash []byte, peerID []byte) {
	// Assuming pieces are split into chunks; tracking missing pieces
	totalPieces := len(torrentFile.Info.Pieces) / 20 // Adjust accordingly
	pieces := make([]bool, totalPieces)

	// Example piece download progress
	for i := 0; i < totalPieces; i++ {
		if !pieces[i] {
			// Download this piece (simulated with sleep for now)
			fmt.Printf("Downloading piece %d...\n", i)
			time.Sleep(time.Second) // Simulate download
			pieces[i] = true
			fmt.Printf("Downloaded piece %d\n", i)
		}
	}

	// Check download status and report progress
	downloadedPieces := 0
	for _, piece := range pieces {
		if piece {
			downloadedPieces++
		}
	}
	fmt.Printf("Download progress: %d/%d pieces\n", downloadedPieces, totalPieces)
}
