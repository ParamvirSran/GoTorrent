package peers

import (
	"fmt"
	"log"
)

// createHandshake creates the initial handshake message to send to a peer when connecting.
// It follows the BitTorrent protocol specification.
func createHandshake(infoHash []byte, clientID []byte) ([]byte, error) {
	var reservedBits = []byte{0, 0, 0, 0, 0, 0, 0, 0} // 8 reserved bytes
	const (
		protocolString = "BitTorrent protocol"
		protocolLength = byte(len(protocolString))
	)
	var handshakeSize = 1 + len(protocolString) + len(reservedBits) + 20 + 20

	// Validate input lengths
	if len(infoHash) != 20 {
		return nil, fmt.Errorf("infoHash length is %d, expected 20", len(infoHash))
	}
	if len(clientID) != 20 {
		return nil, fmt.Errorf("clientID length is %d, expected 20", len(clientID))
	}

	// Create the handshake message
	handshake := make([]byte, handshakeSize)
	offset := 0

	// Set protocol length
	handshake[offset] = protocolLength
	offset++

	// Copy protocol string
	copy(handshake[offset:], protocolString)
	offset += len(protocolString)

	// Copy reserved bits
	copy(handshake[offset:], reservedBits)
	offset += len(reservedBits)

	// Copy info hash
	copy(handshake[offset:], infoHash)
	offset += len(infoHash)

	// Copy client ID
	copy(handshake[offset:], clientID)
	log.Println("handshake message: ", handshake)
	return handshake, nil
}
