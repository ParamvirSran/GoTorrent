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
		fmt.Printf("Failed to open log file: %v", err)
		os.Exit(1)
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
		fmt.Println("Received termination signal")
		cancel()
	}()

	// Start peer connections
	var wg sync.WaitGroup
	peerStatusChan := make(chan string, len(peerAddressList))

	wg.Add(1)
	go func() {
		defer wg.Done()
		connectToPeers(ctx, peerIDList, peerAddressList, infoHash, peerID, peerStatusChan)
	}()

	// receive peer statuses
	wg.Add(1)
	go func() {
		defer wg.Done()
		for status := range peerStatusChan {
			fmt.Println("Peer status:", status)
		}
	}()

	// Wait for all routines to finish
	wg.Wait()
	fmt.Println("Program terminated successfully")
}

// parseArgs handles command-line argument parsing and validation
func parseArgs() string {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <torrent-file>", os.Args[0])
		os.Exit(1)
	}
	return os.Args[1]
}

// initializeTorrent parses and sets up the torrent download process
func initializeTorrent(torrentPath string) (*torrent.Torrent, []byte, []byte) {
	torrentFile, err := torrent.ParseTorrentFile(torrentPath)
	if err != nil {
		fmt.Printf("Error parsing torrent file (%s): %v", torrentPath, err)
		os.Exit(1)
	}

	infoHash, err := torrent.GetInfoHash(torrentFile.Info)
	if err != nil {
		fmt.Printf("Error getting info hash from torrent file: %v", err)
		os.Exit(1)
	}

	peerID, err := torrent.GeneratePeerID()
	if err != nil {
		fmt.Printf("Error generating peer ID: %v", err)
		os.Exit(1)
	}

	return torrentFile, infoHash, []byte(peerID)
}

// getPeers returns the peer list from the trackers
func getPeers(torrentFile *torrent.Torrent, infoHash []byte, peerID []byte) ([]string, []string) {
	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		fmt.Println("No valid trackers found")
		os.Exit(1)
	}

	left := torrentFile.Info.PieceLength * (len(torrentFile.Info.Pieces) / 20)
	uploaded := 0
	downloaded := 0

	peerIDList, peerAddressList, err := torrent.ContactTrackers(trackers, string(infoHash), string(peerID), StartEvent, uploaded, downloaded, left, DefaultPort)
	if err != nil {
		fmt.Printf("Error contacting trackers: %v", err)
		os.Exit(1)
	}

	if len(peerAddressList) == 0 {
		fmt.Printf("No peers found")
		os.Exit(1)
	}

	return peerIDList, peerAddressList
}

// connectToPeers establishes connections to each peer in the peer list concurrently
func connectToPeers(ctx context.Context, peerIDList, peerAddressList []string, infoHash []byte, clientID []byte, peerStatusChan chan string) {
	var wg sync.WaitGroup

	for i := range peerAddressList {
		wg.Add(1)

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
		}(peerID, peerAddress)
	}

	// Close peerStatusChan after all goroutines finish
	go func() {
		wg.Wait()
		close(peerStatusChan)
	}()
}
