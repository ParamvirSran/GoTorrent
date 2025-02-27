package peers

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

// Peer is a struct that holds all the state for a peer the client will communicate with
type Peer struct {
	peer_id    string
	address    string
	peer_state PeerState
}

// PeerState is a struct that holds the flags that represent the clients representation of the bidirectional relationship with a peer
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
	if !isValidPeerAddress(peerAddress) {
		return fmt.Errorf("invalid peer address: %s", peerAddress)
	}

	peer := CreatePeer(peerID, peerAddress)

	handshake, err := CreateHandshake(infoHash, clientID)
	if err != nil {
		return fmt.Errorf("error creating handshake: %v", err)
	}

	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", peer.address)
	if err != nil {
		log.Printf("Failed to dial %s: %v", peer.address, err)
		return err
	}
	defer conn.Close()
	log.Printf("Connected to peer: %s", peer.address)

	// Send the handshake
	if _, err := conn.Write(handshake); err != nil {
		log.Printf("Error writing handshake: %v", err)
		return err
	}

	// Read the handshake response
	response := make([]byte, 68)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		log.Printf("Failed to read handshake response: %v", err)
		return err
	}

	// Validate handshake
	if err := ValidateHandshakeResponse(response[:n], [20]byte(infoHash)); err != nil {
		log.Printf("Invalid handshake response: %v", err)
		return err
	}
	log.Printf("Handshake successful with peer: %s", peer.address)

	// Message handling loop
	for {
		select {
		case <-ctx.Done():
			log.Printf("Disconnecting from peer: %s", peer.address)
			return nil
		default:
			// Read messages from the peer
			msgBuf := make([]byte, 200)
			conn.SetReadDeadline(time.Now().Add(30 * time.Second)) // Timeout for keep-alive

			_, err := conn.Read(msgBuf)
			if err != nil {
				log.Printf("Peer %s disconnected: %v", peer.address, err)
				return err
			}

			// Handle incoming messages
			message, err := ParseMessage(msgBuf)
			if err != nil {
				log.Printf("Error parsing message from %s: %v", peer.address, err)
				continue
			}

			log.Printf("Received message %d from peer %s", message.ID, peer.address)

			// Handle specific message types
			switch message.ID {
			case MsgChoke:
				peer.peer_state.peer_choking = true
			case MsgUnchoke:
				peer.peer_state.peer_choking = false
			case MsgInterested:
				peer.peer_state.peer_interested = true
			case MsgNotInterested:
				peer.peer_state.peer_interested = false
			default:
			}
		}
	}
}

func isValidPeerAddress(address string) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsPrivate() {
		return false
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1024 || portNum > 65535 {
		return false
	}
	return true
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
