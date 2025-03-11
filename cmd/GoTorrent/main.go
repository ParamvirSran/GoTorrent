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
	_defaultPort = "6881"
	_startEvent  = "started"
	_maxWorkers  = 3
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	torrentPath := parseArgs()
	torrentFile, infoHash, peerID, err := initializeTorrent(torrentPath)
	if err != nil {
		log.Fatalf("Failed to initialize torrent: %v", err)
	}
	log.Printf("Infohash - %x", infoHash)
	log.Printf("PeerID - %s", peerID)

	peerIDList, peerAddressList, err := getPeers(torrentFile, infoHash, peerID)
	if err != nil {
		log.Fatalf("Failed to get peers: %v", err)
	}
	log.Printf("PeerID List - %v", peerIDList)
	log.Printf("PeerAddress List - %v", peerAddressList)

	// Graceful shutdown handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Println("Starting peer connections...")
	go workerManager(ctx, peerIDList, peerAddressList, infoHash, peerID)

	log.Println("Waiting for termination signal...")
	<-ctx.Done()
	log.Printf("Program exiting. Context error: %v", ctx.Err())
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
func initializeTorrent(torrentPath string) (*torrent.Torrent, []byte, []byte, error) {
	torrentFile, err := torrent.ParseTorrentFile(torrentPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error parsing Torrent File (%s): %w", torrentPath, err)
	}
	infoHash, err := torrent.GetInfohash(torrentFile.Info)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error getting Infohash: %w", err)
	}
	peerID, err := torrent.GeneratePeerID()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error generating peerID: %w", err)
	}
	return torrentFile, infoHash, []byte(peerID), nil
}

// getPeers returns the peer list from the trackers
func getPeers(torrentFile *torrent.Torrent, infoHash []byte, peerID []byte) ([]string, []string, error) {
	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		return nil, nil, fmt.Errorf("No valid trackers found")
	}
	left := torrentFile.Info.PieceLength * (len(torrentFile.Info.Pieces) / 20)
	log.Printf("Left to Download: %v", left)
	uploaded, downloaded := 0, 0

	peerIDList, peerAddressList, err := torrent.ContactTrackers(trackers, string(infoHash), string(peerID), _startEvent, uploaded, downloaded, left, _defaultPort)
	if err != nil {
		return nil, nil, fmt.Errorf("Error contacting trackers: %w", err)
	}
	if len(peerAddressList) == 0 {
		return nil, nil, fmt.Errorf("No peers found from trackers")
	}
	return peerIDList, peerAddressList, nil
}

// workerManager establishes connections to each peer in the peer list concurrently
func workerManager(ctx context.Context, peerIDList, peerAddressList []string, infoHash, clientID []byte) {
	var wg sync.WaitGroup
	workerPool := make(chan struct{}, _maxWorkers)
	for i := range peerAddressList {
		select {
		case <-ctx.Done():
			return
		case workerPool <- struct{}{}: // Acquire worker slot
		}

		wg.Add(1)
		go func(peerID, peerAddress string) {
			defer wg.Done()
			defer func() { <-workerPool }() // Release slot

			err := peers.Worker(ctx, peerID, infoHash, clientID, peerAddress)
			if err != nil {
				log.Printf("Failed with Peer: %s - %v", peerAddress, err)
			} else {
				log.Printf("Done with Peer: %s", peerAddress)
			}
		}(peerIDList[i], peerAddressList[i])
	}
	wg.Wait()
}
