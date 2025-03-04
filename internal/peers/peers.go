package peers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
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
		return fmt.Errorf("Failed to dial %s: %v", peer.address, err)
	}
	defer conn.Close()

	// Send the handshake
	if _, err := conn.Write(handshake); err != nil {
		return fmt.Errorf("Error writing handshake: %v", err)
	}

	// Read the handshake response
	response := make([]byte, 68)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("Failed to read handshake response: %v", err)
	}

	// Validate handshake
	if err := ValidateHandshakeResponse(peer, response[:n], [20]byte(infoHash)); err != nil {
		return fmt.Errorf("Invalid handshake response: %v", err)
	}

	// Message handling loop
	for {
		if _, err := conn.Write([]byte{0}); err != nil {
			log.Println("error sending keep-alive: ", err)
		}

		select {

		case <-ctx.Done():
			log.Printf("Disconnecting from peer: %s", peer.address)
			return nil

		default:
			conn.SetReadDeadline(time.Now().Add(120 * time.Second))
			msg, err := readLengthPrefixedMessage(conn)
			if err != nil {
				if err == io.EOF {
					continue
				} else {
					log.Printf("error reading message: %v", err)
					continue
				}
			}
			log.Printf("message received: %X", msg)
			message, err := ParseMessage(msg)

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
			case MsgHave:
				log.Println("peer has these pieces", message.Payload)
			default:
			}
		}
	}
}

func readLengthPrefixedMessage(conn net.Conn) ([]byte, error) {
	var length uint32
	err := binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		log.Println("received keep-alive message")
		return nil, nil
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// isValidPeerAddress checks if peers address is valid
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
