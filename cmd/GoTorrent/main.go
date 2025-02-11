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
	peerList := getPeers(torrentFile, infoHash, peerID)

	// Channel to monitor peer status
	peerStatusChan := make(chan string, len(peerList))
	go connectToPeers(peerList, infoHash, peerID, peerStatusChan)
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
		// gracefulShutdown()
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
func getPeers(torrentFile *torrent.Torrent, infoHash []byte, peerID []byte) []string {
	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		log.Fatal("No valid trackers found")
	}
	log.Println(torrentFile.Info.Pieces)
	left := 1
	uploaded := 0
	downloaded := 0
	peerList, err := torrent.ContactTrackers(trackers, string(infoHash), string(peerID), StartEvent, uploaded, downloaded, left, DefaultPort)
	if err != nil {
		log.Fatalf("Error contacting trackers: %v", err)
	}

	if len(peerList) == 0 {
		log.Fatal("No peers found")
	}

	return peerList
}

// connectToPeers establishes connections to each peer in the peer list concurrently
func connectToPeers(peerList []string, infoHash []byte, peerID []byte, peerStatusChan chan string) {
	var wg sync.WaitGroup

	for _, peerAddress := range peerList {
		wg.Add(1)
		go func(peerAddress string) {
			defer wg.Done()
			err := peers.HandlePeerConnection(infoHash, peerID, peerAddress)
			if err != nil {
				peerStatusChan <- "Failed: " + peerAddress + " - " + err.Error()
			} else {
				peerStatusChan <- "Connected: " + peerAddress
			}
		}(peerAddress)
	}

	go func() {
		wg.Wait()
		close(peerStatusChan)
	}()
}

// // trackAndDownload manages the downloading process
// func trackAndDownload(torrentFile *torrent.Torrent, infoHash []byte, peerID []byte) {
// 	pieceHashes := torrentFile.Info.SplitPieces()
// 	totalPieces := len(pieceHashes)
// 	downloadedPieces := make([]bool, totalPieces)
// 	pieceData := make([][]byte, totalPieces)
//
// 	// Concurrently request pieces from available peers
// 	var wg sync.WaitGroup
// 	for i := 0; i < totalPieces; i++ {
// 		if !downloadedPieces[i] {
// 			wg.Add(1)
// 			go func(pieceIndex int) {
// 				defer wg.Done()
// 				err := requestPiece(infoHash, peerID, pieceIndex)
// 				if err != nil {
// 					log.Printf("Failed to request piece %d: %v", pieceIndex, err)
// 					return
// 				}
// 				// Simulate piece download completion and mark it as downloaded
// 				downloadedPieces[pieceIndex] = true
// 				// Once a piece is downloaded, verify its hash
// 				err = verifyPiece(pieceData[pieceIndex], pieceHashes[pieceIndex])
// 				if err != nil {
// 					log.Printf("Piece %d failed verification: %v", pieceIndex, err)
// 					// If the piece fails verification, we may need to request it again
// 					downloadedPieces[pieceIndex] = false
// 				} else {
// 					log.Printf("Piece %d downloaded and verified successfully", pieceIndex)
// 				}
// 			}(i)
// 		}
// 	}
// 	wg.Wait()
// }

// // verifyPiece verifies a downloaded piece using the piece's hash
// func verifyPiece(pieceData []byte, expectedHash []byte) error {
// 	// Verify the piece's integrity by comparing its hash with the expected hash
// 	hash := utils.CalculateSHA1(pieceData)
// 	if !utils.CompareHashes(hash, expectedHash) {
// 		return fmt.Errorf("hash mismatch")
// 	}
// 	return nil
// }

// // requestPiece simulates requesting a piece from a peer
// func requestPiece(infoHash []byte, peerID []byte, pieceIndex int) error {
// 	// Implement communication with peers here. For example:
// 	// - Send a "Request" message for the piece.
// 	// - Handle incoming "Piece" messages from peers.
// 	// This function would involve network communication with peer(s), but let's simulate here:
// 	log.Printf("Requesting piece %d from peer", pieceIndex)
//
// 	// Simulate receiving piece data
// 	pieceData := []byte("dummy piece data") // Replace with actual piece data from peer
// 	fmt.Println(pieceData)
//
// 	// Store the downloaded piece data
// 	// You may want to append this to a global slice or write it to a file.
// 	log.Printf("Received piece %d", pieceIndex)
//
// 	// In real implementation, this is where you would save the piece data.
// 	// For simplicity, we're just logging it here.
// 	return nil
// }

// // gracefulShutdown handles program termination
// func gracefulShutdown() {
// 	log.Println("Shutting down gracefully...")
//
// 	// Clean up resources such as peer connections, file writes, etc.
// 	err := cleanupResources()
// 	if err != nil {
// 		log.Printf("Error during shutdown: %v", err)
// 	}
//
// 	log.Println("Graceful shutdown complete.")
// 	os.Exit(0)
// }
//
// // cleanupResources handles cleanup operations
// func cleanupResources() error {
// 	// Close any open peer connections, stop ongoing downloads, etc.
// 	// You can add a close operation here for connections, files, etc.
// 	return nil
// }
