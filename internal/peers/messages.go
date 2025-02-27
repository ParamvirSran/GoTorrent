package peers

import (
	"bytes"
	"encoding/binary"
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

// // Serialize will prepare a message to be sent over the wire
// func (m *Message) Serialize() []byte {
// 	length := uint32(len(m.Payload) + 1)
// 	buf := make([]byte, 4+length)
// 	binary.BigEndian.PutUint32(buf[0:4], length)
// 	buf[4] = byte(m.ID)
// 	copy(buf[5:], m.Payload)
// 	return buf
// }

func ParseMessage(msgBuf []byte) (Message, error) {
	log.Println(msgBuf)
	// Extract the length prefix (first 4 bytes)
	length_prefix := binary.BigEndian.Uint32(msgBuf[:4])

	// Ensure the length prefix is within valid range (it should not exceed the total message size)
	if length_prefix > uint32(len(msgBuf)) {
		return Message{}, fmt.Errorf("invalid length prefix: %d", length_prefix)
	}

	// If length is 0, it's a keep-alive message
	if length_prefix == 0 {
		return Message{}, nil
	} else {
		// Extract the message ID (next byte after the length prefix)
		message_id := Message_ID(msgBuf[4])

		// Extract the payload (after the ID, accounting for the length prefix and the ID)
		payload := msgBuf[5 : 5+length_prefix-1]

		// Return the parsed message
		message := Message{
			ID:      message_id,
			Payload: payload,
		}
		log.Println("parsed message:", message)
		return message, nil
	}
}

func ValidateHandshakeResponse(response []byte, expectedInfoHash [20]byte) error {
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

// CreateHandshake creates the initial handshake message to send to a peer when connecting.
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
