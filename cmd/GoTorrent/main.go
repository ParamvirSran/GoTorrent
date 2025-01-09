package main

import (
	"fmt"
	"github.com/ParamvirSran/GoTorrent/internal/peers"
	"github.com/ParamvirSran/GoTorrent/internal/torrent"
	"log"
	"net"
	"os"
)

const (
	DefaultPort = 6881
	StartEvent  = "started"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <torrent-file>", os.Args[0])
	}

	torrentPath := os.Args[1]
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

	uploaded, downloaded := 0, 0
	left := torrentFile.Info.PieceLength //TODO calculate correctly
	port := DefaultPort

	trackers := torrent.GatherTrackers(torrentFile)
	if len(trackers) == 0 {
		log.Fatal("No valid trackers found")
	}

	peerList, err := torrent.ContactTrackers(trackers, infoHash, peerID, StartEvent, uploaded, downloaded, left, port)
	if err != nil {
		log.Fatalf("Error contacting trackers: %v", err)
	}

	if len(peerList) == 0 {
		log.Fatal("No peers found")
	}

	fmt.Println("Peers:", peerList)

	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("Error listening on port %d: %v\n", port, err)
		}
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v\n", err)
				continue
			}
			go peers.HandlePeerConnection(conn, infoHash, peerID)
		}
	}()

	for _, peerAddress := range peerList {
		go func(peerAddress string) {
			conn, err := net.Dial("tcp", peerAddress)
			if err != nil {
				log.Printf("Failed to connect to peer %s: %v\n", peerAddress, err)
				return
			}
			peers.HandlePeerConnection(conn, infoHash, peerID)
		}(peerAddress)
	}

	select {}
}
