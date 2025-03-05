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
	_defaultPort = 6881
	_startEvent  = "started"
	_maxWorkers  = 10
)

func main() {
	// Set up logging
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile) // Add timestamps and file info
	log.SetOutput(logFile)

	// Parse torrent file
	torrentPath := parseArgs()
	torrentFile, infoHash, peerID, err := initializeTorrent(torrentPath)
	if err != nil {
		log.Fatalf("Failed to initialize torrent: %v", err)
	}

	peerIDList, peerAddressList, err := getPeers(torrentFile, infoHash, peerID)
	if err != nil {
		log.Fatalf("Failed to get peers: %v", err)
	}

	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start peer connections
	var wg sync.WaitGroup
	peerStatusChan := make(chan string, len(peerAddressList))

	wg.Add(1)
	go func() {
		defer wg.Done()
		connectToPeers(ctx, peerIDList, peerAddressList, infoHash, peerID, peerStatusChan)
	}()

	// Receive peer statuses
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
		fmt.Fprintf(os.Stderr, "Usage: %s <torrent-file>\n", os.Args[0])
		os.Exit(1)
	}
	return os.Args[1]
}

// initializeTorrent parses and sets up the torrent download process
func initializeTorrent(torrentPath string) (*torrent.Torrent, []byte, []byte, error) {
	torrentFile, err := torrent.ParseTorrentFile(torrentPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error parsing torrent file (%s): %v", torrentPath, err)
	}

	infoHash, err := torrent.GetInfohash(torrentFile.Info)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting info hash: %v", err)
	}

	peerID, err := torrent.GeneratePeerID()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error generating peer ID: %v", err)
	}

	return torrentFile, infoHash, []byte(peerID), nil
}

// getPeers returns the peer list from the trackers
func getPeers(torrentFile *torrent.Torrent, infoHash []byte, peerID []byte) ([]string, []string, error) {
	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		return nil, nil, fmt.Errorf("no valid trackers found")
	}

	left := torrentFile.Info.PieceLength * (len(torrentFile.Info.Pieces) / 20)
	uploaded, downloaded := 0, 0

	peerIDList, peerAddressList, err := torrent.ContactTrackers(trackers, string(infoHash), string(peerID), _startEvent, uploaded, downloaded, left, _defaultPort)
	if err != nil {
		return nil, nil, fmt.Errorf("error contacting trackers: %v", err)
	}

	if len(peerAddressList) == 0 {
		return nil, nil, fmt.Errorf("no peers found")
	}

	return peerIDList, peerAddressList, nil
}

// connectToPeers establishes connections to each peer in the peer list concurrently
func connectToPeers(ctx context.Context, peerIDList, peerAddressList []string, infoHash, clientID []byte, peerStatusChan chan<- string) {
	var wg sync.WaitGroup
	defer close(peerStatusChan)

	workerPool := make(chan struct{}, _maxWorkers)

	for i := range peerAddressList {
		select {
		case <-ctx.Done():
			// Exit early if context is canceled
			return
		case workerPool <- struct{}{}: // Acquire worker slot
		}

		wg.Add(1)
		go func(peerID, peerAddress string) {
			defer wg.Done()
			defer func() { <-workerPool }() // Release slot

			select {
			case <-ctx.Done():
				peerStatusChan <- "Canceled: " + peerAddress
				return
			default:
			}

			err := peers.HandlePeerConnection(peerID, infoHash, clientID, peerAddress)
			if err != nil {
				peerStatusChan <- fmt.Sprintf("Failed with Peer: %s - %v", peerAddress, err)
			} else {
				peerStatusChan <- "Done with Peer: " + peerAddress
			}
		}(peerIDList[i], peerAddressList[i])
	}

	wg.Wait()
}
