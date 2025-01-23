package peers

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

func createHandshake(infohash []byte, peer_id []byte) ([]byte, error) {
	// Protocol length should be 19 for "BitTorrent protocol"
	pstrlen := byte(PROTOCOL_LENGTH) // 19
	pstr := []byte(PROTOCOL_STRING)
	reserved := []byte(RESERVED_BITS)

	// Start constructing the handshake
	handshake := append([]byte{pstrlen}, pstr...) // Append the protocol length as a byte
	handshake = append(handshake, reserved...)
	handshake = append(handshake, infohash...)
	handshake = append(handshake, peer_id...)

	if len(handshake) != 68 {
		return nil, fmt.Errorf("handshake is length %d, expected 68", len(handshake))
	}
	return handshake, nil
}

func createKeepAlive() []byte {
	return []byte{0, 0, 0, 0} // Proper keep-alive message with 4 bytes
}

// Message Types (BitTorrent Protocol)
const (
	MSG_TYPE_KEEP_ALIVE = iota
	MSG_TYPE_CHOKE
	MSG_TYPE_UNCHOKE
	MSG_TYPE_INTERESTED
	MSG_TYPE_NOT_INTERESTED
	MSG_TYPE_HAVE
	MSG_TYPE_BITFIELD
	MSG_TYPE_REQUEST
	MSG_TYPE_PIECE
	MSG_TYPE_CANCEL
	MSG_TYPE_PORT
)

// sendChoke sends a choke message
func sendChoke(conn net.Conn) error {
	msg := make([]byte, 5)
	binary.BigEndian.PutUint32(msg[0:4], 1) // length (1 byte for choke)
	msg[4] = MSG_TYPE_CHOKE                 // message type
	_, err := conn.Write(msg)
	if err != nil {
		return fmt.Errorf("error sending choke message: %v", err)
	}
	return nil
}

// sendUnchoke sends an unchoke message
func sendUnchoke(conn net.Conn) error {
	msg := make([]byte, 5)
	binary.BigEndian.PutUint32(msg[0:4], 1) // length (1 byte for unchoke)
	msg[4] = MSG_TYPE_UNCHOKE               // message type
	_, err := conn.Write(msg)
	if err != nil {
		return fmt.Errorf("error sending unchoke message: %v", err)
	}
	return nil
}

// sendInterested sends an interested message
func sendInterested(conn net.Conn) error {
	msg := make([]byte, 5)
	binary.BigEndian.PutUint32(msg[0:4], 1) // length (1 byte for interested)
	msg[4] = MSG_TYPE_INTERESTED            // message type
	_, err := conn.Write(msg)
	if err != nil {
		return fmt.Errorf("error sending interested message: %v", err)
	}
	return nil
}

func handleMessages(conn net.Conn, peer *Peer) {
	buf := make([]byte, 1024) // Initial buffer
	for {
		// Read data from the peer
		_, err := conn.Read(buf)
		if err != nil {
			log.Printf("Error reading from peer: %v", err)
			break
		}

		// Process the message length (first 4 bytes)
		msgLength := binary.BigEndian.Uint32(buf[:4])
		if msgLength == 0 {
			// Keep-Alive message, do nothing
			continue
		}

		buf = make([]byte, msgLength) // Dynamically resize buffer based on message length
		_, err = conn.Read(buf)       // Read the full message
		if err != nil {
			log.Printf("Error reading full message: %v", err)
			break
		}

		msgType := buf[4] // Message type (5th byte onwards)
		switch msgType {
		case MSG_TYPE_CHOKE:
			peer.peer_state.peer_choking = true
			log.Println("Received choke message")
		case MSG_TYPE_UNCHOKE:
			peer.peer_state.peer_choking = false
			log.Println("Received unchoke message")
		case MSG_TYPE_INTERESTED:
			peer.peer_state.peer_interested = true
			log.Println("Received interested message")
		case MSG_TYPE_NOT_INTERESTED:
			peer.peer_state.peer_interested = false
			log.Println("Received not interested message")
		default:
			log.Printf("Received unknown message type: %d", msgType)
		}
	}
}
