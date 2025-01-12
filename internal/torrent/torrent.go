package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/ParamvirSran/GoTorrent/internal/bencode"
)

type TorrentFile struct {
	Announce     string
	AnnounceList [][]string
	Info         Info
	Pieces       []Piece
}

type Info struct {
	Name        string
	PieceLength int
	Pieces      string
	Files       []File
	Length      int
}

type File struct {
	Length int
	Path   []string
}

type Piece struct {
	Hash       []byte
	Downloaded bool
}

const (
	MinPieceLength = 16 * 1024   // 16 KB
	MaxPieceLength = 1024 * 1024 // 1 MB
)

// ParseTorrentFile parses the .torrent file and returns the parsed TorrentFile object
func ParseTorrentFile(torrentPath string) (*TorrentFile, error) {
	// Read the .torrent file content
	content, err := os.ReadFile(torrentPath)
	if err != nil {
		return nil, fmt.Errorf("error reading .torrent file: %v", err)
	}

	// Wrap the []byte content in a bytes.Reader to satisfy the io.Reader interface
	reader := bytes.NewReader(content)

	// Decode the .torrent file using the bencode package
	data, err := bencode.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %w", err)
	}

	// Assert that the decoded data is a map (torrent dictionary)
	torrentDict, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid torrent file format: expected a dictionary but got %T", data)
	}

	// Parse the metainfo from the decoded dictionary
	parsedFile, err := parseMetainfo(torrentDict)
	if err != nil {
		return nil, fmt.Errorf("error parsing metainfo: %v", err)
	}

	// Initialize pieces array
	parsedFile.Pieces, err = parsePieces(parsedFile.Info.Pieces)
	if err != nil {
		return nil, err
	}
	return parsedFile, nil
}

// parsePieces converts the 'pieces' string into individual pieces
func parsePieces(piecesStr string) ([]Piece, error) {
	if len(piecesStr)%20 != 0 {
		return nil, fmt.Errorf("invalid pieces string length: %d (must be a multiple of 20)", len(piecesStr))
	}
	numPieces := len(piecesStr) / 20
	pieces := make([]Piece, numPieces)
	for i := 0; i < numPieces; i++ {
		pieces[i] = Piece{Hash: []byte(piecesStr[i*20 : (i+1)*20])}
	}
	return pieces, nil
}

// parseMetainfo parses the metainfo dictionary
func parseMetainfo(torrentDict map[string]interface{}) (*TorrentFile, error) {
	torrentFile := &TorrentFile{}

	// Parse announce info
	if err := parseAnnounce(torrentDict, torrentFile); err != nil {
		return nil, err
	}

	// Parse info dictionary
	infoDict, ok := torrentDict["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("info dictionary missing or of incorrect type")
	}
	info, err := parseInfo(infoDict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse info dictionary: %w", err)
	}
	torrentFile.Info = *info

	return torrentFile, nil
}

// parseAnnounce parses the "announce" and "announce-list" fields
func parseAnnounce(torrentDict map[string]interface{}, torrentFile *TorrentFile) error {
	if announce, ok := torrentDict["announce"].(string); ok {
		torrentFile.Announce = announce
	} else {
		return fmt.Errorf("announce URL missing or not a string")
	}

	if announceList, ok := torrentDict["announce-list"].([]interface{}); ok {
		for _, tier := range announceList {
			if urlList, ok := tier.([]interface{}); ok {
				var urls []string
				for _, url := range urlList {
					if urlString, ok := url.(string); ok {
						urls = append(urls, urlString)
					} else {
						return fmt.Errorf("announce-list contains non-string URL: %v", url)
					}
				}
				torrentFile.AnnounceList = append(torrentFile.AnnounceList, urls)
			} else {
				return fmt.Errorf("announce-list tier is not a list of URLs: %T", tier)
			}
		}
	}
	return nil
}

