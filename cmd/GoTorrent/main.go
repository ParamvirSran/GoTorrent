package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ParamvirSran/GoTorrent/internal/common"
	"github.com/ParamvirSran/GoTorrent/internal/peers"
	"github.com/ParamvirSran/GoTorrent/internal/torrent"
)

const (
	_defaultPort = "6881"
	_startEvent  = "started"
)

func main() {
	logFile, err := logging()
	if err != nil {
		fmt.Printf("Failed to open log file: %v", err)
		os.Exit(1)
	}
	log.SetOutput(logFile)
	log.Println("Starting")

	torrentPath := parseArgs()
	torrentFile, infoHash, peerID, err := initializeTorrent(torrentPath)
	if err != nil {
		fmt.Printf("Failed to initialize torrent: %v", err)
		os.Exit(1)
	}

	peerIDList, peerAddressList, err := getPeers(torrentFile, infoHash, peerID)
	if err != nil {
		fmt.Printf("Failed to get peers: %v", err)
		os.Exit(1)
	}
	log.Printf("PeerID List - %v", len(peerIDList))
	log.Printf("PeerAddress List - %v", len(peerAddressList))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go peerManager(torrentFile.PieceManager, ctx, peerIDList, peerAddressList, infoHash, peerID)

	<-ctx.Done()
	log.Printf("Exiting. Context error: %v", ctx.Err())
}

// logging sets up a file for logging
func logging() (*os.File, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file: %v", err)
	}
	return logFile, nil
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

// peerManager establishes connections to each peer in the peer list concurrently
func peerManager(pm *common.PieceManager, ctx context.Context, peerIDList, peerAddressList []string, infoHash, clientID []byte) {
	var wg sync.WaitGroup

	for i := range peerAddressList {
		select {
		case <-ctx.Done():
			log.Println("Context canceled, stopping peer connections.")
			return
		default:
			wg.Add(1)
			// Start a new peer handler
			go func(peerID, peerAddress string) {
				defer wg.Done()

				err := peers.HandlePeerConnection(pm, ctx, peerID, infoHash, clientID, peerAddress)
				if err != nil {
					log.Printf("Failed with Peer: %s - %v", peerAddress, err)
				} else {
					log.Printf("Done with Peer: %s", peerAddress)
				}
			}(peerIDList[i], peerAddressList[i])
		}
	}
	wg.Wait()
	log.Println("All peer connections finished.")
}
