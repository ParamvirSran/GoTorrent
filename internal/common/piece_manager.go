package common

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"sync"
)

// Piece represents a torrent piece
type Piece struct {
	Hash           []byte  // the hash for this piece sourced from the .torrent file
	Data           *[]byte // the downloaded data for this piece received from a peer that claims they have the data for this piece
	isDownloaded   bool    // flag once piece is downloaded and verified
	isClaimed      bool    // a worker claims the piece and proceeds to download the piece from a peer and no other worker can handle the piece until unclaimed
	failedAttempts int     // track failed verification attempts
}

// PieceManager tracks which pieces are downloaded
type PieceManager struct {
	pieces          map[int]*Piece // map of piece_index -> Piece struct
	downloadedCount int            // reference of pieces downloaded and verified
	PieceCount      int
	PieceSize       int
	mu              sync.Mutex // lock for concurrent access
}

// NewPieceManager creates a piece manager and returns a pointer to it
func NewPieceManager(pieceCount, pieceLength int) *PieceManager {
	return &PieceManager{
		pieces:          make(map[int]*Piece),
		downloadedCount: 0,
		PieceCount:      pieceCount,
		PieceSize:       pieceLength,
	}
}

// AddPiece adds a piece to the piece manager
func (pm *PieceManager) AddPiece(index int, hash []byte) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pieces[index] = &Piece{Hash: hash, isDownloaded: false, isClaimed: false, Data: nil}
}

// RequeuePiece will requeue a piece at a index
func (pm *PieceManager) RequeuePiece(index int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if piece, exists := pm.pieces[index]; exists {
		if piece.isDownloaded {
			pm.downloadedCount--
		}
		piece.isDownloaded = false
		piece.isClaimed = false
		piece.Data = nil // Clear the pointer to avoid memory leaks
		log.Printf("Piece %d re-queued for download", index)
	}
}

// ClaimPiece marks a piece as claimed by a worker for download
func (pm *PieceManager) ClaimPiece(index int) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if piece, exists := pm.pieces[index]; exists {
		if !piece.isDownloaded && !piece.isClaimed {
			piece.isClaimed = true
			log.Printf("Piece %d claimed", index)
			return true
		}
	}

	return false
}

// MarkPieceDownloaded marks a piece as downloaded
func (pm *PieceManager) MarkPieceDownloaded(index int, data []byte) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if piece, exists := pm.pieces[index]; exists {
		if !piece.isDownloaded {
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)
			piece.Data = &dataCopy
			piece.isDownloaded = true
			piece.isClaimed = false
			pm.downloadedCount++
			log.Printf("Piece %d marked as downloaded", index)
		}
	}
}

// VerifyPiece will take the index provided and verify that the piece is downloaded and the hash matches the .torrent file specified hash
func (pm *PieceManager) VerifyPiece(index int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	piece, exists := pm.pieces[index]
	if !exists {
		return fmt.Errorf("piece %d does not exist", index)
	}

	if !piece.isDownloaded || piece.Data == nil {
		return fmt.Errorf("piece %d is not downloaded or data is nil", index)
	}

	hasher := sha1.New()
	hasher.Write(*piece.Data)
	computedHash := hasher.Sum(nil)

	if !bytes.Equal(computedHash, piece.Hash) {
		piece.failedAttempts++
		if piece.failedAttempts >= 3 { // Threshold for failed attempts
			pm.RequeuePiece(index)
			return fmt.Errorf("piece %d failed verification %d times, re-queued", index, piece.failedAttempts)
		}
		return fmt.Errorf("piece %d hash does not match the expected hash", index)
	}

	piece.failedAttempts = 0 // Reset failed attempts on successful verification
	return nil
}

// IsDownloadComplete checks if all pieces are downloaded
func (pm *PieceManager) IsDownloadComplete() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.downloadedCount == len(pm.pieces) {
		log.Println("Download complete!")
		return true
	}

	log.Printf("Downloaded Pieces: %d\nTotal Pieces: %d", pm.downloadedCount, len(pm.pieces))
	log.Printf("Left: %d", len(pm.pieces)-pm.downloadedCount)
	return false
}

// GetPieceData will return the data stored for the index of a piece
func (pm *PieceManager) GetPieceData(index int) ([]byte, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	piece, exists := pm.pieces[index]
	if !exists {
		return nil, fmt.Errorf("piece %d does not exist", index)
	}

	if !piece.isDownloaded || piece.Data == nil {
		return nil, fmt.Errorf("piece %d is not downloaded or data is nil", index)
	}

	return *piece.Data, nil
}
