package main

import (
	"context"
	"fmt"
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
	// Set up logging
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Parse torrent file
	torrentPath := parseArgs()
	torrentFile, infoHash, peerID := initializeTorrent(torrentPath)
	peerIDList, peerAddressList := getPeers(torrentFile, infoHash, peerID)

	// Context for managing peer goroutines
	ctx, cancel := context.WithCancel(context.Background())

	// Handle OS signals for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("Received termination signal")
		cancel() // Cancel all goroutines
	}()

	// Start peer connections
	var wg sync.WaitGroup
	peerStatusChan := make(chan string, len(peerAddressList))

	wg.Add(1)
	go func() {
		defer wg.Done()
		connectToPeers(ctx, peerIDList, peerAddressList, infoHash, peerID, peerStatusChan)
	}()

	// Log peer statuses
	wg.Add(1)
	go func() {
		defer wg.Done()
		for status := range peerStatusChan {
			fmt.Println("Peer status:", status)
		}
	}()

	// Wait for all routines to finish
	wg.Wait()
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

// getPeers returns the peer list from the trackers
func getPeers(torrentFile *torrent.Torrent, infoHash []byte, peerID []byte) ([]string, []string) {
	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		log.Fatal("No valid trackers found")
	}

	left := torrentFile.Info.PieceLength * (len(torrentFile.Info.Pieces) / 20)
	uploaded := 0
	downloaded := 0

	peerIDList, peerAddressList, err := torrent.ContactTrackers(trackers, string(infoHash), string(peerID), StartEvent, uploaded, downloaded, left, DefaultPort)
	if err != nil {
		log.Fatalf("Error contacting trackers: %v", err)
	}

	if len(peerAddressList) == 0 {
		log.Fatal("No peers found")
	}

	return peerIDList, peerAddressList
}

// connectToPeers establishes connections to each peer in the peer list concurrently
func connectToPeers(ctx context.Context, peerIDList, peerAddressList []string, infoHash []byte, clientID []byte, peerStatusChan chan string) {
	var wg sync.WaitGroup

	for i := range peerAddressList {
		wg.Add(1)

		// Capture loop variables safely
		peerID := peerIDList[i]
		peerAddress := peerAddressList[i]

		go func(peerID, peerAddress string) {
			defer wg.Done()

			// Check if context is canceled before dialing
			select {
			case <-ctx.Done():
				peerStatusChan <- "Canceled: " + peerAddress
				return
			default:
			}

			err := peers.HandlePeerConnection(peerID, infoHash, clientID, peerAddress)
			if err != nil {
				peerStatusChan <- "Failed with Peer: " + peerAddress + " - " + err.Error()
			} else {
				peerStatusChan <- "Done with Peer: " + peerAddress
			}
		}(peerID, peerAddress) // Pass explicitly to avoid loop variable capture issue
	}

	// Close peerStatusChan after all goroutines finish
	go func() {
		wg.Wait()
		close(peerStatusChan) // Close only when all peers are done
	}()
}
