package peers

import "fmt"

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
	return []byte{0000}
}
