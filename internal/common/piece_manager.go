package common

import (
	"log"
	"sync"
)

// Piece represents a torrent piece
type Piece struct {
	Hash         []byte
	isDownloaded bool
	isClaimed    bool
}

// PieceManager tracks which pieces are downloaded
type PieceManager struct {
	pieces          map[int]*Piece
	downloadedCount int
	mu              sync.Mutex
}

// NewPieceManager creates a piece manager
func NewPieceManager() *PieceManager {
	return &PieceManager{
		pieces:          make(map[int]*Piece),
		downloadedCount: 0,
	}
}

// AddPiece adds a piece hash
func (pm *PieceManager) AddPiece(index int, hash []byte) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pieces[index] = &Piece{Hash: hash, isDownloaded: false, isClaimed: false}
}

// ClaimPiece marks a piece as claimed by a worker for download
func (pm *PieceManager) ClaimPiece(index int) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if piece, exists := pm.pieces[index]; exists {
		if !piece.isDownloaded && !piece.isClaimed {
			piece.isClaimed = true
			log.Printf("piece %d claimed", index)
			return true
		}
	}

	return false
}

// MarkPieceDownloaded marks a piece as downloaded
func (pm *PieceManager) MarkPieceDownloaded(index int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if piece, exists := pm.pieces[index]; exists {
		if !piece.isDownloaded {
			piece.isDownloaded = true
			piece.isClaimed = false
			pm.downloadedCount++
			log.Printf("Piece %d marked as downloaded", index)
		}
	}
}

// IsDownloadComplete checks if all pieces are downloaded
func (pm *PieceManager) IsDownloadComplete() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.downloadedCount == len(pm.pieces) {
		log.Println("Download complete!")
		return true
	}
	return false
}
