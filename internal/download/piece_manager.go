package download

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
	downloadedCount int // Keeps track of how many pieces are downloaded
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

// ClaimPiece marks a piece as being downloaded
func (pm *PieceManager) ClaimPiece() (int, []byte, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for index, piece := range pm.pieces {
		if !piece.isDownloaded && !piece.isClaimed {
			piece.isClaimed = true
			return index, piece.Hash, true
		}
	}
	return -1, nil, false
}

// MarkPieceDownloaded marks a piece as downloaded
func (pm *PieceManager) MarkPieceDownloaded(index int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if piece, exists := pm.pieces[index]; exists {
		if !piece.isDownloaded {
			piece.isDownloaded = true
			piece.isClaimed = false
			pm.downloadedCount++ // Increment the counter for downloaded pieces
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
