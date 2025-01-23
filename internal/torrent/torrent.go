package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/ParamvirSran/GoTorrent/internal/bencode"
)

const (
	MinPieceLength = 16 * 1024   // 16 KB
	MaxPieceLength = 1024 * 1024 // 1 MB

	// Keys for the dictionary fields
	KeyAnnounce     = "announce"
	KeyAnnounceList = "announce-list"
	KeyInfo         = "info"
	KeyPieceLength  = "piece length"
	KeyPieces       = "pieces"
	KeyLength       = "length"
	KeyFiles        = "files"
	KeyName         = "name"
	KeyComment      = "comment"
	KeyCreatedBy    = "created by"
	KeyEncoding     = "encoding"
	KeyPrivate      = "private"
)

type Torrent struct {
	Announce     string
	AnnounceList [][]string
	CreationDate int64
	Comment      string
	CreatedBy    string
	Encoding     string
	Info         InfoDictionary
}

type InfoDictionary struct {
	PieceLength int
	Pieces      []byte // Pieces is a []byte, not a string
	Private     *int
	Name        string
	Length      int64
	Files       []File
}

type File struct {
	Length int64
	Md5sum *string
	Path   []string
}

// ParseTorrentFile parses the .torrent file and returns the parsed TorrentFile object
func ParseTorrentFile(torrentPath string) (*Torrent, error) {
	content, err := os.ReadFile(torrentPath)
	if err != nil {
		return nil, fmt.Errorf("error reading .torrent file: %v", err)
	}

	reader := bytes.NewReader(content)
	data, err := bencode.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %w", err)
	}

	torrentDict, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid torrent file format: expected a dictionary but got %T", data)
	}

	parsedFile, err := parseMetainfo(torrentDict)
	if err != nil {
		return nil, fmt.Errorf("error parsing metainfo: %v", err)
	}

	return parsedFile, nil
}

// parseMetainfo parses the metainfo dictionary
func parseMetainfo(torrentDict map[string]interface{}) (*Torrent, error) {
	torrentFile := &Torrent{}
	if err := parseAnnounce(torrentDict, torrentFile); err != nil {
		return nil, err
	}

	infoDict, ok := torrentDict[KeyInfo].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("info dictionary missing or of incorrect type")
	}
	info, err := parseInfo(infoDict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse info dictionary: %w", err)
	}
	torrentFile.Info = *info

	// Optional Fields
	if comment, ok := torrentDict[KeyComment].(string); ok {
		torrentFile.Comment = comment
	}

	if createdBy, ok := torrentDict[KeyCreatedBy].(string); ok {
		torrentFile.CreatedBy = createdBy
	}

	if encoding, ok := torrentDict[KeyEncoding].(string); ok {
		torrentFile.Encoding = encoding
	}

	if private, ok := torrentDict[KeyPrivate].(int); ok {
		torrentFile.Info.Private = &private
	}

	if creationDate, ok := torrentDict["creation date"].(int64); ok {
		torrentFile.CreationDate = creationDate
	}

	return torrentFile, nil
}

// parseAnnounce parses the "announce" and "announce-list" fields
func parseAnnounce(torrentDict map[string]interface{}, torrentFile *Torrent) error {
	if announce, ok := torrentDict[KeyAnnounce].(string); ok {
		torrentFile.Announce = announce
	} else {
		return fmt.Errorf("%s URL missing or not a string", KeyAnnounce)
	}

	if announceList, ok := torrentDict[KeyAnnounceList].([]interface{}); ok {
		for _, tier := range announceList {
			if urlList, ok := tier.([]interface{}); ok {
				var urls []string
				for _, url := range urlList {
					if urlString, ok := url.(string); ok {
						urls = append(urls, urlString)
					} else {
						return fmt.Errorf("%s contains non-string URL: %v", KeyAnnounceList, url)
					}
				}
				torrentFile.AnnounceList = append(torrentFile.AnnounceList, urls)
			} else {
				return fmt.Errorf("%s tier is not a list of URLs: %T", KeyAnnounceList, tier)
			}
		}
	}
	return nil
}

// parseInfo parses the info dictionary from the torrent file
func parseInfo(infoDict map[string]interface{}) (*InfoDictionary, error) {
	info := &InfoDictionary{}
	if err := parseNameAndPieceLength(infoDict, info); err != nil {
		return nil, err
	}

	if err := ValidatePieceLength(info.PieceLength); err != nil {
		return nil, err
	}

	if err := parsePiecesField(infoDict, info); err != nil {
		return nil, err
	}

	if err := parseLengthOrFiles(infoDict, info); err != nil {
		return nil, err
	}

	return info, nil
}

// parseNameAndPieceLength parses the name and piece length from the info dictionary
func parseNameAndPieceLength(infoDict map[string]interface{}, info *InfoDictionary) error {
	name, ok := infoDict[KeyName].(string)
	if !ok {
		return fmt.Errorf("%s field missing or not a string", KeyName)
	}
	info.Name = name

	switch pl := infoDict[KeyPieceLength].(type) {
	case int:
		info.PieceLength = pl
	case int64:
		info.PieceLength = int(pl)
	default:
		return fmt.Errorf("%s field missing or of incorrect type", KeyPieceLength)
	}
	return nil
}

// parsePiecesField parses the "pieces" field
func parsePiecesField(infoDict map[string]interface{}, info *InfoDictionary) error {
	pieces, ok := infoDict[KeyPieces].(string)
	if !ok {
		return fmt.Errorf("%s field missing or not a string", KeyPieces)
	}
	info.Pieces = []byte(pieces) // Convert string to []byte
	return nil
}

// parseLengthOrFiles parses the "length" or "files" field
func parseLengthOrFiles(infoDict map[string]interface{}, info *InfoDictionary) error {
	if length, ok := infoDict[KeyLength].(int); ok {
		info.Length = int64(length)
	} else if length, ok := infoDict[KeyLength].(int64); ok {
		info.Length = length
	} else if files, ok := infoDict[KeyFiles].([]interface{}); ok {
		parsedFiles, err := parseFiles(files)
		if err != nil {
			return fmt.Errorf("failed to parse files: %w", err)
		}
		info.Files = parsedFiles
	} else {
		return fmt.Errorf("neither %s nor %s field found. Torrent may be missing fields", KeyLength, KeyFiles)
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

	if length, ok := fileDict[KeyLength].(int); ok {
		fileInfo.Length = int64(length)
	} else if length, ok := fileDict[KeyLength].(int64); ok {
		fileInfo.Length = length
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

func (info *InfoDictionary) SplitPieces() [][]byte {
	var pieces [][]byte
	for i := 0; i < len(info.Pieces); i += sha1.Size {
		pieces = append(pieces, info.Pieces[i:i+sha1.Size])
	}
	return pieces
}
