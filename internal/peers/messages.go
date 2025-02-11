package peers

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type KeepAliveMessage struct {
	Length uint32
}

func (m *KeepAliveMessage) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, m.Length)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// createHandshake will return the intial handshake to send to a peer when connecting
func createHandshake(infohash []byte, peerID []byte) ([]byte, error) {
	const protocolString = "BitTorrent protocol"
	const protocolLength = 19
	var reservedBits = []byte{0, 0, 0, 0, 0, 0, 0, 0}

	if len(infohash) != 20 {
		return nil, fmt.Errorf("infohash length is %d, expected 20", len(infohash))
	}
	if len(peerID) != 20 {
		return nil, fmt.Errorf("peerID length is %d, expected 20", len(peerID))
	}

	handshake := make([]byte, 68)

	handshake[0] = byte(protocolLength)
	copy(handshake[1:], []byte(protocolString))
	copy(handshake[20:], reservedBits)
	copy(handshake[28:], infohash)
	copy(handshake[48:], peerID)

	return handshake, nil
}
