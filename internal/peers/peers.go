package peers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

const PROTOCOL_STRING = "BitTorrent protocol"
const PROTOCOL_LENGTH = len(PROTOCOL_STRING)
const RESERVED_BITS = "\x00\x00\x00\x00\x00\x00\x00\x00"

type Peer struct {
	peer_state PeerState
}

type PeerState struct {
	am_choking      bool
	am_interested   bool
	peer_choking    bool
	peer_interested bool
}

// CreatePeerConn will return a default peer connection where both sides are choked
func CreatePeerConn() Peer {
	return Peer{PeerState{true, false, true, false}}
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

// UpdatePeerState make sure to update every chance we get
func UpdatePeerState(peer *Peer, receivedInterested bool, receivedChoking bool) {
	// Update the peer's state based on received messages
	peer.peer_state.peer_interested = receivedInterested
	peer.peer_state.peer_choking = receivedChoking

	if peer.peer_state.peer_interested {
		peer.peer_state.am_choking = false // Unchoke if peer is interested
	} else {
		peer.peer_state.am_choking = true // Choke if peer is not interested
	}

	// Log state changes for debugging
	log.Printf("Updated state for peer: %v", peer.peer_state)
}

// CanDownload will return true if we can download from this peer
func CanDownload(peer Peer) bool {
	return peer.peer_state.am_interested && peer.peer_state.peer_choking
}

// CanUpload will return true if we can upload to this peer
func CanUpload(peer Peer) bool {
	return peer.peer_state.peer_interested && peer.peer_state.am_choking
}

func decodeInteger(data []byte) int32 {
	var value int32
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &value)
	if err != nil {
		fmt.Println("Error decoding integer:", err)
	}
	return value
}

func encodeInteger(value int32) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, value)
	if err != nil {
		fmt.Println("Error encoding integer:", err)
	}
	return buf.Bytes()
}

// HandlePeerConnection handles individual peer connections
func HandlePeerConnection(infoHash []byte, peerID []byte, peerAddress string) error {
	// Extract IP and port from the peer address
	peerIP, peerPort, err := net.SplitHostPort(peerAddress)
	if err != nil {
		log.Printf("Error parsing peer address %s: %v", peerAddress, err)
		return err
	}

	log.Printf("Attempting connection to peer %s:%s", peerIP, peerPort)

	// Establish TCP connection with the peer
	conn, err := net.DialTimeout("tcp", peerAddress, 10*time.Second)
	if err != nil {
		log.Printf("Error connecting to peer %s: %v", peerAddress, err)
		return err
	}
	defer conn.Close()

	log.Printf("Connected to peer %s:%s", peerIP, peerPort)

	// Send handshake
	handshake, err := createHandshake(infoHash, peerID)
	if err != nil {
		log.Printf("Error creating handshake: %v", err)
		return err
	}
	_, err = conn.Write(handshake)
	if err != nil {
		log.Printf("Error sending handshake to peer %s: %v", peerAddress, err)
		return err
	}

	// Read the handshake response from the peer
	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		log.Printf("Error reading handshake response from peer %s: %v", peerAddress, err)
		return err
	}

	// Process the response (check protocol, infohash, peer id match)
	if string(response[1:20]) != PROTOCOL_STRING {
		log.Printf("Invalid protocol from peer %s", peerAddress)
		return err
	}
	if string(response[28:48]) != string(infoHash) {
		log.Printf("Invalid infohash from peer %s", peerAddress)
		return err
	}

	log.Printf("Handshake successful with peer %s", peerAddress)
	return nil
}