// parseInfo parses the info dictionary from the torrent file
func parseInfo(infoDict map[string]interface{}) (*Info, error) {
	info := &Info{}

	// Parse name and piece length
	if err := parseNameAndPieceLength(infoDict, info); err != nil {
		return nil, err
	}

	if err := ValidatePieceLength(info.PieceLength); err != nil {
		return nil, err
	}

	// Parse pieces field
	if err := parsePiecesField(infoDict, info); err != nil {
		return nil, err
	}

	// Parse length or files field
	if err := parseLengthOrFiles(infoDict, info); err != nil {
		return nil, err
	}

	return info, nil
}

// parseNameAndPieceLength parses the name and piece length from the info dictionary
func parseNameAndPieceLength(infoDict map[string]interface{}, info *Info) error {
	name, ok := infoDict["name"].(string)
	if !ok {
		return fmt.Errorf("name field missing or not a string")
	}
	info.Name = name

	switch pl := infoDict["piece length"].(type) {
	case int:
		info.PieceLength = pl
	case int64:
		info.PieceLength = int(pl)
	default:
		return fmt.Errorf("piece length field missing or of incorrect type")
	}
	return nil
}

// parsePiecesField parses the "pieces" field
func parsePiecesField(infoDict map[string]interface{}, info *Info) error {
	pieces, ok := infoDict["pieces"].(string)
	if !ok {
		return fmt.Errorf("pieces field missing or not a string")
	}
	info.Pieces = pieces
	return nil
}

// parseLengthOrFiles parses the "length" or "files" field
func parseLengthOrFiles(infoDict map[string]interface{}, info *Info) error {
	if length, ok := infoDict["length"].(int); ok {
		info.Length = length
	} else if length, ok := infoDict["length"].(int64); ok {
		info.Length = int(length)
	} else if files, ok := infoDict["files"].([]interface{}); ok {
		parsedFiles, err := parseFiles(files)
		if err != nil {
			return fmt.Errorf("failed to parse files: %w", err)
		}
		info.Files = parsedFiles
	} else {
		return fmt.Errorf("neither length nor files field found")
	}
	return nil
}

// parseFiles parses the "files" field into a list of File objects
func parseFiles(files []interface{}) ([]File, error) {
	var fileList []File
	for _, file := range files {
		fileDict, ok := file.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("file entry is not a valid dictionary, got %T", file)
		}
		fileInfo, err := parseFile(fileDict)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file: %w", err)
		}
		fileList = append(fileList, fileInfo)
	}
	return fileList, nil
}

// parseFile parses a single file entry
func parseFile(fileDict map[string]interface{}) (File, error) {
	var fileInfo File

	if length, ok := fileDict["length"].(int); ok {
		fileInfo.Length = length
	} else if length, ok := fileDict["length"].(int64); ok {
		fileInfo.Length = int(length)
	} else {
		return fileInfo, fmt.Errorf("file length missing or not a valid type")
	}

	if path, ok := fileDict["path"].([]interface{}); ok {
		for _, p := range path {
			if pStr, ok := p.(string); ok {
				fileInfo.Path = append(fileInfo.Path, pStr)
			} else {
				return fileInfo, fmt.Errorf("path element is not a valid type, got %T", p)
			}
		}
	} else {
		return fileInfo, fmt.Errorf("path missing or of incorrect type")
	}

	return fileInfo, nil
}

// VerifyPiece verifies the downloaded piece against its expected hash
func VerifyPiece(piece []byte, expectedHash []byte) (bool, error) {
	if len(expectedHash) != sha1.Size {
		return false, fmt.Errorf("invalid hash length: expected %d bytes, got %d", sha1.Size, len(expectedHash))
	}
	hash := sha1.Sum(piece)
	return bytes.Equal(hash[:], expectedHash), nil
}

// ValidatePieceLength checks if the piece length is within valid bounds
func ValidatePieceLength(pieceLength int) error {
	if pieceLength < MinPieceLength || pieceLength > MaxPieceLength {
		return fmt.Errorf("invalid piece length: %d (must be between %d and %d)", pieceLength, MinPieceLength, MaxPieceLength)
	}
	return nil
}
