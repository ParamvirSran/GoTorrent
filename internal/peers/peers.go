package peers

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/ParamvirSran/GoTorrent/internal/common"
)

const (
	HandshakeResponseLength = 68
	KeepAliveInterval       = 30 * time.Second
	PeerTimeout             = 120 * time.Second
	BlockSize               = 16384
)

// Peer represents a connected peer
type Peer struct {
	peerID    string
	address   string
	peerState PeerState
}

// PeerState represents the connection state with a peer
type PeerState struct {
	amChoking      bool
	amInterested   bool
	peerChoking    bool
	peerInterested bool
}

// HandlePeerConnection manages a single peer connection
func HandlePeerConnection(pm *common.PieceManager, ctx context.Context, peerID string, infoHash, clientID []byte, peerAddress string) error {
	peerContext, peerCancel := context.WithCancel(ctx)
	defer peerCancel()

	peer := createPeer(peerID, peerAddress)
	handshake, err := CreateHandshake(infoHash, clientID)
	if err != nil {
		return fmt.Errorf("error creating handshake: %v", err)
	}

	conn, err := connectToPeer(peerContext, peer.address)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := sendHandshake(conn, handshake); err != nil {
		return err
	}

	if err := receiveHandshakeResponse(peerID, conn, infoHash); err != nil {
		return err
	}

	stopKeepAlive := startKeepAlive(peerContext, conn)
	defer stopKeepAlive()

	return processMessages(peerContext, conn, peer, pm)
}

// connectToPeer establishes a connection to the peer
func connectToPeer(ctx context.Context, address string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", address, err)
	}
	return conn, nil
}

// sendHandshake sends a handshake to the peer
func sendHandshake(conn net.Conn, handshake []byte) error {
	if _, err := conn.Write(handshake); err != nil {
		return fmt.Errorf("error writing handshake: %v", err)
	}
	return nil
}

// receiveHandshakeResponse reads and validates the handshake response from the peer
func receiveHandshakeResponse(peerID string, conn net.Conn, infoHash []byte) error {
	response := make([]byte, HandshakeResponseLength)
	conn.SetReadDeadline(time.Now().Add(PeerTimeout))
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %v", err)
	}

	if err := ValidateHandshakeResponse(peerID, response[:n], [20]byte(infoHash)); err != nil {
		return fmt.Errorf("invalid handshake response: %v", err)
	}
	return nil
}

// startKeepAlive starts a goroutine to send keep-alive messages to the peer
func startKeepAlive(ctx context.Context, conn net.Conn) func() {
	ctx, cancel := context.WithCancel(ctx)
	var once sync.Once
	go func() {
		ticker := time.NewTicker(KeepAliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := conn.Write(KeepAliveMessage()); err != nil {
					log.Printf("Error sending keep-alive to peer %s: %v", conn.RemoteAddr(), err)
					once.Do(cancel)
					return
				}
			}
		}
	}()
	return func() { once.Do(cancel) }
}

// processMessages processes incoming messages from the peer
func processMessages(ctx context.Context, conn net.Conn, peer *Peer, pm *common.PieceManager) error {
	lastActivity := time.Now()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Disconnecting from peer: %s", peer.address)
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(PeerTimeout))
			msg, err := ReadMessage(conn)
			if err != nil {
				if errors.Is(err, io.EOF) {
					continue
				}
				return fmt.Errorf("error reading message: %v", err)
			}

			if msg.ID != nil {
				handleMessage(conn, ctx, pm, peer, msg)
				lastActivity = time.Now()
			}

			if time.Since(lastActivity) > PeerTimeout {
				return fmt.Errorf("peer %s timed out", peer.address)
			}
		}
	}
}

