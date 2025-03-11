package torrent

import (
	"log"
	"sync"
)

type Piece struct {
	Hash         []byte
	isDownloaded bool
}

type PieceManager struct {
	m  map[int]Piece
	mu sync.RWMutex
}

func NewPieceMap() *PieceManager {
	return &PieceManager{
		m: make(map[int]Piece),
	}
}

// AddPiece safely adds a piece to the map
func (pm *PieceManager) AddPiece(index int, piece Piece) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.m[index] = piece
	log.Printf("Piece added: %d\n", index)
}

// GetPiece safely retrieves a piece by its index
func (pm *PieceManager) GetPiece(index int) (Piece, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	piece, exists := pm.m[index]
	log.Printf("Piece retrieved: %t", exists)
	return piece, exists
}

// UpdatePiece updates the piece at a specific index, modifying the 'isDownloaded' status
func (pm *PieceManager) UpdatePiece(index int, isDownloaded bool, hash []byte) {
	pm.mu.Lock() // Lock for write access
	defer pm.mu.Unlock()

	// Retrieve the current piece if it exists
	piece, exists := pm.m[index]
	if exists {
		// Update the isDownloaded status of the piece
		piece.isDownloaded = isDownloaded
		pm.m[index] = piece // Update the map with the new piece
	} else {
		// Handle the case where the piece does not exist (optional)
		// Create a new piece if it doesn't exist
		pm.m[index] = Piece{Hash: hash, isDownloaded: isDownloaded}
	}
	log.Printf("Updated Piece index:%d, downloaded:%t, hash:%x", index, isDownloaded, hash)
}

// Check if a piece is downloaded
func (pm *PieceManager) IsPieceDownloaded(index int) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	piece, exists := pm.m[index]
	log.Printf("Piece %d status is: %t", index, (exists && piece.isDownloaded))
	return exists && piece.isDownloaded
}
