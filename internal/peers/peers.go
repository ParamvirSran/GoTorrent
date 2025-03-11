package peers

import (
	"context"
	"encoding/binary"
	"fmt"
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

// Worker manages a single peer connection
func Worker(ctx context.Context, peerID string, infoHash []byte, clientID []byte, peerAddress string) error {
	peer := createPeer(peerID, peerAddress)

	handshake, err := CreateHandshake(infoHash, clientID)
	if err != nil {
		return fmt.Errorf("error creating handshake: %v", err)
	}

	var d net.Dialer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", peer.address)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", peer.address, err)
	}
	defer conn.Close()

	// Send the handshake
	if _, err := conn.Write(handshake); err != nil {
		return fmt.Errorf("error writing handshake: %v", err)
	}

	// Read handshake response
	response := make([]byte, 68)
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %v", err)
	}

	if err := ValidateHandshakeResponse(peer, response[:n], [20]byte(infoHash)); err != nil {
		return fmt.Errorf("invalid handshake response: %v", err)
	}

	// Last received keep-alive timestamp
	lastKeepAlive := time.Now()

	// Goroutine cleanup to prevent leaks
	var once sync.Once
	wg := sync.WaitGroup{}
	wg.Add(1)

	// Keep-alive mechanism
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(30 * time.Second) // Send keep-alive every 30 sec
				if _, err := conn.Write([]byte{0}); err != nil {
					log.Println("Error sending keep-alive:", err)
					once.Do(cancel) // Ensure cancel is called only once
					return
				}
			}
		}
	}()

	// Message processing loop
	for {
		select {
		case <-ctx.Done():
			log.Printf("Disconnecting from peer: %s", peer.address)
			wg.Wait() // Ensure keep-alive goroutine exits
			return nil

		default:
			conn.SetReadDeadline(time.Now().Add(120 * time.Second))
			msg, err := ReadMessage(conn)
			if err != nil {
				if err == io.EOF {
					continue
				}
				log.Printf("Error reading message: %v", err)
				once.Do(cancel) // Ensure cancel is only called once
				return err
			}

			if msg.ID == nil {
				continue
			}

			switch *msg.ID {
			case MsgChoke:
				log.Printf("%s - Received CHOKE message", peer.address)
				peer.peerState.peerChoking = true
			case MsgUnchoke:
				log.Printf("%s - Received UNCHOKE message", peer.address)
				peer.peerState.peerChoking = false
			case MsgInterested:
				log.Printf("%s - Received INTERESTED message", peer.address)
				peer.peerState.peerInterested = true
			case MsgNotInterested:
				log.Printf("%s - Received NOT-INTERESTED message", peer.address)
				peer.peerState.peerInterested = false
			case MsgHave:
				pieceIndex := binary.BigEndian.Uint32(msg.Payload)
				log.Printf("%s - Received HAVE message for piece %d\n", peer.address, pieceIndex)
			case MsgBitfield:
				fmt.Printf("%s - Received BITFIELD message: %x", peer.address, msg.Payload)
			case MsgRequest:
				index := binary.BigEndian.Uint32(msg.Payload[0:4])
				begin := binary.BigEndian.Uint32(msg.Payload[4:8])
				length := binary.BigEndian.Uint32(msg.Payload[8:12])
				fmt.Printf("%s - Received REQUEST message for index %d, begin %d, length %d\n", peer.address, index, begin, length)
			case MsgPiece:
				index := binary.BigEndian.Uint32(msg.Payload[0:4])
				begin := binary.BigEndian.Uint32(msg.Payload[4:8])
				block := msg.Payload[8:]
				fmt.Printf("%s - Received PIECE message for index %d, begin %d, block length %d\n", peer.address, index, begin, len(block))
			case MsgCancel:
				index := binary.BigEndian.Uint32(msg.Payload[0:4])
				begin := binary.BigEndian.Uint32(msg.Payload[4:8])
				length := binary.BigEndian.Uint32(msg.Payload[8:12])
				fmt.Printf("%s - Received CANCEL message for index %d, begin %d, length %d\n", peer.address, index, begin, length)
			case MsgPort:
				port := binary.BigEndian.Uint16(msg.Payload)
				fmt.Printf("%s - Received PORT message with port %d\n", peer.address, port)
			default:
				fmt.Printf("%s - Received unknown message ID %d\n", peer.address, *msg.ID)
			}

			// Disconnect peer if no keep-alive received in 2 mins
			if time.Since(lastKeepAlive) > 120*time.Second {
				log.Printf("Peer %s timed out", peer.address)
				once.Do(cancel)
				return nil
			}
		}
	}
}

// ReadMessage reads a message from a connection
func ReadMessage(conn net.Conn) (Message, error) {
	// Read the 4-byte length prefix
	var length uint32
	err := binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		return Message{}, err
	}

	if length == 0 {
		return Message{ID: nil, Payload: nil}, nil // Keep-alive message
	}

	// Read remaining bytes
	buf := make([]byte, length)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return Message{}, err
	}

	return ParseMessage(buf)
}

// SendMessage sends a message over a connection
func SendMessage(conn net.Conn, msg []byte) error {
	_, err := conn.Write(msg)
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
