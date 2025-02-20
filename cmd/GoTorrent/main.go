package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ParamvirSran/GoTorrent/internal/peers"
	"github.com/ParamvirSran/GoTorrent/internal/torrent"
)

const (
	DefaultPort = 6881
	StartEvent  = "started"
	SHA1Size    = 20
)

func main() {
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Set output of the standard logger to the file
	log.SetOutput(logFile)

	// Parse command-line arguments
	torrentPath := parseArgs()

	// Parse .torrent file and initialize download parameters
	torrentFile, infoHash, peerID := initializeTorrent(torrentPath)

	// Gather trackers and get peers
	peerID_list, peer_address_list := getPeers(torrentFile, infoHash, peerID)

	// Channel to monitor peer status
	peerStatusChan := make(chan string, len(peer_address_list))
	go connectToPeers(peerID_list, peer_address_list, infoHash, peerID, peerStatusChan)
	go func() {
		for status := range peerStatusChan {
			log.Println("Peer status:", status)
		}
	}()

	// Signal channel for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("Received termination signal")
		done <- true
	}()

	<-done
	log.Println("Program terminated successfully")
}

// parseArgs handles command-line argument parsing and validation
func parseArgs() string {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <torrent-file>", os.Args[0])
	}
	return os.Args[1]
}

// initializeTorrent parses and sets up the torrent download process
func initializeTorrent(torrentPath string) (*torrent.Torrent, []byte, []byte) {
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
func getPeers(torrentFile *torrent.Torrent, infoHash []byte, peerID []byte) ([]string, []string) {
	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		log.Fatal("No valid trackers found")
	}
	left := torrentFile.Info.PieceLength * (len(torrentFile.Info.Pieces) / 20)
	log.Println("left to download:", left)
	uploaded := 0
	downloaded := 0
	peerID_list, peer_address_list, err := torrent.ContactTrackers(trackers, string(infoHash), string(peerID), StartEvent, uploaded, downloaded, left, DefaultPort)
	if err != nil {
		log.Fatalf("Error contacting trackers: %v", err)
	}

	if len(peer_address_list) == 0 {
		log.Fatal("No peers found")
	}

	return peerID_list, peer_address_list
}

// connectToPeers establishes connections to each peer in the peer list concurrently
func connectToPeers(peerID_list, peer_address_list []string, infoHash []byte, clientID []byte, peerStatusChan chan string) {
	var wg sync.WaitGroup

	for i, peerAddress := range peer_address_list {
		peer_id := peerID_list[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := peers.HandlePeerConnection(peer_id, infoHash, clientID, peerAddress)
			if err != nil {
				peerStatusChan <- "Failed: " + peerAddress + " - " + err.Error()
			} else {
				peerStatusChan <- "Connected: " + peerAddress
			}
		}()
	}

	go func() {
		wg.Wait()
		close(peerStatusChan)
	}()
}
