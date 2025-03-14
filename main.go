package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ParamvirSran/GoTorrent/internal/peers"
	"github.com/ParamvirSran/GoTorrent/internal/torrent"
	"github.com/ParamvirSran/GoTorrent/internal/types"
)

const (
	defaultPort        = "6881"
	startEvent         = "started"
	maxConcurrentPeers = 10
)

var pieceSize int

func main() {
	logFile, err := setupLogging()
	if err != nil {
		fmt.Printf("Failed to open log file: %v", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println("Starting")

	torrentPath := parseArgs()
	torrentFile, infohash, peerID, err := initializeTorrent(torrentPath)
	if err != nil {
		fmt.Printf("Failed to initialize torrent: %v", err)
		os.Exit(1)
	}

	peerIDList, peerAddressList, err := getPeers(torrentFile, infohash, peerID)
	if err != nil {
		fmt.Printf("Failed to get peers: %v", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go peerManager(torrentFile, ctx, peerIDList, peerAddressList, infohash, peerID)
	go monitorDownloadCompletion(ctx, cancel, torrentFile)

	<-ctx.Done()
	log.Printf("Exiting. Context error: %v", ctx.Err())
}

func setupLogging() (*os.File, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logFile, err := os.OpenFile("gotorrent.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	return logFile, nil
}

func parseArgs() string {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <torrent-file>\n", os.Args[0])
		os.Exit(1)
	}
	return os.Args[1]
}

func initializeTorrent(torrentPath string) (*types.Torrent, []byte, []byte, error) {
	torrentFile, err := torrent.ParseTorrentFile(torrentPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error parsing torrent file (%s): %w", torrentPath, err)
	}
	pieceSize = torrentFile.Info.PieceLength

	infohash, err := torrent.GetInfohash(torrentFile.Info)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting infohash: %w", err)
	}

	peerID, err := torrent.GeneratePeerID()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error generating peerID: %w", err)
	}

	return torrentFile, infohash, []byte(peerID), nil
}

func getPeers(torrentFile *types.Torrent, infoHash, peerID []byte) ([]string, []string, error) {
	trackerList := torrent.GatherTrackers(torrentFile)
	if len(trackerList) == 0 {
		return nil, nil, fmt.Errorf("no valid trackers found")
	}

	left := torrentFile.Info.PieceLength * (len(torrentFile.Info.Pieces) / 20)
	log.Printf("Torrent Stats - Piece Count: %d - Piece Size: %d - Left to Download: %d", len(torrentFile.Info.Pieces)/20, torrentFile.Info.PieceLength, left)

	uploaded, downloaded := 0, 0
	peerIDList, peerAddressList, err := torrent.ContactTrackers(trackerList, string(infoHash), string(peerID), startEvent, uploaded, downloaded, left, defaultPort)
	if err != nil {
		return nil, nil, fmt.Errorf("error contacting trackers: %w", err)
	}

	if len(peerAddressList) == 0 {
		return nil, nil, fmt.Errorf("no peers found from trackers")
	}

	return peerIDList, peerAddressList, nil
}

func peerManager(torrentFile *types.Torrent, ctx context.Context, peerIDList, peerAddressList []string, infohash, clientID []byte) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentPeers)
	pm := torrentFile.PieceManager

	for i := range peerAddressList {
		select {
		case <-ctx.Done():
			log.Println("Context canceled, stopping peer connections.")
			return
		default:
			sem <- struct{}{}
			wg.Add(1)
			go func(peerID, peerAddress string) {
				defer wg.Done()
				defer func() { <-sem }()

				if err := peers.HandlePeerConnection(pm, ctx, peerID, infohash, clientID, peerAddress); err != nil {
					log.Printf("Failed with Peer: %s - %v", peerAddress, err)
				} else {
					log.Printf("Done with Peer: %s", peerAddress)
				}
			}(peerIDList[i], peerAddressList[i])
		}
	}
	wg.Wait()
	log.Println("All peer connections finished. Peer manager finished")
}

func monitorDownloadCompletion(ctx context.Context, cancel context.CancelFunc, torrentFile *types.Torrent) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if torrentFile.PieceManager.IsDownloadComplete() {
				log.Println("Torrent download complete.")
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
