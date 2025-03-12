package common

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"log"
	"sync"
)

// Piece represents a torrent piece
type Piece struct {
	Hash         []byte  // the hash for this piece sourced from the .torrent file
	Data         *[]byte // the downloaded data for this piece received from a peer that claims they have the data for this piece
	isDownloaded bool    // flag once piece is downloaded and verified
	isClaimed    bool    // a worker claims the piece and proceeds to download the piece from a peer and no other worker can handle the piece until unclaimed
}

// PieceManager tracks which pieces are downloaded
type PieceManager struct {
	pieces          map[int]*Piece // map of piece_index -> Piece struct
	downloadedCount int            // reference of pieces downloaded and verified
	mu              sync.Mutex     // lock for concurrent access
}

// NewPieceManager creates a piece manager and returns a pointer to it
func NewPieceManager() *PieceManager {
	return &PieceManager{
		pieces:          make(map[int]*Piece),
		downloadedCount: 0,
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
		piece.isDownloaded = false
		piece.isClaimed = false
		piece.Data = nil // Clear the pointer to avoid memory leaks
		pm.downloadedCount--
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
			piece.Data = &data
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
		return errors.New("Piece we are attempting to verify does not exist")
	}

	if !piece.isDownloaded || piece.Data == nil {
		return errors.New("Piece we are attempting to verify is not downloaded/data is nil")
	}

	hasher := sha1.New()
	hasher.Write(*piece.Data)
	computedHash := hasher.Sum(nil)

	if !bytes.Equal(computedHash, piece.Hash) {
		return errors.New("Piece we are verifying does not match the expected hash from .torrent file")
	}

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
	return false
}

// GetPieceData will return the data stored for the index of a piece
func (pm *PieceManager) GetPieceData(index int) ([]byte, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	piece, exists := pm.pieces[index]
	if !exists {
		return nil, errors.New("piece does not exist for which we are trying to get the data for")
	}

	if !piece.isDownloaded || piece.Data == nil {
		return nil, errors.New("piece not downloaded when trying to retreive its data")
	}

	return *piece.Data, nil
}
