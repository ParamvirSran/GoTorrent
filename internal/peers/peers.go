package peers

import (
	"context"
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

// CreatePeer initializes a new peer object
func CreatePeer(peerID, address string) *Peer {
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

// HandlePeerConnection manages a single peer connection
func HandlePeerConnection(peerID string, infoHash []byte, clientID []byte, peerAddress string) error {
	peer := CreatePeer(peerID, peerAddress)
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
			msg, err := ReadLengthPrefixedMessage(conn)
			if err != nil {
				if err == io.EOF {
					continue
				}
				log.Printf("Error reading message: %v", err)
				once.Do(cancel) // Ensure cancel is only called once
				return err
			}

			// Handle keep-alive
			if msg == nil {
				log.Printf("Peer %s sent keep-alive", peer.address)
				lastKeepAlive = time.Now()
				continue
			}

			message, err := ParseMessage(msg)
			if err != nil {
				log.Println("Failed to parse message:", err)
				continue
			}

			// Reset keep-alive on any valid message
			lastKeepAlive = time.Now()

			// Handle different message types
			switch message.ID {
			case MsgChoke:
				peer.peerState.peerChoking = true
			case MsgUnchoke:
				peer.peerState.peerChoking = false
			case MsgInterested:
				peer.peerState.peerInterested = true
			case MsgNotInterested:
				peer.peerState.peerInterested = false
			case MsgHave:
				log.Println("Peer has these pieces:", message.Payload)
			default:
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
