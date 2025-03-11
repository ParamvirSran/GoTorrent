package peers

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/ParamvirSran/GoTorrent/internal/common"
	"io"
	"log"
	"net"
	"sync"
	"time"
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

	// Send handshake
	if err := sendHandshake(conn, handshake); err != nil {
		return err
	}

	// Read handshake response
	response := make([]byte, 68)
	if err := receiveHandshakeResponse(peerID, conn, response, infoHash); err != nil {
		return err
	}

	// Start keep-alive goroutine
	go startKeepAlive(peerContext, conn)

	// Message processing loop
	return processMessages(peerContext, conn, peer, pm, time.Now())
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
func receiveHandshakeResponse(peerID string, conn net.Conn, response []byte, infoHash []byte) error {
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
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
	ctx, cancel := context.WithCancel(ctx) // Create a cancel function
	var once sync.Once
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(30 * time.Second) // Send keep-alive every 30 sec
				if _, err := conn.Write([]byte{0}); err != nil {
					log.Println("Error sending keep-alive:", err)
					once.Do(func() { cancel() }) // Ensure cancel is called once
					return
				}
			}
		}
	}()
	return func() {
		once.Do(func() { cancel() }) // Ensure cleanup
	}
}

// processMessages processes incoming messages from the peer
func processMessages(ctx context.Context, conn net.Conn, peer *Peer, pm *common.PieceManager, lastKeepAlive time.Time) error {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Disconnecting from peer: %s", peer.address)
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(120 * time.Second))
			msg, err := ReadMessage(conn)
			if err != nil {
				if err == io.EOF {
					continue
				}
				log.Printf("Error reading message: %v", err)
				return err
			}

			if msg.ID != nil {
				handleMessage(conn, ctx, pm, peer, msg)
			}

			// Timeout check
			if time.Since(lastKeepAlive) > 120*time.Second {
				log.Printf("Peer %s timed out", peer.address)
				return nil
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
		log.Printf("%s - Received HAVE message for piece %d\n", peer.address, pieceIndex)
		if pm.ClaimPiece(int(pieceIndex)) {
			go Worker(ctx, pm, pieceIndex, conn)
		}
	case MsgBitfield:
		log.Printf("%s - Received BITFIELD message: %x", peer.address, msg.Payload)
	case MsgRequest:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		length := binary.BigEndian.Uint32(msg.Payload[8:12])
		log.Printf("%s - Received REQUEST message for index %d, begin %d, length %d\n", peer.address, index, begin, length)
	case MsgPiece:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		block := msg.Payload[8:]
		log.Printf("%s - Received PIECE message for index %d, begin %d, block length %d\n", peer.address, index, begin, len(block))
	case MsgCancel:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		length := binary.BigEndian.Uint32(msg.Payload[8:12])
		log.Printf("%s - Received CANCEL message for index %d, begin %d, length %d\n", peer.address, index, begin, length)
	case MsgPort:
		port := binary.BigEndian.Uint16(msg.Payload)
		log.Printf("%s - Received PORT message with port %d\n", peer.address, port)
	default:
		log.Printf("%s - Received unknown message ID %d\n", peer.address, *msg.ID)
	}
}

func Worker(ctx context.Context, pm *common.PieceManager, index uint32, conn net.Conn) {
	var block_size uint32 = 16384
	var pieceOffset uint32 = 0
	requestMessage := RequestMessage(index, pieceOffset, block_size)
	_, err := conn.Write(requestMessage)
	if err != nil {
		log.Println("request writing err", err)
		return
	}
	buf := make([]byte, block_size)
	_, err = conn.Read(buf)
	if err != nil {
		log.Println("block err", err)
		return
	}
	log.Printf("Peer (%s) - Sent block for piece (%d) and block received is: %x", conn.RemoteAddr(), index, buf)
}

// ReadMessage reads a message from a connection
func ReadMessage(conn net.Conn) (Message, error) {
	var length uint32
	err := binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		return Message{}, err
	}

	if length == 0 {
		return Message{ID: nil, Payload: nil}, nil // Keep-alive message
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return Message{}, err
	}

	messageId := MessageID(buf[0])
	return Message{
		ID:      &messageId,
		Payload: buf[1:],
	}, nil
}

// SendMessage sends a message over a connection
func SendMessage(conn net.Conn, msg []byte) error {
	log.Printf("Sending message (%x) to Peer (%s)", msg, conn.RemoteAddr())
	_, err := conn.Write(msg)
	log.Printf("Error (%v) when sending to Peer (%s)", err, conn.RemoteAddr())
	return err
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
