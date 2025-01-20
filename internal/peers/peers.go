package peers

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

const PROTOCOL_LENGTH = 19
const PROTOCOL_STRING = "BitTorrent protocol"
const RESERVED_BITS = "00000000"

type Peer struct {
	peer_state PeerState
}

type PeerState struct {
	am_choking      bool
	am_interested   bool
	peer_choking    bool
	peer_interested bool
}

// ExtractPeers will take the peers returned from a tracker and return the parsed peer list
func ExtractPeers(trackerResp map[string]interface{}) ([]string, error) {
	var peerList []string
	var err error

	if peers, ok := trackerResp["peers"].(string); ok {
		peerList, err = ParseCompactPeers([]byte(peers))
	} else if peers, ok := trackerResp["peers"].([]interface{}); ok {
		peerList, err = ParseDictionaryPeers(peers)
	}
	return peerList, err
}

// ParseCompactPeers will parse the compact peer list when we get that format
func ParseCompactPeers(peers []byte) ([]string, error) {
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

// ParseDictionaryPeers will return the peerlist when trackers provide us a standard peerlist in map format
func ParseDictionaryPeers(peers []interface{}) ([]string, error) {
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

// CreatePeerConn will return a default peer connection where both sides are choked
func CreatePeerConn() Peer {
	return Peer{PeerState{true, false, true, false}}
}

// CanDownload will return true if we can download from this peer
func CanDownload(peer Peer) bool {
	return peer.peer_state.am_interested && peer.peer_state.peer_choking
}

// CanUpload will return true if we can upload to this peer
func CanUpload(peer Peer) bool {
	return peer.peer_state.peer_interested && peer.peer_state.am_choking
}

// UpdatePeerState make sure to update every chance we get
func UpdatePeerState(peer *Peer, receivedInterested bool, receivedChoking bool) {
	// Update the peer's state based on received messages
	peer.peer_state.peer_interested = receivedInterested
	peer.peer_state.peer_choking = receivedChoking

	// Make decisions on whether we should choke or unchoke
	// Example: Choking is conditional based on whether the peer is interested
	if peer.peer_state.peer_interested {
		peer.peer_state.am_choking = false // We unchoke if the peer is interested
	} else {
		peer.peer_state.am_choking = true // We choke if the peer is not interested
	}

	// Further logic can be added to make the peer interested or disinterested based on available blocks
}

func createHandshake(infohash []byte, peer_id []byte) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(PROTOCOL_LENGTH)) // integers are in big endian
	pstrlen := buf
	pstr := []byte(PROTOCOL_STRING)
	reserved := []byte(RESERVED_BITS)

	handshake := append(pstrlen, pstr...)
	handshake = append(handshake, reserved...)
	handshake = append(handshake, infohash...)
	handshake = append(handshake, peer_id...)

	return handshake
}

// HandlePeerConnection will handle individual peer connections
func HandlePeerConnection(infoHash []byte, peerID []byte, peerAddress string) {
	// Extract IP and port from the peer address
	peerIP, peerPort, err := net.SplitHostPort(peerAddress)
	if err != nil {
		log.Printf("Error parsing peer address %s: %v", peerAddress, err)
		return
	}

	// Establish TCP connection with the peer
	conn, err := net.DialTimeout("tcp", peerAddress, 10*time.Second) // Timeout in case peer doesn't respond
	if err != nil {
		log.Printf("Error connecting to peer %s: %v", peerAddress, err)
		return
	}
	defer conn.Close()

	log.Printf("Connected to peer %s:%s", peerIP, peerPort)

	// Send handshake
	handshake := createHandshake(infoHash, peerID)
	_, err = conn.Write(handshake)
	if err != nil {
		log.Printf("Error sending handshake to peer %s: %v", peerAddress, err)
		return
	}

	// Read the handshake response from the peer
	response := make([]byte, 68) // 1 byte length prefix + 19 bytes protocol + reserved + infohash + peer id
	_, err = conn.Read(response)
	if err != nil {
		log.Printf("Error reading handshake response from peer %s: %v", peerAddress, err)
		return
	}

	// Process the response (check protocol, infohash, peer id match)
	if string(response[1:20]) != PROTOCOL_STRING {
		log.Printf("Invalid protocol from peer %s", peerAddress)
		return
	}
	if string(response[28:48]) != string(infoHash) {
		log.Printf("Invalid infohash from peer %s", peerAddress)
		return
	}

	log.Printf("Handshake successful with peer %s", peerAddress)

	// After handshake, you can start sending/receiving messages according to the BitTorrent protocol
	// Handle peer messages such as interested, unchoked, request, etc.
	// Here you can implement the logic for requesting pieces or uploading them.
}

// Send an "interested" message to the peer to let them know we're interested in downloading pieces
func sendInterestedMessage(conn net.Conn) error {
	message := []byte{0x00, 0x00, 0x00, 0x01, 0x02} // The "Interested" message is just a 1-byte signal
	_, err := conn.Write(message)
	return err
}

// Read peer's "choking" or "unchoking" message
func readChokeUnchoke(conn net.Conn) (bool, error) {
	message := make([]byte, 5) // "choke" and "unchoke" messages are 5 bytes
	_, err := conn.Read(message)
	if err != nil {
		return false, err
	}
	// Check if peer is choking or not
	if message[4] == 0x00 {
		return true, nil // Peer is choking us
	}
	return false, nil // Peer is unchoking us
}
