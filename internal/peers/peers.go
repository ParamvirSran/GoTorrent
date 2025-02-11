package peers

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

type Peer struct {
	peer_state PeerState
	address    string
	peer_id    []byte
}

type PeerState struct {
	am_choking      bool
	am_interested   bool
	peer_choking    bool
	peer_interested bool
}

// CreatePeer to store info about each peer
func CreatePeer(address string, peerID []byte) *Peer {
	return &Peer{
		PeerState{},
		address,
		peerID,
	}
}

// HandlePeerConnection handles individual peer connections
func HandlePeerConnection(infoHash []byte, peerID []byte, peerAddress string) error {
	peer := CreatePeer(peerAddress, peerID)

	handshake, err := createHandshake(infoHash, peer.peer_id)
	if err != nil {
		log.Printf("Error creating handshake: %v", err)
		return err
	}

	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", peer.address)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
		return err
	}
	defer conn.Close()

	if _, err := conn.Write(handshake); err != nil {
		log.Fatal(err)
	}

	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		log.Printf("Failed to read handshake response: %v", err)
		return err
	}

	for {
		select {

		default:
			keepAlive := KeepAliveMessage{Length: 0}

			data, err := keepAlive.Serialize()
			if err != nil {
				fmt.Println("Error:", err)
				return err
			}
			if _, err := conn.Write(data); err != nil {
				log.Fatal(err)
			}
		}
	}
}

// ExtractPeers will take the peers returned from a tracker and return the parsed peer list
func ExtractPeers(trackerResp map[string]interface{}) ([]string, error) {
	var peerList []string
	var err error

	if peers, ok := trackerResp["peers"].(string); ok {
		peerList, err = parseCompactPeers([]byte(peers))
	} else if peers, ok := trackerResp["peers"].([]interface{}); ok {
		peerList, err = parseDictionaryPeers(peers)
	}
	return peerList, err
}

// parseCompactPeers will parse the compact peer list when we get that format
func parseCompactPeers(peers []byte) ([]string, error) {
	if len(peers)%6 != 0 {
		return nil, fmt.Errorf("invalid compact peers length")
	}

	var peerList []string
	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4]).String()
		port := int(peers[i+4])<<8 + int(peers[i+5])
		peer := fmt.Sprintf("%s:%d", ip, port)
		peerList = append(peerList, peer)
	}

	return peerList, nil
}

// parseDictionaryPeers will return the peerlist when trackers provide us a standard peerlist in map format
func parseDictionaryPeers(peers []interface{}) ([]string, error) {
	var peerList []string
	for _, peer := range peers {
		peerMap, ok := peer.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid peer format")
		}
		ip, ipOk := peerMap["ip"].(string)
		port, portOk := peerMap["port"].(int)
		if ipOk && portOk {
			peerList = append(peerList, fmt.Sprintf("%s:%d", ip, port))
		} else {
			return nil, fmt.Errorf("invalid peer format: missing ip or port")
		}
	}
	return peerList, nil
}
