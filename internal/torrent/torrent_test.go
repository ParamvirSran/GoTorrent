package torrent

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"
)

// TestParseTorrentFile tests the parsing of a torrent file and the computation of its infohash.
func TestParseTorrentFile(t *testing.T) {
	// Path to the test torrent file.
	torrentFilePath := "../../examples/mine.torrent"

	// Expected InfoHash (as bytes).
	expectedInfoHash := []byte{136, 172, 145, 254, 93, 81, 119, 72, 7, 19, 94, 127, 255, 231, 183, 251, 84, 31, 237, 94}

	// Read and parse the torrent file.
	data, err := os.ReadFile(torrentFilePath)
	if err != nil {
		t.Fatalf("Failed to read torrent file: %v", err)
	}

	// Parse the torrent file content.
	torrent, err := ParseTorrentFile(string(data)) // Adjusted to accept []byte.
	if err != nil {
		t.Fatalf("Failed to parse torrent file: %v", err)
	}

	// Compute the infohash.
	infoHash, err := GetInfoHash(torrent.Info)
	if err != nil {
		t.Fatalf("Failed to compute InfoHash: %v", err)
	}

	// Validate the infohash.
	if !bytes.Equal([]byte(infoHash), expectedInfoHash) {
		t.Errorf("InfoHash mismatch. Expected: %s, Got: %s",
			hex.EncodeToString(expectedInfoHash), hex.EncodeToString([]byte(infoHash)))
	}

	// Validate parsed content (e.g., announce URL).
	expectedAnnounce := "http://p4p.arenabg.com:1337/announce"
	if torrent.Announce != expectedAnnounce {
		t.Errorf("Announce URL mismatch. Expected: %s, Got: %s", expectedAnnounce, torrent.Announce)
	}
}