// handleMessage handles a received message
func handleMessage(conn net.Conn, ctx context.Context, pm *common.PieceManager, peer *Peer, msg Message) {
	switch *msg.ID {
	case MsgChoke:
		peer.peerState.peerChoking = true
	case MsgUnchoke:
		peer.peerState.peerChoking = false
	case MsgInterested:
		peer.peerState.peerInterested = true
	case MsgNotInterested:
		peer.peerState.peerInterested = false
	case MsgHave:
		pieceIndex := binary.BigEndian.Uint32(msg.Payload)
		log.Printf("%s - Received HAVE message for piece %d", peer.address, pieceIndex)
		if pm.ClaimPiece(int(pieceIndex)) {
			go worker(peer, ctx, pm, pieceIndex, conn)
		}
	case MsgBitfield:
		log.Printf("%s - Received BITFIELD message: %x", peer.address, msg.Payload)
	case MsgRequest:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		length := binary.BigEndian.Uint32(msg.Payload[8:12])
		log.Printf("%s - Received REQUEST message for index %d, begin %d, length %d", peer.address, index, begin, length)
	case MsgPiece:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		block := msg.Payload[8:]
		log.Printf("%s - Received PIECE message for index %d, begin %d, block length %d", peer.address, index, begin, len(block))
	case MsgCancel:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		length := binary.BigEndian.Uint32(msg.Payload[8:12])
		log.Printf("%s - Received CANCEL message for index %d, begin %d, length %d", peer.address, index, begin, length)
	case MsgPort:
		port := binary.BigEndian.Uint16(msg.Payload)
		log.Printf("%s - Received PORT message with port %d", peer.address, port)
	default:
		log.Printf("%s - Received unknown message ID %d", peer.address, *msg.ID)
	}
}

// worker downloads a piece from the peer
func worker(peer *Peer, ctx context.Context, pm *common.PieceManager, index uint32, conn net.Conn) {
	var offset uint32 = 0
	piece := make([]byte, pm.PieceSize)

	for offset < uint32(pm.PieceSize) {
		select {
		case <-ctx.Done():
			log.Printf("worker: context finished early: piece (%d) from peer (%s)", index, conn.RemoteAddr())
			return
		default:
			// Calculate the block size (usually 16 KB, but smaller for the last block)
			blockSize := uint32(BlockSize)
			if offset+blockSize > uint32(pm.PieceSize) {
				blockSize = uint32(pm.PieceSize) - offset
			}

			// Send a request for the block
			// might need to send interested message and a unchoke meesage
			request := RequestMessage(index, offset, blockSize)
			if _, err := conn.Write(request); err != nil {
				log.Printf("worker: Error sending request for piece %d, offset %d: %v", index, offset, err)
				return
			}

			// Read the block response
			msg, err := ReadMessage(conn)
			if err != nil {
				log.Printf("worker: Error reading block for piece %d, offset %d: %v", index, offset, err)
				return
			}

			// Verify the block response
			if *msg.ID != MsgPiece {
				log.Printf("worker: Expected PIECE message, got message ID %d", *msg.ID)
				break
			}

			// Copy the block into the piece
			receivedIndex := binary.BigEndian.Uint32(msg.Payload[0:4])
			receivedBegin := binary.BigEndian.Uint32(msg.Payload[4:8])
			block := msg.Payload[8:]

			if receivedIndex != index || receivedBegin != offset {
				log.Printf("worker: Mismatch in received block: expected index %d, offset %d; got index %d, offset %d",
					index, offset, receivedIndex, receivedBegin)
				return
			}

			copy(piece[offset:offset+blockSize], block)
			offset += blockSize
		}
	}

	// Mark the piece as downloaded and verify it
	pm.MarkPieceDownloaded(int(index), piece)
	if err := pm.VerifyPiece(int(index)); err != nil {
		log.Printf("worker: Verification failed for piece %d: %v", index, err)
		pm.RequeuePiece(int(index))
	} else {
		log.Printf("worker: Successfully downloaded and verified piece %d from peer %s", index, conn.RemoteAddr())
	}
}

// ReadMessage reads a message from a connection
func ReadMessage(conn net.Conn) (Message, error) {
	var length uint32
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		return Message{}, err
	}

	if length == 0 {
		return Message{ID: nil, Payload: nil}, nil // Keep-alive message
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return Message{}, err
	}

	messageId := MessageID(buf[0])
	return Message{
		ID:      &messageId,
		Payload: buf[1:],
	}, nil
}

// createPeer initializes a new peer object
func createPeer(peerID, address string) *Peer {
	return &Peer{
		peerID:  peerID,
		address: address,
		peerState: PeerState{
			amChoking:      true,
			amInterested:   false,
			peerChoking:    true,
			peerInterested: false,
		},
	}
}
