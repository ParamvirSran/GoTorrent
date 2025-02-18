package peers

import (
	"fmt"
)

// createHandshake will return the intial handshake to send to a peer when connecting
func createHandshake(infohash []byte, clientID []byte) ([]byte, error) {
	const protocolString = "BitTorrent protocol"
	const protocolLength = 19
	var reservedBits = []byte{0, 0, 0, 0, 0, 0, 0, 0}

	if len(infohash) != 20 {
		return nil, fmt.Errorf("infohash length is %d, expected 20", len(infohash))
	} else if len(clientID) != 20 {
		return nil, fmt.Errorf("peerID length is %d, expected 20", len(clientID))
	}

	handshake := make([]byte, 68)
	handshake[0] = byte(protocolLength)
	copy(handshake[1:], []byte(protocolString))
	copy(handshake[20:], reservedBits)
	copy(handshake[28:], infohash)
	copy(handshake[48:], clientID)

	return handshake, nil
}
