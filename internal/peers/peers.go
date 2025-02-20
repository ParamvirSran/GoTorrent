package peers

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

type Peer struct {
	peer_id    string
	address    string
	peer_state PeerState
}

type PeerState struct {
	am_choking      bool
	am_interested   bool
	peer_choking    bool
	peer_interested bool
}

// CreatePeer to store info about each peer
func CreatePeer(peerID, address string) *Peer {
	return &Peer{
		peerID,
		address,
		PeerState{
			am_choking:      true,
			am_interested:   false,
			peer_choking:    true,
			peer_interested: false,
		},
	}
}

// HandlePeerConnection handles individual peer connections
func HandlePeerConnection(peerID string, infoHash []byte, clientID []byte, peerAddress string) error {
	peer := CreatePeer(peerID, peerAddress)

	handshake, err := createHandshake(infoHash, clientID)
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
	log.Printf("handshake: %x", handshake)
	if _, err := conn.Write(handshake); err != nil {
		log.Fatal("error writing handshake:", err)
	}

	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		log.Printf("Failed to read handshake response: %v", err)
		log.Println("response:", response)
		return err
	}
	return nil
}

// ExtractPeers will take the peers returned from a tracker and return the parsed peer list
func ExtractPeers(trackerResp map[string]interface{}) ([]string, []string, error) {
	var peerList []string
	var peer_id_list []string
	var err error

	if peers, ok := trackerResp["peers"].(string); ok {
		peerList, err = parseCompactPeers([]byte(peers))

	} else if peers, ok := trackerResp["peers"].([]interface{}); ok {
		peer_id_list, peerList, err = parseDictionaryPeers(peers)
	}
	log.Println("peer id list:", peer_id_list)
	log.Println("peer address list:", peerList)
	return peer_id_list, peerList, err
}

// parseCompactPeers will parse the compact peer list when we get that format
func parseCompactPeers(peers []byte) ([]string, error) {
	var peerList []string

	if len(peers)%6 != 0 {
		return nil, fmt.Errorf("invalid compact peers length")
	}

	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4]).String()
		port := int(peers[i+4])<<8 + int(peers[i+5])
		peer := fmt.Sprintf("%s:%d", ip, port)
		peerList = append(peerList, peer)
	}

	return peerList, nil
}

// parseDictionaryPeers will return the peerlist when trackers provide us a standard peerlist in map format
func parseDictionaryPeers(peers []interface{}) ([]string, []string, error) {
	var peer_id_list []string
	var peerList []string

	for _, peer := range peers {

		if peerMap, ok := peer.(map[string]interface{}); ok {
			ip, ipOk := peerMap["ip"].(string)
			port, portOk := peerMap["port"].(int)
			peerID, idOk := peerMap["peer id"].(string)

			if ipOk && portOk && idOk {
				peerList = append(peerList, fmt.Sprintf("%s:%d", ip, port))
				peer_id_list = append(peer_id_list, peerID)

			} else {
				return nil, nil, fmt.Errorf("peer ip: %t, peer port: %t, peer id: %t", ipOk, portOk, idOk)
			}

		} else {
			return nil, nil, fmt.Errorf("invalid peer format, expecting dictionary format from tracker")
		}
	}

	return peer_id_list, peerList, nil
}
