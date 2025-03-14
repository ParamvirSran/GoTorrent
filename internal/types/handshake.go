package types

import (
	"bytes"
	"fmt"
	"log"
)

// SerializeHandshake turns the handshake into a byte slice to send over the wire
func (h *Handshake) SerializeHandshake() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(h.ProtocolStringLength)
	buf.WriteString(h.ProtocolString)
	buf.Write(h.Reserved[:])
	buf.Write(h.Infohash[:])
	buf.Write(h.PeerID[:])

	return buf.Bytes()
}

// ValidateHandshakeResponse will check if received handshake is valid
func ValidateHandshakeResponse(peerID string, response []byte, expectedInfoHash [20]byte) error {
	log.Printf("received in handshake peerid (%x) - expected peerid (%x)", response[48:], peerID)

	if len(response) < 68 {
		return fmt.Errorf("invalid handshake response length: %d", len(response))
	}

	protocolStr := string(response[1:20])
	if protocolStr != ProtocolString {
		return fmt.Errorf("invalid protocol string: %s", protocolStr)
	}

	var infoHash [20]byte
	copy(infoHash[:], response[28:48])
	if !bytes.Equal(infoHash[:], expectedInfoHash[:]) {
		return fmt.Errorf("invalid info hash: %x", infoHash)
	}

	return nil
}
