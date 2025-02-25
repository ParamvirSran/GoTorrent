package peers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

type Message_ID byte

const (
	MsgChoke         Message_ID = 0
	MsgUnchoke       Message_ID = 1
	MsgInterested    Message_ID = 2
	MsgNotInterested Message_ID = 3
	MsgHave          Message_ID = 4
	MsgBitfield      Message_ID = 5
	MsgRequest       Message_ID = 6
	MsgPiece         Message_ID = 7
	MsgCancel        Message_ID = 8
)

type Message struct {
	ID      Message_ID
	Payload []byte
}

func (m *Message) Serialize() []byte {
	length := uint32(len(m.Payload) + 1)
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

const (
	PROTOCOL_STRING = "BitTorrent protocol"
	PROTOCOL_LENGTH = byte(len(PROTOCOL_STRING))
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

// Handshake represents the handshake message in the Peer Wire Protocol
type Handshake struct {
	Protocol_String_Len byte
	Protocol_String     string
	Reserved            [8]byte
	Info_Hash           [20]byte
	Peer_ID             [20]byte
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

// Serialize serializes the handshake into a byte slice
func (h *Handshake) Serialize() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(h.Protocol_String_Len)
	buf.WriteString(h.Protocol_String)
	buf.Write(h.Reserved[:])
	buf.Write(h.Info_Hash[:])
	buf.Write(h.Peer_ID[:])
	return buf.Bytes()
}

// DeserializeHandshake deserializes a handshake from a byte slice
func DeserializeHandshake(data []byte) (*Handshake, error) {
	if len(data) < 49 {
		return nil, fmt.Errorf("handshake too short")
	}

	h := &Handshake{
		Protocol_String_Len: data[0],
		Protocol_String:     string(data[1:20]),
	}
	copy(h.Reserved[:], data[20:28])
	copy(h.Info_Hash[:], data[28:48])
	copy(h.Peer_ID[:], data[48:68])

	return h, nil
}

// createHandshake creates the initial handshake message to send to a peer when connecting.
func createHandshake(infoHash []byte, clientID []byte) ([]byte, error) {
	if len(infoHash) != 20 {
		return nil, fmt.Errorf("infoHash length is %d, expected 20", len(infoHash))
	}
	if len(clientID) != 20 {
		return nil, fmt.Errorf("clientID length is %d, expected 20", len(clientID))
	}

	// Create the handshake message
	handshake := &Handshake{
		Protocol_String_Len: PROTOCOL_LENGTH,
		Protocol_String:     PROTOCOL_STRING,
		Reserved:            [8]byte{}, // Reserved bytes are all zero
		Info_Hash:           [20]byte(infoHash),
		Peer_ID:             [20]byte(clientID),
	}

	// Serialize the handshake into a byte slice
	return handshake.Serialize(), nil
}

func validateHandshakeResponse(response []byte, expectedInfoHash [20]byte) error {
	if len(response) < 68 {
		return fmt.Errorf("invalid handshake response length: %d", len(response))
	}

	// Check protocol string
	protocolStr := string(response[1:20])
	if protocolStr != PROTOCOL_STRING {
		return fmt.Errorf("invalid protocol string: %s", protocolStr)
	}

	// Check info hash
	var infoHash [20]byte
	copy(infoHash[:], response[28:48])
	if !bytes.Equal(infoHash[:], expectedInfoHash[:]) {
		return fmt.Errorf("invalid info hash: %x", infoHash)
	}

	return nil
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

// HandlePeerConnection handles individual peer connections
func HandlePeerConnection(peer_id string, info_hash []byte, client_id []byte, peer_address string) error {
	if !isValidPeerAddress(peer_address) {
		return fmt.Errorf("invalid peer address: %s", peer_address)
	}

	peer := CreatePeer(peer_id, peer_address)

	handshake, err := createHandshake(info_hash, client_id)
	if err != nil {
		return fmt.Errorf("error creating handshake: %v", err)
	}

	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to the peer
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
	log.Printf("Handshake response: %x", response[:n])

	// Validate the handshake response
	if err := validateHandshakeResponse(response[:n], [20]byte(info_hash)); err != nil {
		log.Printf("Invalid handshake response: %v", err)
		return err
	}
	log.Printf("Handshake successful with peer: %s", peer.address)

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
