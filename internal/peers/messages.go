package peers

import (
	"bytes"
	"fmt"
	"log"
)

type Message_ID int

const (
	MsgKeepAlive Message_ID = iota
	MsgChoke
	MsgUnchoke
	MsgInterested
	MsgNotInterested
	MsgHave
	MsgBitfield
	MsgRequest
	MsgPiece
	MsgCancel
	MsgPort
)

const (
	PROTOCOL_STRING = "BitTorrent protocol"
	PROTOCOL_LENGTH = byte(len(PROTOCOL_STRING))
)

// Handshake represents the handshake message which starts peer communication in the Peer Wire Protocol
type Handshake struct {
	Protocol_String_Len byte
	Protocol_String     string
	Reserved            [8]byte
	Info_Hash           [20]byte
	Peer_ID             [20]byte
}

// Message represents a message sent or received in the Peer Wire Protocol
type Message struct {
	ID      Message_ID
	Payload []byte
}

func ParseMessage(buf []byte) (Message, error) {
	message := Message{}
	message.ID = Message_ID(buf[0])
	message.Payload = buf[1:]
	log.Println("message is", message)
	return message, nil
}

// ValidateHandshakeResponse will check if received handshake is valid
func ValidateHandshakeResponse(peer *Peer, response []byte, expectedInfoHash [20]byte) error {
	// if peer.peer_id != "" && peer.peer_id != string(response[48:68]) {
	// 	return fmt.Errorf("invalid handshake peer id that doesn't match the recorded peer id: peer id (%s), rec'd id (%s)", peer.peer_id, string(response[48:68]))
	// }

	if len(response) < 68 {
		return fmt.Errorf("invalid handshake response length: %d", len(response))
	}

	protocolStr := string(response[1:20])
	if protocolStr != PROTOCOL_STRING {
		return fmt.Errorf("invalid protocol string: %s", protocolStr)
	}

	var infoHash [20]byte
	copy(infoHash[:], response[28:48])
	if !bytes.Equal(infoHash[:], expectedInfoHash[:]) {
		return fmt.Errorf("invalid info hash: %x", infoHash)
	}

	return nil
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
		Protocol_String_Len: PROTOCOL_LENGTH,
		Protocol_String:     PROTOCOL_STRING,
		Reserved:            [8]byte{}, // Reserved bytes are all zero
		Info_Hash:           [20]byte(infoHash),
		Peer_ID:             [20]byte(clientID),
	}

	// Serialize the handshake into a byte slice
	return handshake.Serialize(), nil
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

// // DeserializeHandshake deserializes a handshake from a byte slice
// func DeserializeHandshake(data []byte) (*Handshake, error) {
// 	if len(data) < 49 {
// 		return nil, fmt.Errorf("handshake too short")
// 	}
//
// 	h := &Handshake{
// 		Protocol_String_Len: data[0],
// 		Protocol_String:     string(data[1:20]),
// 	}
// 	copy(h.Reserved[:], data[20:28])
// 	copy(h.Info_Hash[:], data[28:48])
// 	copy(h.Peer_ID[:], data[48:68])
//
// 	return h, nil
// }
