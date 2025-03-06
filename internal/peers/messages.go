package peers

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// MessageID represents different types of messages in the Peer Wire Protocol
type MessageID byte

const (
	MsgChoke         MessageID = 0
	MsgUnchoke       MessageID = 1
	MsgInterested    MessageID = 2
	MsgNotInterested MessageID = 3
	MsgHave          MessageID = 4
	MsgBitfield      MessageID = 5
	MsgRequest       MessageID = 6
	MsgPiece         MessageID = 7
	MsgCancel        MessageID = 8
	MsgPort          MessageID = 9
	MsgKeepAlive     MessageID = 255
)

// Message represents a parsed message
type Message struct {
	ID      *MessageID // nil for keep-alive message
	Payload []byte
}

// KeepAliveMessage returns KEEP-ALIVE message
func KeepAliveMessage() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(0))
	return buf.Bytes()
}

// FixedLengthMessage creates a message with no payload
func FixedLengthMessage(id MessageID) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(1))
	buf.WriteByte(byte(id))
	return buf.Bytes()
}

// HaveMessage creates a HAVE message with a piece index
func HaveMessage(pieceIndex uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(5))
	buf.WriteByte(byte(MsgHave))
	binary.Write(buf, binary.BigEndian, pieceIndex)
	return buf.Bytes()
}

// RequestMessage creates a REQUEST message
func RequestMessage(index, begin, length uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(13))
	buf.WriteByte(byte(MsgRequest))
	binary.Write(buf, binary.BigEndian, index)
	binary.Write(buf, binary.BigEndian, begin)
	binary.Write(buf, binary.BigEndian, length)
	return buf.Bytes()
}

// CancelMessage creates a CANCEL message
func CancelMessage(index, begin, length uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(13))
	buf.WriteByte(byte(MsgCancel))
	binary.Write(buf, binary.BigEndian, index)
	binary.Write(buf, binary.BigEndian, begin)
	binary.Write(buf, binary.BigEndian, length)
	return buf.Bytes()
}

// PieceMessage creates a PIECE message
func PieceMessage(index, begin uint32, block []byte) []byte {
	buf := new(bytes.Buffer)
	length := uint32(9 + len(block))
	binary.Write(buf, binary.BigEndian, length)
	buf.WriteByte(byte(MsgPiece))
	binary.Write(buf, binary.BigEndian, index)
	binary.Write(buf, binary.BigEndian, begin)
	buf.Write(block)
	return buf.Bytes()
}

// ParseMessage parses a length-prefixed message
func ParseMessage(buf []byte) (Message, error) {
	if len(buf) < 4 {
		return Message{}, fmt.Errorf("buffer too short for length prefix")
	}

	length := binary.BigEndian.Uint32(buf[:4])
	if length == 0 {
		// Keep-alive message (no ID, no payload)
		return Message{ID: nil, Payload: nil}, nil
	}

	if len(buf) < int(4+length) {
		return Message{}, fmt.Errorf("buffer too short for declared length")
	}

	msg := Message{
		ID:      new(MessageID),
		Payload: buf[5 : 4+length], // extract payload after ID
	}
	*msg.ID = MessageID(buf[4]) // first byte after length is ID

	return msg, nil
}
