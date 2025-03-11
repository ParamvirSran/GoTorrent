package peers

import (
	"bytes"
	"fmt"
	"log"
)

const (
	_protocolString = "BitTorrent protocol"
	_protocolLength = byte(len(_protocolString))
)

// Handshake represents the handshake message which starts peer communication in the Peer Wire Protocol
type Handshake struct {
	protocolStringLength byte
	protocolString       string
	reserved             [8]byte
	infohash             [20]byte
	peerID               [20]byte
}

// CreateHandshake creates the initial handshake message to send to a peer when connecting
func CreateHandshake(infoHash []byte, clientID []byte) ([]byte, error) {
	if len(infoHash) != 20 {
		return nil, fmt.Errorf("infoHash length is %d, expected 20", len(infoHash))
	}
	if len(clientID) != 20 {
		return nil, fmt.Errorf("clientID length is %d, expected 20", len(clientID))
	}

	// Create the handshake message
	handshake := &Handshake{
		protocolStringLength: _protocolLength,
		protocolString:       _protocolString,
		reserved:             [8]byte{}, // Reserved bytes are all zero
		infohash:             [20]byte(infoHash),
		peerID:               [20]byte(clientID),
	}

	// Serialize the handshake into a byte slice
	return handshake.serialize(), nil
}

// ValidateHandshakeResponse will check if received handshake is valid
func ValidateHandshakeResponse(peerID string, response []byte, expectedInfoHash [20]byte) error {
	log.Printf("received in handshake peerid (%s) - expected peerid (%s)", string(response[48:]), peerID)

	if len(response) < 68 {
		return fmt.Errorf("invalid handshake response length: %d", len(response))
	}

	protocolStr := string(response[1:20])
	if protocolStr != _protocolString {
		return fmt.Errorf("invalid protocol string: %s", protocolStr)
	}

	var infoHash [20]byte
	copy(infoHash[:], response[28:48])
	if !bytes.Equal(infoHash[:], expectedInfoHash[:]) {
		return fmt.Errorf("invalid info hash: %x", infoHash)
	}

	return nil
}

// serialize serializes the handshake into a byte slice
func (h *Handshake) serialize() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(h.protocolStringLength)
	buf.WriteString(h.protocolString)
	buf.Write(h.reserved[:])
	buf.Write(h.infohash[:])
	buf.Write(h.peerID[:])
	return buf.Bytes()
}
