package peers

import (
	"bytes"
	"encoding/binary"

	"github.com/ParamvirSran/GoTorrent/internal/types"
)

// KeepAliveMessage returns KEEP-ALIVE message
func KeepAliveMessage() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(0))
	return buf.Bytes()
}

// FixedLengthMessage creates a message with no payload
func FixedLengthMessage(id types.MessageID) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(1))
	buf.WriteByte(byte(id))
	return buf.Bytes()
}

// HaveMessage creates a HAVE message with a piece index
func HaveMessage(pieceIndex uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(5))
	buf.WriteByte(byte(types.MsgHave))
	binary.Write(buf, binary.BigEndian, pieceIndex)
	return buf.Bytes()
}

// RequestMessage creates a REQUEST message
func RequestMessage(index, begin, length uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(13))
	buf.WriteByte(byte(types.MsgRequest))
	binary.Write(buf, binary.BigEndian, index)
	binary.Write(buf, binary.BigEndian, begin)
	binary.Write(buf, binary.BigEndian, length)
	return buf.Bytes()
}

// CancelMessage creates a CANCEL message
func CancelMessage(index, begin, length uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(13))
	buf.WriteByte(byte(types.MsgCancel))
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
	buf.WriteByte(byte(types.MsgPiece))
	binary.Write(buf, binary.BigEndian, index)
	binary.Write(buf, binary.BigEndian, begin)
	buf.Write(block)
	return buf.Bytes()
}
